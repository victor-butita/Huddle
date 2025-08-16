package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Task represents a single item in the to-do list.
type Task struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	boardId string
}

// Message is a wrapper for data sent to the Hub's broadcast channel.
type Message struct {
	BoardID string
	Sender  *Client
	Data    []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// serveWs handles websocket requests from the peer.
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
		log.Printf("[WS-ERROR] Failed to upgrade connection for board %s: %v", boardId, err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), boardId: boardId}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()

	// Send the full initial state of the board to the new client.
	board.mu.RLock()
	initialState, err := json.Marshal(map[string]interface{}{
		"type":    "INITIAL_STATE",
		"payload": board,
	})
	if err != nil {
		log.Printf("[WS-ERROR] Failed to marshal initial state for board %s: %v", boardId, err)
		board.mu.RUnlock()
		return
	}
	client.send <- initialState
	board.mu.RUnlock()
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

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
			continue // Board may have been cleaned up
		}

		// Update server state based on message type BEFORE broadcasting
		if msgType, ok := msgData["type"].(string); ok {
			board.mu.Lock()
			switch msgType {
			case "CODE_UPDATE":
				if payload, ok := msgData["payload"].(string); ok {
					board.ContentCode = payload
				}
			case "TASKS_UPDATE":
				// Re-marshal and unmarshal payload to convert it to the correct struct type
				tasksData, err := json.Marshal(msgData["payload"])
				if err == nil {
					var tasks []Task
					json.Unmarshal(tasksData, &tasks)
					board.ContentTasks = tasks
				}
			case "LINK_UPDATE":
				if payload, ok := msgData["payload"].(string); ok {
					board.HuddleLink = payload
				}
			}
			board.mu.Unlock()

			// Persist changes to DB in the background
			go c.hub.persistBoardState(c.boardId)
		}

		broadcastMessage := &Message{BoardID: c.boardId, Sender: c, Data: message}
		c.hub.broadcast <- broadcastMessage
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		message, ok := <-c.send
		if !ok {
			// The hub closed the channel.
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		w, err := c.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		w.Write(message)

		// Add queued chat messages to the current websocket message.
		n := len(c.send)
		for i := 0; i < n; i++ {
			w.Write(<-c.send)
		}

		if err := w.Close(); err != nil {
			return
		}
	}
}
