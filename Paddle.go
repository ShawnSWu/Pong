package main

type GameObject struct {
	row, col, velRow, velCol int
	width, height            int
	symbol                   rune
	nickName                 string
	currentScore             int
}

func (p *GameObject) moveUp() {
	p.row -= 1
}

func (p *GameObject) moveDown() {
	p.row += 1
}
