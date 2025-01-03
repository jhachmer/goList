package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jhachmer/gotocollection/internal/config"
	"golang.org/x/crypto/bcrypt"
)

var secretKey string

func InitAuth() error {
	secretKey = config.GetEnv("GOLIST_JWT", "")
	if secretKey == "" {
		return fmt.Errorf("no jwt env variable")
	}
	return nil
}

func CreateToken(username string) (string, error) {
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"iss": "golist",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now(),
	},
	)
	tokenString, err := claims.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func VerifyToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("error parsing jwt %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return token, nil
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed hashinf password: %w", err)
	}
	return string(hashedPassword), nil
}
