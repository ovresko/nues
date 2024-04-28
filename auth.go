package nues

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IdentityAccess map[string][]string

type Identity struct {
	Id     string `bson:"_id"`
	Name   string
	Access IdentityAccess
}

type Session struct {
	Id    string `bson:"_id"`
	Token string
}

var db *mongo.Database

func initAuth() {
	if nues.IdentityDbUri == "" {
		slog.Error("You must set IdentityDbUri")
		panic("IdentityDbUri uri")
	}
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(nues.IdentityDbUri))
	if err != nil {
		panic(err)
	}

	db = client.Database(nues.IdentityDbName)

	if nues.Reset {
		db.Drop(context.TODO())
	}

}

func AuthCall(token string, route Route) bool {

	if route.Name == "" {
		panic("route name is required")
	}
	if route.Public {
		return true
	}
	if token == "" {
		return false
	}

	if nues.AdminToken == token {
		return true
	}

	var session *Session
	err := db.Collection(fmt.Sprintf("%s_%s", nues.DbPrefix, nues.ColSession)).FindOne(context.TODO(), bson.M{"token": token}).Decode(session)
	if err != nil || session == nil {
		return false
	}
	var identity *Identity
	err = db.Collection(fmt.Sprintf("%s_%s", nues.DbPrefix, nues.ColIdentity)).FindOne(context.TODO(), bson.M{"_id": session.Id}).Decode(identity)
	if err != nil || identity == nil {
		return false
	}

	access, found := identity.Access[nues.ServiceId]
	if !found {
		return false
	}
	// full service access
	return len(access) == 0 || slices.Contains(access, route.Name) || (len(access) == 1 && access[0] == "*")

}
