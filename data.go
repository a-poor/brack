package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const endpoint = "https://8huadblp0h.execute-api.us-east-2.amazonaws.com/puzzles"

type puzzledata struct {
	CompletionText string            `json:"completionText"`
	PuzzleDate     string            `json:"puzzleDate"`
	CompletionURL  string            `json:"completionURL"`
	Solutions      map[string]string `json:"solutions"`
	InitialPuzzle  string            `json:"initialPuzzle"`
	PuzzleSolution string            `json:"puzzleSolution"`
}

func getPuzzleData(d time.Time) (puzzledata, error) {
	url := endpoint + "/" + d.Format("2006-01-02")
	resp, err := http.Get(url)
	if err != nil {
		return puzzledata{}, err
	}
	defer resp.Body.Close()

	var puzzle puzzledata
	if err := json.NewDecoder(resp.Body).Decode(&puzzle); err != nil {
		return puzzledata{}, err
	}
	return puzzle, nil
}

func getActiveQuestions(pd puzzledata, s string) map[string]string {
	qs := make(map[string]string)
	for k, v := range pd.Solutions {
		if strings.Contains(s, "["+k+"]") {
			qs[k] = v
		}
	}
	return qs
}

// ModelToGameState converts a model to a GameState
func modelToGameState(m model) GameState {
	return GameState{
		PuzzleDate: m.data.PuzzleDate,
		State:      m.state,
		Correct:    m.correct,
		Incorrect:  m.incorrect,
		Chars:      m.chars,
		LastPlayed: time.Now(),
		Completed:  m.done,
	}
}

// ApplyGameState applies a GameState to a model
func applyGameState(m *model, gs GameState) {
	m.state = gs.State
	m.correct = gs.Correct
	m.incorrect = gs.Incorrect
	m.chars = gs.Chars
	m.done = gs.Completed
}
