package password

import (
	"golang.org/x/crypto/bcrypt"
)

// Hash хеширует пароль с использованием bcrypt.
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Compare сравнивает хэш пароля и «сырой» пароль.
func Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}


