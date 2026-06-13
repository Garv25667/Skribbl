package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	WSPort   = ":3223"
	wordList = []string{
		"trip", "cobra", "bottle", "curtains", "soap", "mailman", "banana peel", "railroad", "back", "lipstick",
		"knee", "broccoli", "face", "tape", "hot dog", "shadow", "lawnmower", "table", "trash can", "rainbow",
		"hippopotamus", "soda", "laundry basket", "city", "match", "hill", "violin", "mailbox", "tire", "pumpkin",
		"zebra", "shelf", "eel", "beach", "salt and pepper", "ladder", "blue jeans", "address", "radish", "sea turtle",
		"dress", "lid", "family", "ladybug", "window", "cheeseburger", "yo-yo", "frog", "whistle", "glove",
		"magazine", "church", "chameleon", "boot", "tongue", "hospital", "thief", "smile", "potato", "hairbrush",
		"stork", "computer", "school", "heel", "pogo stick", "tent", "cucumber", "fox", "three-toed sloth", "sprinkler",
		"garden", "blowfish", "crib", "wing", "brain", "net", "song", "drums", "bagel", "baby",
		"starfish", "corner", "carpet", "bicycle", "strawberry", "horse", "rug", "puzzle", "snowball", "aircraft",
		"gate", "sidewalk", "pan", "marshmallow", "bell pepper", "watering can", "plate", "jungle", "camera", "forehead",
		"towel", "surfboard", "coin", "watch", "chin", "key", "blimp", "cowboy", "picture frame", "piano",
		"lake", "pirate", "box", "paw", "toast", "swimming pool", "silverware", "salt", "tissue", "shovel",
		"hoof", "dominoes", "roller blading", "base", "rose", "spider web", "hopscotch", "spoon", "elbow", "pinwheel",
		"french fries", "log", "doorknob", "bag", "attic", "beaver", "unicorn", "seahorse", "scar", "snowflake",
		"eraser", "jelly", "battery", "easel", "jar", "barn", "bathtub", "paperclip", "photograph", "maid",
		"ring", "outside", "vase", "electrical outlet", "room", "birthday cake", "map", "coconut", "spool", "chocolate chip cookie",
		"muffin", "ski", "stapler", "t-shirt", "lock", "braid", "seesaw", "half", "paper", "pizza",
		"dock", "shoulder", "lunchbox", "spring", "treasure", "queen", "fang", "round", "dragonfly", "newspaper",
		"mail", "knot", "tusk", "umbrella", "ticket", "lawn mower", "shark", "neck", "toothbrush", "hook",
		"wax", "mop", "beehive", "forest", "money", "napkin", "wreath", "music", "quilt", "chain",
		"backbone", "sheep", "banana split", "baseball", "basket", "printer", "cello", "circus", "whisk", "dimple",
		"hummingbird", "nest", "wrench", "fork", "garage", "stump", "pine tree", "saw", "stove", "toaster",
		"park", "hula hoop", "garbage", "peanut", "daddy longlegs", "hair", "bib", "spare", "light switch", "king",
		"headband", "America", "nature", "milk", "refrigerator", "mattress", "tennis", "popsicle", "stomach", "pajamas",
		"password", "nail", "stamp", "nut", "palace", "gingerbread man", "dog leash", "front porch", "wood", "mitten",
		"rhinoceros", "popcorn", "teeth", "stingray", "happy", "onion", "wall", "pen", "alarm clock", "door",
		"crayon", "swing", "maze", "jewelry", "golf", "gift", "bowtie", "fur", "gumball", "pear",
		"tiger", "peach", "washing machine", "doormat", "desk", "hockey", "crack", "cast", "flashlight", "dustpan",
		"scissors", "skate", "wallet", "sink", "coal", "brick", "hug", "doghouse", "deep", "pelican",
		"page", "lightsaber", "toe", "rake", "tulip", "torch", "teapot", "bucket", "trumpet", "paint",
		"hair dryer", "pineapple", "calendar", "pretzel", "candle", "sailboat", "storm", "tank", "volcano", "flute",
		"ironing board", "clam", "waist", "catfish", "top hat", "skirt", "astronaut", "rain", "button", "dollar",
		"spaceship", "fishing pole", "video camera", "penguin", "lemon", "poodle", "hip", "roof", "state", "claw",
		"clown", "rocking chair", "belt", "mini blinds", "airport", "cheetah", "spine", "pond", "cage", "mouse",
		"bomb", "ice", "cake", "cockroach", "batteries", "fist", "flamingo", "purse", "lighthouse", "manatee",
		"iPad", "telephone", "harp", "eagle", "electricity", "lobster", "cheek", "shallow", "suitcase", "campfire",
		"flagpole", "chalk", "artist", "skunk", "apple pie", "mushroom", "corndog", "smoke", "ship", "grill",
		"food", "cricket", "pencil", "TV", "rolly polly", "dolphin", "bathroom scale", "bubble", "porcupine", "owl",
		"stoplight", "chimney", "light bulb", "deer", "platypus", "globe", "tadpole", "cell phone", "river", "sunflower", "mouth",
	}
)

type Server struct {
	rooms       map[string]*Room
	mu          *sync.RWMutex
	joinRoomCH  chan *Client
	leaveRoomCH chan *Client
	broadcastCH chan *ReqMsg
	startGameCH chan *Room
	guessCH     chan *ReqMsg
}

func newServer() *Server {
	return &Server{
		rooms:       map[string]*Room{},
		mu:          new(sync.RWMutex),
		joinRoomCH:  make(chan *Client, 64),
		leaveRoomCH: make(chan *Client, 64),
		broadcastCH: make(chan *ReqMsg, 64),
		startGameCH: make(chan *Room, 64),
		guessCH:     make(chan *ReqMsg, 64),
	}
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
		case room := <-s.startGameCH:
			s.startGame(room)
		case msg := <-s.guessCH:
			s.guessHandler(msg)

		}
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

func (s *Server) joinRoom(c *Client) {
	room := s.rooms[c.RoomID]
	if room == nil {
		room = newRoom(c.RoomID)
		s.rooms[c.RoomID] = room
	}
	if len(room.clients) > room.maxClients {
		fmt.Println("The room is Full")
		return
	}

	room.clients[c.ID] = c
	room.players = append(room.players, c)
	pljoinPay := PlayerJoinedPayload{
		PlayerID: c.ID,
	}
	s.broadcastToRoom(room, EventPlayerJoined, pljoinPay)
	fmt.Printf("Client Added Successfully ClientId = %s\n", c.ID)
}

func (s *Server) leaveRoom(c *Client) {
	room := s.rooms[c.RoomID]
	if room == nil {
		return
	}
	delete(room.clients, c.ID)
	leftpay := PlayerLeftPayload{
		PlayerID: c.ID,
	}
	s.broadcastExcept(room, c.ID, EventPlayerLeft, leftpay)
	fmt.Printf("Client ID = %s successfully deleted", c.ID)
}

func createNewWSServer() {
	s := newServer()
	go s.AcceptLoop()
	http.HandleFunc("/", s.handleWS)
	log.Fatal(http.ListenAndServe(WSPort, nil))
}

// Put this in a separate words.go file. Then use wordList[rand.IntN(len(wordList))] to pick one.
func (s *Server) startGame(room *Room) {
	room.state = "playing"
	room.currentDrawerIndex = 0
	room.round = 1
	s.startTurn(room)

}
func (s *Server) startTurn(room *Room) {
	room.word = wordList[rand.Intn(len(wordList))]
	room.correctGuesses = map[string]bool{}
	room.currentDrawer = room.players[room.currentDrawerIndex]
	timer := time.NewTimer(80 * time.Second)

	gspay := GameStartedPayload{
		DrawerID:  room.currentDrawer.ID,
		Round:     room.round,
		MaxRounds: room.maxRounds,
	}
	ytpay := YourTurnPayload{
		Word: room.word,
	}
	whpay := WordHintPayload{
		Hint: MaskString(room.word),
	}
	s.sendToClient(room.currentDrawer, EventYourTurn, ytpay)
	s.broadcastExcept(room, room.currentDrawer.ID, EventWordHint, whpay)
	s.broadcastToRoom(room, EventGameStarted, gspay)

	go func() {
		<-timer.C
		s.nextTurn(room)
	}()
	room.timer = timer
}
func (s *Server) nextTurn(room *Room) {
	room.timer.Stop()
	room.currentDrawerIndex += 1
	correctGuesses := []string{}

	for key := range room.correctGuesses {
		correctGuesses = append(correctGuesses, key)
	}
	tepay := TurnEndedPayload{
		Word:            room.word,
		CorrectGuessers: correctGuesses,
	}
	s.broadcastToRoom(room, EventTurnEnded, tepay)

	if room.currentDrawerIndex >= len(room.players) {
		room.round += 1
		room.currentDrawerIndex = 0
		if room.round > room.maxRounds {

			s.endGame(room)
			return
		}

	}
	s.startTurn(room)

}
func (s *Server) endGame(room *Room) {
	room.state = "Finished"
	evendedpay := GameEndedPayload{
		RoomId: room.ID,
	}
	s.broadcastToRoom(room, EventGameEnded, evendedpay)
	delete(s.rooms, room.ID)
}

func (s *Server) guessHandler(msg *ReqMsg) {

	room := s.rooms[msg.Client.RoomID]
	if room.currentDrawer == msg.Client {
		fmt.Println("The Drawer Can't guess")
		return
	}
	if room.correctGuesses[msg.Client.ID] == true {
		fmt.Println("Already guessed Correctly")
		return
	}
	if room.word == msg.Data {
		room.correctGuesses[msg.Client.ID] = true
		cgpay := CorrectGuessPayload{
			PlayerID: msg.Client.ID,
		}
		s.broadcastToRoom(room, EventCorrectGuess, cgpay)

		if len(room.correctGuesses) == len(room.players)-1 {
			s.nextTurn(room)
		}

	}

}

func (s *Server) sendToClient(c *Client, msg MsgType, payload interface{}) {
	data, _ := json.Marshal(payload)
	resp := RespMsg{
		MsgType: msg,
		RoomID:  c.RoomID,
		Data:    data,
	}
	c.conn.WriteJSON(resp)
}

func (s *Server) broadcastToRoom(room *Room, msg MsgType, payload interface{}) {
	data, _ := json.Marshal(payload)
	resp := RespMsg{
		MsgType: msg,
		RoomID:  room.ID,
		Data:    data,
	}
	for _, c := range room.clients {
		c.conn.WriteJSON(resp)
	}

}

func (s *Server) broadcastExcept(room *Room, excludeID string, msgType MsgType, payload interface{}) {
	data, _ := json.Marshal(payload)
	resp := RespMsg{MsgType: msgType, RoomID: room.ID, Data: data}
	for _, c := range room.clients {
		if c.ID != excludeID {
			c.conn.WriteJSON(resp)
		}
	}
}

func MaskString(word string) string {
	n := len(word)
	var s strings.Builder
	for range n {
		s.WriteString("_")
	}
	return s.String()
}
