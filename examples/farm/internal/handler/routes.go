package handler

import (
	"net/http"

	farmhandler "github.com/zeromicro/go-zero/examples/farm/internal/handler/farm"
	userhandler "github.com/zeromicro/go-zero/examples/farm/internal/handler/user"
	"github.com/zeromicro/go-zero/examples/farm/internal/svc"
	"github.com/zeromicro/go-zero/rest"
)

// RegisterRoutes registers all HTTP routes on the server.
//
// Public routes (no auth required):
//   POST /api/v1/user/register
//   POST /api/v1/user/login
//
// Protected routes (JWT required):
//   GET  /api/v1/farm/status
//   POST /api/v1/farm/plant
//   POST /api/v1/farm/water
//   POST /api/v1/farm/harvest
func RegisterRoutes(server *rest.Server, svcCtx *svc.ServiceContext) {
	// Public routes — no JWT middleware.
	server.AddRoutes([]rest.Route{
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/user/register",
			Handler: userhandler.RegisterHandler(svcCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/user/login",
			Handler: userhandler.LoginHandler(svcCtx),
		},
	})

	// Protected routes — JWT authentication is enforced by WithJwt.
	server.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/api/v1/farm/status",
			Handler: farmhandler.StatusHandler(svcCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/farm/plant",
			Handler: farmhandler.PlantHandler(svcCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/farm/water",
			Handler: farmhandler.WaterHandler(svcCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/farm/harvest",
			Handler: farmhandler.HarvestHandler(svcCtx),
		},
	}, rest.WithJwt(svcCtx.Config.Auth.AccessSecret))
}
