package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID       string
	PlayerID int32
	RoomID   string
	mu       *sync.RWMutex
	conn     *websocket.Conn
	isHost   bool
}

func newClient(conn *websocket.Conn, playerID int32) *Client {
	id := rand.Text()[:9]
	return &Client{
		ID:       id,
		PlayerID: playerID,
		RoomID:   "",
		mu:       new(sync.RWMutex),
		conn:     conn,
		isHost:   false,
	}
}

func (c *Client) writeJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

func (c *Client) readMsgLoop(srv *Server) {
	defer func() {
		c.conn.Close()
		srv.leaveRoomCH <- c
	}()
	for {
		_, b, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		msg := new(ReqMsg)
		err = json.Unmarshal(b, msg)
		if err != nil {
			fmt.Printf("Unable to unmarshal the msg %v\n", err)
			continue
		}
		msg.Client = c
		switch msg.MsgType {
		case MsgType_Broadcast:
			srv.broadcastCH <- msg
		case MsgType_Start:
			srv.startGameCH <- c.RoomID

		case MsgType_Guess:
			srv.guessCH <- msg

		}

	}
}
