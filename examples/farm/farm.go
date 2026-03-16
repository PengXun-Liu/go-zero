package main

import (
	"flag"
	"fmt"

	"github.com/zeromicro/go-zero/examples/farm/internal/config"
	"github.com/zeromicro/go-zero/examples/farm/internal/handler"
	"github.com/zeromicro/go-zero/examples/farm/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/farm.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterRoutes(server, ctx)

	fmt.Printf("Starting farm game server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
