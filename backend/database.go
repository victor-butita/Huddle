package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// BoardModel represents the structure of the board in the database.
type BoardModel struct {
	ID           string
	ContentCode  string
	ContentTasks string // Stored as JSON string
	HuddleLink   string
	LastUpdated  time.Time
}

func initDB(filepath string) {
	var err error
	db, err = sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open database: %v", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS boards (
		id TEXT NOT NULL PRIMARY KEY,
		content_code TEXT,
		content_tasks TEXT,
		huddle_link TEXT,
		last_updated TIMESTAMP
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create table: %v", err)
	}
	log.Println("[INFO] Database initialized and table is ready.")
}

func getBoardFromDB(id string) (*BoardModel, error) {
	stmt, err := db.Prepare("SELECT id, content_code, content_tasks, huddle_link, last_updated FROM boards WHERE id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	board := &BoardModel{}
	err = stmt.QueryRow(id).Scan(&board.ID, &board.ContentCode, &board.ContentTasks, &board.HuddleLink, &board.LastUpdated)
	if err == sql.ErrNoRows {
		return nil, nil // Not an error, just means it doesn't exist
	}
	return board, err
}

func createOrUpdateBoard(board *BoardModel) error {
	board.LastUpdated = time.Now()
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT OR REPLACE INTO boards (id, content_code, content_tasks, huddle_link, last_updated) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(board.ID, board.ContentCode, board.ContentTasks, board.HuddleLink, board.LastUpdated)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func cleanupOldBoards() {
	// This function can be called periodically to delete old boards.
	// We'll add a ticker in the hub to call this.
	log.Println("[INFO] Running cleanup for old boards...")
	query := "DELETE FROM boards WHERE last_updated < ?"

	// Delete boards older than 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)

	res, err := db.Exec(query, cutoff)
	if err != nil {
		log.Printf("[ERROR] Failed to clean up old boards: %v", err)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("[INFO] Cleaned up %d old boards.", rowsAffected)
	}
}
