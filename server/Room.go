package main

import (
	"Pong/server/core"
	"net"
)

type Room struct {
	RoomId string

	Player1 *core.Player
	Player2 *core.Player

	Player1Conn *net.Conn
	Player2Conn *net.Conn

	Ball *core.Ball
}
