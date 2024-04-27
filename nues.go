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
	MustNotEmpty(_config.ColCommands, NewError(-1, "ColCommands is required"))
	MustNotEmpty(_config.ColEvents, NewError(-1, "ColEvents is required"))
	MustNotEmpty(_config.ColProjections, NewError(-1, "ColProjections is required"))
	MustNotEmpty(_config.ColSession, NewError(-1, "ColSession is required"))
	MustNotEmpty(_config.ColWatchers, NewError(-1, "ColWatchers is required"))
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
