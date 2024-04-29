package nues

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type NuesApi struct {
	context   context.Context
	server    *http.Server
	throttle  map[string]int
	reqPerSec int
}

func Ping(context.Context, map[string]any) RouteResponse {
	return RouteResponse{"response": true}
}

func (h *NuesApi) httpServe(route Route, r *http.Request) (any, error) {

	body, err := io.ReadAll(r.Body)

	if err != nil {
		return nil, ErrSystemInternal
	}

	switch route.Call {
	case HANDLER:
		handler, ok := route.Handler().(func(context.Context, map[string]any) RouteResponse)
		if !ok {
			return nil, ErrSystemInternal
		}
		reqBody := make(map[string]any)
		if len(body) > 0 {
			err = json.Unmarshal(body, &reqBody)
			if err != nil {
				return nil, ErrBadCommand
			}
		}

		res := handler(h.context, reqBody)
		return res, nil

	case COMMAND:
		var cmdClone = route.Handler()
		cmd, ok := cmdClone.(Command)
		if !ok {
			return nil, ErrSystemInternal
		}
		if len(body) > 0 {
			err = json.Unmarshal(body, cmdClone)
			if err != nil {
				return nil, ErrBadCommand
			}
		}
		callId := r.Header.Get("callId")
		cmdRoot := &CommandRoot{
			Command: cmd,
			CallId:  callId,
		}
		cmdRoot.Execute(h.context)
		return cmdRoot, nil

	case QUERY:
		var queryClone = route.Handler()
		query, ok := queryClone.(Query)
		if !ok {
			return nil, ErrSystemInternal
		}
		if len(body) > 0 {
			err = json.Unmarshal(body, queryClone)
			if err != nil {
				return nil, ErrBadCommand
			}
		}
		queryRoot := &QueryRoot{
			Query: query,
		}
		queryRoot.Execute(h.context)
		return queryRoot, nil
	}
	return nil, ErrSystemInternal

}

// func (h *NuesApi) httpAuth(route Route, r *http.Request) bool {

// 	switch route.auth {
// 	case ROUTE_PUBLIC:
// 		slog.Info("PUBLIC route")
// 		return true
// 	case ROUTE_USER:
// 		slog.Info("USER", "path", route)
// 		return h.AuthUser(r)
// 	case ROUTE_ADMIN:
// 		slog.Info("ADMIN", "path", route)
// 		return h.AuthAdmin(r)
// 	}

// 	return false
// }

func (h *NuesApi) clearthrottle() {

	for {
		select {
		case <-h.context.Done():
			return
		default:
			// slog.Debug("clearing throttle...", "val", h.throttle)
			clear(h.throttle)
			time.Sleep(time.Duration(time.Second * 5))
		}
	}
}

func (h *NuesApi) canCall(r *http.Request) bool {

	ip := GetClientIpAddr(r)

	count, found := h.throttle[ip]
	if !found {
		h.throttle[ip] = 1
	} else {
		h.throttle[ip] = count + 1
	}

	can := h.throttle[ip] < h.reqPerSec*5
	slog.Info("throttle", "attempts", h.throttle[ip], "can", can)
	return can
}

func (h *NuesApi) config() {

	slog.Info("Runing API server configuration")
	http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {

		fullpath := r.URL.Path
		slog.Debug("API call", "path", fullpath)
		var parts []string
		var found bool
		var path string
		var route Route
		var auth bool
		var response any
		var err error
		var callId string
		var called bool
		var token string
		var cookie *http.Cookie

		if !h.canCall(r) {
			http.Error(w, "rate limit exceeded", http.StatusForbidden)
			return
		}

		if r.Method != http.MethodPost {
			goto abort
		}

		if len(fullpath) < 1 {
			goto abort
		}
		fullpath, found = strings.CutPrefix(fullpath, "/")
		if !found {
			goto abort
		}
		parts = strings.Split(fullpath, "/")
		if len(parts) < 2 {
			goto abort
		}
		path = parts[1]
		route, found = nues.Routes[path]
		if !found {
			goto abort
		}
		slog.Info("serving route", "path", path, "route", route)
		cookie, _ = r.Cookie("token")
		if cookie != nil {
			token = cookie.Value
		}
		if token == "" {
			token = r.Header.Get("token")
		}
		auth = authCall(token, route)
		if !auth {
			goto notAuthed
		}

		callId = r.Header.Get("callId")

		if callId != "" {
			// try call history
			var call bson.M
			err := DB.GetCollection(nues.colCommands).FindOne(context.TODO(), bson.M{"_id": callId}).Decode(&call)
			if err != nil && err != mongo.ErrNoDocuments {
				goto abort
			}
			if call != nil {
				//Idempotency detected
				called = true
				response = call["response"]
			}
		}
		if !called {
			response, err = h.httpServe(route, r)
		}

		if err != nil {
			slog.Error("http failed", "err", err)
			goto errored
		} else {
			var responseB []byte
			responseB, err = json.Marshal(response)
			if err != nil {
				goto abort
			}
			w.Header().Add("content-type", "application/json; charset=utf-8")
			_, err = w.Write(responseB)
			if err != nil {
				slog.Error("http response error", "err", err)
			}
		}
		return

	notAuthed:
		http.Error(w, "not logged in", http.StatusUnauthorized)
		return
	errored:
		http.Error(w, "can't read your request!", http.StatusBadRequest)
		return
	abort:
		http.NotFound(w, r)
		return

	})
}

func (h *NuesApi) Close() error {

	return h.server.Close()
}

func (h *NuesApi) Serve(ctx context.Context) {
	h.context = ctx
	h.config()
	h.throttle = make(map[string]int)
	if h.reqPerSec == 0 {
		h.reqPerSec = 2
	}

	go h.clearthrottle()
	h.server = &http.Server{
		Addr:           nues.ApiPort,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	slog.Info("starting API Server ...", "port", nues.ApiPort)

	err := h.server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
