package nues

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Database struct {
	*mongo.Database
	Bus chan Event
}

var DB *Database

var watchMutex sync.Mutex = sync.Mutex{}

func (d *Database) Events() *mongo.Collection {
	return d.GetCollection(nues.colEvents)
}

func (d *Database) WatchEvents(eventName string, callback func(Event) error) error {

	var changeEvent struct {
		ResumeAfter   bson.M `bson:"_id"`
		OperationType string `bson:"operationType"`
		FullDocument  bson.M `bson:"fullDocument"`
	}

	pipe := bson.D{{"$match", bson.D{{"operationType", "insert"}, {"fullDocument.name", eventName}}}}

	var resumeAfter bson.M
	err := d.GetCollection(nues.colWatchers).FindOne(context.TODO(), bson.M{"_id": eventName}).Decode(&resumeAfter)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			slog.Error("watcher failed for event", "event", eventName, "error", err)
			return err
		}
		resumeAfter = bson.M{"_id": eventName, "resume": nil}
		_, err := d.GetCollection(nues.colWatchers).InsertOne(context.TODO(), resumeAfter)
		if err != nil {
			slog.Error("watcher failed to insert watcher doc", "event", eventName, "error", err)
			return err
		}
	}
	st, err := d.GetCollection(nues.colEvents).Watch(context.TODO(), mongo.Pipeline{
		pipe,
	}, options.ChangeStream().SetFullDocument(options.UpdateLookup).SetResumeAfter(resumeAfter["resume"]))

	if err != nil {
		return err
	}

	// get last watch date

	go func() {
		defer st.Close(context.TODO())

		for {
			err := d.Client().Ping(context.Background(), readpref.Primary())
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			break

		}

		for {
			available := st.Next(context.TODO())
			if available {

				if err := st.Decode(&changeEvent); err != nil {
					slog.Error("watch decode failed", err)
					panic(err)
				}

				var ev Event
				pl, err := bson.Marshal(changeEvent.FullDocument)
				if err != nil {
					slog.Error("watch decode failed", err)

					continue
				}
				bson.Unmarshal(pl, &ev)
				slog.Debug("event", "op", ev.Name)
				watchMutex.Lock()
				err = callback(ev)
				if err != nil {
					slog.Error("watcher callback failed", "ev", eventName, "seq", ev, "err", err)
				} else {
					DB.GetCollection(nues.colWatchers).UpdateOne(context.TODO(), bson.M{"_id": eventName},
						bson.D{
							{"$set", bson.M{"resume": changeEvent.ResumeAfter, "changed": time.Now()}}},
					)
				}
				watchMutex.Unlock()
			} else {
				if err := st.Err(); err != nil {
					if errors.Is(err, mongo.ErrClientDisconnected) {
						slog.Info("watcher stopping...", "eventName", eventName)
						return
					}

					slog.Error("watcher db error", err)
					panic(err)

				}
			}
		}
	}()

	return nil

}

// func getInternalDb() *Database {
// 	configDb := "sabil"
// 	db, err := InitNewDb(nues.DbUri, configDb, false)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return db
// }

func GetConfig[T ConfigService](doPanic bool) *T {

	var config *T
	err := DB.Collection("__config").FindOne(context.TODO(), bson.M{"_id": (*config).Name()}).Decode(config)
	if err != nil {
		if doPanic {
			panic(err)
		}
		return nil
	}
	return config
}

func (d *Database) GetCollection(col string) *mongo.Collection {
	if col == "" {
		panic("no collection should be empty, something is seriously wrong")
	}
	return d.Collection(fmt.Sprintf("%s_%s", nues.dbPrefix, col))
}

func (d *Database) GetOne(Collection, field, value string) (*bson.M, error) {
	doc := &bson.M{}
	err := d.GetCollection(Collection).FindOne(context.TODO(), bson.D{{field, value}}).Decode(doc)
	return doc, err
}

func (d *Database) Upsert(Collection, field, value string, doc interface{}) error {
	if value == "" {
		slog.Info("DB Upsert value is empty %s: field %s ", Collection, field)
		return ErrSystemInternal
	}
	res, err := d.GetCollection(Collection).ReplaceOne(context.TODO(), bson.D{{field, value}}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		slog.Error("upsert failed", err)
		return err
	}
	if res.ModifiedCount+res.UpsertedCount == 0 {
		slog.Error("upsert failed", "Collection", Collection, "doc", doc)
		return ErrUpsertFailed
	}
	return nil
}

func (d *Database) Replace(Collection, field, value string, doc interface{}) error {
	if value != "" {
		d.GetCollection(Collection).DeleteMany(context.TODO(), bson.D{{field, value}})
	} else {
		slog.Info("DB Replace value is empty %s: field %s ", Collection, field)
	}
	return d.Upsert(Collection, field, value, doc)
}

func (d *Database) GetValue(Collection, field string, value interface{}, proj string) (interface{}, error) {
	doc := bson.M{}
	err := d.GetCollection(Collection).FindOne(context.TODO(), bson.D{{field, value}}, options.FindOne().SetProjection(bson.D{{proj, 1}})).Decode(&doc)
	if err != nil {
		return nil, err
	}
	val := doc[proj]
	return val, err
}

func (d *Database) SetValue(Collection, id, field string, value interface{}) (interface{}, error) {

	res, err := d.GetCollection(Collection).UpdateOne(context.TODO(), bson.D{{"_id", id}}, bson.D{{"$set", bson.D{{field, value}}}})
	if err != nil {
		return nil, err
	}
	val := res.ModifiedCount
	return val, err
}

func (d *Database) Projections() *mongo.Collection {
	return d.GetCollection(nues.colProjections)
}
func (d *Database) Disconnect() error {
	return d.Client().Disconnect(context.TODO())
}

func (d *Database) AddIndex(field string, collection string) error {
	index := mongo.IndexModel{
		Keys: bson.D{
			{field, 1},
		},
	}
	_, err := d.GetCollection(collection).Indexes().CreateOne(context.Background(), index)
	if err != nil {
		return err
	}
	return nil
}

func initNuesDb() {

	var err error
	DB, err = InitNewDb(nues.DbUri, nues.DbName, nues.reset)
	if err != nil {
		panic(err)
	}

}

func InitNewDb(dbUri, dbName string, reset bool) (*Database, error) {

	if dbUri == "" {
		slog.Error("You must set your 'MONGODB_URI' environment variable. See\n\t https://www.mongodb.com/docs/drivers/go/current/usage-examples/#environment-variable")
		return nil, NewError(-1, "dburi required")
	}
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(dbUri))
	if err != nil {
		return nil, err
	}

	_DB := &Database{
		Database: client.Database(dbName),
		Bus:      make(chan Event),
	}

	if reset {
		_DB.Drop(context.TODO())
	}

	return _DB, nil
}

func initNuesIndexes() {

	index := mongo.IndexModel{
		Keys: bson.M{"name": 1},
	}
	_, err := DB.GetCollection(nues.colEvents).Indexes().CreateOne(context.Background(), index)
	if err != nil {
		slog.Error("create index failed", err)
		panic(err)
	}

	// index sb_commands
	commandsIndexOptions := options.Index().SetExpireAfterSeconds(10 * 60)
	commandsIndex := mongo.IndexModel{
		Keys:    bson.M{"date": 1},
		Options: commandsIndexOptions,
	}
	_, err = DB.GetCollection(nues.colEvents).Indexes().CreateOne(context.Background(), commandsIndex)
	if err != nil {
		slog.Error("create index failed", err)
		panic(err)
	}

}
