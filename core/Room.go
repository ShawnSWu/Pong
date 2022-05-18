package core

const RoomStatusWaiting = "Waiting"
const RoomStatusPlaying = "Playing"

type Room struct {
	RoomId     string
	Name       string
	RoomStatus string
	CreateDate string
	Creator    *Player

	players []*Player

	Ball *Ball
}

func (r *Room) startGame() {

}
