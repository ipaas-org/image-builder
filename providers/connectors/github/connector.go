package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	trasportHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/ipaas-org/image-builder/model"
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
	ErrInvalidUrl         = errors.New("invalid url, check if the url is correct or if the repo is not private")
	ErrMissingRepoName    = errors.New("invalid url, missing repository name")
	ErrMissingUsername    = errors.New("invalid url, missing username")
	ErrGithubRateLimit    = errors.New("github api rate limit exceeded")
	ErrUnauthorizedAccess = errors.New("unauthorized access")
	ErrCommitNotFound     = errors.New("commit not found")
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

// ValidateAndLintUrl check if an url is a valid and existing GitHub repo url
func (g GithubConnector) ValidateAndLintUrl(ctx context.Context, url, token string) (string, error) {
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

	user, repo, err := g.GetUserAndRepo(ctx, url, token)
	if err != nil {
		return "", err
	}
	g.l.Debug(url)
	repoUrl := fmt.Sprintf(baseUrlMetadata, user, repo)
	request, err := http.NewRequestWithContext(ctx, "GET", repoUrl, nil)
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
		switch resp.StatusCode {
		case 403:
			g.l.Errorf("githubConnector.ValidateAndLintUrl: github api rate limit exceeded: %v", jsonBody)
			return "", ErrGithubRateLimit
		case 404:
			g.l.Warnf("%s is not a valid url", url)
			return "", ErrInvalidUrl
		case 401:
			g.l.Errorf("githubConnector.ValidateAndLintUrl: unauthorized access: %v", jsonBody)
			return "", ErrUnauthorizedAccess
		default:
			g.l.Errorf("githubConnector.ValidateAndLintUrl: error getting info for %s [%s]: %v", url, resp.Status, jsonBody)
			return "", fmt.Errorf("error getting info for %s [%s]: %v", url, resp.Status, jsonBody)
		}
	}

	return url, nil
}

// GetUserAndRepo get the username of the creator and the repository's name given a GitHub repository url
func (g GithubConnector) GetUserAndRepo(ctx context.Context, url, token string) (string, string, error) {
	url = strings.TrimSuffix(url, ".git")
	split := strings.Split(url, "/")
	if len(split) < 2 {
		return "", "", fmt.Errorf("invalid url: %s", url)
	}
	return split[len(split)-2], split[len(split)-1], nil
}

// Pull clones the repository from GitHub given the url and save it in the download directory,
// if the download successfully complete the name of the path, name and last commit hash will be returned
func (g GithubConnector) Pull(ctx context.Context, userID, branch, url, commitHash, token string) (*model.PulledRepoInfo, error) {
	var err error
	url, err = g.ValidateAndLintUrl(ctx, url, token)
	if err != nil {
		return nil, err
	}

	//get the name of the repo
	user, repoName, err := g.GetUserAndRepo(ctx, url, token)
	if err != nil {
		g.l.Errorf("githubConnector.Pull: error getting user and repo: %v", err)
		return nil, err
	}

	if branch == "" {
		branch, err = g.getDefaultBranch(ctx, user, repoName, token)
		if err != nil {
			g.l.Errorf("githubConnector.Pull: error getting branch and description: %v", err)
			return nil, err
		}
	}
	g.l.Infof("user: %s, repo: %s, branch: %s\n", user, repoName, branch)

	branches, err := g.getBranches(ctx, user, repoName, token)
	if err != nil {
		return nil, err
	}
	var branchFound bool
	for _, b := range branches {
		if b == branch {
			branchFound = true
			break
		}
	}
	if !branchFound {
		return nil, fmt.Errorf("branch %s not found", branch)
	}
	//get the repo name
	tmpPath := fmt.Sprintf("%s/%s-%s-%s", g.downloadDirectory, userID, repoName, branch)

	if err := os.Mkdir(tmpPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("error creating the tmp folder: %v", err)
	}
	g.l.Infof("downloading repo in %s...", tmpPath)

	r, err := git.PlainClone(tmpPath, false, &git.CloneOptions{
		URL: url,
		// Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		Auth: &trasportHttp.BasicAuth{
			Username: user,
			Password: token,
		},
	})
	if err != nil {
		g.l.Errorf("githubConnector.Pull: error cloning the repo: %v", err)
		return nil, err
	}

	if commitHash == "latest" || commitHash == "" {
		g.l.Info("getting the latest commit hash...")
		head, err := r.Head()
		if err != nil {
			return nil, err
		}
		commitHash = head.Hash().String()
	} else {
		w, err := r.Worktree()
		if err != nil {
			g.l.Errorf("Failed to open work tree for repository: %v", err)
			return nil, err
		}

		commit, err := r.CommitObject(plumbing.NewHash(commitHash))
		if err != nil {
			if err == plumbing.ErrObjectNotFound {
				return nil, ErrCommitNotFound
			}
			return nil, err
		}
		err = w.Reset(&git.ResetOptions{Mode: git.HardReset, Commit: commit.Hash})
		if err != nil {
			g.l.Errorf("Failed to hard reset work tree: %v", err)
			return nil, err
		}
		g.l.Info("Hard reset successful, confirming changes....")
		commitHash = commit.Hash.String()
	}

	if err := os.RemoveAll(fmt.Sprintf("%s/.git", tmpPath)); err != nil {
		g.l.Errorf("githubConnector.Pull: error removing .git folder: %v", err)
		return nil, err
	}

	return &model.PulledRepoInfo{
		Path:         tmpPath,
		PulledCommit: commitHash,
		RepoName:     url,
	}, nil
}

func (g GithubConnector) getDefaultBranch(ctx context.Context, username, repo, token string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(baseUrlMetadata, username, repo), nil)
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
			g.l.Errorf("githubConnector.getBranchAndDescription: github api rate limit exceeded: %v", jsonBody)
			return "", fmt.Errorf("github api rate limit exceeded")
		}
		g.l.Errorf("githubConnector.getBranchAndDescription: error finding release info for %s/%s [%s]: %v", username, repo, resp.Status, jsonBody)
		return "", fmt.Errorf("error finding general info for %s/%s [%s]: %v", username, repo, resp.Status, jsonBody)
	}

	return gjson.Get(jsonBody, "default_branch").String(), nil
}

func (g GithubConnector) getBranches(ctx context.Context, username, repo, token string) ([]string, error) {
	//get the branches
	request, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(branchesBaseUrl, username, repo), nil)
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
