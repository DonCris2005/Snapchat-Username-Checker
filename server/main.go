package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ytax/snapchat-username-checker/checker"
)

type checkRequest struct {
	Usernames  []string `json:"usernames"`
	Goroutines int      `json:"goroutines"`
}

type checkResponse struct {
	Available   []string `json:"available"`
	Unavailable []string `json:"unavailable"`
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("pong"))
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Goroutines <= 0 {
		req.Goroutines = 1
	}
	avail, unavail := checker.CheckUsernames(req.Usernames, req.Goroutines)
	resp := checkResponse{Available: avail, Unavailable: unavail}
	json.NewEncoder(w).Encode(resp)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", handlePing)
	mux.HandleFunc("/check", handleCheck)
	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
