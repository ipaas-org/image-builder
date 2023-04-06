package registry

import "context"

type Registryer interface {
	TagImage(localImageID, newName string) (string, error)
	PushImage(ctx context.Context, localImageID string) error
}
