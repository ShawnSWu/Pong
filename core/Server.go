package core

import (
	"Pong/logger"
	"bufio"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"sync"
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

var mutex sync.RWMutex

//大廳玩家的連線
var lobbyPlayer = make(map[string]*Player)

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

func notifyLobbyPlayerUpdateRoomList() {
	roomInfoPayload := generateRoomsListPayload()
	notifyLobbyPlayer(roomInfoPayload)
}

func notifyRoomPlayerUpdateRoomDetail(room *Room) {
	detailPayload := generateRoomsDetailPayload(*room)
	notifyRoomPlayer(room, detailPayload)
}

func listenPlayerOperation(connP *net.Conn, player *Player) {
	conn := *connP

	for {
		payload, _ := bufio.NewReader(conn).ReadString('~')

		if payload == "" {
			continue
		}

		header := string([]byte(payload)[0:2])
		payload = removeHeaderTerminator(payload)

		switch player.Scene {
		//在大廳 要能進入房間,創建房間與離開大廳
		case SceneLobby:

			switch header {
			//創建房間
			case CreateRoomHeader:
				playerId := conn.RemoteAddr().String()

				creator := lobbyPlayer[playerId]
				roomName := parseCreateRoom(payload)

				r := &Room{
					RoomId:     generateRoomId(),
					Name:       roomName,
					RoomStatus: RoomStatusWaiting,
					Creator:    creator,
					CreateDate: time.Now().Format("2006-02-01 15:01"),
				}

				mutex.Lock()
				lobbyRoom = append(lobbyRoom, r)
				mutex.Unlock()

				logger.Log.Info(fmt.Sprintf("%s 創建新房間！", playerId))

				//通知所有『在大廳』的玩家
				notifyLobbyPlayerUpdateRoomList()
				break
			//進入房間
			case EnterRoomHeader:
				//把玩家加到房間資訊中(更新房間資訊) 並更改玩家場景
				playerId := conn.RemoteAddr().String()
				roomId := parseEnterRoom(payload)

				player := lobbyPlayer[playerId]
				room := findRoomById(roomId)

				if len(room.players) >= 2 {
					//人數已滿 通知！
					payload := generateRoomsFullPayload(roomId)
					sendMsg(player, payload)
					break
				}

				//修改Player Scene
				mutex.Lock()
				player.Scene = SceneRoom
				room.players = append(room.players, player)
				mutex.Unlock()

				// 通知所有在『大廳』的玩家(更新房間人數)
				notifyLobbyPlayerUpdateRoomList()

				// 通知在『房間中』的玩家 Room (如果房間本身沒人 則不會通知)
				notifyRoomPlayerUpdateRoomDetail(room)

				logger.Log.Info(fmt.Sprintf("%s 進入房間 RoomId: %s", playerId, roomId))
				break
			//離開大廳
			case LeaveLobby:
				playerId := conn.RemoteAddr().String()

				mutex.Lock()
				//關閉連線
				disconnectPlayerConn(playerId)
				delete(lobbyPlayer, playerId)
				mutex.Unlock()

				logger.Log.Info(fmt.Sprintf("%s 離開大廳！", playerId))

				notifyLobbyPlayerUpdateRoomList()
				break
			}
			break

		//在房間 要能按下準備開始與離開房間
		case SceneRoom:
			switch header {
			//離開房間
			case LeaveRoomHeader:
				roomId := parseLeaveRoom(payload)
				room := findRoomById(roomId)
				ip := conn.RemoteAddr().String()

				mutex.Lock()
				//移除Room中的此玩家
				removeRoomPlayer(room, ip)
				//更改玩家場景狀態
				player.Scene = SceneLobby
				mutex.Unlock()

				//通知大廳玩家(更新房間List)
				notifyLobbyPlayerUpdateRoomList()

				//通知房間內剩下的玩家
				notifyRoomPlayerUpdateRoomDetail(room)
				logger.Log.Info(fmt.Sprintf("%s 離開房間 Room id:%s", ip, roomId))
				break
			//準備開始&取消準備
			case ReadyStartHeader:
				//(更新房間資訊) 並更改玩家準備狀態(RoomReadyStatus)
				roomId := parseReadyStart(payload)

				room := findRoomById(roomId)
				ip := conn.RemoteAddr().String()

				mutex.Lock()
				//更新準備狀態
				updatePlayerReadyStatus(room, ip)
				mutex.Unlock()

				notifyRoomPlayerUpdateRoomDetail(room)
				logger.Log.Info(fmt.Sprintf("玩家 %s 按下準備按鍵 Room id:%s", ip, roomId))
				break
			}
			break

		case SceneBattle:
			//在戰鬥中 要能移動球拍

			switch header {

			case BattleSituationHeader:
				handleBattleOperation(payload, player)
				break
			}
			time.Sleep(10 * time.Millisecond)
			break

		}

		time.Sleep(10 * time.Millisecond)
	}
}

func updatePlayerReadyStatus(room *Room, ip string) {
	var toChangeIndex int
	for i, p := range room.players {
		if p.IdAkaIpAddress == ip {
			toChangeIndex = i
			break
		}
	}
	player := room.players[toChangeIndex]
	//1變0 0變1
	player.RoomReadyStatus = int(math.Abs(float64(player.RoomReadyStatus - 1)))
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

func sendMsg(player *Player, payload string) int {
	conn := *player.Conn
	ip := conn.RemoteAddr().String()
	logger.Log.Info(fmt.Sprintf(sendMsgContent, conn.RemoteAddr().String(), payload))
	_, err := conn.Write([]byte(payload))

	//斷線了 傳訊息給main goroutine
	if err != nil {
		//CB mean connection broken
		roomChanMsg <- generateConnBrokenPayload(ip)
		logger.Log.Error(ConnBrokenMsg + " => " + fmt.Sprintf("%s_%s", ConnBrokenHeader, ConnBrokenMsg))
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

	payload := generateBattlePayload(ballX, ballY,
		player1X, player1Y, player1Score, player2X, player2Y, player2Score)

	_, err := conn.Write([]byte(payload))

	//斷線了 通知main goroutine
	if err != nil {
		//CB mean connection broken
		roomChanMsg <- generateConnBrokenPayload(conn.RemoteAddr().String())
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

		//產生玩家
		player := generatePlayer(ip, &conn)

		if lobbyPlayer[ip] == nil {
			lobbyPlayer[ip] = player
			logger.Log.Info(fmt.Sprintf("Player ip:%s 進入大廳", ip))
		}

		go listenPlayerOperation(&conn, player)

		//產生遊戲大廳的Room列表
		roomInfoPayload := generateRoomsListPayload()

		//傳送遊戲大廳列表給此新玩家
		sendMsg(player, roomInfoPayload)

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
		//		RoomStatus:  RoomStatusWaiting,
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

//通知所有『在大廳』的玩家
func notifyLobbyPlayer(roomInfoPayload string) {
	for _, player := range lobbyPlayer {
		if player.Scene == SceneLobby {
			sendMsg(player, roomInfoPayload)
		}
	}
}

func notifyRoomPlayer(room *Room, payload string) {
	for _, player := range room.players {
		if player.Scene == SceneRoom {
			sendMsg(player, payload)
		}
	}

}

func listenRoomChannel() {
	for {
		select {
		case msg := <-roomChanMsg:

			header := string([]byte(msg)[0:2])

			switch header {

			case ConnBrokenHeader:
				msg = removeHeaderTerminator(msg) //移除Header與終止符
				//斷線者的ip
				connBrokenIp := msg
				logger.Log.Info("有人斷線！")

				mutex.Lock()
				//斷線處理機制
				connBrokenHandle(connBrokenIp)
				mutex.Lock()

				//處理完後，通知所有玩家，更新大廳與房間資訊
				roomsInfo := generateRoomsListPayload()
				notifyLobbyPlayer(roomsInfo)
				logger.Log.Info("通知大廳玩家 更新房間資訊！")
				break
			}

		}
	}
}

func isPlayerPlaying(playerIp string) (bool, *Room) {
	room := getPlayerRoom(playerIp)
	isPlaying := false

	if room != nil && room.RoomStatus == RoomStatusPlaying {
		isPlaying = true
	}
	return isPlaying, room
}

func getPlayerRoom(playerIp string) *Room {
	var roomIndex int
	found := false
	for i, r := range lobbyRoom {
		for _, player := range r.players {
			if playerIp == player.IdAkaIpAddress {
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

func removeRoomPlayer(room *Room, playerId string) {
	var index int
	for i, player := range room.players {
		if player.IdAkaIpAddress == playerId {
			index = i
			break
		}
	}
	room.players = append(room.players[:index], room.players[index+1:]...)
}

func generatePlayer(ip string, conn *net.Conn) *Player {
	return &Player{
		NickName:       "Player",
		IdAkaIpAddress: ip,
		Conn:           conn,
		Scene:          SceneLobby,
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
		NickName:       "Player",
		IdAkaIpAddress: ip,
		CurrentScore:   0,
		RightOrLeft:    "left",
	}

	player2 := &Player{
		GameObject: GameObject{Row: paddleStart, Col: windowWidth - 2, Width: 1,
			Height: PaddleHeight, Symbol: PaddleSymbol,
			VelRow: 0, VelCol: 0},
		NickName:       "Player two",
		IdAkaIpAddress: ip,
		CurrentScore:   0,
		RightOrLeft:    "right",
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
		if player1.IdAkaIpAddress == connBrokenIp {
			msg := generateConnBrokenPayload(player1.IdAkaIpAddress)
			sendMsg(player2, msg)
			//刪除房間
			delete(lobbyPlayer, player2.IdAkaIpAddress)
		} else {
			msg := generateConnBrokenPayload(player2.IdAkaIpAddress)
			sendMsg(player1, msg)
			//刪除房間
			delete(lobbyPlayer, player1.IdAkaIpAddress)
		}
	}

	//如果沒有正在遊戲中，但在某間房間裡面，則更新那個房間的人數資訊(連線)
	if room != nil {
		var toRemoveIndex = -1
		for i, player := range room.players {
			if connBrokenIp == player.IdAkaIpAddress {
				toRemoveIndex = i
				break
			}
		}
		if toRemoveIndex != -1 {
			//移除此玩家在房間內的資料
			room.players = append(room.players[:toRemoveIndex], room.players[toRemoveIndex+1:]...)
		}
	} else {
		//沒在房間裡 就是在大廳，則更新大廳人數狀況
		delete(lobbyPlayer, connBrokenIp)
	}

	disconnectPlayerConn(connBrokenIp)
}

func insetTestData() {

	player1, player2, ball := generateGameElement("testAddress:testPort")

	var tempList []*net.Conn
	for _, v := range lobbyPlayer {
		tempList = append(tempList, v.Conn)
	}

	players := []*Player{player1, player2}
	lobbyRoom = append(lobbyRoom, &Room{
		RoomId:     generateRoomId(),
		players:    players,
		Name:       fmt.Sprintf("New Room %d", len(lobbyRoom)),
		RoomStatus: RoomStatusWaiting,
		CreateDate: time.Now().Format("2006-02-01 15:01"),
		Ball:       ball,
	})
}

func generateRoomId() string {
	RoomInitialId += 1
	return strconv.Itoa(RoomInitialId)
}

func disconnectPlayerConn(playerId string) {
	(*lobbyPlayer[playerId].Conn).Close()
}
