package core

import "fmt"

const PayloadTerminator = "~"

const ConnBrokenMsgHeader = "CB" // Connection Broken 連線斷線訊息
const RoomInfoMsgHeader = "RM"   // Room列表
const BattleSituation = "BS"     // Battle status 遊戲中的狀態

const CreateRoom = "CR" // Create Room 創建房間
const EnterRoom = "ER"  // Enter Room 進入房間
const LeaveRoom = "LR"  // Leave Room 離開房間
const ReadyStart = "RS" // Ready Start 準備開始

const LeaveLobby = "LL" // Leave Lobby 離開大廳

func generateRoomsInfoPayload() string {
	riList := getRoomList()
	var payload string
	for i := 0; i < len(riList); i += 1 {
		rn := riList[i].roomName
		cd := riList[i].createDate
		pc := riList[i].playerCount

		payload = fmt.Sprintf("%s+%s+%d&", rn, cd, pc)
	}

	return RoomInfoMsgHeader + payload + PayloadTerminator
}

func generateConnBrokenPayload(brokenIp string) string {
	return fmt.Sprintf("%s%s%s", ConnBrokenMsgHeader, brokenIp, PayloadTerminator)
}

func parseEnterRoomPayload(payload string) string {

	// ER_{RoomId}_player

	return ""
}

func removeHeaderTerminator(payload string) string {
	return payload[2 : len(payload)-1]
}
