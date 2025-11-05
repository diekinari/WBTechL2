package main

import (
	"errors"
	"sync"
	"time"
)

type Event struct {
	ID          int    `json:"id"`
	UserID      string `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

type Calendar struct {
	events map[int]*Event
	mu     sync.RWMutex
	nextID int
}

func (c *Calendar) CreateEvent(event Event) (int, error) {
	if event.UserID == "" {
		return 0, errors.New("user ID is required")
	}

	// ID всегда будет пустой, поэтому он будет генерироваться сервером
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.nextID
	c.nextID++
	event.ID = id
	c.events[id] = &event
	return id, nil
}

func (c *Calendar) UpdateEvent(event Event) error {
	if event.ID == 0 {
		return errors.New("event ID is required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Проверяем существование события
	if _, exists := c.events[event.ID]; !exists {
		return errors.New("event not found")
	}

	c.events[event.ID] = &event
	return nil
}

func (c *Calendar) DeleteEvent(id int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Проверяем существование события
	if _, exists := c.events[id]; !exists {
		return errors.New("event not found")
	}

	delete(c.events, id)
	return nil
}

func ValidateEvent(event Event) error {
	if event.UserID == "" {
		return errors.New("user ID is required")
	}
	if event.Title == "" {
		return errors.New("title is required")
	}
	// Date must be in format YYYY-MM-DD
	if _, err := time.Parse("2006-01-02", event.Date); err != nil {
		return errors.New("date must be in format YYYY-MM-DD")
	}
	return nil
}

func (c *Calendar) GetEventsForDay(date string, userID string) ([]Event, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	events := make([]Event, 0)
	for _, event := range c.events {
		if event.Date == date && event.UserID == userID {
			events = append(events, *event)
		}
	}
	return events, nil
}

// addDays adds days to a date
func addDays(date string, days int) string {
	dateTime, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	return dateTime.AddDate(0, 0, days).Format("2006-01-02")
}

func (c *Calendar) GetEventsForWeek(date string, userID string) ([]Event, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	startDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, errors.New("invalid date format")
	}

	// Неделя начинается с указанной даты и длится 7 дней
	endDate := startDate.AddDate(0, 0, 6)
	endDateStr := endDate.Format("2006-01-02")

	events := make([]Event, 0)
	for _, event := range c.events {
		if event.UserID == userID && event.Date >= date && event.Date <= endDateStr {
			events = append(events, *event)
		}
	}
	return events, nil
}

func (c *Calendar) GetEventsForMonth(date string, userID string) ([]Event, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	startDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, errors.New("invalid date format")
	}

	// Получаем первый день месяца
	firstOfMonth := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	// Получаем последний день месяца
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

	startDateStr := firstOfMonth.Format("2006-01-02")
	endDateStr := lastOfMonth.Format("2006-01-02")

	events := make([]Event, 0)
	for _, event := range c.events {
		if event.UserID == userID && event.Date >= startDateStr && event.Date <= endDateStr {
			events = append(events, *event)
		}
	}
	return events, nil
}
