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
	CurrentScore   int
	IdAkaIpAddress string
	RightOrLeft    string

	Scene string

	Conn *net.Conn
}

func (p *Player) MoveUp() {
	p.Row -= 2
}

func (p *Player) MoveDown() {
	p.Row += 2
}
