package model

/*
{
	"applicationID":"id dell'applicazione da builder (per aggiornare lo stato)"
	"pullInfo":{
		"userID":"id dell'utente"
		"token":"per fare la pull"
		"repo":"url della repo"
		"connector":"connettore per la pull: github"
		"branch":"branch della repo"
	}
	"buildPlan":{
		"builder":"dockerfile|nixpacks"
		"rootDirectory":"path in cui fare la build (/ di default, può essere /backend)"

		SE BUILDER DOCKERFILE
		"dockerfilePath":"path del dockerfile (se builder è dockerfile)"

		SE BUILDER NIXPACKS
			"nixpacksPath":"path del nixpacks.toml file (builder nixpacks)"
			"envs":{ ENVS INJECTED IN THE BUILD PROCESS
				"key":"value"
			}
			"nixPkgs":"nixpkgs to add to the final image (added in setup phase)"
			"nixLibs":"nix libs to add to the final image (added in setup phase)"
			"aptPkgs":"base image is ubuntu, it will add the apt packages in the setup phase"
			"installCommand":"command to execute in the install phase"
			"buildCommand": "command to execute in the build phase"
			"startCommand": "command to execute in the start phase"
	}
}
*/

type (
	KeyValue struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	Request struct {
		ApplicationID string           `json:"applicationID"`
		PullInfo      *PullInfoRequest `json:"pullInfo"`
		BuildPlan     *BuildConfig     `json:"buildPlan"`
	}

	BuildConfig struct {
		// must
		RootDirectory string `json:"rootDirectory"`

		// shared
		Builder      BuilderKind `json:"builder"`
		StartCommand string      `json:"startCommand"`

		// docker
		DockerfilePath string `json:"dockerfilePath"`

		// nixpacks
		NixpacksPath   string     `json:"nixpacksPath"`
		Envs           []KeyValue `json:"envs"`
		NixPkgs        []string   `json:"nixPkgs"`
		NixLibs        []string   `json:"nixLibs"`
		AptPkgs        []string   `json:"aptPkgs"`
		InstallCommand string     `json:"installCommand"`
		BuildCommand   string     `json:"buildCommand"`
	}

	PullInfoRequest struct {
		UserID    string `json:"userID"`
		Token     string `json:"token"`
		Repo      string `json:"repo"`
		Connector string `json:"connector"`
		Branch    string `json:"branch"`
		Commit    string `json:"commit"` //commit to build, use latest to build the latest commit
	}

	PulledRepoInfo struct {
		Path         string
		RepoName     string
		PulledCommit string
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
	RegistryHarbor = "harbor"
)
