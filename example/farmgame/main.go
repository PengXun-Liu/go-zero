// Package farmgame is the entry point for the farm-game example server.
//
// Run:
//
//	go run . -f farmgame.yaml
package main

import (
	"flag"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/example/farmgame/handler"
	"github.com/zeromicro/go-zero/example/farmgame/logic"
	"github.com/zeromicro/go-zero/rest"
)

// Config holds the complete server configuration.
type Config struct {
	rest.RestConf        // HTTP server settings (Host, Port, Timeout, etc.)
	Redis  redis.RedisConf // Redis connection settings
}

var configFile = flag.String("f", "farmgame.yaml", "configuration file")

func main() {
	flag.Parse()

	// 1. Load configuration from the YAML file.
	var c Config
	conf.MustLoad(*configFile, &c)

	// 2. Connect to Redis – used to store all game state.
	rds := redis.MustNewRedis(c.Redis)

	// 3. Build the go-zero REST server.
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// 4. Wire up business logic.
	fl := logic.NewFarmLogic(rds)

	// 5. Register routes.
	//
	//   GET  /crops                          – list available crop types
	//   GET  /farm/:playerID                 – get farm state
	//   POST /farm/:playerID/plant           – plant a crop
	//   POST /farm/:playerID/water/:plotID   – water a growing plot
	//   POST /farm/:playerID/harvest/:plotID – harvest a ready plot
	server.AddRoutes([]rest.Route{
		{
			Method:  "GET",
			Path:    "/crops",
			Handler: handler.ListCropsHandler(),
		},
		{
			Method:  "GET",
			Path:    "/farm/:playerID",
			Handler: handler.GetFarmHandler(fl),
		},
		{
			Method:  "POST",
			Path:    "/farm/:playerID/plant",
			Handler: handler.PlantHandler(fl),
		},
		{
			Method:  "POST",
			Path:    "/farm/:playerID/water/:plotID",
			Handler: handler.WaterHandler(fl),
		},
		{
			Method:  "POST",
			Path:    "/farm/:playerID/harvest/:plotID",
			Handler: handler.HarvestHandler(fl),
		},
	})

	// 6. Start serving (blocks until SIGTERM / SIGINT).
	server.Start()
}
