package models

import "time"

type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	Intention    Intention
	CreatedAt    time.Time
}

// DisplayName falls back to the email local part before onboarding captured a name.
func (u User) DisplayName() string {
	if u.Name != "" {
		return u.Name
	}
	if at := indexByte(u.Email, '@'); at > 0 {
		return u.Email[:at]
	}
	return u.Email
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
