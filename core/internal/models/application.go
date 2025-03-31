package models

type Application struct {
	Id             int64  `json:"id"`
	Name           string `json:"name"`
	OwnerId        string `json:"ownerId"`
	RepositoryType string `json:"repoType"`
}
