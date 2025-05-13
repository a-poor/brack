package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	// SchemaVersion is the current version of the database schema
	SchemaVersion = 1

	// Default database file name
	defaultDBFileName = "brack.db"
)

// StorageClient handles SQLite database operations
type StorageClient struct {
	db     *sql.DB
	dbPath string
}

// GameState represents a user's progress in a puzzle
type GameState struct {
	PuzzleDate string    `json:"puzzleDate"`
	State      string    `json:"state"`
	Correct    int       `json:"correct"`
	Incorrect  int       `json:"incorrect"`
	Chars      int       `json:"chars"`
	LastPlayed time.Time `json:"lastPlayed"`
	Completed  bool      `json:"completed"`
}

// NewStorageClient creates a new storage client
func NewStorageClient() (*StorageClient, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, fmt.Errorf("failed to determine database path: %w", err)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	client := &StorageClient{
		db:     db,
		dbPath: dbPath,
	}

	// Initialize the database schema
	if err := client.initializeDB(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return client, nil
}

// Close closes the database connection
func (s *StorageClient) Close() error {
	return s.db.Close()
}

// initializeDB sets up the database schema if it doesn't exist
func (s *StorageClient) initializeDB() error {
	// Create metadata table
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}

	// Create puzzle_data table
	_, err = s.db.Exec(`
	CREATE TABLE IF NOT EXISTS puzzle_data (
		puzzle_date TEXT PRIMARY KEY,
		completion_text TEXT NOT NULL,
		completion_url TEXT NOT NULL,
		solutions TEXT NOT NULL,
		initial_puzzle TEXT NOT NULL,
		puzzle_solution TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to create puzzle_data table: %w", err)
	}

	// Create game_state table
	_, err = s.db.Exec(`
	CREATE TABLE IF NOT EXISTS game_state (
		puzzle_date TEXT PRIMARY KEY,
		state TEXT NOT NULL,
		correct INTEGER NOT NULL,
		incorrect INTEGER NOT NULL,
		chars INTEGER NOT NULL,
		last_played TIMESTAMP NOT NULL,
		completed BOOLEAN NOT NULL,
		FOREIGN KEY (puzzle_date) REFERENCES puzzle_data(puzzle_date)
	)`)
	if err != nil {
		return fmt.Errorf("failed to create game_state table: %w", err)
	}

	// Check if schema version exists
	var version string
	err = s.db.QueryRow("SELECT value FROM metadata WHERE key = 'schema_version'").Scan(&version)
	if err != nil {
		if err == sql.ErrNoRows {
			// Set initial schema version
			_, err = s.db.Exec("INSERT INTO metadata (key, value) VALUES ('schema_version', ?)", fmt.Sprintf("%d", SchemaVersion))
			if err != nil {
				return fmt.Errorf("failed to set schema version: %w", err)
			}
		} else {
			return fmt.Errorf("failed to query schema version: %w", err)
		}
	} else {
		// Handle migrations if needed in the future
		// For now, we're just at version 1
	}

	return nil
}

// SavePuzzleData saves puzzle data to the database
func (s *StorageClient) SavePuzzleData(data puzzledata) error {
	// Convert the solutions map to JSON
	solutionsJSON, err := json.Marshal(data.Solutions)
	if err != nil {
		return fmt.Errorf("failed to marshal solutions: %w", err)
	}

	// Insert or replace the puzzle data
	_, err = s.db.Exec(`
	INSERT OR REPLACE INTO puzzle_data (
		puzzle_date, 
		completion_text, 
		completion_url,
		solutions,
		initial_puzzle,
		puzzle_solution
	) VALUES (?, ?, ?, ?, ?, ?)`,
		data.PuzzleDate,
		data.CompletionText,
		data.CompletionURL,
		string(solutionsJSON),
		data.InitialPuzzle,
		data.PuzzleSolution)

	if err != nil {
		return fmt.Errorf("failed to save puzzle data: %w", err)
	}

	return nil
}

// GetPuzzleData retrieves puzzle data from the database
func (s *StorageClient) GetPuzzleData(puzzleDate string) (puzzledata, error) {
	var data puzzledata
	var solutionsJSON string

	err := s.db.QueryRow(`
	SELECT 
		puzzle_date, 
		completion_text, 
		completion_url,
		solutions,
		initial_puzzle,
		puzzle_solution
	FROM puzzle_data 
	WHERE puzzle_date = ?`, puzzleDate).Scan(
		&data.PuzzleDate,
		&data.CompletionText,
		&data.CompletionURL,
		&solutionsJSON,
		&data.InitialPuzzle,
		&data.PuzzleSolution)

	if err != nil {
		if err == sql.ErrNoRows {
			return puzzledata{}, fmt.Errorf("no puzzle data found for date %s", puzzleDate)
		}
		return puzzledata{}, fmt.Errorf("failed to get puzzle data: %w", err)
	}

	// Unmarshal the solutions JSON
	if err := json.Unmarshal([]byte(solutionsJSON), &data.Solutions); err != nil {
		return puzzledata{}, fmt.Errorf("failed to unmarshal solutions: %w", err)
	}

	return data, nil
}

// SaveGameState saves the user's game state
func (s *StorageClient) SaveGameState(state GameState) error {
	_, err := s.db.Exec(`
	INSERT OR REPLACE INTO game_state (
		puzzle_date,
		state,
		correct,
		incorrect,
		chars,
		last_played,
		completed
	) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		state.PuzzleDate,
		state.State,
		state.Correct,
		state.Incorrect,
		state.Chars,
		state.LastPlayed,
		state.Completed)

	if err != nil {
		return fmt.Errorf("failed to save game state: %w", err)
	}

	return nil
}

// GetGameState retrieves the user's game state
func (s *StorageClient) GetGameState(puzzleDate string) (GameState, error) {
	var state GameState

	err := s.db.QueryRow(`
	SELECT 
		puzzle_date,
		state,
		correct,
		incorrect,
		chars,
		last_played,
		completed
	FROM game_state 
	WHERE puzzle_date = ?`, puzzleDate).Scan(
		&state.PuzzleDate,
		&state.State,
		&state.Correct,
		&state.Incorrect,
		&state.Chars,
		&state.LastPlayed,
		&state.Completed)

	if err != nil {
		if err == sql.ErrNoRows {
			return GameState{}, fmt.Errorf("no game state found for date %s", puzzleDate)
		}
		return GameState{}, fmt.Errorf("failed to get game state: %w", err)
	}

	return state, nil
}

// getDBPath determines the path to the SQLite database file
func getDBPath() (string, error) {
	var configDir string

	// Try XDG_CONFIG_HOME first
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		configDir = filepath.Join(xdgConfigHome, "brack")
	} else {
		// Fall back to HOME directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".brack")
	}

	// Ensure the configuration directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create configuration directory: %w", err)
	}

	return filepath.Join(configDir, defaultDBFileName), nil
}

// HasPuzzleData checks if puzzle data exists for a given date
func (s *StorageClient) HasPuzzleData(puzzleDate string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM puzzle_data WHERE puzzle_date = ?", puzzleDate).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check for puzzle data: %w", err)
	}
	return count > 0, nil
}

// HasGameState checks if game state exists for a given date
func (s *StorageClient) HasGameState(puzzleDate string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM game_state WHERE puzzle_date = ?", puzzleDate).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check for game state: %w", err)
	}
	return count > 0, nil
}