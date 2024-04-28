package nues

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

type Server interface {
	Serve(context.Context) error
	Close() error
}

type Nues struct {
	Debug          bool
	ServiceId      string
	IdentityDbUri  string
	IdentityDbName string
	DbUri          string
	DbName         string
	DbPrefix       string
	ColEvents      string
	ColProjections string
	ColCommands    string
	ColWatchers    string
	ColSession     string
	ColIdentity    string
	AdminToken     string
	Reset          bool
	ApiPort        string
	RpcPort        string
	Routes         Routes
	ReqPerSec      int
}

var nues Nues

func RunServer(_config Nues) error {

	if _config.ColCommands == "" {
		_config.ColCommands = "commands"
	}
	if _config.ColEvents == "" {
		_config.ColEvents = "events"
	}
	if _config.ColProjections == "" {
		_config.ColProjections = "projections"
	}
	if _config.ColSession == "" {
		_config.ColSession = "sessions"
	}
	if _config.ColWatchers == "" {
		_config.ColWatchers = "watchers"
	}
	if _config.ColIdentity == "" {
		_config.ColIdentity = "identity"
	}
	if _config.IdentityDbName == "" {
		_config.IdentityDbName = "identity"
	}

	MustNotEmpty(_config.ServiceId, NewError(-1, "Service Name is required"))
	MustNotEmpty(_config.DbName, NewError(-1, "MongoDb is required"))
	MustNotEmpty(_config.DbUri, NewError(-1, "MongoUri is required"))
	MustNotEmpty(_config.DbPrefix, NewError(-1, "MongoPrefix is required"))
	MustNotEmpty(_config.ApiPort, NewError(-1, "API Port is required"))
	MustNotEmpty(_config.Routes, NewError(-1, "Routes is required"))
	nues = _config

	logL := slog.LevelWarn
	if nues.Debug {
		logL = slog.LevelDebug
	}
	slog.LogAttrs(context.TODO(), logL, nues.ServiceId)
	run()
	return nil
}

func run() {
	initAuth()
	initDb()
	if nues.RpcPort != "" {
		initRpc()
	}

	ctx, cancel := context.WithCancel(context.TODO())
	var api Server = &NuesApi{
		throttle:  make(map[string]int),
		reqPerSec: 4,
		context:   context.TODO(),
	}
	go api.Serve(ctx)

	if nues.RpcPort != "" {
		var rpc Server = &NuesRpc{
			Network: "tcp",
		}
		go rpc.Serve(ctx)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutdown server ...")
	cancel()
	if err := api.Close(); err != nil {
		slog.Error("error stopping API", err)
	}
	if err := rpc.Close(); err != nil {
		slog.Error("error stopping RPC", err)
	}
	slog.Info("server exiting")
}
