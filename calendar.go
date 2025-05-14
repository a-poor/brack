package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Calendar is a component for navigating dates to select puzzles
type Calendar struct {
	cursor     time.Time // current cursor position (selected date)
	currentDay time.Time // today's date
	viewMonth  time.Time // month being viewed
	width      int
	height     int
	storage    *StorageClient
}

// NewCalendar creates a new calendar component
func NewCalendar(storage *StorageClient) *Calendar {
	today := time.Now()
	// Set time to midnight
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	
	return &Calendar{
		cursor:     today,
		currentDay: today,
		viewMonth:  time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location()),
		storage:    storage,
	}
}

// Update handles messages for the calendar
func (c *Calendar) Update(msg tea.Msg) (*Calendar, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			c.moveCursorLeft()
		case "right", "l":
			c.moveCursorRight()
		case "up", "k":
			c.moveCursorUp()
		case "down", "j":
			c.moveCursorDown()
		case "enter", " ":
			// Return the selected date
			return c, nil
		}
	case tea.WindowSizeMsg:
		c.width, c.height = msg.Width, msg.Height
	}
	
	return c, nil
}

// SelectedDate returns the currently selected date
func (c *Calendar) SelectedDate() time.Time {
	return c.cursor
}

// View renders the calendar
func (c *Calendar) View() string {
	// Style definitions
	titleStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	normalDayStyle := lipgloss.NewStyle().Padding(0, 1)
	selectedDayStyle := lipgloss.NewStyle().Padding(0, 1).Underline(true)
	currentDayStyle := lipgloss.NewStyle().Padding(0, 1).Background(lipgloss.Color("#555555"))
	completedDayStyle := lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("#00AA00"))
	inProgressDayStyle := lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("#AAAA00"))
	futureDayStyle := lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("#555555"))
	
	// Calendar title (month and year)
	title := titleStyle.Render(c.viewMonth.Format("January 2006"))
	
	// Days of week header
	header := headerStyle.Render("Su Mo Tu We Th Fr Sa")
	
	// Calculate the first day of the month
	firstDay := time.Date(c.viewMonth.Year(), c.viewMonth.Month(), 1, 0, 0, 0, 0, c.viewMonth.Location())
	
	// Calculate days in month
	daysInMonth := daysInMonth(c.viewMonth)
	
	// Get the weekday (0 = Sunday, 1 = Monday, ..., 6 = Saturday)
	weekday := int(firstDay.Weekday())
	
	// Build the calendar days
	var calendarDays string
	var week string
	
	// Add initial padding for the first week
	for i := 0; i < weekday; i++ {
		week += "   "
	}
	
	// Add each day of the month
	for day := 1; day <= daysInMonth; day++ {
		// Create date for this day
		date := time.Date(c.viewMonth.Year(), c.viewMonth.Month(), day, 0, 0, 0, 0, c.viewMonth.Location())
		
		// Check if this date has a completed game, in-progress game, or is in the future
		var dayStyle lipgloss.Style
		
		// Future dates get gray text
		if date.After(c.currentDay) {
			dayStyle = futureDayStyle
		} else {
			// Check game status in storage
			hasGame := false
			isCompleted := false
			
			if c.storage != nil {
				// Format date for lookup
				dateStr := date.Format("2006-01-02")
				
				// Check if game exists
				hasGameState, _ := c.storage.HasGameState(dateStr)
				if hasGameState {
					hasGame = true
					
					// Check if game is completed
					gameState, err := c.storage.GetGameState(dateStr)
					if err == nil {
						isCompleted = gameState.Completed
					}
				}
			}
			
			if isCompleted {
				dayStyle = completedDayStyle
			} else if hasGame {
				dayStyle = inProgressDayStyle
			} else {
				dayStyle = normalDayStyle
			}
		}
		
		// Check if this is the selected day
		if date.Year() == c.cursor.Year() && date.Month() == c.cursor.Month() && date.Day() == c.cursor.Day() {
			dayStyle = selectedDayStyle
		}
		
		// Check if this is today
		if date.Year() == c.currentDay.Year() && date.Month() == c.currentDay.Month() && date.Day() == c.currentDay.Day() {
			dayStyle = currentDayStyle
		}
		
		// Format and add the day
		dayText := fmt.Sprintf("%2d", day)
		week += dayStyle.Render(dayText)
		
		// If this is Saturday or the last day of the month, start a new line
		if (weekday+day)%7 == 0 || day == daysInMonth {
			calendarDays += week + "\n"
			week = ""
		}
	}
	
	// Combine all parts of the calendar
	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		header,
		calendarDays,
	)
}

// moveCursorLeft moves the cursor one day left
func (c *Calendar) moveCursorLeft() {
	newCursor := c.cursor.AddDate(0, 0, -1)
	
	// Don't allow moving past the beginning of time or into the future
	if !newCursor.After(c.currentDay) && newCursor.Year() >= 2000 {
		c.cursor = newCursor
		
		// If we move to the previous month, update the viewMonth
		if c.cursor.Month() != c.viewMonth.Month() {
			c.viewMonth = time.Date(c.cursor.Year(), c.cursor.Month(), 1, 0, 0, 0, 0, c.cursor.Location())
		}
	}
}

// moveCursorRight moves the cursor one day right
func (c *Calendar) moveCursorRight() {
	newCursor := c.cursor.AddDate(0, 0, 1)
	
	// Don't allow moving into the future
	if !newCursor.After(c.currentDay) {
		c.cursor = newCursor
		
		// If we move to the next month, update the viewMonth
		if c.cursor.Month() != c.viewMonth.Month() {
			c.viewMonth = time.Date(c.cursor.Year(), c.cursor.Month(), 1, 0, 0, 0, 0, c.cursor.Location())
		}
	}
}

// moveCursorUp moves the cursor up one week
func (c *Calendar) moveCursorUp() {
	newCursor := c.cursor.AddDate(0, 0, -7)
	
	// Don't allow moving past the beginning of time or into the future
	if !newCursor.After(c.currentDay) && newCursor.Year() >= 2000 {
		c.cursor = newCursor
		
		// If we move to the previous month, update the viewMonth
		if c.cursor.Month() != c.viewMonth.Month() {
			c.viewMonth = time.Date(c.cursor.Year(), c.cursor.Month(), 1, 0, 0, 0, 0, c.cursor.Location())
		}
	}
}

// moveCursorDown moves the cursor down one week
func (c *Calendar) moveCursorDown() {
	newCursor := c.cursor.AddDate(0, 0, 7)
	
	// Don't allow moving into the future
	if !newCursor.After(c.currentDay) {
		c.cursor = newCursor
		
		// If we move to the next month, update the viewMonth
		if c.cursor.Month() != c.viewMonth.Month() {
			c.viewMonth = time.Date(c.cursor.Year(), c.cursor.Month(), 1, 0, 0, 0, 0, c.cursor.Location())
		}
	}
}

// daysInMonth returns the number of days in the given month
func daysInMonth(date time.Time) int {
	// Go to the first day of the next month, then subtract 1 day
	nextMonth := date.AddDate(0, 1, 0)
	lastDay := nextMonth.AddDate(0, 0, -1)
	return lastDay.Day()
}