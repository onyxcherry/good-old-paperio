package main

import (
	"log"
	"net/http"
	"sync"
)

type Server struct {
	Games map[string]*Game
	mu    sync.RWMutex
}


func main() {
	server := &Server{
		Games: make(map[string]*Game),
	}

	http.HandleFunc("/api/games", handleGetGames(server))
	http.HandleFunc("/api/games/create", handleCreateGame(server))
	http.HandleFunc("/api/session", handleGetSession)
	http.HandleFunc("/ws", handleWebSocket(server))

	log.Println("Server running on 0.0.0.0:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
