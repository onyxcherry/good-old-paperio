package main

import (
	"math/rand"
	"sync"
	"time"

	"paperio/pb" // Ensure this matches your Go module name

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

const (
	GridWidth  = 210
	GridHeight = 210
	TotalCells = GridWidth * GridHeight
	WinCells   = TotalCells * 99 / 100
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
	DirQ  []byte // Input queue to prevent missing intermediate states
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

func (g *Game) Join(token string, conn *websocket.Conn) *Player {
	g.mu.Lock()
	for _, player := range g.Players {
		if player.Token == token {
			player.mu.Lock()
			player.Conn = conn
			player.mu.Unlock()
			g.mu.Unlock()
			player.sendInit()
			return player
		}
	}

	p := &Player{
		ID:    rand.Uint32(),
		Token: token,
		Conn:  conn,
	}
	g.spawnPlayer(p)
	g.Players[p.ID] = p
	g.mu.Unlock()

	p.sendInit()
	return p
}

func (g *Game) Run() {
	ticker := time.NewTicker(TickRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.tick()
			g.broadcastState()
		}
	}
}

func (g *Game) tick() {
	g.mu.Lock()
	defer g.mu.Unlock()

	territoryCaptured := false

	for _, p := range g.Players {
		if !p.Alive {
			continue
		}

		if len(p.DirQ) > 0 {
			p.Dir = p.DirQ[0]
			p.DirQ = p.DirQ[1:]
		}

		// Move player
		switch p.Dir {
		case Up:
			p.Y--
		case Right:
			p.X++
		case Down:
			p.Y++
		case Left:
			p.X--
		}

		// Check bounds (Death)
		if p.X < 0 || p.X >= GridWidth || p.Y < 0 || p.Y >= GridHeight {
			g.killPlayer(p)
			continue
		}

		// Check tail collisions (Killing others or self)
		for _, other := range g.Players {
			if !other.Alive {
				continue
			}
			for _, tp := range other.Tail {
				if p.X == tp.X && p.Y == tp.Y {
					if p.ID == other.ID {
						g.killPlayer(p) // Hit own tail
					} else {
						g.killPlayer(other) // Killed other
					}
					break
				}
			}
		}

		if !p.Alive {
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
			territoryCaptured = true
		}
	}

	if territoryCaptured {
		counts := make(map[uint32]int)
		for x := 0; x < GridWidth; x++ {
			for y := 0; y < GridHeight; y++ {
				if owner := g.Grid[x][y]; owner != 0 {
					counts[owner]++
				}
			}
		}
		for id, count := range counts {
			if count >= WinCells {
				g.win(id)
			}
		}
	}
}

func (g *Game) win(winnerID uint32) {
	msg := &pb.ServerMessage{
		Payload: &pb.ServerMessage_Win{
			Win: &pb.WinMsg{
				WinnerId: winnerID,
			},
		},
	}
	buf, _ := proto.Marshal(msg)

	activePlayers := make([]*Player, 0, len(g.Players))
	for _, p := range g.Players {
		if p.Conn != nil {
			activePlayers = append(activePlayers, p)
		}
	}

	go func() {
		broadcast(activePlayers, buf)
	}()
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
		queue = append(queue, Point{x, 0}, Point{x, GridHeight-1})
	}
	for y := 0; y < GridHeight; y++ {
		queue = append(queue, Point{0, y}, Point{GridWidth-1, y})
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

func (p *Player) sendInit() {
	msg := &pb.ServerMessage{
		Payload: &pb.ServerMessage_Init{
			Init: &pb.InitMsg{
				Id:         p.ID,
				GridWidth:  uint32(GridWidth),
				GridHeight: uint32(GridHeight),
			},
		},
	}
	buf, _ := proto.Marshal(msg)
	p.mu.Lock()
	p.Conn.WriteMessage(websocket.BinaryMessage, buf)
	p.mu.Unlock()
}

func (g *Game) broadcastState() {
	g.mu.RLock()

	players := make([]*pb.Player, 0, len(g.Players))
	for _, p := range g.Players {
		tail := make([]*pb.Point, 0, len(p.Tail))
		for _, tp := range p.Tail {
			tail = append(tail, &pb.Point{X: int32(tp.X), Y: int32(tp.Y)})
		}
		players = append(players, &pb.Player{
			Id:   p.ID,
			X:    int32(p.X),
			Y:    int32(p.Y),
			Dir:  uint32(p.Dir),
			Tail: tail,
		})
	}

	grid := make([]uint32, 0, GridWidth*GridHeight)
	for x := 0; x < GridWidth; x++ {
		for y := 0; y < GridHeight; y++ {
			grid = append(grid, g.Grid[x][y])
		}
	}

	msg := &pb.ServerMessage{
		Payload: &pb.ServerMessage_State{
			State: &pb.StateMsg{
				Players: players,
				Grid:    grid,
			},
		},
	}
	buf, _ := proto.Marshal(msg)

	activePlayers := make([]*Player, 0, len(g.Players))
	for _, p := range g.Players {
		if p.Conn != nil {
			activePlayers = append(activePlayers, p)
		}
	}
	g.mu.RUnlock() // Release game lock before blocking on network I/O

	broadcast(activePlayers, buf)
}

func broadcast(players []*Player, buf []byte) {
	for _, p := range players {
		p.mu.Lock()
		p.Conn.WriteMessage(websocket.BinaryMessage, buf)
		p.mu.Unlock()
	}
}