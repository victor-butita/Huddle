package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Client struct {
	hub               *Hub
	conn              *websocket.Conn
	send              chan []byte
	boardId, clientId string
}
type Message struct {
	BoardID string
	Sender  *Client
	Data    []byte
}

var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(r *http.Request) bool { return true }}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	boardId := vars["boardId"]
	board := hub.getOrCreateBoard(boardId)
	if board == nil {
		log.Printf("[WS-ERROR] Could not get or create board: %s", boardId)
		http.Error(w, "Could not initialize board", http.StatusInternalServerError)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS-ERROR] Failed to upgrade connection: %v", err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), boardId: boardId, clientId: uuid.New().String()}
	client.hub.register <- client
	go client.writePump()
	go client.readPump()
	board.mu.RLock()
	initialState, _ := json.Marshal(map[string]interface{}{"type": "INITIAL_STATE", "payload": board, "clientId": client.clientId})
	client.send <- initialState
	board.mu.RUnlock()
}
func (c *Client) readPump() {
	defer func() { c.hub.unregister <- c; c.conn.Close() }()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS-ERROR] Unexpected close error on board %s: %v", c.boardId, err)
			}
			break
		}
		var msgData map[string]interface{}
		json.Unmarshal(message, &msgData)
		c.hub.mu.RLock()
		board := c.hub.boards[c.boardId]
		c.hub.mu.RUnlock()
		if board == nil {
			continue
		}
		if msgType, ok := msgData["type"].(string); ok {
			board.mu.Lock()
			payloadBytes, _ := json.Marshal(msgData["payload"])
			switch msgType {
			case "CODE_UPDATE":
				if payload, ok := msgData["payload"].(string); ok {
					board.ContentCode = payload
				}
			case "TASKS_UPDATE":
				var tasks []Task
				json.Unmarshal(payloadBytes, &tasks)
				board.ContentTasks = tasks
			case "NOTES_UPDATE":
				if payload, ok := msgData["payload"].(string); ok {
					board.ContentNotes = payload
				}
			case "LINK_UPDATE":
				if payload, ok := msgData["payload"].(string); ok {
					board.HuddleLink = payload
				}
			case "TEAM_UPDATE":
				var team []Member
				json.Unmarshal(payloadBytes, &team)
				board.Team = team
			}
			board.mu.Unlock()
			go c.hub.persistBoardState(c.boardId)
		}
		broadcastMessage := &Message{BoardID: c.boardId, Sender: c, Data: message}
		c.hub.broadcast <- broadcastMessage
	}
}
func (c *Client) writePump() {
	defer func() { c.conn.Close() }()
	for {
		message, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		w, err := c.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		w.Write(message)
		n := len(c.send)
		for i := 0; i < n; i++ {
			w.Write(<-c.send)
		}
		if err := w.Close(); err != nil {
			return
		}
	}
}
