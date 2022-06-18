package core

import (
	"Pong/logger"
	"bufio"
	"fmt"
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

const MaxRoomCount = 100

var RoomInitialId = 0

var mutex sync.RWMutex

//大廳玩家的連線
var lobbyPlayer = make(map[string]*Player)

var lobbyRoom = make([]*Room, 0, MaxRoomCount)

// roomChanMsg Room跟main goroutine的溝通channel
var roomChanMsg = make(chan string)

func notifyLobbyPlayerUpdateRoomList() {
	roomInfoPayload := generateRoomsListPayload()
	notifyLobbyPlayer(roomInfoPayload)
}

func notifyRoomPlayerUpdateRoomDetail(room *Room) {
	detailPayload := generateRoomsDetailPayload(*room)
	notifyRoomPlayer(room, detailPayload)
}

func notifyRoomPlayerBattleOver(room *Room) {
	detailPayload := generateBattleOver(room.RoomId)
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
			//設玩家名字
			case PlayerNameSetting:
				playerId := conn.RemoteAddr().String()
				playerName := parseSetPlayerName(payload)
				player := lobbyPlayer[playerId]
				player.NickName = playerName
				break

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
					CreateDate: time.Now().Format("2006-01-02 15:04"),
				}

				mutex.Lock()
				// 將此創建房間的玩家加入到房間中
				r.players = append(r.players, creator)
				creator.SetScene(SceneRoom)

				lobbyRoom = append(lobbyRoom, r)
				mutex.Unlock()

				// 傳送房間資訊該房間創建者
				notifyRoomPlayerUpdateRoomDetail(r)

				logger.Log.Info(fmt.Sprintf("%s 創建新房間！", playerId))

				//通知所有『在大廳』的玩家
				notifyLobbyPlayerUpdateRoomList()
				logger.Log.Info(fmt.Sprintf("玩家 %s 創建房間", playerId))
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

				logger.Log.Info(fmt.Sprintf(logger.PlayerEnterRoomMsg, playerId, roomId))
				break
			//離開大廳
			case LeaveLobby:
				playerId := conn.RemoteAddr().String()

				//通知玩家已成功離開大廳
				player := lobbyPlayer[playerId]
				payload := generateLeaveLobbySuccessPayload()
				sendMsg(player, payload)

				mutex.Lock()
				//關閉連線
				disconnectPlayerConn(playerId)
				delete(lobbyPlayer, playerId)
				mutex.Unlock()

				logger.Log.Info(fmt.Sprintf("%s 離開大廳！", playerId))
				logger.Log.Info(fmt.Sprintf("當下人數：%d", len(lobbyPlayer)))

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

				//當房間沒人時 移除空房間
				if isRoomEmpty(room) {
					roomIndex := getRoomIndex(room)
					lobbyRoom = append(lobbyRoom[:roomIndex], lobbyRoom[roomIndex+1:]...)
				}
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
						roomChanMsg <- generateStartBattlePayload(roomId)
					}
				}
				//通知房間玩家準備開始戰鬥
				go notifyRoomPlayerUpdateRoomDetail(room)
				break
			}
			break

		case SceneBattle:
			//戰鬥中的操作(e.g.移動與終止遊戲)
			switch header {

			case BattleActionHeader:
				battleAction := parsePlayerBattleAction(payload)
				handleBattleOperation(battleAction, player)
				break

			case GiveUpBattleHeader:
				roomId, _ := parseInterruptBattle(payload)
				playerId := conn.RemoteAddr().String()
				roomChanMsg <- generateOpponentGiveUpBattle(roomId, playerId)
				break
			}
			time.Sleep(10 * time.Millisecond)
			break
		}

		//接收Client心跳封包
		handleHeartBeatPayload(header, conn)

		time.Sleep(10 * time.Millisecond)
	}
}

func handleHeartBeatPayload(header string, conn net.Conn) {
	//客戶端傳來的心跳封包 將count歸0
	if header == HeartBeatHeader {
		playerId := conn.RemoteAddr().String()
		player := lobbyPlayer[playerId]
		player.resetHeartbeat()

		fmt.Println(fmt.Sprintf("連線仍然存活！玩家 %s 心跳計數歸零", playerId))
	}
}

func isRoomEmpty(room *Room) bool {
	return len(room.players) == 0
}

func getRoomIndex(room *Room) int {
	var index int
	for i, r := range lobbyRoom {
		if r.RoomId == room.RoomId {
			index = i
			break
		}
	}
	return index
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
	env := os.Getenv("PONG_ENV")
	host, port := ReadProperties(env)

	tcpAddr, _ := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%s", host, port))
	listener, _ := net.ListenTCP("tcp", tcpAddr)

	go listenRoomChannel()

	//心跳封包機制
	go heartBeatJob()

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
			//通知所有玩家 有新玩家家加入
			go notifyOnlinePlayerCount()
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

func notifyOnlinePlayerCount() {
	for {
		onlinePlayerCount := len(lobbyPlayer)
		payload := generateOnlinePlayerCountPayload(onlinePlayerCount)
		// 在大廳才傳送
		notifyLobbyPlayer(payload)
		time.Sleep(2000 * time.Millisecond)
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
				msg = removeHeaderTerminator(msg)
				//斷線者的ip
				connBrokenIp := msg

				//斷線處理機制
				connBrokenHandle(connBrokenIp)

				//處理完後，通知所有玩家，更新大廳與房間資訊
				roomsInfo := generateRoomsListPayload()
				notifyLobbyPlayer(roomsInfo)
				logger.Log.Info(logger.NotifyLobbyConnBrokenMsg)
				break

			case StartBattleHeader:
				payload := removeHeaderTerminator(msg) //移除Header與終止符
				room := findRoomById(payload)

				//通知玩家準備開始
				notifyRoomPlayer(room, msg)

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

				// 直接設置Loser
				room.setLoser(playerId)
				logger.Log.Info(fmt.Sprintf("玩家 %s 已經發起投降！", playerId))
				break

			case BattleOverHeader:

				roomId := parseBattleOver(msg)
				room := findRoomById(roomId)
				room.resetRoomStatus()

				updateRoomPlayerScene(room, SceneRoom)

				//通知玩家遊戲結束
				notifyRoomPlayerUpdateRoomDetail(room)
				notifyRoomPlayerBattleOver(room)
			}

		}
	}
}

func heartBeatJob() {
	//when count < 5，count += 1
	//whe count >= 5，說明已超過15秒未收到該玩家心跳封包，則判定該玩家已經斷線；
	for {
		for _, player := range lobbyPlayer {
			if player.HearBeatCount < 5 {
				fmt.Println(fmt.Sprintf("玩家 %s 心跳跳一下, 當前倒數心跳:%d", player.IdAkaIpAddress, player.HearBeatCount))
				player.Heartbeat()
			}
			if player.HearBeatCount >= 5 {
				//斷線處理
				fmt.Println(fmt.Sprintf("玩家%s斷線了", player.IdAkaIpAddress))
				connBrokenHandle(player.IdAkaIpAddress)
				//移除斷線者房間
				room := findPlayerRoom(player.IdAkaIpAddress)
				if room != nil {
					removeRoom(lobbyRoom, room.RoomId)
					logger.Log.Info(fmt.Sprintf("移除斷線者房間 名稱：%s, id: %s", room.Name, room.RoomId))
				}
			}
		}
		time.Sleep(3000 * time.Millisecond)
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

func findPlayerRoom(playerId string) *Room {
	var targetRoom *Room
	for i, room := range lobbyRoom {
		for _, player := range room.players {
			if player.IdAkaIpAddress == playerId {
				targetRoom = lobbyRoom[i]
			}
		}
	}
	return targetRoom
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

func updateRoomPlayerScene(room *Room, scene string) {
	for i, _ := range room.players {
		room.players[i].SetScene(scene)
	}
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
	mutex.Lock()
	//關閉連線
	disconnectPlayerConn(connBrokenIp)
	delete(lobbyPlayer, connBrokenIp)
	mutex.Unlock()

	logger.Log.Error(fmt.Sprintf("玩家 %s 連線異常，中止連線", connBrokenIp))
}

func generateRoomId() string {
	RoomInitialId += 1
	return strconv.Itoa(RoomInitialId)
}

func disconnectPlayerConn(playerId string) {
	(*lobbyPlayer[playerId].Conn).Close()
}
