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

func main() {
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

Bracket City: https://theatlantic.com/games/bracket-city
		`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Is there a date argument?
			d, err := parseDateArg(cmd.Args().Get(0))
			if err != nil {
				return err
			}

			// Fetch the puzzle data
			puzzle, err := getPuzzleData(d)
			if err != nil {
				return err
			}

			// Run the puzzle
			m := newModel(puzzle)
			p := tea.NewProgram(m, tea.WithAltScreen())
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
