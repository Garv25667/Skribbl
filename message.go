package main

import "encoding/json"

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
	MsgType MsgType
	RoomID  string
	Data    json.RawMessage
}

func newRespMsg(msg *ReqMsg) *RespMsg {
	data, _ := json.Marshal(msg.Data)
	return &RespMsg{
		MsgType: msg.MsgType,
		RoomID:  msg.Client.RoomID,
		Data:    data,
	}
}
