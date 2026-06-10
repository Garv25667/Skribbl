package main

type MsgType string

const (
	MsgType_Broadcast MsgType = "broadcast"
	MsgType_Start     MsgType = "StartGame"
	MsgType_Guess     MsgType = "Guess"
)

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
