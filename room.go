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
}

func newRoom(id string) *Room {
	return &Room{
		ID:         id,
		clients:    map[string]*Client{},
		mu:         new(sync.RWMutex),
		maxClients: 8,
		GameState: GameState{
			state:     "Waiting",
			maxRounds: 3,
		},
	}
}
