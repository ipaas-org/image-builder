package connectors

type Connector interface {
	Pull(userID, branch, url, token string) (path, repoName, lastCommit string, err error)
	GetUserAndRepo(url, token string) (username string, repoName string, err error)
	ValidateAndLintUrl(url, token string) (linted string, err error)
	GetMetadata(url, token string) (metadata map[string][]string, err error)
}
