package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type TokenType string

const TokenTypeAccess TokenType = "chirpy-access"

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	signingKey := []byte(tokenSecret)
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    string(TokenTypeAccess),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * expiresIn)),
		Subject:   userID.String(),
	})

	return newToken.SignedString(signingKey)
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claimsStruct := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(token *jwt.Token) (interface{}, error) { return []byte(tokenSecret), nil },
	)
	fmt.Println(err)
	if err != nil {
		return uuid.Nil, err
	}

	userIDString, err := token.Claims.GetSubject()
	fmt.Printf("UserIDString: %s\n", userIDString)
	if err != nil {
		return uuid.Nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	fmt.Printf("Issuer: %s\n", issuer)
	if err != nil {
		return uuid.Nil, err
	}
	if issuer != string(TokenTypeAccess) {
		return uuid.Nil, errors.New("invalid issuer")
	}

	id, err := uuid.Parse(userIDString)
	fmt.Printf("Parsed ID: %v\n", id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID: %w", err)
	}
	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("There is not token String")
	}
	splitAuth := strings.Split(authHeader, " ")

	if len(splitAuth) < 2 || splitAuth[0] != "Bearer" {
		return "", fmt.Errorf("Not correctly formated BearerToken")
	}

	return splitAuth[1], nil
}

func MakeRefreshToken() (string, error) {
	buff := make([]byte, 32)
	_, err := rand.Read(buff)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buff), nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password string, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(password), []byte(hash))
	return err
}
