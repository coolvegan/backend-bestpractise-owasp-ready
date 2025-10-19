package models

type ErrUserLogin struct {
	Message string `json:"message"`
}

type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
