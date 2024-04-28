package utils

import (
	"net/mail"
)

func EmailValid(email string) bool {
	emailAddress, err := mail.ParseAddress(email)
	return err == nil && emailAddress.Address == email
}
