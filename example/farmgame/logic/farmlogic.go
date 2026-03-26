// Package logic contains the business logic for the farm game, backed by
// Redis for fast, in-memory game state storage.
package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/example/farmgame/types"
)

const (
	// farmKey builds the Redis key that stores a player's farm JSON blob.
	farmKeyPrefix = "farm:"

	// Number of plots every new farm starts with.
	defaultPlotCount = 6

	// Starting coins for a new player.
	startingCoins = 100
)

// Catalog is the authoritative list of crop types and their attributes.
var Catalog = []types.CropInfo{
	{Kind: types.CropWheat, Name: "Wheat", GrowSeconds: 30},
	{Kind: types.CropCorn, Name: "Corn", GrowSeconds: 60},
	{Kind: types.CropCarrot, Name: "Carrot", GrowSeconds: 45},
}

// coinReward returns the number of coins awarded when harvesting a crop.
var coinReward = map[types.CropKind]int64{
	types.CropWheat:  10,
	types.CropCorn:   20,
	types.CropCarrot: 15,
}

// growSeconds looks up how long a crop takes to mature.
func growSeconds(kind types.CropKind) (int64, bool) {
	for _, c := range Catalog {
		if c.Kind == kind {
			return c.GrowSeconds, true
		}
	}
	return 0, false
}

// FarmLogic provides all game operations for managing a player's farm.
type FarmLogic struct {
	rds   *redis.Redis
	nowFn func() time.Time // injectable for tests; defaults to time.Now
}

// NewFarmLogic creates a FarmLogic that stores state in the given Redis
// instance.
func NewFarmLogic(rds *redis.Redis) *FarmLogic {
	return &FarmLogic{rds: rds, nowFn: time.Now}
}

// WithNow returns a copy of FarmLogic that uses the provided clock function.
// Intended for testing only.
func (l *FarmLogic) WithNow(fn func() time.Time) *FarmLogic {
	return &FarmLogic{rds: l.rds, nowFn: fn}
}

// farmKey returns the Redis key for a player's farm.
func farmKey(playerID string) string {
	return farmKeyPrefix + playerID
}

// loadFarm reads a player's farm from Redis; creates a fresh farm on first
// access.
func (l *FarmLogic) loadFarm(ctx context.Context, playerID string) (types.Farm, error) {
	key := farmKey(playerID)
	raw, err := l.rds.GetCtx(ctx, key)
	if err != nil {
		return types.Farm{}, fmt.Errorf("redis get: %w", err)
	}
	if raw == "" {
		return l.initFarm(ctx, playerID)
	}

	var farm types.Farm
	if err := json.Unmarshal([]byte(raw), &farm); err != nil {
		return types.Farm{}, fmt.Errorf("unmarshal farm: %w", err)
	}
	return farm, nil
}

// initFarm creates and persists a brand-new farm for playerID.
func (l *FarmLogic) initFarm(ctx context.Context, playerID string) (types.Farm, error) {
	plots := make([]types.Plot, defaultPlotCount)
	for i := range plots {
		plots[i] = types.Plot{ID: i, State: types.PlotEmpty}
	}
	farm := types.Farm{
		PlayerID: playerID,
		Coins:    startingCoins,
		Plots:    plots,
	}
	return farm, l.saveFarm(ctx, farm)
}

// saveFarm persists a farm to Redis.
func (l *FarmLogic) saveFarm(ctx context.Context, farm types.Farm) error {
	data, err := json.Marshal(farm)
	if err != nil {
		return fmt.Errorf("marshal farm: %w", err)
	}
	return l.rds.SetCtx(ctx, farmKey(farm.PlayerID), string(data))
}

// refreshPlots updates any growing plots that have matured since the last
// load, so the client always sees up-to-date states.
func refreshPlots(farm *types.Farm, now int64) {
	for i := range farm.Plots {
		p := &farm.Plots[i]
		if p.State == types.PlotGrowing && now >= p.ReadyAt {
			p.State = types.PlotReady
		}
	}
}

// GetFarm returns the current state of a player's farm, refreshing any
// crop-growth timers first.
func (l *FarmLogic) GetFarm(ctx context.Context, playerID string) (types.Farm, error) {
	farm, err := l.loadFarm(ctx, playerID)
	if err != nil {
		return types.Farm{}, err
	}
	refreshPlots(&farm, l.nowFn().Unix())
	if err := l.saveFarm(ctx, farm); err != nil {
		return types.Farm{}, err
	}
	return farm, nil
}

// Plant places a crop on an empty plot and starts the growth timer.
func (l *FarmLogic) Plant(ctx context.Context, playerID string, plotID int, crop types.CropKind) (types.Plot, error) {
	grow, ok := growSeconds(crop)
	if !ok {
		return types.Plot{}, fmt.Errorf("unknown crop kind %q", crop)
	}

	farm, err := l.loadFarm(ctx, playerID)
	if err != nil {
		return types.Plot{}, err
	}
	now := l.nowFn().Unix()
	refreshPlots(&farm, now)

	if plotID < 0 || plotID >= len(farm.Plots) {
		return types.Plot{}, fmt.Errorf("plot %d does not exist", plotID)
	}
	plot := &farm.Plots[plotID]
	if plot.State != types.PlotEmpty {
		return types.Plot{}, fmt.Errorf("plot %d is not empty (state: %s)", plotID, plot.State)
	}

	plot.State = types.PlotGrowing
	plot.Crop = crop
	plot.PlantedAt = now
	plot.ReadyAt = now + grow

	if err := l.saveFarm(ctx, farm); err != nil {
		return types.Plot{}, err
	}
	return *plot, nil
}

// Water reduces a crop's remaining grow time by 20 % (minimum 1 second
// remaining after watering).
func (l *FarmLogic) Water(ctx context.Context, playerID string, plotID int) (types.Plot, error) {
	farm, err := l.loadFarm(ctx, playerID)
	if err != nil {
		return types.Plot{}, err
	}
	now := l.nowFn().Unix()
	refreshPlots(&farm, now)

	if plotID < 0 || plotID >= len(farm.Plots) {
		return types.Plot{}, fmt.Errorf("plot %d does not exist", plotID)
	}
	plot := &farm.Plots[plotID]
	if plot.State != types.PlotGrowing {
		return types.Plot{}, fmt.Errorf("plot %d is not growing (state: %s)", plotID, plot.State)
	}

	remaining := plot.ReadyAt - now
	if remaining > 1 {
		reduction := remaining / 5 // cut 20 %
		if reduction < 1 {
			reduction = 1
		}
		plot.ReadyAt -= reduction
	}

	if err := l.saveFarm(ctx, farm); err != nil {
		return types.Plot{}, err
	}
	return *plot, nil
}

// Harvest collects a ready crop, awards coins, and resets the plot to empty.
func (l *FarmLogic) Harvest(ctx context.Context, playerID string, plotID int) (types.Plot, int64, error) {
	farm, err := l.loadFarm(ctx, playerID)
	if err != nil {
		return types.Plot{}, 0, err
	}
	refreshPlots(&farm, l.nowFn().Unix())

	if plotID < 0 || plotID >= len(farm.Plots) {
		return types.Plot{}, 0, fmt.Errorf("plot %d does not exist", plotID)
	}
	plot := &farm.Plots[plotID]
	if plot.State != types.PlotReady {
		return types.Plot{}, 0, fmt.Errorf("plot %d is not ready for harvest (state: %s)", plotID, plot.State)
	}

	earned := coinReward[plot.Crop]
	farm.Coins += earned

	plot.State = types.PlotEmpty
	plot.Crop = ""
	plot.PlantedAt = 0
	plot.ReadyAt = 0

	if err := l.saveFarm(ctx, farm); err != nil {
		return types.Plot{}, 0, err
	}
	return *plot, earned, nil
}

// ListCrops returns all available crop types from the catalog.
func ListCrops() []types.CropInfo {
	return Catalog
}
