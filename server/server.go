package server

import (
	core "Pong/core"
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

const FinalScore = 9        // 遊戲結束分數
const BallSymbol = 0x25CF   // 球符號
const PaddleSymbol = 0x2588 // 球拍符號
const PaddleHeight = 6      // 球拍高度
const BallVelocityRow = 1
const BallVelocityCol = 1

const windowHeight = 60
const windowWidth = 150

var onlinePlayerCount = 0

var player1 *core.Paddle
var player2 *core.Paddle
var ball *core.Ball

var paddles []*core.Paddle

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

func updateState() {
	//兩個球拍
	for i := range paddles {
		paddles[i].Row += paddles[i].VelRow
		paddles[i].Col += paddles[i].VelCol
	}
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
	ball.Row = windowHeight / 2
	ball.Col = windowWidth / 2
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
	return ball.Col < 0 || ball.Col > windowWidth
}

func calculateScore(ball *core.Ball) {
	if ball.Col < 0 {
		player2.CurrentScore += 1
	}
	if ball.Col > windowWidth {
		player1.CurrentScore += 1
	}
}

func isTouchPaddle(ball *core.Ball) bool {
	if ball.Col+ball.VelCol <= player1.Col &&
		(ball.Row > player1.Row && ball.Row <= player1.Row+PaddleHeight) {
		return true
	} else if ball.Col+ball.VelCol >= player2.Col &&
		(ball.Row > player2.Row && ball.Row <= player2.Row+PaddleHeight) {
		return true
	}
	return false
}

func isTouchBottomBorder(paddle *core.Paddle) bool {
	return (paddle.Row + paddle.Height) < windowHeight
}

func isTouchTopBorder(paddle *core.Paddle) bool {
	return paddle.Row > 0
}

func isCollidesWithWall(ball *core.Ball) bool {
	return ball.Row+ball.VelRow < 0 || ball.Row+ball.VelRow >= windowHeight
}

func readClientInput(connP *net.Conn) {
	conn := *connP

	for {
		userCommand, _ := bufio.NewReader(conn).ReadString('.')
		//fmt.Print("Message Address:", conn.RemoteAddr(), "\n")
		//fmt.Print("Message Received:", userCommand, "\n")

		handleInput(userCommand)
		time.Sleep(10 * time.Millisecond)
	}
}

func startService() {

	tcpAddr, _ := net.ResolveTCPAddr("tcp4", "127.0.0.1:4321")
	listener, _ := net.ListenTCP("tcp", tcpAddr)

	// accept connection
	conn, _ := listener.Accept()

	if player1.IpAddress != "" {
		player1.IpAddress = conn.RemoteAddr().String()
	} else {
		player2.IpAddress = conn.RemoteAddr().String()
	}

	go readClientInput(&conn)

	for {
		updateState()
		sendToClient(&conn)
		time.Sleep(65 * time.Millisecond)
	}
}

func sendToClient(connP *net.Conn) {
	ballX := ball.Col
	ballY := ball.Row
	player1X := player1.Col
	player1Y := player1.Row
	player1Score := player1.CurrentScore
	player2X := player2.Col
	player2Y := player2.Row
	player2Score := player2.CurrentScore

	//ballX, ballY, player1X, player1Y,player1Score, player2X, player2Y,player2Score
	payload := fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d.", ballX, ballY,
		player1X, player1Y, player1Score, player2X, player2Y, player2Score)
	fmt.Println(payload)
	conn := *connP
	conn.Write([]byte(payload))
}

func handleInput(userCommand string) {
	if userCommand == "" {
		return
	}
	userCommand = string([]byte(userCommand)[:len(userCommand)-1])
	switch userCommand {
	case "U":
		player2.MoveUp()
		break

	case "D":
		player2.MoveDown()
		break

	case "w":
		player1.MoveUp()
		break

	case "s":
		player1.MoveDown()
		break
	}
}

func startOnline() {
	initGameState()
	startService()
}

func main() {
	startOnline()
}
