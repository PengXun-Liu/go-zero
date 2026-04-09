package farm

import (
	"errors"
	"net/http"

	farmlogic "github.com/zeromicro/go-zero/examples/farm/internal/logic/farm"
	"github.com/zeromicro/go-zero/examples/farm/internal/svc"
	"github.com/zeromicro/go-zero/examples/farm/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// userIdFromCtx extracts the userId that was embedded in the context by the JWT middleware.
func userIdFromCtx(r *http.Request) (string, error) {
	userId, ok := r.Context().Value("userId").(string)
	if !ok || userId == "" {
		return "", errors.New("unauthorized: missing userId in token")
	}
	return userId, nil
}

// PlantHandler handles POST /api/v1/farm/plant.
func PlantHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, err := userIdFromCtx(r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		var req types.PlantRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := farmlogic.NewPlantLogic(r.Context(), svcCtx)
		resp, err := l.Plant(userId, &req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

// WaterHandler handles POST /api/v1/farm/water.
func WaterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, err := userIdFromCtx(r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		var req types.WaterRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := farmlogic.NewWaterLogic(r.Context(), svcCtx)
		resp, err := l.Water(userId, &req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

// HarvestHandler handles POST /api/v1/farm/harvest.
func HarvestHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, err := userIdFromCtx(r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		var req types.HarvestRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := farmlogic.NewHarvestLogic(r.Context(), svcCtx)
		resp, err := l.Harvest(userId, &req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

// StatusHandler handles GET /api/v1/farm/status.
func StatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, err := userIdFromCtx(r)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := farmlogic.NewStatusLogic(r.Context(), svcCtx)
		resp, err := l.Status(userId)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
