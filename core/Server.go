package core

import (
	"Pong/logger"
	"bufio"
	"fmt"
	"github.com/google/uuid"
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

const MaxRoomCount = 10

//大廳玩家
var lobbyPlayer = make(map[string]*net.Conn)

//最多同時10間房間
var lobbyRoom = make([]*Room, 0, 10)

func updateState(room *Room) {
	player1 := room.Player1
	player2 := room.Player2
	ball := room.Ball

	//玩家ㄧ球拍
	player1.Row += player1.VelRow
	player1.Col += player1.VelCol
	//玩家二球拍
	player2.Row += player2.VelRow
	player2.Col += player2.VelCol

	//球
	ball.Row += ball.VelRow
	ball.Col += ball.VelCol

	//檢查有沒有撞到上下牆壁
	if isCollidesWithWall(ball) {
		ball.VelRow = -ball.VelRow
	}
	//檢查是否有碰到球拍
	if isTouchPaddle(room) {
		ball.VelCol = -ball.VelCol
	}

	if isBallOutSide(ball) {
		calculateScore(room)
		resetNewRound(ball)
	}

	over, _ := isGameOver(room)
	if over {
		os.Exit(0)
	}
}

func resetNewRound(ball *Ball) {
	ball.Row = windowHeight / 2
	ball.Col = windowWidth / 2
}

func isGameOver(room *Room) (bool, *Player) {
	player1 := room.Player1
	player2 := room.Player2

	if player1.CurrentScore == FinalScore {
		return true, player1
	}
	if player2.CurrentScore == FinalScore {
		return true, player2
	}
	return false, nil
}

func isBallOutSide(ball *Ball) bool {
	return ball.Col < 0 || ball.Col > windowWidth
}

func calculateScore(room *Room) {
	player1 := room.Player1
	player2 := room.Player2
	ball := room.Ball

	if ball.Col < 0 {
		player2.CurrentScore += 1
	}
	if ball.Col > windowWidth {
		player1.CurrentScore += 1
	}
}

func isTouchPaddle(room *Room) bool {
	player1 := room.Player1
	player2 := room.Player2
	ball := room.Ball

	if ball.Col+ball.VelCol <= player1.Col &&
		(ball.Row > player1.Row && ball.Row <= player1.Row+PaddleHeight) {
		return true
	} else if ball.Col+ball.VelCol >= player2.Col &&
		(ball.Row > player2.Row && ball.Row <= player2.Row+PaddleHeight) {
		return true
	}
	return false
}

func isTouchBottomBorder(paddle *Player) bool {
	return (paddle.Row + paddle.Height) < windowHeight
}

func isTouchTopBorder(paddle *Player) bool {
	return paddle.Row > 0
}

func isCollidesWithWall(ball *Ball) bool {
	return ball.Row+ball.VelRow < 0 || ball.Row+ball.VelRow >= windowHeight
}

func readClientInput(room *Room, connP *net.Conn, player *Player) {
	conn := *connP

	for {
		userCommand, _ := bufio.NewReader(conn).ReadString('.')
		handleInput(room, userCommand, player)
		time.Sleep(10 * time.Millisecond)
	}
}

func startService(room *Room) {

	player1 := room.Player1
	player2 := room.Player2
	conn1 := room.Player1Conn
	conn2 := room.Player2Conn

	go readClientInput(room, conn1, player1)
	go readClientInput(room, conn2, player2)

	for {
		updateState(room)
		sendToClient(room, conn1)
		sendToClient(room, conn2)
		time.Sleep(65 * time.Millisecond)
	}
}

func sendToClient(room *Room, connP *net.Conn) {
	player1 := room.Player1
	player2 := room.Player2
	ball := room.Ball

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

func handleInput(room *Room, userCommand string, player *Player) {
	if userCommand == "" {
		return
	}
	player1 := room.Player1
	player2 := room.Player2

	userCommand = string([]byte(userCommand)[:len(userCommand)-1])

	switch userCommand {
	case "U":
		if player.RightOrLeft == "left" && isTouchTopBorder(player1) {
			player1.MoveUp()
		}

		if player.RightOrLeft == "right" && isTouchTopBorder(player2) {
			player2.MoveUp()
		}
		break

	case "D":
		if player.RightOrLeft == "left" && isTouchBottomBorder(player1) {
			player1.MoveDown()
		}

		if player.RightOrLeft == "right" && isTouchBottomBorder(player2) {
			player2.MoveDown()
		}
		break
	}
}

func waitingPlayer() {

	tcpAddr, _ := net.ResolveTCPAddr("tcp4", "127.0.0.1:4321")
	listener, _ := net.ListenTCP("tcp", tcpAddr)

	for {
		logger.Log.Info("Server launch 等待連線...")

		conn, _ := listener.Accept()
		ip := conn.RemoteAddr().String()
		logger.Log.Info(fmt.Sprintf("Player已連線 (ip:%s)", ip))

		if len(lobbyRoom) >= MaxRoomCount {
			//sent message to client say full room
			//close connection
			conn.Close()
			logger.Log.Info(fmt.Sprintf("連線已滿，關閉ip:%s 的連線", ip))
			continue
		}

		if lobbyPlayer[ip] == nil {
			lobbyPlayer[ip] = &conn
			logger.Log.Info(fmt.Sprintf("Player ip:%s 進入大廳", ip))
		}

		if len(lobbyPlayer) < 2 {
			continue
		}

		if len(lobbyPlayer) == 2 {
			player1, player2, ball := generateGameElement(ip)

			var tempList []*net.Conn
			for _, v := range lobbyPlayer {
				tempList = append(tempList, v)
			}

			room := &Room{
				RoomId:      uuid.New().String(),
				Player1:     player1,
				Player2:     player2,
				Ball:        ball,
				Player1Conn: tempList[0],
				Player2Conn: tempList[1],
			}

			go startOnline(room)

			lobbyRoom = append(lobbyRoom, room)

			//開始遊戲後移除LobbyPlayer等待下一波玩家
			lobbyPlayer = make(map[string]*net.Conn)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func generateGameElement(ip string) (*Player, *Player, *Ball) {
	paddleStart := windowHeight/2 - PaddleHeight/2

	player1 := &Player{
		GameObject: GameObject{Row: paddleStart, Col: 0,
			Width: 1, Height: PaddleHeight,
			Symbol: PaddleSymbol,
			VelRow: 0, VelCol: 0,
		},
		NickName:     "Player",
		IpAddress:    ip,
		CurrentScore: 0,
		RightOrLeft:  "left",
	}

	player2 := &Player{
		GameObject: GameObject{Row: paddleStart, Col: windowWidth - 2, Width: 1,
			Height: PaddleHeight, Symbol: PaddleSymbol,
			VelRow: 0, VelCol: 0},
		NickName:     "Player two",
		IpAddress:    ip,
		CurrentScore: 0,
		RightOrLeft:  "right",
	}

	ball := &Ball{
		GameObject: GameObject{Row: windowHeight / 2, Col: windowWidth / 2, Width: 1, Height: 1, Symbol: BallSymbol,
			VelRow: BallVelocityRow, VelCol: BallVelocityCol},
	}

	return player1, player2, ball
}

func startOnline(room *Room) {
	logger.Log.Info(fmt.Sprintf("Room id:%s 遊戲開始！", room.RoomId))
	startService(room)
}

func Start() {
	waitingPlayer()
}

/*
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
*/
