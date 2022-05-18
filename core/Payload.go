package core

import "fmt"

const PayloadTerminator = "~"

const ConnBrokenHeader = "CB" // Connection Broken 連線斷線訊息

const RoomInfoHeader = "RL" // Room列表
const LeaveLobby = "LL"     // Leave Lobby 離開大廳

const CreateRoomHeader = "CR" // Create Room 創建房間
const RoomDetailHeader = "RD" // Room Detail 房間詳細內容
const RoomFullHeader = "RF"   // Room 房間人數已滿
const EnterRoomHeader = "ER"  // Enter Room 進入房間
const LeaveRoomHeader = "LR"  // Leave Room 離開房間
const ReadyStartHeader = "RS" // Ready Start 準備開始

const BattleSituationHeader = "BS" // Battle status 遊戲中的狀態

func generateRoomsListPayload() string {
	riList := getRoomList()
	var payload string
	for i := 0; i < len(riList); i += 1 {
		ri := riList[i].roomId
		rn := riList[i].roomName
		cd := riList[i].createDate
		pc := riList[i].playerCount

		payload += fmt.Sprintf("%s,%s,%s,%d", ri, rn, cd, pc)

		if i != len(riList)-1 {
			payload += "&"
		}
	}

	return RoomInfoHeader + payload + PayloadTerminator
}

func generateRoomsDetailPayload(room Room) string {
	roomId := room.RoomId
	roomName := room.Name

	var payload string

	if len(room.players) == 0 {
		payload = fmt.Sprintf("%s,%s,,,,,,", roomId, roomName)
	} else if len(room.players) == 1 {
		player1 := room.players[0]
		payload = fmt.Sprintf("%s,%s,%s,%s,%d,,,", roomId, roomName, player1.IdAkaIpAddress, player1.NickName, player1.RoomReadyStatus)
	} else {
		player1 := room.players[0]
		player2 := room.players[1]

		payload = fmt.Sprintf("%s,%s,%s,%s,%d,%s,%s,%d", roomId, roomName,
			player1.IdAkaIpAddress,
			player1.NickName,
			player1.RoomReadyStatus,
			player2.IdAkaIpAddress,
			player2.NickName,
			player2.RoomReadyStatus)
	}

	return RoomDetailHeader + payload + PayloadTerminator
}

func generateRoomsFullPayload(roomId string) string {
	return RoomFullHeader + roomId + PayloadTerminator
}

func parseCreateRoom(payload string) string {
	return payload
}

func parseEnterRoom(payload string) string {
	return payload
}

func parseLeaveRoom(payload string) string {
	roomId := payload
	return roomId
}

func parseReadyStart(payload string) string {
	roomId := payload
	return roomId
}

func generateConnBrokenPayload(brokenIp string) string {
	return fmt.Sprintf("%s%s%s", ConnBrokenHeader, brokenIp, PayloadTerminator)
}

func parseEnterRoomPayload(payload string) string {

	// ER_{RoomId}_player

	return ""
}

func removeHeaderTerminator(payload string) string {
	return payload[2 : len(payload)-1]
}

//ballX, ballY, player1X, player1Y,player1Score, player2X, player2Y,player2Score
func generateBattlePayload(ballX, ballY,
	player1X, player1Y, player1Score,
	player2X, player2Y, player2Score int) string {
	payload := fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d", ballX, ballY,
		player1X, player1Y, player1Score, player2X, player2Y, player2Score)
	return BattleSituationHeader + payload + PayloadTerminator
}
