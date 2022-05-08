package main

type GameObject struct {
	row, col       int
	width, height  int
	velRow, velCol int
	symbol         rune
}

type Ball struct {
	GameObject
}

type Paddle struct {
	GameObject
	nickName     string
	currentScore int
}

func (p *Paddle) moveUp() {
	p.row -= 1
}

func (p *Paddle) moveDown() {
	p.row += 1
}
