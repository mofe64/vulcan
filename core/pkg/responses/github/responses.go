package responses_github

type Repository struct {
	Name  string `json:"name"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type Branch struct {
	Name string `json:"name"`
}

type WebhookResponse struct {
	ID int `json:"id"`
}
