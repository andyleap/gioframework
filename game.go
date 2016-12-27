package gioframework

import (
	"encoding/json"
)

type Game struct {
	c  *Client
	ID string

	chatroom string
	replayID string

	QueueUpdate func(playercount, forcestartcount int)
	PreStart    func()
	Start       func(playerindex int, users []string)
	Update      func(update GameUpdate)
	Won         func()
	Lost        func()
	Chat        func(user int, message string)

	lastAttack  int
	attackIndex int

	PlayerIndex int
	Width       int
	Height      int
	GameMap     []Cell
	inited      bool
	TurnCount   int

	mapRaw    []int
	citiesRaw []int

	Scores []struct {
		Armies int  `json:"total"`
		Tiles  int  `json:"tiles"`
		Index  int  `json:"i"`
		Dead   bool `json:"dead"`
	}
}

type GameUpdate struct {
	AttackIndex int   `json:"attackIndex"`
	CitiesDiff  []int `json:"cities_diff"`
	Generals    []int `json:"generals"`
	MapDiff     []int `json:"map_diff"`
	Scores      []struct {
		Armies int  `json:"total"`
		Tiles  int  `json:"tiles"`
		Index  int  `json:"i"`
		Dead   bool `json:"dead"`
	}
	Stars *[]float64 `json:"stars"`
	Turn  int        `json:"turn"`
}

func (g *Game) registerEvents() {
	g.c.events["queue_update"] = func(data json.RawMessage) {
		playercount := 0
		forcestartcount := 0
		decode := []interface{}{nil, &playercount, &forcestartcount}
		json.Unmarshal(data, &decode)
		if g.QueueUpdate != nil {
			g.QueueUpdate(playercount, forcestartcount)
		}
	}
	g.c.events["pre_game_start"] = func(data json.RawMessage) {
		if g.PreStart != nil {
			g.PreStart()
		}
	}
	g.c.events["game_start"] = func(data json.RawMessage) {
		gameinfo := struct {
			PlayerIndex int      `json:"playerIndex"`
			ReplayID    string   `json:"replay_id"`
			ChatRoom    string   `json:"chat_room"`
			Usernames   []string `json:"usernames"`
		}{}
		decode := []interface{}{nil, &gameinfo}
		json.Unmarshal(data, &decode)
		g.PlayerIndex = gameinfo.PlayerIndex
		g.chatroom = gameinfo.ChatRoom
		g.replayID = gameinfo.ReplayID
		if g.Start != nil {
			g.Start(gameinfo.PlayerIndex, gameinfo.Usernames)
		}
	}
	g.c.events["game_update"] = func(data json.RawMessage) {
		update := GameUpdate{}
		decode := []interface{}{nil, &update}
		json.Unmarshal(data, &decode)

		newRaw := []int{}
		difPos := 0
		oldPos := 0
		for difPos < len(update.MapDiff) {
			getOld := update.MapDiff[difPos]
			difPos++
			for l1 := 0; l1 < getOld; l1++ {
				newRaw = append(newRaw, g.mapRaw[oldPos])
				oldPos++
			}
			if difPos >= len(update.MapDiff) {
				break
			}
			getNew := update.MapDiff[difPos]
			difPos++
			for l1 := 0; l1 < getNew; l1++ {
				newRaw = append(newRaw, update.MapDiff[difPos])
				oldPos++
				difPos++
			}
		}

		g.mapRaw = newRaw

		newRaw = []int{}
		difPos = 0
		oldPos = 0
		for difPos < len(update.CitiesDiff) {
			getOld := update.CitiesDiff[difPos]
			difPos++
			for l1 := 0; l1 < getOld; l1++ {
				newRaw = append(newRaw, g.citiesRaw[oldPos])
				oldPos++
			}
			if difPos >= len(update.CitiesDiff) {
				break
			}
			getNew := update.CitiesDiff[difPos]
			difPos++
			for l1 := 0; l1 < getNew; l1++ {
				newRaw = append(newRaw, update.CitiesDiff[difPos])
				oldPos++
				difPos++
			}
		}
		g.citiesRaw = newRaw

		if !g.inited {
			g.Width = g.mapRaw[0]
			g.Height = g.mapRaw[1]
			g.GameMap = make([]Cell, g.Width*g.Height)
			g.inited = true
		}

		g.TurnCount = update.Turn
		g.attackIndex = update.AttackIndex

		g.Scores = update.Scores

		for x := 0; x < g.Width; x++ {
			for y := 0; y < g.Height; y++ {
				g.GameMap[y*g.Width+x].Armies = g.mapRaw[y*g.Width+x+2]
			}
		}
		for x := 0; x < g.Width; x++ {
			for y := 0; y < g.Height; y++ {
				g.GameMap[y*g.Width+x].Faction = g.mapRaw[y*g.Width+x+2+g.Width*g.Height]
			}
		}
		for _, city := range g.citiesRaw {
			if city >= 0 {
				g.GameMap[city].Type = City
			}
		}
		for _, general := range update.Generals {
			if general >= 0 {
				g.GameMap[general].Type = General
			}
		}

		if g.Update != nil {
			g.Update(update)
		}
	}
	g.c.events["game_won"] = func(data json.RawMessage) {
		if g.Won != nil {
			g.Won()
		}
	}
	g.c.events["game_lost"] = func(data json.RawMessage) {
		if g.Lost != nil {
			g.Lost()
		}
	}
	g.c.events["chat_message"] = func(data json.RawMessage) {
		message := struct {
			Text        string `json:"text"`
			PlayerIndex *int   `json:"playerIndex"`
		}{}
		decode := []interface{}{nil, nil, &message}
		json.Unmarshal(data, &decode)
		if g.Chat != nil {
			index := -1
			if message.PlayerIndex != nil {
				index = *message.PlayerIndex
			}
			g.Chat(index, message.Text)
		}
	}
}

func (g *Game) GetAdjacents(from int) (adjacent []int) {
	if from >= g.Width {
		adjacent = append(adjacent, from-g.Width)
	}
	if from < g.Width*(g.Height-1) {
		adjacent = append(adjacent, from+g.Width)
	}
	if from%g.Width > 0 {
		adjacent = append(adjacent, from-1)
	}
	if from%g.Width < g.Width-1 {
		adjacent = append(adjacent, from+1)
	}
	return
}

func (g *Game) GetNeighborhood(from int) (adjacent []int) {
	if from >= g.Width {
		if from%g.Width > 0 {
			adjacent = append(adjacent, (from-g.Width)-1)
		}
		adjacent = append(adjacent, from-g.Width)
		if from%g.Width < g.Width-1 {
			adjacent = append(adjacent, (from-g.Width)+1)
		}
	}
	if from < g.Width*(g.Height-1) {
		if from%g.Width > 0 {
			adjacent = append(adjacent, (from+g.Width)-1)
		}
		adjacent = append(adjacent, from+g.Width)
		if from%g.Width < g.Width-1 {
			adjacent = append(adjacent, (from+g.Width)+1)
		}
	}
	if from%g.Width > 0 {
		adjacent = append(adjacent, from-1)
	}
	if from%g.Width < g.Width-1 {
		adjacent = append(adjacent, from+1)
	}
	return
}

func (g *Game) GetDistance(from, to int) int {

	x1, y1 := from%g.Width, from/g.Width
	x2, y2 := to%g.Width, to/g.Width
	dx := x1 - x2
	dy := y1 - y2
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

func (g *Game) SendChat(msg string) {
	g.c.sendMessage("chat_message", g.chatroom, msg)
}

func (g *Game) QueueLength() int {
	return g.lastAttack - g.attackIndex
}

func (g *Game) Walkable(cell int) bool {
	return g.GameMap[cell].Faction != -2 && g.GameMap[cell].Faction != -4
}

func (g *Game) SetForceStart(start bool) {
	g.c.sendMessage("set_force_start", g.ID, start)
}

func (g *Game) Attack(from, to int, is50 bool) {
	g.lastAttack++
	g.c.sendMessage("attack", from, to, is50, g.lastAttack)
}
