package nues

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Server interface {
	Serve(context.Context)
	Close() error
}

type Nues struct {
	Debug       bool
	ServiceId   string
	ServiceName string
	DbUri       string
	IP          string
	ApiPort     string
	RpcPort     string
	Routes      Routes

	dbName         string
	dbPrefix       string
	adminToken     string
	reset          bool
	colCommands    string
	colIdentity    string
	colSessions    string
	colEvents      string
	colWatchers    string
	colProjections string
	services       []NuesService
}

var nues Nues

func RunServer(_config Nues) error {

	MustNotEmpty(_config.IP, NewError(-1, "IP is required"))
	MustNotEmpty(_config.ServiceId, NewError(-1, "Service ID is required"))
	MustNotEmpty(_config.ServiceName, NewError(-1, "Service Name is required"))
	MustNotEmpty(_config.DbUri, NewError(-1, "MongoUri is required"))
	MustNotEmpty(_config.ApiPort, NewError(-1, "API Port is required"))
	MustNotEmpty(_config.Routes, NewError(-1, "Routes is required"))

	nues = _config

	logL := slog.LevelWarn
	if nues.Debug {
		slog.Info("setting log level to debug")
		logL = slog.LevelDebug
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logL,
		AddSource: true,
	}))

	slog.SetDefault(l) // configures log package to print with LevelInfo

	if !slog.Default().Enabled(context.TODO(), logL) {
		panic("error setting logger")
	}
	slog.LogAttrs(context.TODO(), logL, nues.ServiceId)
	run()
	return nil
}

func registerCustomValidators() {
	validate.RegisterValidation("phone_dz", phoneValidator)
}

func initConfig() {
	var config bson.M = GetConfig("nues")

	var ok bool
	nues.reset, ok = config["reset"].(bool)
	if !ok {
		panic("init config error")
	}
	nues.adminToken, ok = config["admin_token"].(string)
	if !ok {
		panic("init config error")
	}
	nues.colCommands, ok = config["col_commands"].(string)
	if !ok {
		panic("init config error")
	}
	nues.colEvents, ok = config["col_events"].(string)
	if !ok {
		panic("init config error")
	}
	nues.colIdentity, ok = config["col_identity"].(string)
	if !ok {
		panic("init config error")
	}
	nues.colProjections, ok = config["col_projections"].(string)
	if !ok {
		panic("init config error")
	}
	nues.colSessions, ok = config["col_sessions"].(string)
	if !ok {
		panic("init config error")
	}
	nues.colWatchers, ok = config["col_watchers"].(string)
	if !ok {
		panic("init config error")
	}
	nues.dbName, ok = config["db_name"].(string)
	if !ok {
		panic("init config error")
	}
	nues.dbPrefix, ok = config["db_prefix"].(string)
	if !ok {
		panic("init config error")
	}

	slog.Debug("config loaded successfully", config)
	insertSelfService()
	loadServices()

}
func insertSelfService() {
	db := getInternalDb()
	defer db.Disconnect()
	selfService := NuesService{
		Id:   nues.ServiceId,
		Name: nues.ServiceName,
		Ip:   nues.IP,
		Port: nues.RpcPort,
	}
	_, err := db.Collection("__services").UpdateOne(context.Background(), bson.M{"_id": selfService.Id}, bson.M{"$set": selfService}, options.Update().SetUpsert(true))
	if err != nil {
		panic(err)
	}
	slog.Debug("self service injected successfully")
}
func loadServices() {
	go func() {
		for {
			db := getInternalDb()
			defer db.Disconnect()
			var services []NuesService
			cur, err := db.Collection("__services").Find(context.TODO(), bson.M{})
			if err != nil {
				panic(err)
			}
			err = cur.All(context.TODO(), &services)
			if err != nil && err != mongo.ErrNoDocuments {
				panic(err)
			}
			nues.services = services
			slog.Debug("services loaded successfully", "services", services)
			time.Sleep(time.Duration(time.Second * 60 * 2))
		}
	}()
}

func run() {
	initConfig()
	initNuesDb()
	initAuth()
	registerCustomValidators()
	var rpc Server
	if nues.RpcPort != "" {
		initRpc()
	}

	ctx, cancel := context.WithCancel(context.TODO())
	var api Server = &NuesApi{
		context: context.TODO(),
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
