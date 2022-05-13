package core

import (
	"net"
)

type Room struct {
	RoomId string

	Player1 *Player
	Player2 *Player

	Player1Conn *net.Conn
	Player2Conn *net.Conn

	Ball *Ball
}
