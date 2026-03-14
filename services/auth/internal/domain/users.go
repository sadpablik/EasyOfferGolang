package domain

import (
	"golang.org/x/crypto/bcrypt"
	"time"
)

type User struct{
	ID       string `json:"id" db:"id"`
	Email	string `json:"email" db:"email"`
	PasswordHash string `json:"-" db:"password_hash"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Role	 string `json:"role" db:"role"` // e.g., "user", "admin"
}

func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}

func (u *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(bytes)
	return nil
}	

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}