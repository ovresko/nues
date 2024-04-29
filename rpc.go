package nues

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/rpc"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type NuesRpcCall struct{}
type NuesRpcArgs struct {
	CommandName string
	Payload     []byte
	Token       string
	CallId      string
}
type NuesRpcResponse struct {
	ServiceId string
	Response  any
}

type NuesRpc struct {
	Network string
	context context.Context
}

type NuesService struct {
	Ip   string
	Port string
	Name string
}

var services map[string]*NuesService = make(map[string]*NuesService)

func initRpc() {
	loadServices()
	nrc := new(NuesRpcCall)
	rpc.RegisterName("NuesRpcCall", nrc)
	rpc.HandleHTTP()
}

func (n *NuesRpcCall) Call(args *NuesRpcArgs, reply *NuesRpcResponse) error {

	ctx := context.Background()

	var err error
	route, found := nues.Routes[args.CommandName]
	if !found {
		return ErrBadCommand
	}
	auth := authCall(args.Token, route)
	if !auth {
		return ErrUserNotAuth
	}

	callId := args.CallId
	var called bool = false
	var response any
	if callId != "" {
		// try call history
		var call bson.M
		err := DB.GetCollection(nues.colCommands).FindOne(context.TODO(), bson.M{"_id": callId}).Decode(&call)
		if err != nil && err != mongo.ErrNoDocuments {
			return ErrSystemInternal
		}
		if call != nil {
			//Idempotency detected
			called = true
			response = call["response"]
		}
	}
	if !called {
		response, err = rpcServe(ctx, route, args)
	}

	if err != nil {
		slog.Error("http failed", "err", err)
		return ErrBadCommand
	} else {
		reply = &NuesRpcResponse{
			Response:  response,
			ServiceId: nues.ServiceId,
		}
	}
	return nil

}

func rpcServe(ctx context.Context, route Route, args *NuesRpcArgs) (any, error) {

	body := args.Payload

	if body == nil {
		return nil, ErrBadCommand
	}

	switch route.Call {
	case HANDLER:
		handler, ok := route.Handler().(func(context.Context, map[string]any) RouteResponse)
		if !ok {
			return nil, ErrSystemInternal
		}
		reqBody := make(map[string]any)
		if len(body) > 0 {
			err := json.Unmarshal(body, &reqBody)
			if err != nil {
				return nil, ErrBadCommand
			}
		}

		res := handler(ctx, reqBody)
		return res, nil

	case COMMAND:
		var cmdClone = route.Handler()
		cmd, ok := cmdClone.(Command)
		if !ok {
			return nil, ErrSystemInternal
		}
		if len(body) > 0 {
			err := json.Unmarshal(body, cmdClone)
			if err != nil {
				return nil, ErrBadCommand
			}
		}
		cmdRoot := &CommandRoot{
			Command: cmd,
			CallId:  args.CallId,
		}
		cmdRoot.Execute(ctx)
		return cmdRoot, nil

	case QUERY:
		var queryClone = route.Handler()
		query, ok := queryClone.(Query)
		if !ok {
			return nil, ErrSystemInternal
		}
		if len(body) > 0 {
			err := json.Unmarshal(body, queryClone)
			if err != nil {
				return nil, ErrBadCommand
			}
		}
		queryRoot := &QueryRoot{
			Query: query,
		}
		queryRoot.Execute(ctx)
		return queryRoot, nil
	}
	return nil, ErrSystemInternal
}

func (n *NuesRpc) Close() error {
	return nil
}
func (n *NuesRpc) Serve(ctx context.Context) {
	slog.Info("starting RPC server...")
	n.context = ctx

	l, err := net.Listen(n.Network, nues.RpcPort)
	if err != nil {
		slog.Error("listen error:", err)
		panic(err)
	}
	err = http.Serve(l, nil)
	if err != nil {
		panic(err)
	}
}

func getService(name string) *NuesService {
	service, found := services[name]
	if !found {
		slog.Error("service %s not found", name)
		panic(fmt.Sprintf("service %v not found", name))
	}
	return service

}
func loadServices() {

	go func() {
		c := http.DefaultClient
		for {
			res, err := c.Get(nues.ServicesFileUrl)
			if err != nil {
				slog.Error("load service failed", "err", err)
			} else {
				body, err := io.ReadAll(res.Body)
				if err != nil {
					slog.Error("load service failed", "err", err)
				} else {
					var newServices map[string]map[string]string = make(map[string]map[string]string)
					err = json.Unmarshal(body, &newServices)
					if err != nil {
						slog.Error("load service failed", "err", err)
					} else {
						for k := range services {
							delete(services, k)
						}
						for k, v := range newServices {
							slog.Debug("registering new service", "name", k, "ip", v["ip"], "port", v["port"])
							services[k] = &NuesService{
								Ip:   v["ip"],
								Port: v["port"],
								Name: k,
							}
						}
					}
				}
			}
			time.Sleep(time.Duration(time.Second * 60))
		}
	}()

}

func RequestRpc(serviceName string, args NuesRpcArgs) (*NuesRpcResponse, error) {

	service := getService(serviceName)
	client, err := rpc.DialHTTP("tcp", service.Ip+service.Port)
	if err != nil {
		slog.Error("dialing:", err)
		return nil, err
	}
	reply := &NuesRpcResponse{}
	err = client.Call("NuesRpcCall.Call", args, reply)
	if err != nil {
		slog.Error("dialing:", err)
		return nil, err
	}

	client.Close()
	return reply, nil
}
