package models

type Link interface {
	GetRepoLink() string
	GetRepoName() string
	GetRepoType() string
}

type GithubLink struct {
	RepoLink string `json:"repoLink"`
	RepoName string `json:"repoName"`
	RepoType string `json:"repoType"`
}

func (g *GithubLink) GetRepoLink() string {
	return g.RepoLink
}

func (g *GithubLink) GetRepoName() string {
	return g.RepoName
}

func (g *GithubLink) GetRepoType() string {
	return g.RepoType
}
