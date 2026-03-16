package farm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/examples/farm/internal/svc"
	"github.com/zeromicro/go-zero/examples/farm/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// maxPlots is the number of farm plots per user.
	maxPlots = 5

	// cropGrowSeconds is how long each crop takes to be ready for harvest.
	cropGrowSeconds = 60
)

// farmKey returns the Redis hash key that stores all plot states for a user.
func farmKey(userId string) string {
	return fmt.Sprintf("farm:%s", userId)
}

// plotField returns the Redis hash field name for one plot.
func plotField(plotId int) string {
	return fmt.Sprintf("plot:%d", plotId)
}

// PlantLogic handles planting a crop on a farm plot.
type PlantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewPlantLogic creates a new PlantLogic.
func NewPlantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PlantLogic {
	return &PlantLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// Plant plants a crop on the specified plot.
func (l *PlantLogic) Plant(userId string, req *types.PlantRequest) (*types.PlantResponse, error) {
	if req.PlotId < 0 || req.PlotId >= maxPlots {
		return nil, fmt.Errorf("invalid plotId: must be between 0 and %d", maxPlots-1)
	}
	if req.CropType == "" {
		return nil, errors.New("cropType is required")
	}

	// Check if the plot is already occupied.
	existing, err := getPlot(l.ctx, l.svcCtx, userId, req.PlotId)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.CropType != "" {
		return nil, fmt.Errorf("plot %d already has a crop; harvest it first", req.PlotId)
	}

	now := time.Now().Unix()
	plot := types.PlotState{
		PlotId:    req.PlotId,
		CropType:  req.CropType,
		PlantedAt: now,
		HarvestAt: now + cropGrowSeconds,
	}

	if err := savePlot(l.ctx, l.svcCtx, userId, plot); err != nil {
		return nil, err
	}

	return &types.PlantResponse{
		PlotId:    plot.PlotId,
		CropType:  plot.CropType,
		PlantedAt: plot.PlantedAt,
		HarvestAt: plot.HarvestAt,
	}, nil
}

// WaterLogic handles watering a crop.
type WaterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewWaterLogic creates a new WaterLogic.
func NewWaterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WaterLogic {
	return &WaterLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// Water marks the given plot as watered.
func (l *WaterLogic) Water(userId string, req *types.WaterRequest) (*types.WaterResponse, error) {
	plot, err := getPlot(l.ctx, l.svcCtx, userId, req.PlotId)
	if err != nil {
		return nil, err
	}
	if plot == nil || plot.CropType == "" {
		return nil, fmt.Errorf("no crop planted in plot %d", req.PlotId)
	}
	if plot.Watered {
		return nil, fmt.Errorf("plot %d is already watered", req.PlotId)
	}

	plot.Watered = true
	if err := savePlot(l.ctx, l.svcCtx, userId, *plot); err != nil {
		return nil, err
	}

	return &types.WaterResponse{
		PlotId:    req.PlotId,
		WateredAt: time.Now().Unix(),
	}, nil
}

// HarvestLogic handles harvesting a crop.
type HarvestLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewHarvestLogic creates a new HarvestLogic.
func NewHarvestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HarvestLogic {
	return &HarvestLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// Harvest harvests the crop from the given plot.
func (l *HarvestLogic) Harvest(userId string, req *types.HarvestRequest) (*types.HarvestResponse, error) {
	plot, err := getPlot(l.ctx, l.svcCtx, userId, req.PlotId)
	if err != nil {
		return nil, err
	}
	if plot == nil || plot.CropType == "" {
		return nil, fmt.Errorf("no crop planted in plot %d", req.PlotId)
	}
	if time.Now().Unix() < plot.HarvestAt {
		return nil, fmt.Errorf("crop in plot %d is not ready yet", req.PlotId)
	}

	cropType := plot.CropType
	yield := 10
	if plot.Watered {
		yield = 15 // Bonus yield for watered crops.
	}

	// Clear the plot after harvesting.
	cleared := types.PlotState{PlotId: req.PlotId}
	if err := savePlot(l.ctx, l.svcCtx, userId, cleared); err != nil {
		return nil, err
	}

	return &types.HarvestResponse{
		PlotId:   req.PlotId,
		CropType: cropType,
		Yield:    yield,
	}, nil
}

// StatusLogic returns the current state of all farm plots.
type StatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewStatusLogic creates a new StatusLogic.
func NewStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StatusLogic {
	return &StatusLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// Status returns the state of all farm plots for a user.
func (l *StatusLogic) Status(userId string) (*types.StatusResponse, error) {
	plots := make([]types.PlotState, maxPlots)
	now := time.Now().Unix()

	for i := range maxPlots {
		plot, err := getPlot(l.ctx, l.svcCtx, userId, i)
		if err != nil {
			return nil, err
		}
		if plot != nil {
			plot.Ready = plot.CropType != "" && now >= plot.HarvestAt
			plots[i] = *plot
		} else {
			plots[i] = types.PlotState{PlotId: i}
		}
	}

	return &types.StatusResponse{
		UserId: userId,
		Plots:  plots,
	}, nil
}

// getPlot loads a single plot state from Redis.
func getPlot(ctx context.Context, svcCtx *svc.ServiceContext, userId string, plotId int) (*types.PlotState, error) {
	data, err := svcCtx.Redis.HgetCtx(ctx, farmKey(userId), plotField(plotId))
	if err != nil || data == "" {
		return nil, err
	}

	var plot types.PlotState
	if err := json.Unmarshal([]byte(data), &plot); err != nil {
		return nil, fmt.Errorf("failed to decode plot state: %w", err)
	}

	return &plot, nil
}

// savePlot persists a single plot state to Redis.
func savePlot(ctx context.Context, svcCtx *svc.ServiceContext, userId string, plot types.PlotState) error {
	data, err := json.Marshal(plot)
	if err != nil {
		return fmt.Errorf("failed to encode plot state: %w", err)
	}

	return svcCtx.Redis.HsetCtx(ctx, farmKey(userId), plotField(plot.PlotId), string(data))
}
