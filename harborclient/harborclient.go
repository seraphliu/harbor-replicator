package harborclient

import (
	"fmt"
	"github.com/json-iterator/go"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"time"
)

type Event struct {
	Repo string
	Tag  string
}

type RepoStatus map[string]time.Time // map[tag]update_time

type HarborCLI struct {
	url      string
	scheme   string
	user     string
	pass     string
	projects []string
	repos    map[string]*RepoStatus
}

func NewHarborClient(url, user, pass, project string, insecure bool) *HarborCLI {
	scheme := "https"
	if insecure {
		scheme = "http"
	}
	return &HarborCLI{
		url:      url,
		scheme:   scheme,
		user:     user,
		pass:     pass,
		projects: []string{project},
		repos:    map[string]*RepoStatus{},
	}
}

func (h *HarborCLI) GetRepoNames() []string {
	keys := make([]string, 0, len(h.repos))
	for k := range h.repos {
		keys = append(keys, k)
	}
	return keys
}

func (h *HarborCLI) queryAPI(url string) ([]byte, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "new http request error")
	}
	req.SetBasicAuth(h.user, h.pass)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http request error")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read response error")
	}
	return body, nil
}

func (h *HarborCLI) getProjectRepos(project string) ([]string, error) {
	var (
		listProjectUrl = "%s://%s/api/projects"
		listRepoUrl    = "%s://%s/api/repositories?project_id=%d"
	)

	url := fmt.Sprintf(listProjectUrl, h.scheme, h.url)
	res, err := h.queryAPI(url)
	if err != nil {
		return nil, err
	}

	data := jsoniter.Get(res, '*')

	id := -1
	for i := 0; i < data.Size(); i++ {
		p := data.Get(i)
		if p.Get("name").ToString() == project {
			id = p.Get("project_id").ToInt()
		}
	}
	if id < 0 {
		return nil, fmt.Errorf("can't find project %s in harbor", project)
	}

	url = fmt.Sprintf(listRepoUrl, h.scheme, h.url, id)
	res, err = h.queryAPI(url)
	if err != nil {
		return nil, err
	}
	//fmt.Println(string(res))

	data = jsoniter.Get(res, '*', "name")

	//fmt.Println(data.ToString())
	var repos []string
	for i := 0; i < data.Size(); i++ {
		r := data.Get(i).ToString()
		repos = append(repos, r)
	}

	return repos, nil
}

func (h *HarborCLI) RefreshRepoTags(repo string) (RepoStatus, error) {
	var (
		listTagUrl = "%s://%s/api/repositories/%s/tags"
	)

	url := fmt.Sprintf(listTagUrl, h.scheme, h.url, repo)
	res, err := h.queryAPI(url)
	if err != nil {
		return nil, err
	}

	data := jsoniter.Get(res, '*')

	last := *h.repos[repo]
	update := RepoStatus{}
	for i := 0; i < data.Size(); i++ {
		t := data.Get(i)
		tag := t.Get("name").ToString()
		ctime, err := time.Parse(time.RFC3339Nano, t.Get("created").ToString())
		if err != nil {
			return nil, errors.Wrap(err, "parse image created time error")
		}
		if v, ok := last[tag]; !ok || ctime.After(v) {
			update[tag] = ctime
			last[tag] = ctime
		}
	}

	return update, nil
}

func (h *HarborCLI) RefreshRepos() error {
	var err error
	for _, project := range h.projects {
		repos, err := h.getProjectRepos(project)
		if err != nil {
			err = fmt.Errorf("get project repos error: %v", err)
			continue
		}
		for _, r := range repos {
			if _, ok := h.repos[r]; !ok {
				h.repos[r] = &RepoStatus{}
			}
		}
	}
	return err
}
