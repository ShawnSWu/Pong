package core

import (
	"fmt"
	"strings"
)

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

const StartBattleHeader = "SB"     // Start battle 開始戰鬥
const BattleSituationHeader = "BS" // Battle status 戰鬥中的狀態
const BattleOperationHeader = "BO" // Battle operation 戰鬥中玩家的操作
const InterruptBattleHeader = "IB" // Interrupt Battle 中斷戰鬥

func generateRoomsListPayload() string {
	riList := getRoomList()
	var payload string
	for i := 0; i < len(riList); i += 1 {
		ri := riList[i].roomId
		rn := riList[i].roomName
		cd := riList[i].createDate
		pc := riList[i].playerCount
		rs := riList[i].RoomStatus

		payload += fmt.Sprintf("%s,%s,%s,%d,%d", ri, rn, cd, pc, rs)

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

func generateConnBrokenPayload(brokenIp string) string {
	return fmt.Sprintf("%s%s%s", ConnBrokenHeader, brokenIp, PayloadTerminator)
}

func generateStartBattlePayload(roomId string) string {
	return fmt.Sprintf("%s%s%s", StartBattleHeader, roomId, PayloadTerminator)
}

//ballX, ballY, player1X, player1Y,player1Score, player2X, player2Y,player2Score
func generateBattlePayload(ballX, ballY,
	player1X, player1Y, player1Score,
	player2X, player2Y, player2Score int) string {
	payload := fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d,%d", ballX, ballY,
		player1X, player1Y, player1Score, player2X, player2Y, player2Score)
	return BattleSituationHeader + payload + PayloadTerminator
}

func generateInterruptBattle(roomId string, interruptSponsor string) string {
	payload := fmt.Sprintf("%s,%s", roomId, interruptSponsor)
	return fmt.Sprintf("%s%s%s", InterruptBattleHeader, payload, PayloadTerminator)
}

func parsePlayerBattleOperation(payload string) string {
	battleOperation := payload
	return battleOperation
}

func parseCreateRoom(payload string) string {
	roomName := payload
	return roomName
}

func parseEnterRoom(payload string) string {
	roomId := payload
	return roomId
}

func parseLeaveRoom(payload string) string {
	roomId := payload
	return roomId
}

func parseReadyStart(payload string) string {
	roomId := payload
	return roomId
}

func parseInterruptBattle(payload string) (string, string) {
	split := strings.Split(payload, ",")
	roomId := split[0]
	playerId := split[1]
	return roomId, playerId
}

func removeHeaderTerminator(payload string) string {
	return payload[2 : len(payload)-1]
}
