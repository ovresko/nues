package nues

import "context"

type Nues struct {
	Debug          bool
	MongoUri       string
	MongoDb        string
	MongoPrefix    string
	ColEvents      string
	ColProjections string
	ColCommands    string
	ColWatchers    string
	ColSession     string
	AdminToken     string
	Reset          bool
	Port           string
	Routes         Routes
	ReqPerSec      int
}

var nues Nues

func NewServer(_config Nues) error {

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

	MustNotEmpty(_config.MongoDb, NewError(-1, "MongoDb is required"))
	MustNotEmpty(_config.MongoUri, NewError(-1, "MongoUri is required"))
	MustNotEmpty(_config.MongoPrefix, NewError(-1, "MongoPrefix is required"))
	MustNotEmpty(_config.Port, NewError(-1, "Port is required"))
	MustNotEmpty(_config.Routes, NewError(-1, "Routes is required"))
	nues = _config
	return nil
}

func (n *Nues) Run() {
	api := NuesApi{
		throttle:  make(map[string]int),
		reqPerSec: 4,
		context:   context.TODO(),
	}
	api.Serve()
}
