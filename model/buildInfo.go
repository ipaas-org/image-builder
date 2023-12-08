package model

type (
	BuildRequest struct {
		ApplicationID string `json:"applicationID"`
		Token         string `json:"token"`
		UserID        string `json:"userID"`
		Type          string `json:"type"`      // repo, tag, release, ...
		Connector     string `json:"connector"` //github, gitlab, ...
		Repo          string `json:"repo,omitempty"`
		Branch        string `json:"branch,omitempty"`
		Tag           string `json:"tag,omitempty"`
		Release       string `json:"release,omitempty"`
		// Binary     string `json:"binary, omitempty"`
	}

	PulledRepoInfo struct {
		Path       string
		RepoName   string
		LastCommit string
	}
)

const (
	TypeRepo    = "repo"
	TypeTag     = "tag"
	TypeRelease = "release"
	// TypeBinary  = "binary"

	ConnectorGithub = "github"

	DownloaderNixpacks = "nixpacks"

	RegistryDocker = "docker"
)
