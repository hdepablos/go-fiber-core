package main

import (
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := "tu_super_secreto_para_access_tokens" // Valor real de .env

	claims := jwt.MapClaims{
		"sub":  "1",
		"sid":  "session-uuid-123", // Session ID requerido
		"typ":  "access",           // Tipo de token requerido
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
		"iat":  time.Now().Unix(),
		"role": "admin",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		panic(err)
	}

	fmt.Println(tokenString)
}
