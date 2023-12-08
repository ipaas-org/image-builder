package controller

import "github.com/ipaas-org/image-builder/model"

func (b *Controller) PullRepo(info *model.BuildRequest) (*model.PulledRepoInfo, error) {
	if info.Token == "" {
		return nil, ErrEmptyToken
	}

	connector, ok := b.connectors[info.Connector]
	if !ok {
		return nil, ErrConnectorNotFound
	}

	b.l.Debugf("info received: %+v", info)
	b.l.Infof("%s is pulling %s at branch %s", info.UserID, info.Repo, info.Branch)

	pullInfo := new(model.PulledRepoInfo)
	var err error
	pullInfo.Path, pullInfo.RepoName, pullInfo.LastCommit, err = connector.Pull(info.UserID, info.Branch, info.Repo, info.Token)
	if err != nil {
		b.l.Errorf("error pulling %s: %v", info.Repo, err)
		return nil, err
	}
	b.l.Infof("pulled %s successfully in %q", info.Repo, pullInfo.Path)
	return pullInfo, nil
}
