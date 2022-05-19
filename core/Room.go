package core

import (
	"Pong/logger"
	"fmt"
	"math"
	"os"
	"time"
)

const FinalScore = 9        // 遊戲結束分數
const BallSymbol = 0x25CF   // 球符號
const PaddleSymbol = 0x2588 // 球拍符號
const PaddleHeight = 6      // 球拍高度
const BallVelocityRow = 1
const BallVelocityCol = 1

const windowHeight = 60
const windowWidth = 150

const RoomStatusWaiting = 0
const RoomStatusPlaying = 1

type Room struct {
	RoomId     string
	Name       string
	RoomStatus int
	CreateDate string
	Creator    *Player

	players []*Player

	Ball *Ball
}

func (r *Room) startGame() {
	logger.Log.Info(fmt.Sprintf("Room id:%s 遊戲開始！", r.RoomId))

	player1 := r.players[0]
	player2 := r.players[1]

	player1.Scene = SceneBattle
	player2.Scene = SceneBattle

	//產生遊戲元素
	r.spawnGameElement()

	for {
		//檢查房間狀態(是否有人離線)
		state := r.updateState()
		if state == false {
			return
		}
		conn1SendStatus := sendGameState(player1.Conn, r)
		conn2SendStatus := sendGameState(player2.Conn, r)

		if conn1SendStatus == ConnBroken || conn2SendStatus == ConnBroken {
			return
		}
		time.Sleep(2000 * time.Millisecond)
	}
}

func (r *Room) spawnGameElement() {
	paddleStart := windowHeight/2 - PaddleHeight/2

	player1 := r.players[0]
	player2 := r.players[1]

	//初始化玩家狀態
	player1.Row = paddleStart
	player1.Col = 0
	player1.Width = 1
	player1.Height = PaddleHeight
	player1.Symbol = PaddleSymbol
	player1.VelRow = 0
	player1.VelCol = 0
	player1.CurrentScore = 0
	player1.RightOrLeft = "left"

	player2.Row = paddleStart
	player2.Col = windowWidth - 2
	player2.Width = 1
	player2.Height = PaddleHeight
	player2.Symbol = PaddleSymbol
	player2.VelRow = 0
	player2.VelCol = 0
	player2.CurrentScore = 0
	player2.RightOrLeft = "right"

	//產生球
	ball := &Ball{
		GameObject: GameObject{Row: windowHeight / 2, Col: windowWidth / 2, Width: 1, Height: 1, Symbol: BallSymbol,
			VelRow: BallVelocityRow, VelCol: BallVelocityCol},
	}

	r.Ball = ball
}

func (r *Room) updateState() bool {
	if len(r.players) < 2 {
		return false
	}
	player1 := r.players[0]
	player2 := r.players[1]
	ball := r.Ball

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
	if r.isCollidesWithWall() {
		ball.VelRow = -ball.VelRow
	}
	//檢查是否有碰到球拍
	if r.isTouchPaddle() {
		ball.VelCol = -ball.VelCol
	}

	if r.isBallOutSide() {
		r.calculateScore()
		r.resetNewRound()
	}

	over, _ := r.isGameOver()
	if over {
		os.Exit(0)
	}

	return true
}

func (r *Room) resetNewRound() {
	ball := r.Ball
	ball.Row = windowHeight / 2
	ball.Col = windowWidth / 2
}

func (r *Room) isGameOver() (bool, *Player) {
	player1 := r.players[0]
	player2 := r.players[1]

	if player1.CurrentScore == FinalScore {
		return true, player1
	}
	if player2.CurrentScore == FinalScore {
		return true, player2
	}
	return false, nil
}

func (r *Room) isBallOutSide() bool {
	ball := r.Ball
	return ball.Col < 0 || ball.Col > windowWidth
}

func (r *Room) calculateScore() {
	player1 := r.players[0]
	player2 := r.players[1]
	ball := r.Ball

	if ball.Col < 0 {
		player2.CurrentScore += 1
	}
	if ball.Col > windowWidth {
		player1.CurrentScore += 1
	}
}

func (r *Room) isTouchPaddle() bool {
	player1 := r.players[0]
	player2 := r.players[1]
	ball := r.Ball

	if ball.Col+ball.VelCol <= player1.Col &&
		(ball.Row > player1.Row && ball.Row <= player1.Row+PaddleHeight) {
		return true
	} else if ball.Col+ball.VelCol >= player2.Col &&
		(ball.Row > player2.Row && ball.Row <= player2.Row+PaddleHeight) {
		return true
	}
	return false
}

func (r *Room) isCollidesWithWall() bool {
	ball := r.Ball
	return ball.Row+ball.VelRow < 0 || ball.Row+ball.VelRow >= windowHeight
}

func (r *Room) updatePlayerReadyStatus(playerId string) {
	var toChangeIndex int
	for i, p := range r.players {
		if p.IdAkaIpAddress == playerId {
			toChangeIndex = i
			break
		}
	}
	player := r.players[toChangeIndex]
	//1變0 0變1
	player.RoomReadyStatus = int(math.Abs(float64(player.RoomReadyStatus - 1)))

	//Log
	if player.RoomReadyStatus == 1 {
		logger.Log.Info(fmt.Sprintf(logger.PlayerPressReadyMsg, playerId, r.RoomId))
	} else {
		logger.Log.Info(fmt.Sprintf(logger.PlayerCancelReadyMsg, playerId, r.RoomId))
	}
}

func (r *Room) updateRoomStatus(roomStatus int) {
	//修改房間狀態
	r.RoomStatus = roomStatus
}

func (r *Room) removeRoomPlayer(playerId string) {
	var index int
	for i, player := range r.players {
		if player.IdAkaIpAddress == playerId {
			index = i
			break
		}
	}
	r.players = append(r.players[:index], r.players[index+1:]...)
}

func isTouchBottomBorder(paddle *Player) bool {
	return (paddle.Row + paddle.Height) < windowHeight
}

func isTouchTopBorder(paddle *Player) bool {
	return paddle.Row > 0
}
