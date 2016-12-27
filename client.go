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
	dialer.EnableCompression = true
	url := "ws://ws.generals.io/socket.io/?EIO=3&transport=websocket"

	if server == "eu" {
		url = "ws://euws.generals.io/socket.io/?EIO=3&transport=websocket"
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
		for data := range c.sendChan {
			err := c.c.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				c.c.Close()
			}
		}
	}()
	for {
		_, message, err := c.c.ReadMessage()
		if err != nil {
			return err
		}
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

func (c *Client) JoinCustomGame(ID string) *Game {
	c.sendMessage("join_private", ID, c.username, c.userID)
	time.Sleep(50 * time.Millisecond)
	c.sendMessage("set_username", c.userID, c.username)
	g := &Game{c: c, ID: ID}
	g.registerEvents()
	return g
}

func (c *Client) JoinClassic() *Game {
	c.sendMessage("play", c.username, c.userID)
	time.Sleep(50 * time.Millisecond)
	c.sendMessage("set_username", c.userID, c.username)
	g := &Game{c: c}
	g.registerEvents()
	return g
}

func (c *Client) JoinTeam(team string) *Game {
	c.sendMessage("join_team", team, c.username, c.userID)
	time.Sleep(50 * time.Millisecond)
	c.sendMessage("set_username", c.userID, c.username)
	g := &Game{c: c}
	g.registerEvents()
	return g
}
