package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"paperio/pb"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for VPS hosting flexibility
	},
}

var adjs = []string{"swift", "brave", "mighty", "clever", "silent", "happy", "lucky", "fierce"}
var nouns = []string{"tiger", "eagle", "dragon", "panther", "wolf", "bear", "fox", "shark"}

func generateGameName() string {
	adj := adjs[rand.Intn(len(adjs))]
	noun := nouns[rand.Intn(len(nouns))]
	return fmt.Sprintf("%s-%s-%d", adj, noun, rand.Intn(1000))
}

type GameInfo struct {
	ID         string `json:"id"`
	Players    int    `json:"players"`
	MaxPlayers int    `json:"max_players"`
	HasSession bool   `json:"has_session"`
}

func handleGetGames(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		server.mu.RLock()
		defer server.mu.RUnlock()

		games := []GameInfo{}
		for id, g := range server.Games {
			g.mu.RLock()
			hasSession := false
			for _, p := range g.Players {
				if p.Token == token {
					hasSession = true
					break
				}
			}
			games = append(games, GameInfo{
				ID:         id,
				Players:    len(g.Players),
				MaxPlayers: 10,
				HasSession: hasSession,
			})
			g.mu.RUnlock()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(games)
	}
}

func handleCreateGame(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		server.mu.Lock()
		defer server.mu.Unlock()

		var gameID string
		for {
			gameID = generateGameName()
			if _, exists := server.Games[gameID]; !exists {
				break
			}
		}

		game := NewGame(gameID)
		server.Games[gameID] = game

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": gameID})
	}
}

func handleGetSession(w http.ResponseWriter, r *http.Request) {
	token := fmt.Sprintf("t_%x%x", time.Now().UnixNano(), rand.Int63())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func handleWebSocket(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gameID := r.URL.Query().Get("game")
		if gameID == "" {
			http.Error(w, "Missing game parameter", http.StatusBadRequest)
			return
		}

		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "Missing token parameter", http.StatusBadRequest)
			return
		}

		server.mu.RLock()
		game, exists := server.Games[gameID]
		server.mu.RUnlock()

		if !exists {
			http.Error(w, "Game not found", http.StatusNotFound)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}

		player := game.Join(token, conn)

		// Listen for client directions
	loop:
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var clientMsg pb.ClientMessage
			if err := proto.Unmarshal(msg, &clientMsg); err != nil {
				continue
			}

			switch p := clientMsg.Payload.(type) {
			case *pb.ClientMessage_Direction:
				newDir := byte(p.Direction.Dir)

				player.mu.Lock()
				lastDir := player.Dir
				if len(player.DirQ) > 0 {
					lastDir = player.DirQ[len(player.DirQ)-1]
				}

				// Prevent 180-degree immediate turns
				if (lastDir == Up && newDir != Down) ||
					(lastDir == Down && newDir != Up) ||
					(lastDir == Left && newDir != Right) ||
					(lastDir == Right && newDir != Left) {
					
					if len(player.DirQ) < 3 { // Limit queue size to prevent memory exhaustion
						player.DirQ = append(player.DirQ, newDir)
					}
				}
				player.mu.Unlock()
			case *pb.ClientMessage_Leave:
				break loop
			}
		}
		conn.Close()
	}
}