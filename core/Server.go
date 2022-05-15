package core

import (
	"Pong/logger"
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const PayloadTerminator = "~"

const GameStatusWaiting = "Waiting"
const GameStatusPlaying = "Playing"
const ConnBrokenStatus = "CB"

const ConnWorking = 1
const ConnBroken = 0

const FinalScore = 9        // 遊戲結束分數
const BallSymbol = 0x25CF   // 球符號
const PaddleSymbol = 0x2588 // 球拍符號
const PaddleHeight = 6      // 球拍高度
const BallVelocityRow = 1
const BallVelocityCol = 1

const windowHeight = 60
const windowWidth = 150

const MaxRoomCount = 10

//大廳玩家的連線
var lobbyPlayer = make(map[string]*net.Conn)

//最多同時10間房間
var lobbyRoom = make([]*Room, 0, 10)

//Room跟main goroutine的溝通channel
var roomChanMsg = make(chan string)

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

func roomGameStart(room *Room) {

	player1 := room.Player1
	player2 := room.Player2
	conn1 := room.Player1Conn
	conn2 := room.Player2Conn

	go readClientInput(room, conn1, player1)
	go readClientInput(room, conn2, player2)

	for {
		updateState(room)
		conn1SendStatus := sendGameState(conn1, room)
		conn2SendStatus := sendGameState(conn2, room)

		if conn1SendStatus == ConnBroken || conn2SendStatus == ConnBroken {
			return
		}
		time.Sleep(65 * time.Millisecond)
	}
}

func sendMsg(connP *net.Conn, payload string) int {
	conn := *connP

	logger.Log.Info(fmt.Sprintf(sendMsgContent, conn.RemoteAddr().String(), payload))
	_, err := conn.Write([]byte(payload))

	//斷線了 傳訊息給main goroutine
	if err != nil {
		//CB mean connection broken
		roomChanMsg <- fmt.Sprintf("%s_%s%s", ConnBrokenStatus, ConnBrokenMsg, PayloadTerminator)
		logger.Log.Error(ConnBrokenMsg + " => " + fmt.Sprintf("%s_%s", ConnBrokenStatus, ConnBrokenMsg))
		return ConnBroken
	}
	return ConnWorking
}

func sendGameState(connP *net.Conn, room *Room) int {
	conn := *connP
	player1 := room.Player1
	player2 := room.Player2
	ball := room.Ball

	ballX, ballY := ball.Col, ball.Row
	player1X, player1Y, player1Score := player1.Col, player1.Row, player1.CurrentScore
	player2X, player2Y, player2Score := player2.Col, player2.Row, player2.CurrentScore

	//ballX, ballY, player1X, player1Y,player1Score, player2X, player2Y,player2Score
	payload := fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d%s", ballX, ballY,
		player1X, player1Y, player1Score, player2X, player2Y, player2Score, PayloadTerminator)

	_, err := conn.Write([]byte(payload))

	//斷線了 通知main goroutine
	if err != nil {
		//CB mean connection broken
		roomChanMsg <- fmt.Sprintf("%s_%s_%s%s", ConnBrokenStatus, room.RoomId, conn.RemoteAddr().String(), PayloadTerminator)
		return ConnBroken
	}
	return ConnWorking
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

func StartService() {

	tcpAddr, _ := net.ResolveTCPAddr("tcp4", "127.0.0.1:4321")
	listener, _ := net.ListenTCP("tcp", tcpAddr)

	go listenRoomChannel()

	for {
		logger.Log.Info("等待新玩家連線...")

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

		//產生遊戲大廳的Room列表
		roomInfoPayload := generateRoomInfoPayload()
		//傳送遊戲大廳列表給所有玩家
		notifyAllPlayer(roomInfoPayload)
		logger.Log.Info("All- " + roomInfoPayload)

		//暫時註解，這裡是Room創建後的邏輯
		//if len(lobbyPlayer) < 2 {
		//	continue
		//}
		//
		//if len(lobbyPlayer) == 2 {
		//	player1, player2, ball := generateGameElement(ip)
		//
		//	var tempList []*net.Conn
		//	for _, v := range lobbyPlayer {
		//		tempList = append(tempList, v)
		//	}
		//
		//	now := time.Now()
		//
		//	room := &Room{
		//		RoomId:      uuid.New().String(),
		//		Player1:     player1,
		//		Player2:     player2,
		//		Name:        fmt.Sprintf("New Room %d", len(lobbyRoom)),
		//		GameStatus:  GameStatusWaiting,
		//		CreateDate:  now.Format("2006-02-01 15:01"),
		//		Ball:        ball,
		//		Player1Conn: tempList[0],
		//		Player2Conn: tempList[1],
		//	}
		//
		//	go startOnline(room)
		//
		//	lobbyRoom = append(lobbyRoom, room)
		//
		//	//開始遊戲後移除LobbyPlayer等待下一波玩家
		//	lobbyPlayer = make(map[string]*net.Conn)
		//}
		time.Sleep(10 * time.Millisecond)
	}
}

func notifyAllPlayer(roomInfoPayload string) {
	for _, playerConn := range lobbyPlayer {
		sendMsg(playerConn, roomInfoPayload)
	}
}

func listenRoomChannel() {
	for {
		select {
		case msg := <-roomChanMsg:

			if strings.HasPrefix(msg, ConnBrokenStatus) {
				split := strings.Split(msg, "_")
				roomId := split[1]
				//斷線者的ip
				connBrokenIp := split[2]

				room := findRoomById(roomId)

				//有人斷線 刪除大廳中房間
				lobbyRoom = removeRoom(lobbyRoom, roomId)

				//通知另一方對手已斷線
				if room.Player1.IpAddress == connBrokenIp {
					m := fmt.Sprintf("%s_%s_%s%s", ConnBrokenStatus, room.RoomId, room.Player1.IpAddress, PayloadTerminator)

					sendMsg(room.Player2Conn, m)
					delete(lobbyPlayer, room.Player2.IpAddress)
				} else {
					m := fmt.Sprintf("%s_%s_%s%s", ConnBrokenStatus, room.RoomId, room.Player2.IpAddress, PayloadTerminator)

					sendMsg(room.Player1Conn, m)
					delete(lobbyPlayer, room.Player1.IpAddress)
				}
			}
		}
	}
}

func findRoomById(id string) *Room {
	var targetIndex int
	for i, room := range lobbyRoom {
		if room.RoomId == id {
			targetIndex = i
			break
		}
	}
	return lobbyRoom[targetIndex]
}

func removeRoom(rooms []*Room, roomId string) []*Room {
	var index int
	for i, v := range rooms {
		if v.RoomId == roomId {
			index = i
			break
		}
	}
	return append(rooms[:index], rooms[index+1:]...)
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
	roomGameStart(room)
}

type RoomInfo struct {
	roomName    string
	createDate  string
	playerCount int
}

func getRoomList() []RoomInfo {
	var roomInfoSlice = make([]RoomInfo, 0, 10)

	for i := 0; i < len(lobbyRoom); i++ {
		roomName := lobbyRoom[i].Name
		createDate := lobbyRoom[i].CreateDate

		playerCount := 0
		if lobbyRoom[i].Player1Conn != nil {
			playerCount += 1
		}
		if lobbyRoom[i].Player2Conn != nil {
			playerCount += 1
		}

		roomInfoSlice = append(roomInfoSlice, RoomInfo{roomName, createDate, playerCount})
	}

	return roomInfoSlice
}

func generateRoomInfoPayload() string {
	riList := getRoomList()
	var payload string
	for i := 0; i < len(riList); i += 1 {
		rn := riList[i].roomName
		cd := riList[i].createDate
		pc := riList[i].playerCount

		payload = fmt.Sprintf("%s+%s+%d&", rn, cd, pc)
	}

	return payload + PayloadTerminator
}
