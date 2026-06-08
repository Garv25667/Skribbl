package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	WSPort = ":3223"
)

type MsgType string

const (
	MsgType_Broadcast MsgType = "broadcast"
)

type Client struct {
	ID     string
	RoomID string
	mu     *sync.RWMutex
	conn   *websocket.Conn
}

type Room struct {
	ID         string
	clients    map[string]*Client
	mu         *sync.RWMutex
	maxClients int
}
type ReqMsg struct {
	MsgType MsgType
	Client  *Client
	Data    string
}
type RespMsg struct {
	MsgType  MsgType
	ClientID string
	RoomID   string
	Data     string
}

func newRespMsg(msg *ReqMsg) *RespMsg {
	return &RespMsg{
		MsgType:  msg.MsgType,
		ClientID: msg.Client.ID,
		RoomID:   msg.Client.RoomID,
		Data:     msg.Data,
	}
}
func newRoom(id string) *Room {
	return &Room{
		ID:         id,
		clients:    map[string]*Client{},
		mu:         new(sync.RWMutex),
		maxClients: 8,
	}
}

type Server struct {
	rooms       map[string]*Room
	mu          *sync.RWMutex
	joinRoomCH  chan *Client
	leaveRoomCH chan *Client
	broadcastCH chan *ReqMsg
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

func newServer() *Server {
	return &Server{
		rooms:       map[string]*Room{},
		mu:          new(sync.RWMutex),
		joinRoomCH:  make(chan *Client, 64),
		leaveRoomCH: make(chan *Client, 64),
		broadcastCH: make(chan *ReqMsg, 64),
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
		srv.broadcastCH <- msg
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Error Upgrading the connection Error = %s\n", err)
		return
	}
	client := newClient(conn)
	roomID := r.URL.Query().Get("room")
	client.RoomID = roomID
	s.joinRoomCH <- client
	go client.readMsgLoop(s)

}
func (s *Server) AcceptLoop() {
	for {
		select {
		case c := <-s.joinRoomCH:
			s.joinRoom(c)
		case c := <-s.leaveRoomCH:
			s.leaveRoom(c)
		case msg := <-s.broadcastCH:
			s.broadcast(msg)

		}
	}
}
func (s *Server) broadcast(msg *ReqMsg) {
	cls := []*Client{}
	room := s.rooms[msg.Client.RoomID]
	if room == nil {
		fmt.Println("Cannot broadcast message as room doesn't exist")
		return
	}
	room.mu.RLock()
	for _, c := range room.clients {
		if c.ID != msg.Client.ID {
			cls = append(cls, c)
		}
	}
	room.mu.RUnlock()
	resp := newRespMsg(msg)
	for _, c := range cls {
		err := c.conn.WriteJSON(resp)
		if err != nil {
			fmt.Printf("Error sending msg to ClientID = %s\n", c.ID)
			continue
		}
	}
	fmt.Printf("BroadCast sent Successfully to RoomID = %s\n", room.ID)
}

func createNewWSServer() {
	s := newServer()
	go s.AcceptLoop()
	http.HandleFunc("/", s.handleWS)
	log.Fatal(http.ListenAndServe(WSPort, nil))
}

func (s *Server) joinRoom(c *Client) {
	room := s.rooms[c.RoomID]
	if room == nil {
		room = newRoom(c.RoomID)
		s.rooms[c.RoomID] = room
	}
	if len(room.clients) > room.maxClients {
		fmt.Println("The room is Full\n")
		return
	}

	room.clients[c.ID] = c
	fmt.Printf("Client Added Successfully ClientId = %s\n", c.ID)
}

func (s *Server) leaveRoom(c *Client) {
	room := s.rooms[c.RoomID]
	if room == nil {
		return
	}
	delete(room.clients, c.ID)
	fmt.Printf("Client ID = %s successfully deleted", c.ID)
}

// to do
// [x] Create a server
// [x]connection upgrade
// [x]  create a client and help him get into a room
// [x] make a room of 6 people or more then get that in a server
// [] go channel run to accept loop

func main() {
	createNewWSServer()
}
