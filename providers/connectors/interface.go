package connectors

type Connector interface {
	Pull(userID, branch, url string) (path, repoName, lastCommit string, err error)
	GetUserAndRepo(url string) (username string, repoName string, err error)
	ValidateAndLintUrl(url string) (linted string, err error)
	GetMetadata(url string) (metadata map[string][]string, err error)
}