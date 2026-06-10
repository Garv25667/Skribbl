package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID     string
	RoomID string
	mu     *sync.RWMutex
	conn   *websocket.Conn
}

func newClient(conn *websocket.Conn) *Client {
	id := rand.Text()[:9]
	return &Client{
		ID:     id,
		RoomID: "",
		mu:     new(sync.RWMutex),
		conn:   conn,
	}
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
			srv.startGameCH <- srv.rooms[c.RoomID]
		case MsgType_Guess:
			srv.guessCH <- msg

		}

	}
}
