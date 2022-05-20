package core

import (
	"Pong/logger"
	"bufio"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

const SceneLobby = "Lobby"
const SceneRoom = "Room"
const SceneBattle = "Battle"

const ConnWorking = 1
const ConnBroken = 0

const MaxRoomCount = 6

var RoomInitialId = 0

var mutex sync.RWMutex

//大廳玩家的連線
var lobbyPlayer = make(map[string]*Player)

//最多同時6間房間
var lobbyRoom = make([]*Room, 0, MaxRoomCount)

//Room跟main goroutine的溝通channel
var roomChanMsg = make(chan string)

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
		//大廳中的操作(e.g.進入房間,創建房間與離開大廳)
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
				player.SetScene(SceneRoom)
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

		//房間中的操作(e.g.準備開始與離開房間)
		case SceneRoom:
			switch header {
			//離開房間
			case LeaveRoomHeader:
				roomId := parseLeaveRoom(payload)
				room := findRoomById(roomId)
				playerId := conn.RemoteAddr().String()

				mutex.Lock()
				//移除Room中的此玩家
				removeRoomPlayer(room, playerId)
				//更改玩家場景狀態
				player.SetScene(SceneLobby)
				mutex.Unlock()

				//通知大廳玩家(更新房間List)
				notifyLobbyPlayerUpdateRoomList()

				//通知房間內剩下的玩家
				notifyRoomPlayerUpdateRoomDetail(room)
				logger.Log.Info(fmt.Sprintf("%s 離開房間 Room id:%s", playerId, roomId))
				break

			//準備開始&取消準備
			case ReadyStartHeader:
				//(更新房間資訊) 並更改玩家準備狀態(RoomReadyStatus)
				roomId := parseReadyStart(payload)

				room := findRoomById(roomId)
				playerId := conn.RemoteAddr().String()

				mutex.Lock()
				//更新準備狀態
				room.updatePlayerReadyStatus(playerId)
				mutex.Unlock()

				//檢查是否兩個都按開始了，若是就開始了
				if len(room.players) == 2 {
					player1 := room.players[0]
					player2 := room.players[1]

					if player1.RoomReadyStatus == 1 && player2.RoomReadyStatus == 1 {
						//TODO 即將開始遊戲，通知兩個玩家三秒後開始遊戲

						roomChanMsg <- generateStartBattlePayload(roomId)
					}
				}

				notifyRoomPlayerUpdateRoomDetail(room)
				break
			}
			break

		case SceneBattle:
			//戰鬥中的操作(e.g.移動與終止遊戲)
			switch header {

			case BattleOperationHeader:
				battleOperation := parsePlayerBattleOperation(payload)
				handleBattleOperation(battleOperation, player)
				break

			case GiveUpBattleHeader:
				roomId, _ := parseInterruptBattle(payload)
				playerId := conn.RemoteAddr().String()
				roomChanMsg <- generateGiveUpBattle(roomId, playerId)
				break
			}
			time.Sleep(10 * time.Millisecond)
			break

		}

		time.Sleep(10 * time.Millisecond)
	}
}

func sendMsg(player *Player, payload string) int {
	conn := *player.Conn
	playerId := conn.RemoteAddr().String()
	logger.Log.Info(fmt.Sprintf(logger.SendMsgContentMsg, conn.RemoteAddr().String(), payload))
	_, err := conn.Write([]byte(payload))

	//斷線了 傳訊息給main goroutine
	if err != nil {
		//CB mean connection broken
		roomChanMsg <- generateConnBrokenPayload(playerId)
		logger.Log.Error(logger.ConnBrokenMsg + " => " + fmt.Sprintf("%s_%s", ConnBrokenHeader, logger.ConnBrokenMsg))
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
		playerId := conn.RemoteAddr().String()
		logger.Log.Info(fmt.Sprintf("Player已連線 (ip:%s)", playerId))

		if len(lobbyRoom) >= MaxRoomCount {
			//sent message to client say full room
			//close connection
			conn.Close()
			logger.Log.Info(fmt.Sprintf("連線已滿，關閉ip:%s 的連線", playerId))
			continue
		}

		//產生玩家
		player := generatePlayer(playerId, &conn)

		if lobbyPlayer[playerId] == nil {
			lobbyPlayer[playerId] = player
			logger.Log.Info(fmt.Sprintf("Player ip:%s 進入大廳", playerId))
		}

		//開始監聽玩家操作事件
		go listenPlayerOperation(&conn, player)

		roomInfoPayload := generateRoomsListPayload()
		//傳送遊戲大廳給此新玩家
		sendMsg(player, roomInfoPayload)

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

			case StartBattleHeader:
				msg = removeHeaderTerminator(msg) //移除Header與終止符
				room := findRoomById(msg)
				//倒數三秒
				time.Sleep(3000 * time.Millisecond)
				room.updateRoomStatus(RoomStatusPlaying)
				notifyLobbyPlayerUpdateRoomList()

				go room.startGame()
				break

			case GiveUpBattleHeader:
				payload := removeHeaderTerminator(msg) //移除Header與終止符
				roomId, playerId := parseInterruptBattle(payload)

				room := findRoomById(roomId)

				//移除房間內此玩家
				room.removeRoomPlayer(playerId)

				//改變房間狀態 Waiting
				room.updateRoomStatus(RoomStatusWaiting)

				//剩下來的玩家
				anotherPlayer := room.players[0]

				//修改另一個玩家Scene
				anotherPlayer.SetScene(SceneRoom)

				//通知另個玩家並讓此玩家回到房間內等待
				payload = generateGiveUpBattle(room.RoomId, anotherPlayer.IdAkaIpAddress)

				//通知房間內剩餘玩家有人已離線
				notifyRoomPlayer(room, payload)

				time.Sleep(30 * time.Millisecond)

				//讓玩家回到房間時，獲得當前房間狀態
				notifyRoomPlayerUpdateRoomDetail(room)

				logger.Log.Info(fmt.Sprintf(logger.CompetitorConnBrokenMsg, playerId))
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

type RoomInfo struct {
	roomId      string
	roomName    string
	createDate  string
	playerCount int
	RoomStatus  int
}

func getRoomList() []RoomInfo {
	var roomInfoSlice = make([]RoomInfo, 0, 10)

	for i := 0; i < len(lobbyRoom); i++ {
		roomId := lobbyRoom[i].RoomId
		roomName := lobbyRoom[i].Name
		createDate := lobbyRoom[i].CreateDate
		playerCount := len(lobbyRoom[i].players)
		roomStatus := lobbyRoom[i].RoomStatus

		roomInfoSlice = append(roomInfoSlice, RoomInfo{roomId, roomName,
			createDate, playerCount, roomStatus})
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

func generateRoomId() string {
	RoomInitialId += 1
	return strconv.Itoa(RoomInitialId)
}

func disconnectPlayerConn(playerId string) {
	(*lobbyPlayer[playerId].Conn).Close()
}
