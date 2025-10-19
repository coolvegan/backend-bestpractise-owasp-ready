package main

import (
	"encoding/json"
	"fmt"
	"foodshop/internal/models"
	"net/http"
)

var (
	server = "127.0.0.1"
	port   = "8080"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		return
	}

}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		return
	}
	var ul models.UserLogin
	if err := json.NewDecoder(r.Body).Decode(&ul); err != nil {
		json.NewEncoder(w).Encode(models.ErrUserLogin{Message: "The provided data is wrong!"})
	}
	if ul.Username == "" {
		json.NewEncoder(w).Encode(models.ErrUserLogin{Message: "The provided data has missing pieces!"})
	}

}

func main() {
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/logout", IndexHandler)
	http.HandleFunc("/registration", IndexHandler)
	http.ListenAndServe(fmt.Sprintf("%s:%s", server, port), nil)
}
