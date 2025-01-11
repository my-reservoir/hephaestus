package main

import (
	"flag"
	"fmt"
	"github.com/go-kratos/kratos/v2/registry"
	lua "github.com/yuin/gopher-lua"
	"hephaestus/internal/conf"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
	_ "net/http/pprof"
)

var (
	Name       = "hephaestus"
	Version    = "1.1.0"
	flagConf   string
	id, _      = os.Hostname()
	startTime  = time.Now()
	timeFormat = "2006-01-02 15:04:05.000"
)

func init() {
	flag.StringVar(&flagConf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(
	logger log.Logger, reg registry.Registrar,
	gs *grpc.Server, hs *http.Server,
) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{
			"description": "A powerful distributed Lua scripting middleware.",
			"lua_version": lua.LuaVersion,
		}),
		kratos.Logger(logger),
		kratos.Server(
			gs, hs,
		),
		kratos.Registrar(reg),
	)
}

func main() {
	// Parse arguments from the command line
	flag.Parse()
	logger := NewLogger()
	log.SetLogger(logger)
	// Initialize the config with the source
	// Note that a source may on the local disk, but it can also be fetched from remote one
	c := config.New(
		config.WithSource(
			file.NewSource(flagConf),
		),
	)
	defer func(c config.Config) {
		err := c.Close()
		if err != nil {
			log.Warn(err)
		}
	}(c) // We should make sure the config file would finally be closed

	if err := c.Load(); err != nil { // Try to load the config file from the given source
		panic(err)
	}

	// Bootstrap is the configuration structure of the config file
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil { // Try to deserialize the config into the corresponding structure
		panic(err)
	}

	// Inject dependencies into the service
	app, cleanup, err := wireApp(bc.Registry, bc.Server, bc.Telemetry, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup() // Clean up the injected dependencies before exits

	builder := strings.Builder{}
	builder.WriteString(
		fmt.Sprintf(
			"\n    \033[36;1m%s\033[0m \033[96m%s\033[0m  ready in \033[33;1m%v\033[0m\n",
			Name, Version, time.Now().Sub(startTime),
		),
	)
	fmt.Println(builder.String())

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
