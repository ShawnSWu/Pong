package main

import (
	"fmt"
	"github.com/gdamore/tcell"
	"os"
	"time"
)

var screen tcell.Screen
var player1 *Paddle
var player2 *Paddle
var ball *Ball

var paddles []*Paddle

const BallSymbol = 0x25CF   // 球符號
const PaddleSymbol = 0x2588 // 球拍符號
const PaddleHeight = 6      // 球拍高度
const BallVelocityRow = 1
const BallVelocityCol = 1

func userOperationSetting() {
	inputChan := initUserInput()
	for {
		updateState()
		drawView()
		time.Sleep(75 * time.Millisecond)

		key := readInput(inputChan)
		if key == "Rune[w]" && isTouchTopBorder(player1) {
			player1.moveUp()
		}

		if key == "Rune[s]" && isTouchBottomBorder(player1) {
			player1.moveDown()
		}

		if key == "Up" && isTouchTopBorder(player2) {
			player2.moveUp()
		}

		if key == "Down" && isTouchBottomBorder(player2) {
			player2.moveDown()
		}
	}
}

func initUserInput() chan string {
	//初始化channel，去接另一個goroutine丟回來的資料
	inputChan := make(chan string)

	//建立一個goroutine去監聽鍵盤的事件
	go func() {
		for {
			switch ev := screen.PollEvent().(type) {
			case *tcell.EventKey:
				inputChan <- ev.Name()
			}
		}
	}()

	return inputChan
}

func initScreen() {
	var err error
	screen, err = tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if e := screen.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	defaultStyle := tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite)
	screen.SetStyle(defaultStyle)
}

func initGameState() {
	width, height := screen.Size()
	paddleStart := height/2 - PaddleHeight/2

	player1 = &Paddle{
		GameObject: GameObject{row: paddleStart, col: 0, width: 1,
			height: PaddleHeight, symbol: PaddleSymbol,
			velRow: 0, velCol: 0},
		nickName:     "Player one",
		currentScore: 0,
	}

	player2 = &Paddle{
		GameObject: GameObject{row: paddleStart, col: width - 2, width: 1,
			height: PaddleHeight, symbol: PaddleSymbol,
			velRow: 0, velCol: 0},
		nickName:     "Player two",
		currentScore: 0,
	}

	ball = &Ball{
		GameObject: GameObject{row: height / 2, col: width / 2, width: 1, height: 1, symbol: BallSymbol,
			velRow: BallVelocityRow, velCol: BallVelocityCol},
	}

	paddles = []*Paddle{
		player1,
		player2,
	}
}

func updateState() {
	//兩個球拍
	for i := range paddles {
		paddles[i].row += paddles[i].velRow
		paddles[i].col += paddles[i].velCol
	}
	//球
	ball.row += ball.velRow
	ball.col += ball.velCol

	//檢查有沒有撞到上下牆壁
	if isCollidesWithWall(ball) {
		ball.velRow = -ball.velRow
	}
	//檢查是否有碰到球拍
	if isTouchPaddle(ball) {
		ball.velCol = -ball.velCol
	}
}

func isBallOutSide(ball *Ball) bool {
	width, _ := screen.Size()
	return ball.col < 0 || ball.col > width
}

func isTouchPaddle(ball *Ball) bool {
	if ball.col+ball.velCol == player1.col &&
		(ball.row > player1.row && ball.row <= player1.row+PaddleHeight) {
		return true
	} else if ball.col+ball.velCol == player2.col &&
		(ball.row > player2.row && ball.row <= player2.row+PaddleHeight) {
		return true
	}
	return false
}

func isTouchBottomBorder(paddle *Paddle) bool {
	_, screenHeight := screen.Size()
	return (paddle.row + paddle.height) < screenHeight
}

func isTouchTopBorder(paddle *Paddle) bool {
	return paddle.row > 0
}

func isCollidesWithWall(ball *Ball) bool {
	_, screenHeight := screen.Size()
	return ball.row+ball.velRow < 0 || ball.row+ball.velRow >= screenHeight
}

func readInput(inputChan chan string) string {
	var key string
	select {
	case key = <-inputChan:
	default:
		key = ""
	}
	return key
}

func drawView() {
	screen.Clear()
	//兩個球拍
	for _, obj := range paddles {
		Print(obj.row, obj.col, obj.width, obj.height, PaddleSymbol)
	}
	//球
	Print(ball.row, ball.col, ball.width, ball.height, PaddleSymbol)
	screen.Show()
}

func Print(row, col, width, height int, ch rune) {
	for r := 0; r < height; r++ {
		for c := 0; c < width; c++ {
			screen.SetContent(col+c, row+r, ch, nil, tcell.StyleDefault)
		}
	}
	screen.Show()
}
