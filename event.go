package nues

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var EvAttemptName string = "EvAttempt"

type EvAttempt struct {
	EvName  string
	Command interface{}
}

var evMutex sync.Mutex

type Event struct {
	Id        string      `bson:"_id"`
	Name      string      `bson:"name"`
	Sequence  int64       `bson:"sequence"`
	Timestamp time.Time   `bson:"timestamp"`
	Data      interface{} `bson:"data"`
}

func (e *Event) save(ctx context.Context) error {
	defer evMutex.Unlock()
	evMutex.Lock()

	if len(e.Id) == 0 {
		return fmt.Errorf("event id is empty")
	}
	if len(e.Name) == 0 {
		return fmt.Errorf("event name is empty")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("event tiemstamp is not valid")
	}

	_, err := DB.GetCollection(nues.colEvents).InsertOne(ctx, e)

	return err
}

func RegisterEvents(ctx context.Context, evs ...interface{}) error {

	last := GetLastSequence()
	for _, ev := range evs {
		last = last + 1
		evName := reflect.TypeOf(ev).Name()
		if len(evName) == 0 {
			panic("unknown event name")
		}
		e := &Event{
			Id:        GenerateId(),
			Name:      evName,
			Sequence:  last,
			Timestamp: time.Now(),
			Data:      ev,
		}
		if err := e.save(ctx); err != nil {
			slog.Error("event save failed", err)
			return ErrSystemInternal
		}
	}
	return nil

}

func GetLastSequence() int64 {
	defer evMutex.Unlock()
	evMutex.Lock()

	var res bson.M
	err := DB.GetCollection(nues.colEvents).FindOne(context.TODO(), bson.D{}, options.FindOne().SetProjection(bson.D{{"sequence", 1}}).SetSort(bson.D{{"sequence", -1}})).Decode(&res)
	if err != nil {
		switch err {
		case mongo.ErrNoDocuments:
			return 0
		default:
			panic(err)
		}
	}
	return res["sequence"].(int64)
}
