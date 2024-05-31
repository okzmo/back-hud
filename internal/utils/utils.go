package utils

import (
	"crypto/rand"
	"math/big"
	"net/mail"
)

func EmailValid(email string) bool {
	emailAddress, err := mail.ParseAddress(email)
	return err == nil && emailAddress.Address == email
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomId(length ...int) (string, error) {
	idLength := 8
	if len(length) > 0 {
		idLength = length[0]
	}

	id := make([]byte, idLength)
	for i := range id {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}

		id[i] = charset[num.Int64()]
	}

	return string(id), nil
}
