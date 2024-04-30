package nues

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var projMutex map[string]*sync.Mutex = make(map[string]*sync.Mutex)

type ProjectionRoot struct {
	Id       string    `bson:"_id"`
	Sequence int64     `bson:"sequence"`
	Modified time.Time `bson:"modified"`
}

type Projection interface {
	Name() string
	Update(events []Event) (int64, error)
	Steams() []string
	CreateIndexes() error
}

func GetProjectionFirst[T Projection](pipeline mongo.Pipeline) (*T, error) {
	pipeline = append(pipeline, bson.D{{"$limit", 1}})
	res, err := GetProjection[T](pipeline)
	if err != nil {
		return nil, err
	}
	if len(res) > 0 {
		return &res[0], nil
	}
	return nil, nil
}
func UpdateProjection[T Projection](id string, valuesMap interface{}, upsert bool) error {

	if err := AssertNotEmpty(id, ErrMissingReuiredFields); err != nil {
		slog.Error("project update failed due to missing id")
		return ErrMissingReuiredFields
	}

	err := validate.Struct(valuesMap)
	if err != nil {
		slog.Error("saving projection failed", "err", err)
		return err
	}

	var p T
	res, err := DB.GetCollection(p.Name()).UpdateOne(context.TODO(), bson.M{"_id": id}, bson.M{"$set": valuesMap}, options.Update().SetUpsert(upsert))
	if err != nil {
		slog.Error("updating projection failed", err)
		return err
	}
	if !upsert && res.MatchedCount == 0 {
		slog.Error("updating projection failed, not matching docs", "doc", valuesMap)
		return ErrProjectionFailed
	}
	if upsert && (res.UpsertedCount+res.MatchedCount) == 0 {
		slog.Error("upserting projection failed, not upserted docs", "doc", valuesMap)
		return ErrProjectionFailed
	}

	return nil

}

func BuildProjection[T Projection]() error {
	var p T

	proj := &ProjectionRoot{}
	if err := DB.GetCollection(nues.colProjections).FindOne(context.TODO(), bson.D{{"_id", p.Name()}}).Decode(proj); err != nil {
		slog.Error("no projection exist, creating new projection", "proj", p.Name(), "error", err)
		if err == mongo.ErrNoDocuments {
			// no projection, so save one and remove collection
			DB.GetCollection(p.Name()).Drop(context.TODO())
			proj = &ProjectionRoot{
				Id:       p.Name(),
				Sequence: 0,
				Modified: time.Now(),
			}
			if err := DB.Upsert(nues.colProjections, "_id", proj.Id, proj); err != nil {
				slog.Error("upsert new projection failed", err)
				return err
			}

			err = p.CreateIndexes()
			if err != nil {
				slog.Error("index creation faild", err)
				panic(err)
			}

		} else {
			slog.Error("error", err)
			return err

		}
	}

	projSeq := proj.Sequence
	streamQuery := buildStreamQuery(p.Steams())
	lastSeq, err := getLastSeq(streamQuery)
	if err != nil {
		slog.Error("error getting last sequence", err)
		return err
	}

	if lastSeq > projSeq {
		// we need to update
		eventsQuery := append(streamQuery, bson.D{{"sequence", bson.D{{"$gt", projSeq}}}}...)
		cur, err := DB.GetCollection(nues.colEvents).Find(context.TODO(), eventsQuery, options.Find().SetSort(bson.D{{"sequence", 1}}))
		if err != nil && err != mongo.ErrNoDocuments {
			slog.Error("error", err)
			return err
		}
		events := []Event{}
		err = cur.All(context.TODO(), &events)
		if err != nil {
			slog.Error("error", err)
			return err
		}
		seq, err := p.Update(events)
		// update project
		_, errUpdate := DB.SetValue(nues.colProjections, proj.Id, "sequence", seq)
		if err != nil {
			slog.Error("updating projection failed", "err", err)
			return err
		}
		if errUpdate != nil {
			slog.Error("error", errUpdate)
			return errUpdate
		}
		_, errUpdate = DB.SetValue(nues.colProjections, proj.Id, "modified", time.Now())
		if errUpdate != nil {
			slog.Error("error", errUpdate)
			return errUpdate
		}

	}
	return nil
}

func GetProjection[T Projection](pipeline mongo.Pipeline) ([]T, error) {

	var p T
	var m *sync.Mutex = &sync.Mutex{}
	if f, s := projMutex[p.Name()]; s {
		m = f
	} else {
		projMutex[p.Name()] = m
	}
	m.Lock()
	defer m.Unlock()
	err := BuildProjection[T]()
	if err != nil {
		slog.Error("get projection faild", "err", err, "pipline", pipeline)
		return nil, err
	}

	result := []T{}
	cur, err := DB.GetCollection(p.Name()).Aggregate(context.TODO(), pipeline)
	if err != nil {
		slog.Error("", err)
		if err == mongo.ErrNoDocuments {
			return result, nil
		}
		return nil, ErrSystemInternal
	}

	err = cur.All(context.TODO(), &result)

	if err != nil {
		slog.Error("", err)
		return nil, err
	}

	return result, nil
}

func buildStreamQuery(streams []string) bson.D {

	ora := bson.A{}
	for _, st := range streams {
		q := bson.D{
			{"name", st},
		}
		ora = append(ora, q)
	}

	return bson.D{{"$or", ora}}
}

func getLastSeq(query bson.D) (int64, error) {

	seq := bson.M{}
	err := DB.GetCollection(nues.colEvents).FindOne(context.TODO(), query, options.FindOne().SetSort(bson.D{{"sequence", -1}}).SetProjection(bson.D{{"sequence", 1}})).Decode(seq)
	if err == mongo.ErrNoDocuments {
		slog.Error("error", err)
		return 0, nil
	}
	s, ok := seq["sequence"].(int64)
	if !ok {
		panic("wrong sequence type")
	}
	return int64(s), err
}
