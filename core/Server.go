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

const SceneLobby = "Lobby"
const SceneRoom = "Room"
const SceneBattle = "Battle"

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

var RoomInitialId = 0

//大廳玩家的連線
var lobbyPlayer = make(map[string]*net.Conn)

//最多同時6間房間
var lobbyRoom = make([]*Room, 0, 6)

var lobbyRoomMap = make(map[string]*Room)

//Room跟main goroutine的溝通channel
var roomChanMsg = make(chan string)

func updateState(room *Room) {
	player1 := room.players[0]
	player2 := room.players[1]
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
	player1 := room.players[0]
	player2 := room.players[1]

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
	player1 := room.players[0]
	player2 := room.players[1]
	ball := room.Ball

	if ball.Col < 0 {
		player2.CurrentScore += 1
	}
	if ball.Col > windowWidth {
		player1.CurrentScore += 1
	}
}

func isTouchPaddle(room *Room) bool {
	player1 := room.players[0]
	player2 := room.players[1]
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

func readBattleOperation(room *Room, connP *net.Conn, player *Player) {
	conn := *connP

	for {
		userCommand, _ := bufio.NewReader(conn).ReadString('.')
		handleBattleOperation(userCommand, player)
		time.Sleep(10 * time.Millisecond)
	}
}

func listenPlayerOperation(connP *net.Conn, player *Player) {
	conn := *connP

	for {
		payload, _ := bufio.NewReader(conn).ReadString('~')
		header := string([]byte(payload)[0:2])
		payload = removeHeaderTerminator(payload)

		switch player.Scene {
		//在大廳 要能進入房間,創造房間與離開大廳
		case SceneLobby:

			switch header {
			case CreateRoom:
				//TODO 定義創建房間封包
				r := &Room{
					RoomId:     uuid.New().String(),
					Name:       fmt.Sprintf("New Room %d", len(lobbyRoom)),
					GameStatus: GameStatusWaiting,
					CreateDate: time.Now().Format("2006-02-01 15:01"),
				}
				lobbyRoom = append(lobbyRoom, r)
				break

			case EnterRoom:
				//TODO 定義進入房間封包

				break

			case LeaveLobby:
				//TODO 定義離開大廳封包
				break
			}
			break

		//在房間 要能按下準備與離開房間
		case SceneRoom:
			switch header {
			case LeaveRoom:
				//TODO 定義離開房間封包
				break

			case ReadyStart:
				//TODO 定義準備開始遊戲封包
				break
			}

			break

		case SceneBattle:
			//在戰鬥中 要能移動球拍

			switch header {

			case BattleSituation:
				handleBattleOperation(payload, player)
				break
			}
			time.Sleep(10 * time.Millisecond)
			break

		}

		time.Sleep(10 * time.Millisecond)
	}
}

func roomGameStart(room *Room) {

	player1 := room.players[0]
	player2 := room.players[1]
	conn1 := room.players[0].Conn
	conn2 := room.players[1].Conn

	go readBattleOperation(room, conn1, player1)
	go readBattleOperation(room, conn2, player2)

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
		roomChanMsg <- fmt.Sprintf("%s%s%s", ConnBrokenMsgHeader, conn.RemoteAddr().String(), PayloadTerminator)
		logger.Log.Error(ConnBrokenMsg + " => " + fmt.Sprintf("%s_%s", ConnBrokenMsgHeader, ConnBrokenMsg))
		return ConnBroken
	}
	return ConnWorking
}

func sendGameState(connP *net.Conn, room *Room) int {
	conn := *connP
	player1 := room.players[0]
	player2 := room.players[1]
	ball := room.Ball

	ballX, ballY := ball.Col, ball.Row
	player1X, player1Y, player1Score := player1.Col, player1.Row, player1.CurrentScore
	player2X, player2Y, player2Score := player2.Col, player2.Row, player2.CurrentScore

	//ballX, ballY, player1X, player1Y,player1Score, player2X, player2Y,player2Score
	payload := fmt.Sprintf("%s%d,%d,%d,%d,%d,%d,%d,%d%s", BattleSituation, ballX, ballY,
		player1X, player1Y, player1Score, player2X, player2Y, player2Score, PayloadTerminator)

	_, err := conn.Write([]byte(payload))

	//斷線了 通知main goroutine
	if err != nil {
		//CB mean connection broken
		roomChanMsg <- fmt.Sprintf("%s%s%s", ConnBrokenMsgHeader, conn.RemoteAddr().String(), PayloadTerminator)
		return ConnBroken
	}
	return ConnWorking
}

func handleBattleOperation(userCommand string, player *Player) {
	if userCommand == "" {
		return
	}

	switch userCommand {
	case "U":
		if isTouchTopBorder(player) {
			player.MoveUp()
		}
		break

	case "D":
		if isTouchBottomBorder(player) {
			player.MoveDown()
		}
		break
	}
}

func StartService() {

	//測試資料
	insetTestData()

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

		//產生玩家
		player := generatePlayer(ip, &conn)

		go listenPlayerOperation(&conn, player)

		//產生遊戲大廳的Room列表
		roomInfoPayload := generateRoomsInfoPayload()

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

			header := string([]byte(msg)[0:2])

			switch header {

			case ConnBrokenMsgHeader:
				msg = removeHeaderTerminator(msg) //移除Header與終止符
				//斷線者的ip
				connBrokenIp := msg
				logger.Log.Info("有人斷線！")

				//斷線處理機制
				connBrokenHandle(connBrokenIp)

				//處理完後，通知所有玩家，更新大廳與房間資訊
				roomsInfo := generateRoomsInfoPayload()
				notifyAllPlayer(roomsInfo)
				logger.Log.Info("已通知所有玩家！")
				break
			}

		}
	}
}

func isPlayerPlaying(playerIp string) (bool, *Room) {
	room := getPlayerRoom(playerIp)
	isPlaying := false

	if room != nil && room.GameStatus == GameStatusPlaying {
		isPlaying = true
	}
	return isPlaying, room
}

func getPlayerRoom(playerIp string) *Room {
	var roomIndex int
	found := false
	for i, r := range lobbyRoom {
		for _, player := range r.players {
			if playerIp == player.IpAddress {
				roomIndex = i
				found = true
				break
			}
		}
		if found == true {
			break
		}
	}
	return lobbyRoom[roomIndex]
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

func generatePlayer(ip string, conn *net.Conn) *Player {
	return &Player{
		NickName:  "Player",
		IpAddress: ip,
		Conn:      conn,
		Scene:     SceneLobby,
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
	roomGameStart(room)
}

type RoomInfo struct {
	roomId      string
	roomName    string
	createDate  string
	playerCount int
}

func getRoomList() []RoomInfo {
	var roomInfoSlice = make([]RoomInfo, 0, 10)

	for i := 0; i < len(lobbyRoom); i++ {
		roomId := lobbyRoom[i].RoomId
		roomName := lobbyRoom[i].Name
		createDate := lobbyRoom[i].CreateDate

		playerCount := len(lobbyRoom[i].players)

		roomInfoSlice = append(roomInfoSlice, RoomInfo{roomId, roomName, createDate, playerCount})
	}

	return roomInfoSlice
}

func connBrokenHandle(connBrokenIp string) {
	//如果此玩家正在遊戲中 則通知對手，然後刪除Room
	playing, room := isPlayerPlaying(connBrokenIp)

	//遊戲中的話則通知對手，然後刪除Room
	if playing {
		//有人斷線 刪除大廳中房間
		lobbyRoom = removeRoom(lobbyRoom, room.RoomId)

		player1 := room.players[0]
		player2 := room.players[1]

		//通知另一方對手已斷線
		if player1.IpAddress == connBrokenIp {
			msg := generateConnBrokenPayload(player1.IpAddress)
			sendMsg(player2.Conn, msg)
			//刪除房間
			delete(lobbyPlayer, player2.IpAddress)
		} else {
			msg := generateConnBrokenPayload(player2.IpAddress)
			sendMsg(player1.Conn, msg)
			//刪除房間
			delete(lobbyPlayer, player1.IpAddress)
		}
	}

	//如果沒有正在遊戲中，但在某間房間裡面，則更新那個房間的人數資訊(連線)
	if room != nil {
		var toRemoveIndex = 0
		for i, player := range room.players {
			if connBrokenIp == player.IpAddress {
				toRemoveIndex = i
				break
			}
		}
		//移除此玩家在房間內的資料
		room.players = append(room.players[:toRemoveIndex], room.players[toRemoveIndex+1:]...)
	} else {
		//沒在房間裡 就是在大廳，則更新大廳人數狀況
		delete(lobbyPlayer, connBrokenIp)
	}
}

func insetTestData() {

	player1, player2, ball := generateGameElement("testAddress:testPort")

	var tempList []*net.Conn
	for _, v := range lobbyPlayer {
		tempList = append(tempList, v)
	}

	players := []*Player{player1, player2}
	lobbyRoom = append(lobbyRoom, &Room{
		RoomId:     generateRoomId(),
		players:    players,
		Name:       fmt.Sprintf("New Room %d", len(lobbyRoom)),
		GameStatus: GameStatusWaiting,
		CreateDate: time.Now().Format("2006-02-01 15:01"),
		Ball:       ball,
	})
}

func generateRoomId() string {
	RoomInitialId += 1
	return string(rune(RoomInitialId))
}
