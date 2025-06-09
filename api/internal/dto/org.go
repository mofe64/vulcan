package dto

import "time"

type Org struct {
	Id         string    `json:"id"`
	Name       string    `json:"name"`
	OwnerId    string    `json:"owner_id"`
	OwnerEmail string    `json:"owner_email"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateOrgRequest struct {
	Name       string `json:"name"`
	OwnerId    string `json:"owner_id"`
	OwnerEmail string `json:"owner_email"`
}
