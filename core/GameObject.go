package core

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
	NickName     string
	CurrentScore int
	IpAddress    string
	RightOrLeft  string
}

func (p *Player) MoveUp() {
	p.Row -= 2
}

func (p *Player) MoveDown() {
	p.Row += 2
}
