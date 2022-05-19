package core

import "net"

type GameObject struct {
	Row, Col       int
	Width, Height  int
	VelRow, VelCol int
	Symbol         rune
}

type Ball struct {
	GameObject
}

type Player struct {
	GameObject
	NickName       string
	IdAkaIpAddress string

	RightOrLeft     string
	CurrentScore    int
	RoomReadyStatus int

	Scene string

	Conn *net.Conn
}

func (p *Player) SetScene(scene string) {
	p.Scene = scene
}

func (p *Player) MoveUp() {
	p.Row -= 2
}

func (p *Player) MoveDown() {
	p.Row += 2
}
