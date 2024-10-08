package controller

import (
	"context"

	"github.com/ipaas-org/image-builder/model"
)

func (b *Controller) PullRepo(ctx context.Context, info *model.PullInfoRequest) (*model.PulledRepoInfo, error) {
	if info.Token == "" {
		return nil, ErrEmptyToken
	}

	connector, ok := b.connectors[info.Connector]
	if !ok {
		return nil, ErrConnectorNotFound
	}

	b.l.Debugf("info received: %+v", info)
	b.l.Infof("%s is pulling %s at branch %s", info.UserID, info.Repo, info.Branch)

	pullInfo, err := connector.Pull(ctx, info.UserID, info.Branch, info.Repo, info.Commit, info.Token)
	if err != nil {
		b.l.Errorf("error pulling %s: %v", info.Repo, err)
		return nil, err
	}
	b.l.Infof("pulled %s successfully in %q", info.Repo, pullInfo.Path)
	return pullInfo, nil
}
