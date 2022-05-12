package main

import (
	core "Pong/core"
	"bufio"
	"fmt"
	"io"
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

var player1 *core.Paddle
var player2 *core.Paddle
var ball *core.Ball

var playerList = make(map[string]string)

var paddles = make(map[string]*core.Paddle)

var playerConn []*net.Conn

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
		handleInput(userCommand, connP)
		time.Sleep(10 * time.Millisecond)
	}
}

func startService() {

	conn1 := playerConn[0]
	conn2 := playerConn[1]

	go readClientInput(conn1)
	go readClientInput(conn2)

	for {
		updateState()
		sendToClient(conn1)
		sendToClient(conn2)
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

func handleInput(userCommand string, connP *net.Conn) {
	conn := *connP
	if userCommand == "" {
		return
	}
	userCommand = string([]byte(userCommand)[:len(userCommand)-1])

	paddle := playerList[conn.RemoteAddr().String()]
	fmt.Println("@@@@@@@@@@@@ ", paddle)

	switch userCommand {
	case "U":
		if paddle == "player1" {
			player1.MoveUp()
		} else {
			player2.MoveUp()
		}
		break

	case "D":
		if paddle == "player1" {
			player1.MoveDown()
		} else {
			player2.MoveDown()
		}
		break
	}
}

func waitingPlayer() {

	tcpAddr, _ := net.ResolveTCPAddr("tcp4", "127.0.0.1:4321")
	listener, _ := net.ListenTCP("tcp", tcpAddr)

	var c = 0
	for {
		fmt.Println("等待連線....")

		conn, _ := listener.Accept()

		if len(paddles) < 2 {
			ip := conn.RemoteAddr().String()
			c += 1
			if playerList[ip] == "" {
				playerList[ip] = fmt.Sprintf("player%d", c)
			}

			fmt.Println("玩家人數 ", playerList)
			playerConn = append(playerConn, &conn)

			if len(playerList) == 2 {
				startOnline()
			}

			//監視玩家連線狀態
			//go monitorConnectionStatus(conn)

			continue
		} else {
			//人數已滿
			conn.Write([]byte("caf."))
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func generatePlayer(ip string) *core.Paddle {
	return &core.Paddle{IpAddress: ip}
}

func startOnline() {
	fmt.Println("遊戲開始！！！")
	initGameState()
	startService()
}

func main() {
	waitingPlayer()
}

func monitorConnectionStatus(conn net.Conn) {
	defer conn.Close()
	notify := make(chan error)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				notify <- err
				return
			}
			if n > 0 {
				fmt.Println("unexpected data: %s", buf[:n])
			}
		}
	}()
	connectAlive := true

	for {
		select {
		case err := <-notify:
			if io.EOF == err {
				ip := conn.RemoteAddr().String()
				fmt.Println(fmt.Sprintf("%s斷線,", ip), err)
				delete(paddles, ip)
				fmt.Println("目前人數:", len(paddles))
				connectAlive = false
				break
			}
		case <-time.After(time.Second * 1):
			cm := fmt.Sprintf("%s, still alive", conn.RemoteAddr().String())
			fmt.Println(cm)
		}

		if connectAlive == false {
			break
		}
	}

}
