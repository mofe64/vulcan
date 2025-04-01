package models

type Auth struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	Firstname    string `json:"firstname"`
	Lastname     string `json:"lastname"`
}
