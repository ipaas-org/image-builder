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
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/vano2903/image-builder/providers/connectors"
)

const (
	DescriptionMeta   = "description"
	BranchesMeta      = "branches"
	DefaultBranchMeta = "default_branch"
	TagsMeta          = "tags"
	ReleasesMeta      = "releases"
)

var _ connectors.Connector = new(GithubConnector)

type GithubConnector struct {
	l                 *logrus.Logger
	userAgent         string
	token             string
	downloadDirectory string
}

func NewGithubConnector(downloadDirectory, userAgent, token string, l *logrus.Logger) *GithubConnector {
	return &GithubConnector{
		l:                 l,
		token:             token,
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
func (g GithubConnector) ValidateAndLintUrl(url string) (string, error) {
	//sanitize the url
	g.l.Debugf("validating url: %s", url)
	url = strings.TrimSpace(url)
	url = strings.ToLower(url)

	if !strings.Contains(url, "/") {
		return "", errors.New("invalid url")
	}

	urlSplit := strings.Split(url, "/")
	if urlSplit[len(urlSplit)-1] == "" || urlSplit[len(urlSplit)-1] == "github.com" {
		return "", errors.New("invalid url, missing repository name")
	}

	if urlSplit[len(urlSplit)-2] == "" || urlSplit[len(urlSplit)-2] == "github.com" {
		return "", errors.New("invalid url, missing username")
	}

	//we allow users to not specify the github.com part of the url
	//so vano2903/ipaas is a valid url
	if !g.checkPrefix(url) {
		if strings.HasPrefix(url, "github.com") {
			url = "https://" + url
		} else {
			url = "https://github.com/" + url
		}
	}

	g.l.Debug(url)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	request.Header.Set("User-Agent", g.userAgent)
	request.Header.Set("Authorization", "token "+g.token)
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
			g.l.Errorf("githubConnector.getReleases: github api rate limit exceeded: %v", jsonBody)
			return "", fmt.Errorf("github api rate limit exceeded")
		} else if resp.StatusCode == 404 {
			g.l.Warnf("%s is not a valid url", url)
			return "", errors.New("invalid url, check if the url is correct or if the repo is not private")
		}
		g.l.Errorf("githubConnector.getReleases: error getting info for %s [%s]: %v", url, resp.Status, jsonBody)
		return "", fmt.Errorf("error getting info for %s [%s]: %v", url, resp.Status, jsonBody)
	}

	return url, nil
}

// GetUserAndRepo get the username of the creator and the repository's name given a GitHub repository url
func (g GithubConnector) GetUserAndRepo(url string) (string, string, error) {
	url, err := g.ValidateAndLintUrl(url)
	if err != nil {
		return "", "", err
	}

	url = strings.TrimSuffix(url, ".git")
	split := strings.Split(url, "/")

	return split[len(split)-2], split[len(split)-1], nil
}

// Pull clones the repository from GitHub given the url and save it in the download directory,
// if the download successfully complete the name of the path, name and last commit hash will be returned
func (g GithubConnector) Pull(userID, branch, url string) (string, string, string, error) {
	var err error
	url, err = g.ValidateAndLintUrl(url)
	if err != nil {
		return "", "", "", err
	}

	//get the name of the repo
	g.l.Info("getting repo name...")
	user, repoName, err := g.GetUserAndRepo(url)
	if err != nil {
		g.l.Errorf("githubConnector.Pull: error gettingerr")
		return "", "", "", err
	}

	if branch == "" {
		branch, _, err = g.getBranchAndDescription(user, repoName)
		if err != nil {
			return "", "", "", err
		}
	}
	fmt.Println("ok")
	fmt.Println("repo name:", repoName)
	fmt.Println("user:", user)
	fmt.Println("branch:", branch)

	branches, err := g.getBranches(user, repoName)
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
	fmt.Printf("downloading repo in %s...", tmpPath)

	r, err := git.PlainClone(tmpPath, false, &git.CloneOptions{
		URL:           url,
		Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		// Progress: os.Stdout,
	})
	if err != nil {
		fmt.Println("err")
		return "", "", "", err
	}
	fmt.Println("ok")

	logs, err := r.Log(&git.LogOptions{})
	if err != nil {
		return "", "", "", err
	}
	defer logs.Close()

	//get the last commit hash
	fmt.Print("getting last commit hash...")
	commitHash, err := logs.Next()
	if err != nil {
		fmt.Println("err")
		return "", "", "", err
	}
	fmt.Println("ok")

	//remove the .git folder
	fmt.Print("removing .git...")
	if err := os.RemoveAll(fmt.Sprintf("%s/.git", tmpPath)); err != nil {
		fmt.Println("err")
		return "", "", "", err
	}
	fmt.Println("ok")
	return tmpPath, repoName, commitHash.Hash.String(), nil
}

// GetMetadata gets the description, default branch and all the branches of a GitHub repository
func (g GithubConnector) GetMetadata(url string) (map[string][]string, error) {
	meta := make(map[string][]string)
	username, repoName, err := g.GetUserAndRepo(url)
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}

	var defaultBranch, description string
	var branches, tags, releases []string

	errch := make(chan error, 1)
	wg.Add(4)
	go func() {
		defer wg.Done()
		var err error
		defaultBranch, description, err = g.getBranchAndDescription(username, repoName)
		if err != nil {
			errch <- err
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		branches, err = g.getBranches(username, repoName)
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
		tags, err = g.getTags(username, repoName)
		if err != nil {
			errch <- err
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		releases, err = g.getReleases(username, repoName)
		if err != nil {
			errch <- err
		}
	}()

	wg.Wait()
	select {
	case err := <-errch:
		return nil, err
	default:
	}
	close(errch)

	meta[DefaultBranchMeta] = []string{defaultBranch}
	meta[DescriptionMeta] = []string{description}
	meta[ReleasesMeta] = releases
	meta[TagsMeta] = tags
	meta[BranchesMeta] = branches

	return meta, nil
}

func (g GithubConnector) getBranches(username, repo string) ([]string, error) {
	//get the branches
	baseUrl := "https://api.github.com/repos/%s/%s/branches"
	request, err := http.NewRequest("GET", fmt.Sprintf(baseUrl, username, repo), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", g.userAgent)
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

func (g GithubConnector) getBranchAndDescription(username, repo string) (string, string, error) {
	baseUrl := "https://api.github.com/repos/%s/%s"
	request, err := http.NewRequest("GET", fmt.Sprintf(baseUrl, username, repo), nil)
	if err != nil {
		return "", "", err
	}

	request.Header.Set("User-Agent", g.userAgent)
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

func (g GithubConnector) getTags(username, repo string) ([]string, error) {
	baseUrl := "https://api.github.com/repos/%s/%s/tags"
	request, err := http.NewRequest("GET", fmt.Sprintf(baseUrl, username, repo), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", g.userAgent)
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

func (g GithubConnector) getReleases(username, repo string) ([]string, error) {
	baseUrl := "https://api.github.com/repos/%s/%s/releases"
	request, err := http.NewRequest("GET", fmt.Sprintf(baseUrl, username, repo), nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", g.userAgent)
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
