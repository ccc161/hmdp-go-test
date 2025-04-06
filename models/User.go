package models

import "fmt"

type User struct {
	ID    int    `json:"id"`
	Phone string `json:"phone"`
	Auth  string `json:"auth"`
}

func (u User) String() string {
	return fmt.Sprintf("id: %d, phone: %s, auth: %s", u.ID, u.Phone, u.Auth)
}
