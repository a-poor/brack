package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var _ tea.Model = model{}

var headerStyle = lipgloss.NewStyle().
	Bold(true)

var bodyStyle = lipgloss.NewStyle().
	Width(100)

var activeStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#0f0f0f")).
	Background(lipgloss.Color("#e8c566"))

type model struct {
	done      bool
	correct   int
	incorrect int
	chars     int
	state     string
	data      puzzledata
	txtin     textinput.Model
	w, h      int
	storage   *StorageClient
}

func newModel(d puzzledata, storage *StorageClient) model {
	tin := textinput.New()
	tin.Focus()
	
	m := model{
		data:    d,
		txtin:   tin,
		state:   d.InitialPuzzle,
		storage: storage,
	}
	
	// Try to load existing game state
	if storage != nil {
		gameState, err := storage.GetGameState(d.PuzzleDate)
		if err == nil {
			// Found existing state, restore it
			applyGameState(&m, gameState)
		}
	}
	
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			// Get the current input value
			in := m.txtin.Value()
			if in == "" {
				return m, nil
			}

			// Reset the input
			m.txtin.Reset()

			// Is that value a correct answer?
			for q, a := range getActiveQuestions(m.data, m.state) {
				if !strings.EqualFold(in, a) {
					continue
				}

				// If we got here, the answer is correct
				m.correct++

				// Replace the question with the correct answer
				m.state = strings.Replace(m.state, "["+q+"]", a, 1)

				// Save game state
				if m.storage != nil {
					gameState := modelToGameState(m)
					gameState.Completed = m.correct == len(m.data.Solutions)
					_ = m.storage.SaveGameState(gameState)
				}

				// Done?
				if m.correct == len(m.data.Solutions) {
					m.done = true
					return m, tea.Quit
				}

				// Good.
				return m, nil
			}

			// If we got here, the answer is incorrect
			m.incorrect++
			
			// Save game state
			if m.storage != nil {
				gameState := modelToGameState(m)
				gameState.Completed = false
				_ = m.storage.SaveGameState(gameState)
			}
			
			return m, nil

		default:
			if txt := msg.String(); len(txt) == 1 && unicode.IsLetter(rune(txt[0])) {
				m.chars++
				
				// Update character count in game state
				if m.storage != nil {
					gameState := modelToGameState(m)
					_ = m.storage.SaveGameState(gameState)
				}
			}
			tin, cmd := m.txtin.Update(msg)
			m.txtin = tin
			return m, cmd
		}
	}
	return m, nil
}

func (m model) View() string {
	var s string
	rest := m.state
	re := regexp.MustCompile(`\[([^\[\]]+)\]`)

	for re.MatchString(rest) {
		// Get the (first) match
		q := re.FindStringSubmatch(rest)[0]

		// Split on the match
		parts := re.Split(rest, 2)
		left, right := parts[0], parts[1]

		// Add the left part to the string as is
		s += left

		// Format and add the question
		s += activeStyle.Render(q)

		// Set the rest of the string to the right part
		rest = right
	}

	// Add the rest of the string as is
	s += rest

	// Format the score
	score := fmt.Sprintf(
		"✅ %d ❌ %d ⌨️ %d",
		m.correct,
		m.incorrect,
		m.chars,
	)

	if m.done {
		return lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render(
				"[ Bracket City | "+m.data.PuzzleDate+" ]",
			),
			score,
			"---",
			bodyStyle.Width(min(m.w, 100)).Render(s),
			"---",
			"🎉 You win! 🎉",
			"URL: "+m.data.CompletionURL,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Render(
			"[ Bracket City | "+
				m.data.PuzzleDate+
				" ]",
		),
		score,
		"---",
		bodyStyle.Width(min(m.w, 100)).Render(s),
		"---",
		m.txtin.View(),
	)
}
