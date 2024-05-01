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
		Debug:       true,
		ServiceId:   "sabilwallet_1",
		DbUri:       "mongodb://localhost:27017",
		ServiceName: "Test",
		IP:          "localhost",
		ApiPort:     ":8080",
		RpcPort:     "",
		Routes:      routes,
		ReqPerSec:   3,
	})
}
