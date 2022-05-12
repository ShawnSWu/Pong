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

type Paddle struct {
	GameObject
	NickName     string
	CurrentScore int
	IpAddress    string
}

func (p *Paddle) MoveUp() {
	p.Row -= 2
}

func (p *Paddle) MoveDown() {
	p.Row += 2
}
