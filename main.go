package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
)

// View modes
const (
	ModeGame     = "game"
	ModeCalendar = "calendar"
)

func main() {
	// Initialize storage client
	storage, err := NewStorageClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize storage: %s\n", err)
		os.Exit(1)
	}
	defer storage.Close()

	// Initialize calendar
	calendar := NewCalendar(storage)

	cmd := &cli.Command{
		Name:      "brack",
		Version:   "0.0.3",
		Usage:     "Play Bracket City on the command line.",
		ArgsUsage: "[DATE]",
		Description: `Play Bracket City, by the Atlantic.

Bracket City is a daily puzzle game published by The Atlantic.

DATE is an optional argument that specifies the date of the puzzle to play.
If no date is provided, the current date will be used.

Examples:

$ # Play the current day's puzzle
$ brack

$ # Play the puzzle for January 2, 2024
$ brack 2024-01-02

$ # Play the puzzle for the previous day
$ brack -1

$ # Open the calendar view
$ brack --calendar

Bracket City: https://theatlantic.com/games/bracket-city
		`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "calendar",
				Aliases: []string{"c"},
				Usage:   "Open the calendar view",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Check if calendar view is requested
			showCalendar := cmd.Bool("calendar")
			
			// Initial view mode
			viewMode := ModeGame
			if showCalendar {
				viewMode = ModeCalendar
			}
			
			// Is there a date argument?
			d, err := parseDateArg(cmd.Args().Get(0))
			if err != nil {
				return err
			}

			// Try to load puzzle data from local storage first
			puzzleDate := d.Format("2006-01-02")
			hasPuzzle, _ := storage.HasPuzzleData(puzzleDate)
			
			var puzzle puzzledata
			if hasPuzzle {
				// Load from storage
				puzzle, err = storage.GetPuzzleData(puzzleDate)
				if err != nil {
					return fmt.Errorf("failed to load puzzle from storage: %w", err)
				}
			} else {
				// Fetch from API
				puzzle, err = getPuzzleData(d)
				if err != nil {
					return err
				}
				
				// Save to storage
				if err := storage.SavePuzzleData(puzzle); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to save puzzle data: %s\n", err)
				}
			}

			// Create initial application model
			appModel := &AppModel{
				mode:     viewMode,
				model:    newModel(puzzle, storage),
				calendar: calendar,
				storage:  storage,
			}
			
			// Run the program
			p := tea.NewProgram(appModel, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return err
			}

			// Done!
			return nil
		},
	}

	ctx := context.Background()
	if err := cmd.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func parseDateArg(s string) (time.Time, error) {
	// If no date is provided, use the current date
	if s == "" {
		return time.Now(), nil
	}

	// Try to parse it as a negative number
	if n, err := strconv.Atoi(s); err == nil && n < 0 {
		return time.Now().AddDate(0, 0, n), nil
	}

	// Parse the date
	return time.Parse("2006-01-02", s)
}

// AppModel is the top-level application model that manages different views
type AppModel struct {
	mode     string         // current view mode
	model    model          // game model
	calendar *Calendar      // calendar model
	storage  *StorageClient // storage client
}

// Init initializes the application
func (a AppModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the application
func (a AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key handlers
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "tab", "c":
			// Toggle between game and calendar view
			if a.mode == ModeGame {
				a.mode = ModeCalendar
			} else {
				a.mode = ModeGame
			}
			return a, nil
		}

		if a.mode == ModeCalendar {
			// Handle calendar-specific updates
			calendar, cmd := a.calendar.Update(msg)
			a.calendar = calendar
			
			// If enter was pressed in calendar view, switch to game view with selected date
			if msg.String() == "enter" || msg.String() == " " {
				selectedDate := a.calendar.SelectedDate()
				dateStr := selectedDate.Format("2006-01-02")
				
				// Try to load the puzzle for the selected date
				hasPuzzle, _ := a.storage.HasPuzzleData(dateStr)
				var puzzle puzzledata
				
				if hasPuzzle {
					puzzle, _ = a.storage.GetPuzzleData(dateStr)
				} else {
					// Fetch from API
					puzzle, _ = getPuzzleData(selectedDate)
					// Save to storage
					_ = a.storage.SavePuzzleData(puzzle)
				}
				
				// Create a new model for the selected date
				a.model = newModel(puzzle, a.storage)
				a.mode = ModeGame
			}
			
			return a, cmd
		} else {
			// Forward messages to the game model
			newModel, cmd := a.model.Update(msg)
			updatedModel, ok := newModel.(model)
			if ok {
				a.model = updatedModel
			}
			return a, cmd
		}
		
	case tea.WindowSizeMsg:
		// Forward window size messages to both models
		newModel, _ := a.model.Update(msg)
		updatedModel, ok := newModel.(model)
		if ok {
			a.model = updatedModel
		}
		
		calendar, _ := a.calendar.Update(msg)
		a.calendar = calendar
	}
	
	return a, nil
}

// View renders the current view
func (a AppModel) View() string {
	// Instructions for switching between views
	instructions := "\nPress 'tab' to toggle between game and calendar view"
	
	if a.mode == ModeCalendar {
		return a.calendar.View() + instructions
	} else {
		return a.model.View() + instructions
	}
}
