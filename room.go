package main

import (
	"sync"
	"time"
)

type Room struct {
	ID         string
	clients    map[string]*Client
	mu         *sync.RWMutex
	maxClients int
	scores     map[string]int
	turnEnded  bool
	GameState
}
type GameState struct {
	state              string
	players            []*Client
	currentDrawerIndex int
	currentDrawer      *Client
	word               string
	correctGuesses     map[string]bool
	round              int
	maxRounds          int
	timer              *time.Timer
	turnStartTime      time.Time
}

func newRoom(id string) *Room {
	return &Room{
		ID:         id,
		clients:    map[string]*Client{},
		mu:         new(sync.RWMutex),
		maxClients: 8,
		scores:     map[string]int{},

		GameState: GameState{
			state:          "Waiting",
			maxRounds:      3,
			correctGuesses: make(map[string]bool),
		},
	}
}
