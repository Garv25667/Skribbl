package main

const (
	EventGameStarted  MsgType = "game_started"
	EventYourTurn     MsgType = "your_turn"
	EventWordHint     MsgType = "word_hint"
	EventCorrectGuess MsgType = "correct_guess"
	EventTurnEnded    MsgType = "turn_ended"
	EventGameEnded    MsgType = "game_ended"
	EventPlayerJoined MsgType = "player_joined"
	EventPlayerLeft   MsgType = "player_left"
	EventRoomInfo     MsgType = "room_info"
)

type GameStartedPayload struct {
	DrawerID  string
	Round     int
	MaxRounds int
}

type YourTurnPayload struct {
	Word string
}

type WordHintPayload struct {
	Hint string
}

type CorrectGuessPayload struct {
	PlayerID string
}
type TurnEndedPayload struct {
	Word            string
	CorrectGuessers []string
}
type GameEndedPayload struct {
	RoomId string
	Scores map[string]int
}

type PlayerJoinedPayload struct {
	PlayerID string
	Username string
}

type PlayerLeftPayload struct {
	PlayerID string
}

type RoomInfoPayload struct {
	CurrentPlayers []PlayerInfo
}
