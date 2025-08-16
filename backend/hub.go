package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Board represents the live state of a board in memory.
type Board struct {
	ID           string           `json:"id"`
	Clients      map[*Client]bool `json:"-"`
	ContentCode  string           `json:"contentCode"`
	ContentTasks []Task           `json:"contentTasks"`
	HuddleLink   string           `json:"huddleLink"`
	mu           sync.RWMutex
}

// Hub maintains the set of active boards.
type Hub struct {
	boards     map[string]*Board
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mu         sync.RWMutex
}

func newHub() *Hub {
	hub := &Hub{
		boards:     make(map[string]*Board),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
	}
	// Start a background task to clean up old boards every hour
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

func (h *Hub) run() { /* ...Identical to previous correct version... */ }

// getOrCreateBoard is the primary entry point for a client.
func (h *Hub) getOrCreateBoard(boardId string) *Board {
	h.mu.Lock()
	defer h.mu.Unlock()

	// If board is already in memory, return it.
	if board, ok := h.boards[boardId]; ok {
		log.Printf("[HUB] Client joined existing in-memory board %s", boardId)
		return board
	}

	// Board is not in memory, try loading from DB.
	dbBoard, err := getBoardFromDB(boardId)
	if err != nil {
		log.Printf("[ERROR] Failed to get board from DB: %v", err)
		return nil
	}

	var tasks []Task
	// If the board exists in the DB, load its state.
	if dbBoard != nil {
		json.Unmarshal([]byte(dbBoard.ContentTasks), &tasks)
		board := &Board{
			ID:           dbBoard.ID,
			Clients:      make(map[*Client]bool),
			ContentCode:  dbBoard.ContentCode,
			ContentTasks: tasks,
			HuddleLink:   dbBoard.HuddleLink,
		}
		h.boards[boardId] = board
		log.Printf("[HUB] Loaded board %s from database into memory.", boardId)
		return board
	}

	// Board does not exist in DB either, create a new one.
	log.Printf("[HUB] No board found for %s. Creating a new one.", boardId)
	board := &Board{
		ID:           boardId,
		Clients:      make(map[*Client]bool),
		ContentCode:  "// Welcome to your Huddle!\n// Start coding here.",
		ContentTasks: []Task{},
		HuddleLink:   "",
	}
	h.boards[boardId] = board

	// Persist the newly created board to the DB
	tasksJSON, _ := json.Marshal(board.ContentTasks)
	dbModel := &BoardModel{
		ID:           board.ID,
		ContentCode:  board.ContentCode,
		ContentTasks: string(tasksJSON),
		HuddleLink:   board.HuddleLink,
	}
	createOrUpdateBoard(dbModel)

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
	dbModel := &BoardModel{
		ID:           board.ID,
		ContentCode:  board.ContentCode,
		ContentTasks: string(tasksJSON),
		HuddleLink:   board.HuddleLink,
	}
	board.mu.RUnlock()

	err := createOrUpdateBoard(dbModel)
	if err != nil {
		log.Printf("[ERROR] Failed to persist board %s: %v", boardId, err)
	} else {
		log.Printf("[HUB] Persisted state for board %s", boardId)
	}
}
