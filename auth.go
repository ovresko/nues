package nues

import (
	"context"
	"fmt"
	"slices"
	"strings"

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
	IdentityId string `validate:"required" bson:"_id" json:"identity_id"`
	Token      string `validate:"required" json:"token"`
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

func ClearSessions(identityId string) error {
	_, err := DB.GetCollection(nues.colSessions).DeleteMany(context.TODO(), bson.M{"_id": identityId})
	return err

}
func RegisterNewSession(identityId string) (*Session, error) {
	if identityId == "" {
		return nil, NewError(-1, "identity id is required")
	}
	var identity Identity
	err := DB.GetCollection(nues.colIdentity).FindOne(context.TODO(), bson.M{"_id": identityId}).Decode(&identity)
	if err != nil {

		return nil, err
	}
	if identity.IdentityId == "" {
		return nil, ErrIdentityNotFound
	}
	session := &Session{
		IdentityId: identityId,
		Token:      fmt.Sprintf("%s:%s", identityId, GenerateId()),
	}
	err = validate.Struct(session)
	if err != nil {
		return nil, err
	}
	_, err = DB.GetCollection(nues.colSessions).InsertOne(context.TODO(), session)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			err = DB.GetCollection(nues.colSessions).FindOne(context.TODO(), bson.M{"_id": identityId}).Decode(session)
			if err != nil {
				return nil, ErrSystemInternal
			}
		} else {
			return nil, ErrSystemInternal
		}
	}
	return session, err
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

func authCall(headerToken string, route Route) bool {

	if route.Name == "" {
		panic("route name is required")
	}
	if route.Public {
		return true
	}
	if headerToken == "" {
		return false
	}

	if nues.AdminToken == headerToken {
		return true
	}

	parts := strings.Split(headerToken, ":")
	if len(parts) != 2 {
		return false
	}
	identityId := parts[0]
	token := parts[1]

	var session *Session
	err := DB.GetCollection(nues.colSessions).FindOne(context.TODO(), bson.M{"token": token, "_id": identityId}).Decode(session)
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
