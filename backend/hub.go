package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

type Task struct {
	ID, Text, Assignee string
	Completed          bool
}
type Member struct{ ID, Name, Color string }
type Board struct {
	ID           string           `json:"id"`
	Clients      map[*Client]bool `json:"-"`
	ContentCode  string           `json:"contentCode"`
	ContentTasks []Task           `json:"contentTasks"`
	ContentNotes string           `json:"contentNotes"`
	HuddleLink   string           `json:"huddleLink"`
	Team         []Member         `json:"team"`
	mu           sync.RWMutex
}
type Hub struct {
	boards     map[string]*Board
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mu         sync.RWMutex
}

func newHub() *Hub {
	hub := &Hub{boards: make(map[string]*Board), register: make(chan *Client), unregister: make(chan *Client), broadcast: make(chan *Message)}
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			<-ticker.C
			cleanupOldBoards()
		}
	}()
	return hub
}
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.RLock()
			board := h.boards[client.boardId]
			h.mu.RUnlock()
			if board != nil {
				board.mu.Lock()
				board.Clients[client] = true
				board.mu.Unlock()
			}
		case client := <-h.unregister:
			h.mu.RLock()
			board := h.boards[client.boardId]
			h.mu.RUnlock()
			if board != nil {
				board.mu.Lock()
				if _, ok := board.Clients[client]; ok {
					delete(board.Clients, client)
					close(client.send)
				}
				if len(board.Clients) == 0 {
					go h.scheduleBoardDeletion(board.ID)
				} else {
					var remainingTeam []Member
					for _, member := range board.Team {
						if member.ID != client.clientId {
							remainingTeam = append(remainingTeam, member)
						}
					}
					board.Team = remainingTeam
					go h.persistBoardState(board.ID)
					payloadBytes, _ := json.Marshal(board.Team)
					teamUpdateMsg, _ := json.Marshal(map[string]interface{}{"type": "TEAM_UPDATE", "payload": json.RawMessage(payloadBytes)})
					message := &Message{BoardID: board.ID, Sender: nil, Data: teamUpdateMsg}
					h.broadcast <- message
				}
				board.mu.Unlock()
			}
		case message := <-h.broadcast:
			h.mu.RLock()
			board := h.boards[message.BoardID]
			h.mu.RUnlock()
			if board != nil {
				board.mu.RLock()
				for client := range board.Clients {
					if client != message.Sender {
						select {
						case client.send <- message.Data:
						default:
							close(client.send)
							delete(board.Clients, client)
						}
					}
				}
				board.mu.RUnlock()
			}
		}
	}
}
func (h *Hub) getOrCreateBoard(boardId string) *Board {
	h.mu.Lock()
	defer h.mu.Unlock()
	if board, ok := h.boards[boardId]; ok {
		return board
	}
	dbBoard, _ := getBoardFromDB(boardId)
	var tasks []Task
	var team []Member
	if dbBoard != nil {
		json.Unmarshal([]byte(dbBoard.ContentTasks), &tasks)
		json.Unmarshal([]byte(dbBoard.TeamData), &team)
		board := &Board{
			ID: dbBoard.ID, Clients: make(map[*Client]bool), ContentCode: dbBoard.ContentCode,
			ContentTasks: tasks, ContentNotes: dbBoard.ContentNotes, HuddleLink: dbBoard.HuddleLink, Team: team,
		}
		h.boards[boardId] = board
		return board
	}
	board := &Board{
		ID: boardId, Clients: make(map[*Client]bool), ContentCode: "// Welcome to your Huddle!\n// Select a language and click an AI action for a code review.",
		ContentTasks: []Task{}, ContentNotes: "## Meeting Notes\n\n- Start typing here...", HuddleLink: "", Team: []Member{},
	}
	h.boards[boardId] = board
	go h.persistBoardState(boardId)
	return board
}
func (h *Hub) persistBoardState(boardId string) {
	h.mu.RLock()
	board, ok := h.boards[boardId]
	h.mu.RUnlock()
	if !ok {
		return
	}
	board.mu.RLock()
	tasksJSON, _ := json.Marshal(board.ContentTasks)
	teamJSON, _ := json.Marshal(board.Team)
	dbModel := &BoardModel{
		ID: board.ID, ContentCode: board.ContentCode, ContentTasks: string(tasksJSON),
		ContentNotes: board.ContentNotes, HuddleLink: board.HuddleLink, TeamData: string(teamJSON),
	}
	board.mu.RUnlock()
	if err := createOrUpdateBoard(dbModel); err != nil {
		log.Printf("[ERROR] Failed to persist board %s: %v", boardId, err)
	}
}
func (h *Hub) scheduleBoardDeletion(boardID string) {
	time.Sleep(10 * time.Minute)
	h.mu.Lock()
	defer h.mu.Unlock()
	if board, ok := h.boards[boardID]; ok {
		if len(board.Clients) == 0 {
			delete(h.boards, boardID)
			log.Printf("[HUB] Unloaded board %s from memory due to inactivity.", boardID)
		}
	}
}
