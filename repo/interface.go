package repo

import (
	"context"
	"errors"

	"github.com/ipaas-org/image-builder/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ApplicationRepoer interface {
	UpdateStateByID(ctx context.Context, state model.ApplicationState, id primitive.ObjectID) (bool, error)
	GetStateByID(ctx context.Context, id primitive.ObjectID) (model.ApplicationState, error)
}

var (
	ErrNotFound error = errors.New("not found")
)
