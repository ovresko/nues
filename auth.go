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

type Identity struct {
	IdentityId      string `bson:"_id"`
	Name            string
	AllowedServices map[string][]string
}

type Session struct {
	IdentityId string `bson:"_id"`
	Token      string
}

var db *mongo.Database
var identityCol string
var sessionCol string

func initAuth() {

	if nues.IdentityDbUri == "" {
		slog.Error("You must set IdentityDbUri")
		panic("IdentityDbUri uri")
	}
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(nues.IdentityDbUri))
	if err != nil {
		panic(err)
	}

	identityCol = fmt.Sprintf("%s_%s", nues.DbPrefix, nues.ColIdentity)
	sessionCol = fmt.Sprintf("%s_%s", nues.DbPrefix, nues.ColSession)

	db = client.Database(nues.IdentityDbName)

	if nues.Reset {
		db.Drop(context.TODO())
	}

	// register service as identity
	identity := Identity{
		Name:            nues.ServiceId,
		IdentityId:      nues.ServiceId,
		AllowedServices: map[string][]string{},
	}
	err = RegisterNewIdentity(identity)
	if err != nil {
		panic(err)
	}
}

func RegisterNewIdentity(identity Identity) error {

	if err := AssertNotEmpty(identity.IdentityId, NewError(-1, "identity id is required")); err != nil {
		return err
	}

	if err := AssertNotEmpty(identity.Name, NewError(-1, "identity name is required")); err != nil {
		return err
	}

	_, err := db.Collection(identityCol).UpdateOne(context.TODO(), bson.M{"_id": identity.IdentityId}, identity, options.Update().SetUpsert(true))

	if err != nil {
		return err
	}
	return nil
}

func authCall(token string, route Route) bool {

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
	err := db.Collection(sessionCol).FindOne(context.TODO(), bson.M{"token": token}).Decode(session)
	if err != nil || session == nil {
		return false
	}
	var identity *Identity
	err = db.Collection(identityCol).FindOne(context.TODO(), bson.M{"_id": session.IdentityId}).Decode(identity)
	if err != nil || identity == nil {
		return false
	}

	if len(identity.AllowedServices) == 0 {
		return true
	}
	access, found := identity.AllowedServices[nues.ServiceId]
	if !found {
		return false
	}
	// full service access
	return len(access) == 0 || slices.Contains(access, route.Name) || (len(access) == 1 && access[0] == "*")

}
