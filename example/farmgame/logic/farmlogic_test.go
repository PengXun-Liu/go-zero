package logic_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/example/farmgame/logic"
	"github.com/zeromicro/go-zero/example/farmgame/types"
)

// newTestRedis starts an in-process miniredis server and returns a connected
// go-zero Redis client and the underlying miniredis instance.
func newTestRedis(t *testing.T) (*redis.Redis, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rds, err := redis.NewRedis(redis.RedisConf{
		Host: mr.Addr(),
		Type: "node",
	})
	require.NoError(t, err)
	return rds, mr
}

func newFarmLogic(t *testing.T) *logic.FarmLogic {
	t.Helper()
	rds, _ := newTestRedis(t)
	return logic.NewFarmLogic(rds)
}

// futureLogic returns a FarmLogic whose clock is set to `offset` seconds in
// the future, sharing the same Redis as `base`.
func futureLogic(base *logic.FarmLogic, offset time.Duration) *logic.FarmLogic {
	return base.WithNow(func() time.Time { return time.Now().Add(offset) })
}

func TestGetFarm_NewPlayer(t *testing.T) {
	fl := newFarmLogic(t)
	farm, err := fl.GetFarm(context.Background(), "player1")
	require.NoError(t, err)
	assert.Equal(t, "player1", farm.PlayerID)
	assert.Equal(t, int64(100), farm.Coins)
	assert.Len(t, farm.Plots, 6)
	for _, p := range farm.Plots {
		assert.Equal(t, types.PlotEmpty, p.State)
	}
}

func TestGetFarm_Idempotent(t *testing.T) {
	fl := newFarmLogic(t)
	farm1, err := fl.GetFarm(context.Background(), "player1")
	require.NoError(t, err)
	farm2, err := fl.GetFarm(context.Background(), "player1")
	require.NoError(t, err)
	assert.Equal(t, farm1.Coins, farm2.Coins)
	assert.Equal(t, len(farm1.Plots), len(farm2.Plots))
}

func TestPlant_Success(t *testing.T) {
	fl := newFarmLogic(t)
	plot, err := fl.Plant(context.Background(), "p1", 0, types.CropWheat)
	require.NoError(t, err)
	assert.Equal(t, types.PlotGrowing, plot.State)
	assert.Equal(t, types.CropWheat, plot.Crop)
	assert.Greater(t, plot.ReadyAt, plot.PlantedAt)
}

func TestPlant_UnknownCrop(t *testing.T) {
	fl := newFarmLogic(t)
	_, err := fl.Plant(context.Background(), "p1", 0, types.CropKind("mango"))
	require.Error(t, err)
}

func TestPlant_InvalidPlot(t *testing.T) {
	fl := newFarmLogic(t)
	_, err := fl.Plant(context.Background(), "p1", 99, types.CropCorn)
	require.Error(t, err)
}

func TestPlant_PlotNotEmpty(t *testing.T) {
	fl := newFarmLogic(t)
	_, err := fl.Plant(context.Background(), "p1", 0, types.CropCorn)
	require.NoError(t, err)
	_, err = fl.Plant(context.Background(), "p1", 0, types.CropCorn)
	require.Error(t, err)
}

func TestWater_ReducesGrowTime(t *testing.T) {
	fl := newFarmLogic(t)
	ctx := context.Background()

	plot0, err := fl.Plant(ctx, "p1", 0, types.CropCarrot)
	require.NoError(t, err)
	originalReady := plot0.ReadyAt

	plot1, err := fl.Water(ctx, "p1", 0)
	require.NoError(t, err)
	assert.Equal(t, types.PlotGrowing, plot1.State)
	assert.Less(t, plot1.ReadyAt, originalReady)
}

func TestWater_NotGrowing(t *testing.T) {
	fl := newFarmLogic(t)
	_, err := fl.Water(context.Background(), "p1", 0)
	require.Error(t, err)
}

func TestHarvest_Success(t *testing.T) {
	rds, _ := newTestRedis(t)
	ctx := context.Background()

	fl := logic.NewFarmLogic(rds)
	_, err := fl.Plant(ctx, "p1", 0, types.CropWheat)
	require.NoError(t, err)

	// Use a clock set 60 s in the future so the crop has matured.
	flFuture := futureLogic(fl, 60*time.Second)

	plot, earned, err := flFuture.Harvest(ctx, "p1", 0)
	require.NoError(t, err)
	assert.Equal(t, types.PlotEmpty, plot.State)
	assert.Greater(t, earned, int64(0))

	farm, err := flFuture.GetFarm(ctx, "p1")
	require.NoError(t, err)
	assert.Equal(t, int64(100)+earned, farm.Coins)
}

func TestHarvest_NotReady(t *testing.T) {
	fl := newFarmLogic(t)
	_, err := fl.Plant(context.Background(), "p1", 0, types.CropCorn)
	require.NoError(t, err)
	_, _, err = fl.Harvest(context.Background(), "p1", 0)
	require.Error(t, err)
}

func TestListCrops(t *testing.T) {
	crops := logic.ListCrops()
	assert.NotEmpty(t, crops)
	for _, c := range crops {
		assert.NotEmpty(t, c.Kind)
		assert.NotEmpty(t, c.Name)
		assert.Greater(t, c.GrowSeconds, int64(0))
	}
}

// TestGrowthRefresh verifies that a growing plot transitions to PlotReady once
// time has elapsed, without an explicit harvest call.
func TestGrowthRefresh(t *testing.T) {
	rds, _ := newTestRedis(t)
	ctx := context.Background()

	fl := logic.NewFarmLogic(rds)
	_, err := fl.Plant(ctx, "p1", 0, types.CropWheat)
	require.NoError(t, err)

	flFuture := futureLogic(fl, 60*time.Second)
	farm, err := flFuture.GetFarm(ctx, "p1")
	require.NoError(t, err)
	assert.Equal(t, types.PlotReady, farm.Plots[0].State)
}

// TestFarmPersistence verifies that game state is stored in Redis and survives
// across separate FarmLogic instances pointing at the same Redis.
func TestFarmPersistence(t *testing.T) {
	rds, _ := newTestRedis(t)
	ctx := context.Background()

	fl1 := logic.NewFarmLogic(rds)
	_, err := fl1.Plant(ctx, "p1", 2, types.CropCarrot)
	require.NoError(t, err)

	fl2 := logic.NewFarmLogic(rds)
	farm, err := fl2.GetFarm(ctx, "p1")
	require.NoError(t, err)
	assert.Equal(t, types.CropCarrot, farm.Plots[2].Crop)
}

// TestFarmJSON verifies that the Farm struct marshals to the expected JSON
// shape (used by the HTTP API).
func TestFarmJSON(t *testing.T) {
	fl := newFarmLogic(t)
	farm, err := fl.GetFarm(context.Background(), "alice")
	require.NoError(t, err)

	data, err := json.Marshal(farm)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, "alice", m["player_id"])
	assert.Equal(t, float64(100), m["coins"])
	assert.NotNil(t, m["plots"])
}

