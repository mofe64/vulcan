package models

type Application struct {
	Id             int64          `json:"id"`
	OwnerId        int64          `json:"ownerId"`
	Name           string         `json:"name"`
	SourceCodeLink SourceCodeLink `json:"sourceCodeLink"`
}

type SourceCodeLink struct {
	RepoUrl           string `json:"repoUrl"`
	RepoName          string `json:"repoName"`
	RepoType          string `json:"repoType"`
	RepoBranchName    string `json:"repoBranchName"`
	WebhookIdentifier string `json:"webhookIdentifier"`
}
