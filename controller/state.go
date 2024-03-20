package controller

import (
	"context"

	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/repo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (c *Controller) ShouldBuild(ctx context.Context, applicationID string) (bool, error) {
	c.l.Println("ShouldBuild")
	c.l.Printf("applicationID: %q\n", applicationID)
	appID, err := primitive.ObjectIDFromHex(applicationID)
	if err != nil {
		c.l.Errorf("invalid applicationID")
		return false, err
	}

	state, err := c.ApplicationRepo.GetStateByID(ctx, appID)
	if err != nil {
		c.l.Errorf("error getting application state: %v", err)
		if err == repo.ErrNotFound {
			return false, nil
		} else {
			return false, err
		}
	}
	if state == "deleting" {
		return false, nil
	}
	return true, nil
}

func (c *Controller) UpdateApplicationStateToBuilding(ctx context.Context, applicationID string) error {
	appID, err := primitive.ObjectIDFromHex(applicationID)
	if err != nil {
		return err
	}
	_, err = c.ApplicationRepo.UpdateStateByID(ctx, model.ApplicationStateBuilding, appID)
	return err
}

func (c *Controller) UpdateApplicationStateToFailed(ctx context.Context, applicationID string) error {
	appID, err := primitive.ObjectIDFromHex(applicationID)
	if err != nil {
		return err
	}
	_, err = c.ApplicationRepo.UpdateStateByID(ctx, model.ApplicationStateFailed, appID)
	return err
}
