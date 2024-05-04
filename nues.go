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
	DbName      string
	IP          string
	ApiPort     string
	RpcPort     string
	Routes      Routes

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
	MustNotEmpty(_config.DbName, NewError(-1, "DbName is required"))
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
	validate.RegisterValidation("identity", identityValidator)
}

func initConfig() {
	config := GetConfig[ConfigNues](false)
	if config == nil {
		// init default config
		config = &ConfigNues{
			Id:             "nues",
			Reset:          false,
			AdminToken:     GenerateId(),
			ColCommands:    "commands",
			ColEvents:      "events",
			ColWatchers:    "watchers",
			ColSessions:    "sessions",
			ColIdentity:    "identities",
			ColProjections: "projections",
			DbPrefix:       "sb",
		}
		_, err := DB.Collection("__config").InsertOne(context.TODO(), config)
		if err != nil {
			panic(err)
		}
	}

	nues.reset = config.Reset
	nues.adminToken = config.AdminToken
	nues.colCommands = config.ColCommands
	nues.colEvents = config.ColEvents
	nues.colIdentity = config.ColIdentity
	nues.colProjections = config.ColProjections
	nues.colSessions = config.ColSessions
	nues.colWatchers = config.ColWatchers
	nues.dbPrefix = config.DbPrefix

	slog.Debug("config loaded successfully", config)
	insertSelfService()
	loadServices()

}
func insertSelfService() {
	selfService := NuesService{
		Id:   nues.ServiceId,
		Name: nues.ServiceName,
		Ip:   nues.IP,
		Port: nues.RpcPort,
	}
	_, err := DB.Collection("__services").UpdateOne(context.Background(), bson.M{"_id": selfService.Id}, bson.M{"$set": selfService}, options.Update().SetUpsert(true))
	if err != nil {
		panic(err)
	}
	slog.Debug("self service injected successfully")
}
func loadServices() {
	go func() {
		for {
			var services []NuesService
			cur, err := DB.Collection("__services").Find(context.TODO(), bson.M{})
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
	initNuesDb()
	initConfig()
	initNuesIndexes()
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
