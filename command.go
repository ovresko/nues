package nues

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CommandResponse any

var validate = validator.New(validator.WithRequiredStructEnabled())

type Command interface {
	Handle(context.Context) (CommandResponse, error)
}

type CommandRoot struct {
	Response CommandResponse `json:"response"`
	Executed bool            `json:"executed"`
	Error    SysError        `json:"error"`
	Command  Command         `json:"command"`
	Ts       string          `json:"ts"`
	CallId   string          `json:"callId"`
}

func (cr *CommandRoot) validate() SysError {
	err := validate.Struct(cr.Command)
	if err != nil {
		var errMsg string
		for _, err := range err.(validator.ValidationErrors) {

			errMsg = fmt.Sprintf("%s\n%s", errMsg, fmt.Sprintf("%s %s condition failed.", err.Field(), err.Tag()))

		}
		return NewError(-1, errMsg)
	}
	return nil
}

func (cr *CommandRoot) Execute(ctx context.Context) {

	start := time.Now()
	defer func() {
		cr.Ts = time.Since(start).String()
	}()

	err := cr.validate()
	if err != nil {
		cr.Error = err
		return
	}
	session, err := DB.Client().StartSession()

	if err != nil {
		slog.Error("command session creation error", err)
		cr.Error = ErrSystemInternal
		return
	}
	if err = session.StartTransaction(); err != nil {
		slog.Error("command session creation error", err)
		cr.Error = ErrSystemInternal
		return
	}

	defer session.EndSession(context.TODO())
	ctxMongo := mongo.NewSessionContext(ctx, session)
	cr.Response, cr.Error = cr.Command.Handle(ctxMongo)
	if cr.Error == nil {
		// validate response
		err = validate.Struct(cr.Response)
		if err != nil {
			var errMsg string
			for _, err := range err.(validator.ValidationErrors) {
				errMsg = fmt.Sprintf("%s\n%s", errMsg, fmt.Sprintf("%s %s condition failed.", err.Field(), err.Tag()))
			}
			cr.Error = NewError(-1, errMsg)
		}
	}
	if cr.Error != nil {
		slog.Error("command error", cr.Error)
		evName := reflect.TypeOf(cr.Command).Elem()
		slog.Debug("fail attempt", "CMD", evName)
		evAttempt := EvAttempt{
			Command: cr,
			EvName:  evName.Name(),
		}
		if err := RegisterEvents(nil, evAttempt); err != nil {
			slog.Error("attempt event register failed", err)
			cr.Error = ErrSystemInternal
			return
		}

		if err := session.AbortTransaction(context.TODO()); err != nil {
			slog.Error("can't abort transaction", err)
			cr.Error = ErrSystemInternal
			return
		}
	} else {
		cr.Executed = true
		if cr.CallId != "" {
			// save command result for Idempotent check
			_, err := DB.GetCollection(nues.colCommands).InsertOne(context.TODO(), bson.M{"_id": cr.CallId, "response": cr, "date": time.Now()})
			if err != nil {
				cr.Error = ErrSystemInternal
				cr.Executed = false
				return
			}
		}
		if err := session.CommitTransaction(context.TODO()); err != nil {
			slog.Error("error commiting transaction", err)
			cr.Error = ErrSystemInternal
			cr.Executed = false
			return
		}

	}

}
