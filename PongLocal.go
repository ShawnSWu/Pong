package main

import (
	"Pong/core"
	"fmt"
	"github.com/gdamore/tcell"
	"os"
	"strconv"
	"time"
)

var screen tcell.Screen
var player1 *core.Paddle
var player2 *core.Paddle
var ball *core.Ball

const FinalScore = 9        // 遊戲結束分數
const BallSymbol = 0x25CF   // 球符號
const PaddleSymbol = 0x2588 // 球拍符號
const PaddleHeight = 6      // 球拍高度
const BallVelocityRow = 1
const BallVelocityCol = 1

func startGameLoop() {
	inputChan := initUserInput()
	for {
		updateState()
		drawView()
		time.Sleep(45 * time.Millisecond)
		userOperationHandle(inputChan)
	}
}

func userOperationHandle(inputChan chan string) {
	key := readInput(inputChan)
	if key == "Rune[w]" && isTouchTopBorder(player1) {
		player1.MoveUp()
	}

	if key == "Rune[s]" && isTouchBottomBorder(player1) {
		player1.MoveDown()
	}

	if key == "Up" && isTouchTopBorder(player2) {
		player2.MoveUp()
	}

	if key == "Down" && isTouchBottomBorder(player2) {
		player2.MoveDown()
	}
}

func initUserInput() chan string {
	//初始化channel，去接另一個goroutine丟回來的資料
	inputChan := make(chan string)

	//建立一個goroutine去監聽鍵盤的事件
	go func() {
		for {
			switch ev := screen.PollEvent().(type) {
			case *tcell.EventResize:
				drawView()
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

	player1 = &core.Paddle{
		GameObject: core.GameObject{Row: paddleStart, Col: 0, Width: 1,
			Height: PaddleHeight, Symbol: PaddleSymbol,
			VelRow: 0, VelCol: 0},
		NickName:     "Player one",
		CurrentScore: 0,
	}

	player2 = &core.Paddle{
		GameObject: core.GameObject{Row: paddleStart, Col: width - 2, Width: 1,
			Height: PaddleHeight, Symbol: PaddleSymbol,
			VelRow: 0, VelCol: 0},
		NickName:     "Player two",
		CurrentScore: 0,
	}

	ball = &core.Ball{
		GameObject: core.GameObject{Row: height / 2, Col: width / 2, Width: 1, Height: 1, Symbol: BallSymbol,
			VelRow: BallVelocityRow, VelCol: BallVelocityCol},
	}

}

func updateState() {
	//兩個球拍
	player1.Row += player1.VelRow
	player2.Row += player2.VelRow

	//球
	ball.Row += ball.VelRow
	ball.Col += ball.VelCol

	//檢查有沒有撞到上下牆壁
	if isCollidesWithWall(ball) {
		ball.VelRow = -ball.VelRow
	}
	//檢查是否有碰到球拍
	if isTouchPaddle(ball) {
		ball.VelCol = -ball.VelCol
	}

	if isBallOutSide(ball) {
		calculateScore(ball)
		resetNewRound(ball)
	}

	over, _ := isGameOver()
	if over {
		os.Exit(0)
	}
}

func resetNewRound(ball *core.Ball) {
	width, height := screen.Size()
	ball.Row = height / 2
	ball.Col = width / 2
}

func isGameOver() (bool, *core.Paddle) {
	if player1.CurrentScore == FinalScore {
		return true, player1
	}
	if player2.CurrentScore == FinalScore {
		return true, player2
	}
	return false, nil
}

func isBallOutSide(ball *core.Ball) bool {
	width, _ := screen.Size()
	return ball.Col < 0 || ball.Col > width
}

func calculateScore(ball *core.Ball) {
	if ball.Col < 0 {
		player2.CurrentScore += 1
	}
	width, _ := screen.Size()
	if ball.Col > width {
		player1.CurrentScore += 1
	}
}

func isTouchPaddle(ball *core.Ball) bool {
	if ball.Col+ball.VelCol == player1.Col &&
		(ball.Row > player1.Row && ball.Row <= player1.Row+PaddleHeight) {
		return true
	} else if ball.Col+ball.VelCol == player2.Col &&
		(ball.Row > player2.Row && ball.Row <= player2.Row+PaddleHeight) {
		return true
	}
	return false
}

func isTouchBottomBorder(paddle *core.Paddle) bool {
	_, screenHeight := screen.Size()
	return (paddle.Row + paddle.Height) < screenHeight
}

func isTouchTopBorder(paddle *core.Paddle) bool {
	return paddle.Row > 0
}

func isCollidesWithWall(ball *core.Ball) bool {
	_, screenHeight := screen.Size()
	return ball.Row+ball.VelRow < 0 || ball.Row+ball.VelRow >= screenHeight
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
	windowWidth, windowheight := screen.Size()
	//刷新畫面時，確保player2球拍能在最右邊
	player2.Col = windowWidth - 2
	Print(player1.Row, player1.Col, player1.Width, player1.Height, PaddleSymbol)
	Print(player2.Row, player2.Col, player2.Width, player2.Height, PaddleSymbol)

	//球
	Print(ball.Row, ball.Col, ball.Width, ball.Height, PaddleSymbol)

	//中線
	Print(0, windowWidth/2, 1, windowheight, 0x2590)

	//分數更新
	drawLetters(windowWidth/4, 1, strconv.Itoa(player1.CurrentScore))
	drawLetters((windowWidth/4)*3, 1, strconv.Itoa(player2.CurrentScore))

	screen.Show()
}

func Print(row, col, width, height int, ch rune) {
	for r := 0; r < height; r++ {
		for c := 0; c < width; c++ {
			screen.SetContent(col+c, row+r, ch, nil, tcell.StyleDefault)
		}
	}
}

func drawLetters(x int, y int, word string) {
	letterWidth, letterHeight := 1, 1
	letterNum := len(word)
	totalLen := letterNum*letterWidth + (letterNum - 1)
	startX := x - totalLen/2
	offsetY := y - letterHeight/2

	for i, letter := range word {
		letterCells := getCellsFromChar(string(letter))

		offsetX := startX + totalLen/letterNum*i
		offsetX += i - 1

		for _, cell := range letterCells {
			finalX := offsetX + int(cell[0])
			finalY := offsetY + int(cell[1])
			screen.SetContent(finalX, finalY, BallSymbol, nil, tcell.Style(tcell.ColorWhite))
		}
	}
}

func start() {
	initScreen()
	initGameState()
	startGameLoop()
}
