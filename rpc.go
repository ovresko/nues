package nues

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strings"

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
	Ip      string
	Port    string
	Name    string

	context context.Context
}

type NuesService struct {
	Ip   string
	Port string
	Name string
}

var services map[string]NuesService = make(map[string]NuesService)

func init() {
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
	auth := AuthCall(args.Token, route)
	if !auth {
		return ErrUserNotAuth
	}

	callId := args.CallId
	var called bool = false
	var response any
	if callId != "" {
		// try call history
		var call bson.M
		err := DB.GetCollection(nues.ColCommands).FindOne(context.TODO(), bson.M{"_id": callId}).Decode(&call)
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

	switch route.call {
	case HANDLER:
		handler, ok := route.handler().(func(context.Context, map[string]any) RouteResponse)
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
		var cmdClone = route.handler()
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
		var queryClone = route.handler()
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
func (n *NuesRpc) Serve(ctx context.Context) error {
	n.context = ctx

	serviceEnv := n.Name + "#" + n.Ip + "#" + n.Port + "|"
	env := os.Getenv("NUES_SERVICES")
	env = strings.ReplaceAll(env, serviceEnv, "")
	serviceEnv = env + serviceEnv
	os.Setenv("NUES_SERVICES", serviceEnv)
	l, err := net.Listen(n.Network, n.Port)
	if err != nil {
		slog.Error("listen error:", err)
		return err
	}
	return http.Serve(l, nil)
}

func getService(name string) *NuesService {
	service, found := services[name]
	if !found {
		err := loadServices()
		if err != nil {
			panic(err)
		}
		service, found = services[name]
	}
	if !found {
		slog.Error("service %s not found", name)
		panic("service not found")
	}
	return &service

}
func loadServices() error {
	for k := range services {
		delete(services, k)
	}
	servicestr := os.Getenv("NUES_SERVICES")
	if servicestr == "" {
		return NewError(-1, fmt.Sprintln("service %v not found", servicestr))
	}
	servicesarr := strings.Split(servicestr, "|")
	for _, v := range servicesarr {
		if v == "" {
			continue
		}
		keys := strings.Split(v, "#")
		serviceName := keys[0]
		serviceIp := keys[1]
		servicePort := keys[2]
		services[serviceName] = NuesService{
			Ip:   serviceIp,
			Port: servicePort,
			Name: serviceName,
		}
	}

	return nil
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
