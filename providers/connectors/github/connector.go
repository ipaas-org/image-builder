package github

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	trasportHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/ipaas-org/image-builder/providers/connectors"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

const (
	MetaDescription   connectors.MetaType = "description" //description will come with defaul_branch
	MetaBranches      connectors.MetaType = "branches"
	MetaDefaultBranch connectors.MetaType = "default_branch" //default branch will come with description
	MetaTags          connectors.MetaType = "tags"
	MetaReleases      connectors.MetaType = "releases"

	branchesBaseUrl = "https://api.github.com/repos/%s/%s/branches"
	baseUrlMetadata = "https://api.github.com/repos/%s/%s"
	baseUrlTag      = "https://api.github.com/repos/%s/%s/tags"
	baseUrlRelease  = "https://api.github.com/repos/%s/%s/releases"
)

var (
	ErrInvalidUrl      = errors.New("invalid url, check if the url is correct or if the repo is not private")
	ErrMissingRepoName = errors.New("invalid url, missing repository name")
	ErrMissingUsername = errors.New("invalid url, missing username")
	ErrGithubRateLimit = errors.New("github api rate limit exceeded")
)

var _ connectors.Connector = new(GithubConnector)

type GithubConnector struct {
	l                 *logrus.Logger
	userAgent         string
	downloadDirectory string
}

func NewGithubConnector(downloadDirectory, userAgent string, l *logrus.Logger) *GithubConnector {
	return &GithubConnector{
		l:                 l,
		userAgent:         userAgent,
		downloadDirectory: downloadDirectory,
	}
}

func (g GithubConnector) checkPrefix(url string) bool {
	return strings.HasPrefix(url, "https://github.com") ||
		strings.HasPrefix(url, "http://github.com") ||
		strings.HasPrefix(url, "https://www.github.com") ||
		strings.HasPrefix(url, "http://www.github.com")
}

// ValidateAndLintUrl check if an url is a valid and existing GitHub repo url
func (g GithubConnector) ValidateAndLintUrl(url, token string) (string, error) {
	//sanitize the url
	// defer fmt.Println()
	g.l.Debugf("validating url: %s", url)
	url = strings.TrimSpace(url)
	url = strings.ToLower(url)

	if !strings.Contains(url, "/") {
		return "", ErrInvalidUrl
	}

	urlSplit := strings.Split(url, "/")
	// g.l.Debugf("url split: %v", urlSplit)
	if len(urlSplit) != 2 {
		return "", ErrInvalidUrl
	}
	if urlSplit[len(urlSplit)-1] == "" || urlSplit[len(urlSplit)-1] == "github.com" {
		return "", ErrMissingRepoName
	}

	if urlSplit[len(urlSplit)-2] == "" || urlSplit[len(urlSplit)-2] == "github.com" {
		return "", ErrMissingUsername
	}

	url = "https://github.com/" + url

	// if !g.checkPrefix(url) {
	// 	if strings.HasPrefix(url, "github.com") {
	// 		url = "https://" + url
	// 	} else {
	// 		url = "https://github.com/" + url
	// 	}
	// }

	g.l.Debugf("url after sanitization: %s", url)

	user, repo, err := g.GetUserAndRepo(url, token)
	if err != nil {
		return "", err
	}
	g.l.Debug(url)
	repoUrl := fmt.Sprintf(baseUrlMetadata, user, repo)
	request, err := http.NewRequest("GET", repoUrl, nil)
	if err != nil {
		return "", err
	}

	request.Header.Set("User-Agent", g.userAgent)
	request.Header.Set("Authorization", "token "+token)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	jsonBody := string(body)

	if resp.StatusCode != 200 {
		if resp.StatusCode == 403 {
			g.l.Errorf("githubConnector.ValidateAndLintUrl: github api rate limit exceeded: %v", jsonBody)
			return "", ErrGithubRateLimit
		} else if resp.StatusCode == 404 {
			g.l.Warnf("%s is not a valid url", url)
			return "", ErrInvalidUrl
		}
		g.l.Errorf("githubConnector.ValidateAndLintUrl: error getting info for %s [%s]: %v", url, resp.Status, jsonBody)
		return "", fmt.Errorf("error getting info for %s [%s]: %v", url, resp.Status, jsonBody)
	}

	return url, nil
}

// GetUserAndRepo get the username of the creator and the repository's name given a GitHub repository url
func (g GithubConnector) GetUserAndRepo(url, token string) (string, string, error) {
	url = strings.TrimSuffix(url, ".git")
	split := strings.Split(url, "/")
	if len(split) < 2 {
		return "", "", fmt.Errorf("invalid url: %s", url)
	}
	return split[len(split)-2], split[len(split)-1], nil
}

// Pull clones the repository from GitHub given the url and save it in the download directory,
// if the download successfully complete the name of the path, name and last commit hash will be returned
func (g GithubConnector) Pull(userID, branch, url, token string) (string, string, string, error) {
	var err error
	url, err = g.ValidateAndLintUrl(url, token)
	if err != nil {
		return "", "", "", err
	}

	//get the name of the repo
	user, repoName, err := g.GetUserAndRepo(url, token)
	if err != nil {
		g.l.Errorf("githubConnector.Pull: error getting user and repo: %v", err)
		return "", "", "", err
	}

	if branch == "" {
		branch, _, err = g.getBranchAndDescription(user, repoName, token)
		if err != nil {
			g.l.Errorf("githubConnector.Pull: error getting branch and description: %v", err)
			return "", "", "", err
		}
	}
	g.l.Infof("user: %s, repo: %s, branch: %s\n", user, repoName, branch)

	branches, err := g.getBranches(user, repoName, token)
	if err != nil {
		return "", "", "", err
	}
	var branchFound bool
	for _, b := range branches {
		if b == branch {
			branchFound = true
			break
		}
	}
	if !branchFound {
		return "", "", "", fmt.Errorf("branch %s not found", branch)
	}
	//get the repo name
	tmpPath := fmt.Sprintf("%s/%s-%s-%s", g.downloadDirectory, userID, repoName, branch)
	err = os.Mkdir(tmpPath, os.ModePerm)
	if err != nil {
		return "", "", "", fmt.Errorf("error creating the tmp folder: %v", err)
	}
	g.l.Infof("downloading repo in %s...", tmpPath)

	r, err := git.PlainClone(tmpPath, false, &git.CloneOptions{
		URL:           url,
		Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		Auth: &trasportHttp.BasicAuth{
			Username: user,
			Password: token,
		},
		// Progress: os.Stdout,
	})
	if err != nil {
		g.l.Errorf("githubConnector.Pull: error cloning the repo: %v", err)
		return "", "", "", err
	}
	g.l.Info("ok")

	logs, err := r.Log(&git.LogOptions{})
	if err != nil {
		return "", "", "", err
	}
	defer logs.Close()

	//get the last commit hash
	g.l.Info("getting last commit hash...")
	commitHash, err := logs.Next()
	if err != nil {
		g.l.Errorf("githubConnector.Pull: error getting the last commit hash: %v", err)
		return "", "", "", err
	}
	g.l.Infoln("ok")

	//remove the .git folder
	g.l.Info("removing .git...")
	if err := os.RemoveAll(fmt.Sprintf("%s/.git", tmpPath)); err != nil {
		g.l.Errorf("githubConnector.Pull: error removing .git folder: %v", err)
		return "", "", "", err
	}
	g.l.Infoln("ok")
	return tmpPath, repoName, commitHash.Hash.String(), nil
}

// GetMetadata gets the description, default branch and all the branches of a GitHub repository
// if meta is not nil then only the specified metadata will be returned
func (g GithubConnector) GetMetadata(url, token string, meta ...connectors.MetaType) (map[connectors.MetaType][]string, error) {
	metaInfo := make(map[connectors.MetaType][]string)
	username, repoName, err := g.GetUserAndRepo(url, token)
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}

	var defaultBranch, description string
	var branches, tags, releases []string
	errch := make(chan error, 1)
	defer close(errch)

	if len(meta) == 0 {
		wg.Add(4)
		go func() {
			defer wg.Done()
			var err error
			defaultBranch, description, err = g.getBranchAndDescription(username, repoName, token)
			if err != nil {
				errch <- err
			}
		}()

		go func() {
			defer wg.Done()
			var err error
			branches, err = g.getBranches(username, repoName, token)
			if err != nil {
				errch <- err
			}
			if len(branches) == 0 {
				branches = []string{defaultBranch}
			}
		}()

		go func() {
			defer wg.Done()
			var err error
			tags, err = g.getTags(username, repoName, token)
			if err != nil {
				errch <- err
			}
		}()

		go func() {
			defer wg.Done()
			var err error
			releases, err = g.getReleases(username, repoName, token)
			if err != nil {
				errch <- err
			}
		}()
	} else {
		for _, m := range meta {
			switch m {
			case MetaDefaultBranch:
				if description != "" {
					continue
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					var err error
					defaultBranch, description, err = g.getBranchAndDescription(username, repoName, token)
					if err != nil {
						errch <- err
					}
				}()
			case MetaDescription:
				if defaultBranch != "" {
					continue
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					var err error
					defaultBranch, description, err = g.getBranchAndDescription(username, repoName, token)
					if err != nil {
						errch <- err
					}
				}()
			case MetaBranches:
				wg.Add(1)
				go func() {
					defer wg.Done()
					var err error
					branches, err = g.getBranches(username, repoName, token)
					if err != nil {
						errch <- err
					}
					if len(branches) == 0 {
						branches = []string{defaultBranch}
					}
				}()
			case MetaTags:
				wg.Add(1)
				go func() {
					defer wg.Done()
					var err error
					tags, err = g.getTags(username, repoName, token)
					if err != nil {
						errch <- err
					}
				}()
			case MetaReleases:
				wg.Add(1)
				go func() {
					defer wg.Done()
					var err error
					releases, err = g.getReleases(username, repoName, token)
					if err != nil {
						errch <- err
					}
				}()
			}
		}
	}

	wg.Wait()
	select {
	case err := <-errch:
		return nil, err
	default:
	}

	metaInfo[MetaDefaultBranch] = []string{defaultBranch}
	metaInfo[MetaDescription] = []string{description}
	metaInfo[MetaReleases] = releases
	metaInfo[MetaTags] = tags
	metaInfo[MetaBranches] = branches

	return metaInfo, nil
}

func (g GithubConnector) getBranches(username, repo, token string) ([]string, error) {
	//get the branches
	request, err := http.NewRequest("GET", fmt.Sprintf(branchesBaseUrl, username, repo), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", g.userAgent)
	request.Header.Set("Authorization", "token "+token)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	jsonBody := string(body)

	if resp.StatusCode != 200 {
		if resp.StatusCode == 403 {
			g.l.Errorf("githubConnector.getBranches: github api rate limit exceeded: %v", jsonBody)
			return nil, fmt.Errorf("github api rate limit exceeded")
		}
		g.l.Errorf("githubConnector.getBranches: error finding release info for %s/%s [%s]: %v", username, repo, resp.Status, jsonBody)
		return nil, fmt.Errorf("error finding branches info for %s/%s [%s]: %v", username, repo, resp.Status, jsonBody)
	}

	branchesRes := gjson.Get(jsonBody, "@this.#.name").Array()
	branches := make([]string, len(branchesRes))
	for i, r := range branchesRes {
		branches[i] = r.String()
	}
	return branches, nil
}

func (g GithubConnector) getBranchAndDescription(username, repo, token string) (string, string, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf(baseUrlMetadata, username, repo), nil)
	if err != nil {
		return "", "", err
	}

	request.Header.Set("User-Agent", g.userAgent)
	request.Header.Set("Authorization", "token "+token)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	jsonBody := string(body)

	if resp.StatusCode != 200 {
		if resp.StatusCode == 403 {
			g.l.Errorf("githubConnector.getBranchAndDescription: github api rate limit exceeded: %v", jsonBody)
			return "", "", fmt.Errorf("github api rate limit exceeded")
		}
		g.l.Errorf("githubConnector.getBranchAndDescription: error finding release info for %s/%s [%s]: %v", username, repo, resp.Status, jsonBody)
		return "", "", fmt.Errorf("error finding general info for %s/%s [%s]: %v", username, repo, resp.Status, jsonBody)
	}

	return gjson.Get(jsonBody, "default_branch").String(), gjson.Get(jsonBody, "description").String(), nil
}

func (g GithubConnector) getTags(username, repo, token string) ([]string, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf(baseUrlTag, username, repo), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", g.userAgent)
	request.Header.Set("Authorization", "token "+token)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	jsonBody := string(body)

	if resp.StatusCode != 200 {
		if resp.StatusCode == 403 {
			g.l.Errorf("githubConnector.getTags: github api rate limit exceeded: %v", jsonBody)
			return nil, fmt.Errorf("github api rate limit exceeded")
		}
		g.l.Errorf("githubConnector.getTags: error finding release info for %s/%s [%s]: %v", username, repo, resp.Status, jsonBody)
		return nil, fmt.Errorf("error finding tags info for %s/%s [%s]: %v", username, repo, resp.Status, jsonBody)
	}

	tagsRes := gjson.Get(jsonBody, "@this.#.name").Array()
	tags := make([]string, len(tagsRes))
	for i, r := range tagsRes {
		tags[i] = r.String()
	}
	return tags, nil
}

func (g GithubConnector) getReleases(username, repo, token string) ([]string, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf(baseUrlRelease, username, repo), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", g.userAgent)
	request.Header.Set("Authorization", "token "+token)
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	jsonBody := string(body)

	if resp.StatusCode != 200 {
		if resp.StatusCode == 403 {
			g.l.Errorf("githubConnector.getReleases: github api rate limit exceeded: %v", jsonBody)
			return nil, fmt.Errorf("github api rate limit exceeded")
		}
		g.l.Errorf("githubConnector.getReleases: error finding release info for %s/%s [%s]: %v", username, repo, resp.Status, jsonBody)
		return nil, fmt.Errorf("error finding release info for %s/%s [%s]: %v", username, repo, resp.Status, jsonBody)
	}

	releasesRes := gjson.Get(jsonBody, "@this.#.name").Array()
	releases := make([]string, len(releasesRes))
	for i, r := range releasesRes {
		releases[i] = r.String()
	}
	return releases, nil
}
