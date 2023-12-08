package mongo

import (
	"context"

	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/repo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewApplicationRepoer(collection *mongo.Collection) repo.ApplicationRepoer {
	return &ApplicationRepoerMongo{
		collection: collection,
	}
}

type ApplicationRepoerMongo struct {
	collection *mongo.Collection
}

func (r *ApplicationRepoerMongo) UpdateStateByID(ctx context.Context, state model.ApplicationState, _id primitive.ObjectID) (bool, error) {
	result, err := r.collection.UpdateOne(ctx, bson.M{
		"_id": _id,
	}, bson.M{
		"$set": bson.M{
			"state": state,
		},
	})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, repo.ErrNotFound
		}
		return false, err
	}
	return result.MatchedCount > 0, err
}

func (r *ApplicationRepoerMongo) GetStateByID(ctx context.Context, id primitive.ObjectID) (model.ApplicationState, error) {
	type State struct {
		State string `bson:"state"`
	}
	var state State
	err := r.collection.FindOne(ctx, bson.M{
		"_id": id,
	}, &options.FindOneOptions{
		Projection: bson.M{
			"state": 1,
			"_id":   0,
		},
	}).Decode(&state)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", repo.ErrNotFound
		}
		return "", err
	}
	return model.ApplicationState(state.State), nil
}
