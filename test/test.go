package main

import "github.com/ovresko/nues"

var routes nues.Routes = nues.Routes{
	"ping": nues.Route{
		Public:  false,
		Call:    nues.HANDLER,
		Handler: func() any { return nues.Ping },
		Name:    "ping",
	},
}

func main() {
	_ = nues.RunServer(nues.Nues{
		Debug:          true,
		ServiceId:      "sabilwallet_1",
		IdentityDbUri:  "mongodb://localhost:27017",
		IdentityDbName: "identity",
		DbUri:          "mongodb://localhost:27017",
		DbName:         "sabil_ms",
		DbPrefix:       "sb",
		ColEvents:      "events",
		ColProjections: "projections",
		ColCommands:    "commands",
		ColWatchers:    "watchers",
		ColSession:     "sessions",
		ColIdentity:    "identity",
		AdminToken:     "TOKEN",
		Reset:          false,
		ApiPort:        ":8080",
		RpcPort:        "",
		Routes:         routes,
		ReqPerSec:      3,
	})
}
