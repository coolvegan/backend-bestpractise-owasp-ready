package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v4"
)

func main() {
	var tokenString string
	var secretKey string

	flag.StringVar(&tokenString, "token", "", "JWT Token zum Prüfen")
	flag.StringVar(&secretKey, "key", "", "Geheimer JWT Key")
	flag.Parse()

	if tokenString == "" || secretKey == "" {
		fmt.Println("Usage: shell -token <JWT> -key <SECRET>")
		os.Exit(1)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		fmt.Printf("Token ungültig: %v\n", err)
		os.Exit(1)
	}

	if !token.Valid {
		fmt.Println("Token ist nicht valid!")
		os.Exit(1)
	}

	fmt.Println("Token ist gültig!")
}
