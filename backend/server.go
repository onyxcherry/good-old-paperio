package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	GridWidth  = 100
	GridHeight = 100
	TickRate   = 100 * time.Millisecond // 10 ticks per second
)

// Directions
const (
	Up byte = iota
	Right
	Down
	Left
)

type Point struct {
	X, Y int
}

type Player struct {
	ID    uint32
	Token string // Used for reconnection
	X, Y  int
	Dir   byte
	Tail  []Point
	Alive bool
	Conn  *websocket.Conn
	mu    sync.Mutex
}

type Game struct {
	ID       string
	Grid     [][]uint32 // 0 = empty, otherwise Player ID
	Players  map[uint32]*Player
	mu       sync.RWMutex
	JoinChan chan *Player
}

type Server struct {
	Games map[string]*Game
	mu    sync.RWMutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for VPS hosting flexibility
	},
}

func NewGame(id string) *Game {
	grid := make([][]uint32, GridWidth)
	for i := range grid {
		grid[i] = make([]uint32, GridHeight)
	}

	g := &Game{
		ID:       id,
		Grid:     grid,
		Players:  make(map[uint32]*Player),
		JoinChan: make(chan *Player),
	}
	go g.Run()
	return g
}

// Spawns a player and assigns initial 3x3 territory
func (g *Game) spawnPlayer(p *Player) {
	p.X = rand.Intn(GridWidth-10) + 5
	p.Y = rand.Intn(GridHeight-10) + 5
	p.Dir = Right
	p.Tail = []Point{}
	p.Alive = true

	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			g.Grid[p.X+dx][p.Y+dy] = p.ID
		}
	}
}

// killPlayer removes a player from the game, freeing their territory and closing the connection
func (g *Game) killPlayer(p *Player) {
	p.Alive = false
	// Free territory and tail
	for x := 0; x < GridWidth; x++ {
		for y := 0; y < GridHeight; y++ {
			if g.Grid[x][y] == p.ID {
				g.Grid[x][y] = 0
			}
		}
	}
	if p.Conn != nil {
		p.mu.Lock()
		p.Conn.Close()
		p.Conn = nil
		p.mu.Unlock()
	}
	delete(g.Players, p.ID)
}

func (g *Game) Run() {
	ticker := time.NewTicker(TickRate)
	defer ticker.Stop()

	for {
		select {
		case p := <-g.JoinChan:
			g.mu.Lock()
			// Handle Reconnection vs New Player
			var existing *Player
			for _, player := range g.Players {
				if player.Token == p.Token {
					existing = player
					break
				}
			}

			if existing != nil {
				existing.Conn = p.Conn
				// Send initial state to reconnected client
				existing.sendInit()
			} else {
				g.spawnPlayer(p)
				g.Players[p.ID] = p
				p.sendInit()
			}
			g.mu.Unlock()

		case <-ticker.C:
			g.tick()
			g.broadcastState()
		}
	}
}

func (g *Game) tick() {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, p := range g.Players {
		if !p.Alive {
			continue
		}

		// Move player
		switch p.Dir {
		case Up: p.Y--
		case Right: p.X++
		case Down: p.Y++
		case Left: p.X--
		}

		// Check bounds (Death)
		if p.X < 0 || p.X >= GridWidth || p.Y < 0 || p.Y >= GridHeight {
			g.killPlayer(p)
			continue
		}

		// Check tail collisions (Killing others or self)
		hitTail := false
		for _, other := range g.Players {
			if !other.Alive { continue }
			for _, tp := range other.Tail {
				if p.X == tp.X && p.Y == tp.Y {
					if p.ID == other.ID {
						g.killPlayer(p) // Hit own tail
					} else {
						g.killPlayer(other) // Killed other
					}
					hitTail = true
					break
				}
			}
		}

		if hitTail && !p.Alive {
			continue
		}

		// Territory Logic
		currentOwner := g.Grid[p.X][p.Y]
		if currentOwner != p.ID {
			// Outside territory, grow tail
			p.Tail = append(p.Tail, Point{p.X, p.Y})
		} else if len(p.Tail) > 0 {
			// Back in own territory, capture area
			g.captureTerritory(p)
			p.Tail = []Point{} // Reset tail
		}
	}
}

// Flood-fill algorithm to determine captured area
func (g *Game) captureTerritory(p *Player) {
	// 1. Mark player tail as their territory on the grid
	for _, tp := range p.Tail {
		g.Grid[tp.X][tp.Y] = p.ID
	}

	// 2. Flood fill from edges to find "outside" space
	visited := make([][]bool, GridWidth)
	for i := range visited {
		visited[i] = make([]bool, GridHeight)
	}

	queue := []Point{}
	// Add edges to queue
	for x := 0; x < GridWidth; x++ {
		queue = append(queue, Point{x, 0}, Point{x, GridHeight - 1})
	}
	for y := 0; y < GridHeight; y++ {
		queue = append(queue, Point{0, y}, Point{GridWidth - 1, y})
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.X < 0 || curr.X >= GridWidth || curr.Y < 0 || curr.Y >= GridHeight {
			continue
		}
		if visited[curr.X][curr.Y] || g.Grid[curr.X][curr.Y] == p.ID {
			continue
		}

		visited[curr.X][curr.Y] = true
		queue = append(queue, 
			Point{curr.X + 1, curr.Y}, Point{curr.X - 1, curr.Y},
			Point{curr.X, curr.Y + 1}, Point{curr.X, curr.Y - 1},
		)
	}

	// 3. Any unvisited cell not owned by player becomes theirs
	for x := 0; x < GridWidth; x++ {
		for y := 0; y < GridHeight; y++ {
			if !visited[x][y] {
				g.Grid[x][y] = p.ID
			}
		}
	}
}

// --- Binary Protocol Implementations ---

func (p *Player) sendInit() {
	buf := make([]byte, 9)
	buf[0] = 0 // MsgType: Init
	binary.BigEndian.PutUint32(buf[1:5], p.ID)
	binary.BigEndian.PutUint16(buf[5:7], GridWidth)
	binary.BigEndian.PutUint16(buf[7:9], GridHeight)
	p.mu.Lock()
	p.Conn.WriteMessage(websocket.BinaryMessage, buf)
	p.mu.Unlock()
}

func (g *Game) broadcastState() {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Calculate payload size
	// [MsgType 1 byte] [NumPlayers 2 bytes] -> Player Data -> [NumGridUpdates 2 bytes] -> Grid Data
	size := 1 + 2
	for _, p := range g.Players {
		size += 4 + 2 + 2 + 1 + 2 + (len(p.Tail) * 4) // ID(4), X(2), Y(2), Dir(1), TailLen(2), Tail points
	}
	
	// For optimization, a production app sends grid *deltas*. 
	// For this snippet, we send the full grid state compactly.
	size += 2 + (GridWidth * GridHeight * 4) 

	buf := make([]byte, size)
	buf[0] = 1 // MsgType: State Update
	
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(g.Players)))
	offset := 3

	for _, p := range g.Players {
		binary.BigEndian.PutUint32(buf[offset:], p.ID)
		binary.BigEndian.PutUint16(buf[offset+4:], uint16(p.X))
		binary.BigEndian.PutUint16(buf[offset+6:], uint16(p.Y))
		buf[offset+8] = p.Dir
		binary.BigEndian.PutUint16(buf[offset+9:], uint16(len(p.Tail)))
		offset += 11

		for _, tp := range p.Tail {
			binary.BigEndian.PutUint16(buf[offset:], uint16(tp.X))
			binary.BigEndian.PutUint16(buf[offset+2:], uint16(tp.Y))
			offset += 4
		}
	}

	binary.BigEndian.PutUint16(buf[offset:], uint16(GridWidth*GridHeight))
	offset += 2
	for x := 0; x < GridWidth; x++ {
		for y := 0; y < GridHeight; y++ {
			binary.BigEndian.PutUint32(buf[offset:], g.Grid[x][y])
			offset += 4
		}
	}

	for _, p := range g.Players {
		if p.Conn != nil {
			p.mu.Lock()
			p.Conn.WriteMessage(websocket.BinaryMessage, buf)
			p.mu.Unlock()
		}
	}
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

func handleWebSocket(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade error:", err)
			return
		}

		// Reconnection / Lobby logic via Query Params
		gameID := r.URL.Query().Get("game")
		if gameID == "" {
			gameID = "default"
		}
		
		token := r.URL.Query().Get("token")
		if token == "" {
			token = fmt.Sprintf("token_%d", time.Now().UnixNano())
		}

		server.mu.Lock()
		game, exists := server.Games[gameID]
		if !exists {
			game = NewGame(gameID)
			server.Games[gameID] = game
		}

		// Check player limit before allowing connection
		game.mu.RLock()
		playerCount := len(game.Players)
		var isReconnecting bool
		for _, p := range game.Players {
			if p.Token == token {
				isReconnecting = true
				break
			}
		}
		game.mu.RUnlock()

		if !isReconnecting && playerCount >= 10 {
			server.mu.Unlock()
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Game is full"))
			conn.Close()
			return
		}
		server.mu.Unlock()

		player := &Player{
			ID:    rand.Uint32(),
			Token: token,
			Conn:  conn,
		}

		game.JoinChan <- player

		// Listen for client directions
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if len(msg) >= 2 && msg[0] == 2 { // MsgType 2 = Direction Change
				newDir := msg[1]

				game.mu.Lock()
				var activePlayer *Player
				for _, p := range game.Players {
					if p.Token == token {
						activePlayer = p
						break
					}
				}

				if activePlayer != nil {
					// Prevent 180-degree immediate turns
					if (activePlayer.Dir == Up && newDir != Down) ||
					   (activePlayer.Dir == Down && newDir != Up) ||
					   (activePlayer.Dir == Left && newDir != Right) ||
					   (activePlayer.Dir == Right && newDir != Left) {
						activePlayer.Dir = newDir
					}
				}
				game.mu.Unlock()
			}
		}
		conn.Close()
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	server := &Server{
		Games: make(map[string]*Game),
	}

	fs := http.FileServer(http.Dir("../frontend/dist"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Ensure index.html is revalidated, but allow aggressive caching for hashed assets
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fs.ServeHTTP(w, r)
	})
	http.HandleFunc("/api/games", handleGetGames(server))
	http.HandleFunc("/ws", handleWebSocket(server))

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
