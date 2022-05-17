package core

const GameStatusWaiting = "Waiting"
const GameStatusPlaying = "Playing"

type Room struct {
	RoomId     string
	Name       string
	GameStatus string
	CreateDate string

	players []*Player

	Ball *Ball
}

func (r *Room) startGame() {

}
