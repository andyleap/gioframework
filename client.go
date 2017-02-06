package gioframework

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	c      *websocket.Conn
	events map[string]func(json.RawMessage)

	userID   string
	username string

	sendChan chan []byte
}

func Connect(server string, userid string, username string) (*Client, error) {

	dialer := &websocket.Dialer{}
	dialer.EnableCompression = false
	url := "ws://ws.generals.io/socket.io/?EIO=3&transport=websocket"

	if server == "eu" {
		url = "ws://euws.generals.io/socket.io/?EIO=3&transport=websocket"
	}
	if server == "bot" {
		url = "ws://botws.generals.io/socket.io/?EIO=3&transport=websocket"
	}

	c, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		c:        c,
		events:   map[string]func(json.RawMessage){},
		userID:   userid,
		username: username,
		sendChan: make(chan []byte, 10),
	}, nil

}

func (c *Client) Run() error {
	go func() {
		for range time.Tick(5 * time.Second) {
			c.sendChan <- []byte("2")
		}
	}()
	go func() {
		time.Sleep(100 * time.Millisecond)
		for data := range c.sendChan {
			err := c.c.WriteMessage(websocket.TextMessage, data)
			//log.Println("Sending: ", string(data))
			if err != nil {
				c.c.Close()
			}
		}
	}()
	c.sendMessage("set_username", c.userID, c.username)
	for {
		_, message, err := c.c.ReadMessage()
		if err != nil {
			return err
		}
		//log.Println("Got: ", string(message))
		dec := json.NewDecoder(bytes.NewBuffer(message))
		var msgType int
		dec.Decode(&msgType)

		if msgType == 42 {
			var raw json.RawMessage
			dec.Decode(&raw)
			eventname := ""
			data := []interface{}{&eventname}
			json.Unmarshal(raw, &data)
			if f, ok := c.events[eventname]; ok {
				f(raw)
			}
		}
	}
}

func (c *Client) sendMessage(v ...interface{}) {
	buf, _ := json.Marshal(v)
	newbuf := []byte("42" + string(buf))
	c.sendChan <- newbuf
}

func (c *Client) Close() {
	c.c.Close()
}

func (c *Client) JoinCustomGame(ID string) *Game {
	c.sendMessage("join_private", ID, c.userID)
	g := &Game{c: c, ID: ID}
	g.registerEvents()
	return g
}

func (c *Client) Join1v1() *Game {
	c.sendMessage("join_1v1", c.userID)
	g := &Game{c: c, ID: "1v1"}
	g.registerEvents()
	return g
}

func (c *Client) JoinClassic() *Game {
	c.sendMessage("play", c.userID)
	g := &Game{c: c}
	g.registerEvents()
	return g
}

func (c *Client) JoinTeam(team string) *Game {
	c.sendMessage("join_team", team, c.userID)
	g := &Game{c: c, ID: "2v2"}
	g.registerEvents()
	return g
}

type Ranking struct {
	Name  string  `json:"name"`
	Stars float64 `json:"stars"`
}

type Replay struct {
	Type    string    `json:"type"`
	ID      string    `json:"id"`
	Started int64     `json:"started"`
	Turns   int       `json:"turns"`
	Ranking []Ranking `json:"ranking"`
}

func (c *Client) GetReplaysForUser(userID string) []Replay {
	c.sendMessage("replay_list", struct {
		UserID string `json:"user_id"`
	}{UserID: userID})
	reply := make(chan []Replay)
	c.events["replay_list"] = func(data json.RawMessage) {
		replays := []Replay{}
		decode := []interface{}{nil, &replays}
		json.Unmarshal(data, &decode)
		reply <- replays
	}
	return <-reply
}

func (c *Client) GetReplays() []Replay {
	c.sendMessage("replay_list", map[string]interface{}{})
	reply := make(chan []Replay)
	c.events["replay_list"] = func(data json.RawMessage) {
		replays := []Replay{}
		decode := []interface{}{nil, &replays}
		json.Unmarshal(data, &decode)
		reply <- replays
	}
	return <-reply
}
