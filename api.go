package nues

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type RouteResponse map[string]any
type RouteAuthType int
type RouteCallType int
type Routes map[string]Route

const (
	ROUTE_PUBLIC RouteAuthType = iota
	ROUTE_USER
	ROUTE_ADMIN

	COMMAND RouteCallType = iota
	QUERY
	HANDLER
)

type Route struct {
	auth    RouteAuthType
	call    RouteCallType
	handler func() any
}

type NuesApi struct {
	context   context.Context
	throttle  map[string]int
	reqPerSec int
}

func Ping(context.Context, map[string]any) RouteResponse {
	return RouteResponse{"response": true}
}

func (h *NuesApi) httpServe(ctx context.Context, route Route, r *http.Request) (any, error) {

	body, err := io.ReadAll(r.Body)

	if err != nil {
		return nil, ErrSystemInternal
	}

	switch route.call {
	case HANDLER:
		handler, ok := route.handler().(func(context.Context, map[string]any) RouteResponse)
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

		res := handler(ctx, reqBody)
		return res, nil

	case COMMAND:
		var cmdClone = route.handler()
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
		cmdRoot.Execute(ctx)
		return cmdRoot, nil

	case QUERY:
		var queryClone = route.handler()
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
		queryRoot.Execute(ctx)
		return queryRoot, nil
	}
	return nil, ErrSystemInternal

}

func (h *NuesApi) httpAuth(route Route, r *http.Request) bool {

	switch route.auth {
	case ROUTE_PUBLIC:
		slog.Info("PUBLIC route")
		return true
	case ROUTE_USER:
		slog.Info("USER", "path", route)
		return h.AuthUser(r)
	case ROUTE_ADMIN:
		slog.Info("ADMIN", "path", route)
		return h.AuthAdmin(r)
	}

	return false
}

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
	http.HandleFunc("/api/*", func(w http.ResponseWriter, r *http.Request) {

		ctx := context.Background()
		fullpath := r.URL.Path
		var parts []string
		var found bool
		var path string
		var route Route
		var auth bool
		var response any
		var err error
		var callId string
		var called bool

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

		auth = h.httpAuth(route, r)
		if !auth {
			goto notAuthed
		}

		callId = r.Header.Get("callId")

		if callId != "" {
			// try call history
			var call bson.M
			err := DB.GetCollection(nues.ColCommands).FindOne(context.TODO(), bson.M{"_id": callId}).Decode(&call)
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
			response, err = h.httpServe(ctx, route, r)
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

func (h *NuesApi) Serve() {
	h.config()
	ctxCancel, cancelListener := context.WithCancel(context.TODO())
	h.context = ctxCancel
	h.throttle = make(map[string]int)
	if h.reqPerSec == 0 {
		h.reqPerSec = 2
	}

	go h.clearthrottle()
	srv := &http.Server{
		Addr:           nues.Port,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		// service connections
		slog.Info("starting Server ...")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen: %s\n", err)
			panic("server error")
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutdown server ...")
	cancelListener()

	ctx, cancel := context.WithTimeout(context.Background(), 0*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown Error:", err)
		panic("server error")
	}
	// catching ctx.Done(). timeout of 5 seconds.
	<-ctx.Done()
	slog.Info("server exiting")

}

func (h *NuesApi) AuthAdmin(r *http.Request) bool {
	sysToken := os.Getenv("ADMIN_TOKEN")
	var authToken = ""
	cookie, _ := r.Cookie("token")
	if cookie == nil {
		authToken = r.Header.Get("token")
	}
	if authToken == "" {
		return false
	}

	if err := AssertNotEmpty(authToken, ErrUserNotAuth); err != nil {
		return false
	}
	if err := AssertNotEmpty(sysToken, ErrUserNotAuth); err != nil {
		return false
	}
	if err := AssertTrue(authToken == sysToken, ErrUserNotAuth); err != nil {
		return false
	}

	return true
}

func (h *NuesApi) AuthUser(r *http.Request) bool {
	var session bson.M
	var userid = ""
	var token = ""
	var vals []string
	var authToken = ""
	cookie, _ := r.Cookie("token")
	if cookie == nil {
		authToken = r.Header.Get("token")
	}
	if authToken == "" {
		return false
	}

	vals = strings.Split(authToken, ":")
	if err := AssertTrue(len(vals) == 2, ErrUserNotAuth); err != nil {
		return false
	}
	userid = vals[0]
	token = vals[1]

	if err := DB.GetCollection(nues.ColSession).FindOne(context.TODO(), bson.D{{"token", token}, {"userid", userid}}).Decode(&session); err != nil {
		if err != nil || session == nil {
			return false
		}
	}
	return true

}
