package nues

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/go-playground/validator/v10"
)

type QueryResponse map[string]interface{}

type Query interface {
	Handle(context.Context) (QueryResponse, error)
}

type QueryRoot struct {
	Response QueryResponse `json:"response"`
	Executed bool          `json:"executed"`
	Error    SysError      `json:"error"`
	Query    Query         `json:"query"`
	Ts       string        `json:"ts"`
}

func (cr *QueryRoot) validate() SysError {
	err := validate.Struct(cr.Query)
	if err != nil {
		slog.Error("query validate failed", "err", err)
		var errMsg string
		for _, err := range err.(validator.ValidationErrors) {

			errMsg = fmt.Sprintf("%s\n%s", errMsg, fmt.Sprintf("%s %s condition failed.", err.Field(), err.Tag()))

		}
		return NewError(-1, errMsg)
	}
	return nil
}

func (q *QueryRoot) Execute(ctx context.Context) {

	start := time.Now()

	slog.Debug("validating query")
	err := q.validate()
	if err != nil {
		q.Error = err
		return
	}

	q.Response, q.Error = q.Query.Handle(ctx)
	q.Ts = time.Since(start).String()
	slog.Debug("QUERY", "target", reflect.TypeOf(q.Query).Elem(), "ts", q.Ts)
	q.Executed = true
	if q.Error != nil {
		slog.Error("query failed", q.Error)
	}

}
