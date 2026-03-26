// Package handler wires HTTP requests to farm-game business logic.
package handler

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/example/farmgame/logic"
	"github.com/zeromicro/go-zero/example/farmgame/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// GetFarmHandler returns the full state of a player's farm.
//
// GET /farm/:playerID
func GetFarmHandler(fl *logic.FarmLogic) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := r.PathValue("playerID")
		farm, err := fl.GetFarm(r.Context(), playerID)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, farm)
	}
}

// PlantHandler plants a crop on the specified plot.
//
// POST /farm/:playerID/plant
// Body: {"plot_id": 0, "crop": "wheat"}
func PlantHandler(fl *logic.FarmLogic) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := r.PathValue("playerID")

		var req types.PlantRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		plot, err := fl.Plant(r.Context(), playerID, req.PlotID, req.Crop)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, types.PlantResponse{Plot: plot})
	}
}

// WaterHandler waters a growing plot, reducing its grow time.
//
// POST /farm/:playerID/water/:plotID
func WaterHandler(fl *logic.FarmLogic) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := r.PathValue("playerID")
		plotID, err := strconv.Atoi(r.PathValue("plotID"))
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		plot, err := fl.Water(r.Context(), playerID, plotID)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, types.WaterResponse{Plot: plot})
	}
}

// HarvestHandler harvests a ready plot and awards coins.
//
// POST /farm/:playerID/harvest/:plotID
func HarvestHandler(fl *logic.FarmLogic) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		playerID := r.PathValue("playerID")
		plotID, err := strconv.Atoi(r.PathValue("plotID"))
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		plot, earned, err := fl.Harvest(r.Context(), playerID, plotID)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, types.HarvestResponse{Plot: plot, Earned: earned})
	}
}

// ListCropsHandler returns all available crop types.
//
// GET /crops
func ListCropsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.OkJsonCtx(r.Context(), w, logic.ListCrops())
	}
}
