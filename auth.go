package nues

import (
	"context"
	"slices"

	"go.mongodb.org/mongo-driver/bson"
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

func initAuth() {

	// register service as identity
	identity := Identity{
		Name:            nues.ServiceId,
		IdentityId:      nues.ServiceId,
		AllowedServices: map[string][]string{},
	}
	err := RegisterNewIdentity(identity)
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

	_, err := DB.GetCollection(nues.colIdentity).UpdateOne(context.TODO(), bson.M{"_id": identity.IdentityId}, bson.M{"$set": identity}, options.Update().SetUpsert(true))

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
	err := DB.GetCollection(nues.colSessions).FindOne(context.TODO(), bson.M{"token": token}).Decode(session)
	if err != nil || session == nil {
		return false
	}
	var identity *Identity
	err = DB.GetCollection(nues.colIdentity).FindOne(context.TODO(), bson.M{"_id": session.IdentityId}).Decode(identity)
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
