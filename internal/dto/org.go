package dto

import "time"

type Org struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	OwnerId   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateOrgRequest struct {
	Name    string `json:"name"`
	OwnerId string `json:"owner_id"`
}
