package nues

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

type Server interface {
	Serve(context.Context)
	Close() error
}

type Nues struct {
	Debug           bool
	ServiceId       string
	ServicesFileUrl string
	DbUri           string
	DbName          string
	DbPrefix        string
	AdminToken      string
	Reset           bool
	ApiPort         string
	RpcPort         string
	Routes          Routes
	ReqPerSec       int
	colCommands     string
	colIdentity     string
	colSessions     string
	colEvents       string
	colWatchers     string
	colProjections  string
}

var nues Nues

func RunServer(_config Nues) error {

	MustNotEmpty(_config.ServiceId, NewError(-1, "Service Name is required"))
	MustNotEmpty(_config.DbName, NewError(-1, "MongoDb is required"))
	MustNotEmpty(_config.DbUri, NewError(-1, "MongoUri is required"))
	MustNotEmpty(_config.DbPrefix, NewError(-1, "MongoPrefix is required"))
	MustNotEmpty(_config.ApiPort, NewError(-1, "API Port is required"))
	MustNotEmpty(_config.Routes, NewError(-1, "Routes is required"))
	MustNotEmpty(_config.ServicesFileUrl, NewError(-1, "ServicesFileUrl Ip is required"))

	_config.colCommands = "commands"
	_config.colIdentity = "identities"
	_config.colSessions = "sessions"
	_config.colEvents = "events"
	_config.colWatchers = "watchers"
	_config.colProjections = "projections"
	nues = _config

	logL := slog.LevelWarn
	if nues.Debug {
		logL = slog.LevelDebug
	}
	slog.LogAttrs(context.TODO(), logL, nues.ServiceId)
	run()
	return nil
}

func registerCustomValidators() {
	validate.RegisterValidation("phone_dz", phoneValidator)
}

func run() {
	initDb()
	initAuth()
	registerCustomValidators()
	var rpc Server
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
		rpc = &NuesRpc{
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
	if rpc != nil {
		if err := rpc.Close(); err != nil {
			slog.Error("error stopping RPC", err)
		}
	}
	slog.Info("server exiting")
}
