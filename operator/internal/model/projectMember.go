package model

type ProjectMember struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	Email     string `json:"email"`
}
