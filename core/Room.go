package core

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
