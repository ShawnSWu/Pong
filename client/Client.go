package main

import (
	"Pong/client/core"
	"bufio"
	"fmt"
	"github.com/gdamore/tcell"
	"net"
	"os"
	"strconv"
	"strings"
)

var screen tcell.Screen

var player1 *core.Paddle
var player2 *core.Paddle
var ball *core.Ball

var paddles []*core.Paddle

const PaddleSymbol = 0x2588 // 球拍符號
const PaddleHeight = 6      // 球拍高度
const windowHeight = 60
const windowWidth = 150
const BallVelocityRow = 1
const BallVelocityCol = 1
const BallSymbol = 0x25CF // 球符號

func initGameState() {

	paddleStart := windowHeight/2 - PaddleHeight/2

	player1 = &core.Paddle{
		GameObject: core.GameObject{Row: paddleStart, Col: 0, Width: 1,
			Height: PaddleHeight, Symbol: PaddleSymbol,
			VelRow: 0, VelCol: 0},
		NickName:     "Player one",
		CurrentScore: 0,
	}

	player2 = &core.Paddle{
		GameObject: core.GameObject{Row: paddleStart, Col: windowWidth - 2, Width: 1,
			Height: PaddleHeight, Symbol: PaddleSymbol,
			VelRow: 0, VelCol: 0},
		NickName:     "Player two",
		CurrentScore: 0,
	}

	ball = &core.Ball{
		GameObject: core.GameObject{Row: windowHeight / 2, Col: windowWidth / 2, Width: 1, Height: 1, Symbol: BallSymbol,
			VelRow: BallVelocityRow, VelCol: BallVelocityCol},
	}

	paddles = []*core.Paddle{
		player1,
		player2,
	}
}

func main() {

	conn := connect()

	initScreen()
	initGameState()

	go listenOperation(conn)

	readServerMsg(conn)

}

func listenOperation(conn *net.TCPConn) {
	var sendMessage string

	for {
		switch ev := screen.PollEvent().(type) {
		case *tcell.EventKey:

			switch ev.Name() {

			case "Up":
				sendMessage = "U"
				break

			case "Down":
				sendMessage = "D"
				break

			case "Rune[w]":
				sendMessage = "w"
				break

			case "Rune[s]":
				sendMessage = "s"
				break
			}
		}

		if sendMessage != "" {
			// send to server
			sendData(conn, sendMessage)
		}
	}
}

func connect() *net.TCPConn {
	service := "127.0.0.1:4321"
	tcpAddr, _ := net.ResolveTCPAddr("tcp", service)
	conn, _ := net.DialTCP("tcp", nil, tcpAddr)
	return conn
}

func sendData(connP *net.TCPConn, msg string) {
	conn := *connP
	conn.Write([]byte(msg + "."))
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

func updateState(payload string) {
	//ballX, ballY, player1X, player1Y,player1Score, player2X, player2Y,player2Score
	p := strings.Split(payload, ",")

	ballCol, _ := strconv.Atoi(p[0])
	ball.Col = ballCol

	ballRow, _ := strconv.Atoi(p[1])
	ball.Row = ballRow

	player1Col, _ := strconv.Atoi(p[2])
	player1.Col = player1Col

	player1Row, _ := strconv.Atoi(p[3])
	player1.Row = player1Row

	player1CurrentScore, _ := strconv.Atoi(p[4])
	player1.CurrentScore = player1CurrentScore

	player2Col, _ := strconv.Atoi(p[5])
	player2.Col = player2Col

	player2Row, _ := strconv.Atoi(p[6])
	player2.Row = player2Row

	player2CurrentScore, _ := strconv.Atoi(p[7])
	player2.CurrentScore = player2CurrentScore
}

func readServerMsg(conn *net.TCPConn) {

	for {
		payload, _ := bufio.NewReader(conn).ReadString('.')
		if payload == "" {
			continue
		}
		payload = string([]byte(payload)[:len(payload)-1])
		if payload == "caf" {
			fmt.Println("目前滿人")
		} else {
			updateState(payload)
			drawView()
		}
	}
}

func drawView() {
	screen.Clear()
	//兩個球拍
	for _, obj := range paddles {
		Print(obj.Row, obj.Col, obj.Width, obj.Height, obj.Symbol)
	}
	//球
	Print(ball.Row, ball.Col, ball.Width, ball.Height, ball.Symbol)

	//中線
	width, height := screen.Size()
	Print(0, width/2, 1, height, 0x2590)

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
		letterCells := core.GetCellsFromChar(string(letter))

		offsetX := startX + totalLen/letterNum*i
		offsetX += i - 1

		for _, cell := range letterCells {
			finalX := offsetX + int(cell[0])
			finalY := offsetY + int(cell[1])
			screen.SetContent(finalX, finalY, ball.Symbol, nil, tcell.Style(tcell.ColorWhite))
		}
	}
}
