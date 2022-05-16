package core

import "fmt"

const ConnBrokenMsgHeader = "CB"
const RoomInfoMsgHeader = "RM"
const BattleSituation = "BS"

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
