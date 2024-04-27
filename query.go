package nues

import (
	"context"
	"log/slog"
	"reflect"
	"time"
)

type QueryResponse map[string]interface{}

type Query interface {
	Handle(context.Context) (QueryResponse, error)
}

type QueryRoot struct {
	Response QueryResponse `json:"response"`
	Executed bool          `json:"executed"`
	Err      error         `json:"error"`
	Query    Query         `json:"query"`
	Ts       string        `json:"ts"`
}

func (q *QueryRoot) Execute(ctx context.Context) {

	start := time.Now()

	q.Response, q.Err = q.Query.Handle(ctx)
	q.Ts = time.Since(start).String()
	slog.Debug("QUERY", "target", reflect.TypeOf(q.Query).Elem(), "ts", q.Ts)
	q.Executed = true
	if q.Err != nil {
		slog.Error("query failed", q.Err)
	}

}
