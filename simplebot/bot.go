package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/andyleap/gioframework"
)

func main() {

	c, _ := gioframework.Connect("us", "botbotbot", "SimpleBot")
	go c.Run()

	for {
		game := c.JoinCustomGame("botbotbot")
		game.SetForceStart(true)
		started := false
		game.Start = func(playerIndex int, users []string) {
			log.Println("Game started with ", users)
			started = true
		}
		done := false
		game.Won = func() {
			log.Println("Won game!")
			done = true
		}
		game.Lost = func() {
			log.Println("Lost game...")
			done = true
		}
		for !started {
			time.Sleep(1 * time.Second)
		}

		time.Sleep(1 * time.Second)

		for !done {
			time.Sleep(100 * time.Millisecond)
			if game.QueueLength() > 0 {
				continue
			}
			mine := []int{}
			for i, tile := range game.GameMap {
				if tile.Faction == game.PlayerIndex && tile.Armies > 1 {
					mine = append(mine, i)
				}
			}
			if len(mine) == 0 {
				continue
			}
			cell := rand.Intn(len(mine))
			move := []int{}
			for _, adjacent := range game.GetAdjacents(mine[cell]) {
				if game.Walkable(adjacent) {
					move = append(move, adjacent)
				}
			}
			if len(move) == 0 {
				continue
			}
			movecell := rand.Intn(len(move))
			game.Attack(mine[cell], move[movecell], false)

		}
	}
}
