package model

type (
	RepoAnalisys struct {
		IsBuildable bool   `json:"isBuildable"`
		Reason      string `json:"reason"` // Reason why the repo is not buildable
		RepoInfo    *DetectedInfo
	}

	DetectedInfo struct {
		Builders []BuilderKind `json:"builders"`
		Docker   *DockerInfo   `json:"docker,omitempty"`
		NixPacks *NixPacksInfo `json:"nixpacks,omitempty"`
	}

	DockerInfo struct {
		DockerIgnoreFound bool     `json:"dockerIngoreFound"` //true if .dockerignore is found
		Dockerfiles       []string `json:"dockerfiles"`       //path to the detected dockerfiles
	}

	NixPacksInfo struct {
		NixPacksConfigPath string            `json:"nixPacksConfigPath"` //path to nixpacks.[json|toml]
		NixPacksProviders  []string          `json:"providers"`          //detected nixpack
		NixPackages        []string          `json:"nixPackages" `
		NixLibraries       []string          `json:"nixLibraries"`
		AptPackages        []string          `json:"aptPackages"`
		InstallCommands    []string          `json:"installCommands"`
		BuildCommands      []string          `json:"buildCommands"`
		StartCommand       string            `json:"startCommand"`
		Variables          map[string]string `json:"variables"`
	}

	BuilderKind string
)
