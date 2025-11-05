package main

import (
	"testing"
)

func TestCreateEvent(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	event := Event{
		UserID:      "user1",
		Title:       "Test Event",
		Description: "Test Description",
		Date:        "2023-12-31",
	}

	id, err := calendar.CreateEvent(event)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if id != 1 {
		t.Errorf("Expected ID 1, got %d", id)
	}

	// Проверяем, что событие создано
	calendar.mu.RLock()
	createdEvent, exists := calendar.events[id]
	calendar.mu.RUnlock()

	if !exists {
		t.Fatal("Event was not created")
	}

	if createdEvent.UserID != "user1" {
		t.Errorf("Expected UserID 'user1', got '%s'", createdEvent.UserID)
	}

	if createdEvent.Title != "Test Event" {
		t.Errorf("Expected Title 'Test Event', got '%s'", createdEvent.Title)
	}
}

func TestCreateEventWithoutUserID(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	event := Event{
		Title: "Test Event",
		Date:  "2023-12-31",
	}

	_, err := calendar.CreateEvent(event)
	if err == nil {
		t.Fatal("Expected error for missing UserID, got nil")
	}

	if err.Error() != "user ID is required" {
		t.Errorf("Expected error 'user ID is required', got '%s'", err.Error())
	}
}

func TestUpdateEvent(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	// Создаем событие
	event := Event{
		UserID:      "user1",
		Title:       "Original Title",
		Description: "Original Description",
		Date:        "2023-12-31",
	}

	id, err := calendar.CreateEvent(event)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Обновляем событие
	updatedEvent := Event{
		ID:          id,
		UserID:      "user1",
		Title:       "Updated Title",
		Description: "Updated Description",
		Date:        "2023-12-31",
	}

	err = calendar.UpdateEvent(updatedEvent)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Проверяем, что событие обновлено
	calendar.mu.RLock()
	storedEvent := calendar.events[id]
	calendar.mu.RUnlock()

	if storedEvent.Title != "Updated Title" {
		t.Errorf("Expected Title 'Updated Title', got '%s'", storedEvent.Title)
	}
}

func TestUpdateNonExistentEvent(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	event := Event{
		ID:     999,
		UserID: "user1",
		Title:  "Test",
		Date:   "2023-12-31",
	}

	err := calendar.UpdateEvent(event)
	if err == nil {
		t.Fatal("Expected error for non-existent event, got nil")
	}

	if err.Error() != "event not found" {
		t.Errorf("Expected error 'event not found', got '%s'", err.Error())
	}
}

func TestDeleteEvent(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	// Создаем событие
	event := Event{
		UserID: "user1",
		Title:  "Test Event",
		Date:   "2023-12-31",
	}

	id, err := calendar.CreateEvent(event)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	// Удаляем событие
	err = calendar.DeleteEvent(id)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Проверяем, что событие удалено
	calendar.mu.RLock()
	_, exists := calendar.events[id]
	calendar.mu.RUnlock()

	if exists {
		t.Fatal("Event was not deleted")
	}
}

func TestDeleteNonExistentEvent(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	err := calendar.DeleteEvent(999)
	if err == nil {
		t.Fatal("Expected error for non-existent event, got nil")
	}

	if err.Error() != "event not found" {
		t.Errorf("Expected error 'event not found', got '%s'", err.Error())
	}
}

func TestGetEventsForDay(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	// Создаем несколько событий
	event1 := Event{UserID: "user1", Title: "Event 1", Date: "2023-12-31"}
	event2 := Event{UserID: "user1", Title: "Event 2", Date: "2023-12-31"}
	event3 := Event{UserID: "user1", Title: "Event 3", Date: "2024-01-01"}
	event4 := Event{UserID: "user2", Title: "Event 4", Date: "2023-12-31"}

	calendar.CreateEvent(event1)
	calendar.CreateEvent(event2)
	calendar.CreateEvent(event3)
	calendar.CreateEvent(event4)

	events, err := calendar.GetEventsForDay("2023-12-31", "user1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
}

func TestGetEventsForWeek(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	// Создаем события на неделю
	calendar.CreateEvent(Event{UserID: "user1", Title: "Day 1", Date: "2023-12-31"})
	calendar.CreateEvent(Event{UserID: "user1", Title: "Day 3", Date: "2024-01-02"})
	calendar.CreateEvent(Event{UserID: "user1", Title: "Day 7", Date: "2024-01-06"})
	calendar.CreateEvent(Event{UserID: "user1", Title: "Day 8", Date: "2024-01-07"}) // Вне недели
	calendar.CreateEvent(Event{UserID: "user2", Title: "Other User", Date: "2024-01-02"})

	events, err := calendar.GetEventsForWeek("2023-12-31", "user1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}
}

func TestGetEventsForMonth(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	// Создаем события в декабре 2023
	calendar.CreateEvent(Event{UserID: "user1", Title: "Dec 1", Date: "2023-12-01"})
	calendar.CreateEvent(Event{UserID: "user1", Title: "Dec 15", Date: "2023-12-15"})
	calendar.CreateEvent(Event{UserID: "user1", Title: "Dec 31", Date: "2023-12-31"})
	calendar.CreateEvent(Event{UserID: "user1", Title: "Jan 1", Date: "2024-01-01"}) // Вне месяца
	calendar.CreateEvent(Event{UserID: "user2", Title: "Other User", Date: "2023-12-15"})

	events, err := calendar.GetEventsForMonth("2023-12-15", "user1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}
}

func TestValidateEvent(t *testing.T) {
	tests := []struct {
		name    string
		event   Event
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid event",
			event: Event{
				UserID: "user1",
				Title:  "Test",
				Date:   "2023-12-31",
			},
			wantErr: false,
		},
		{
			name: "missing user ID",
			event: Event{
				Title: "Test",
				Date:  "2023-12-31",
			},
			wantErr: true,
			errMsg:  "user ID is required",
		},
		{
			name: "missing title",
			event: Event{
				UserID: "user1",
				Date:   "2023-12-31",
			},
			wantErr: true,
			errMsg:  "title is required",
		},
		{
			name: "invalid date format",
			event: Event{
				UserID: "user1",
				Title:  "Test",
				Date:   "2023/12/31",
			},
			wantErr: true,
			errMsg:  "date must be in format YYYY-MM-DD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEvent(tt.event)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	// Запускаем несколько горутин для одновременного доступа
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			event := Event{
				UserID: "user1",
				Title:  "Concurrent Event",
				Date:   "2023-12-31",
			}
			_, err := calendar.CreateEvent(event)
			if err != nil {
				t.Errorf("Error creating event: %v", err)
			}
			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	// Проверяем, что все события созданы
	calendar.mu.RLock()
	eventCount := len(calendar.events)
	calendar.mu.RUnlock()

	if eventCount != 10 {
		t.Errorf("Expected 10 events, got %d", eventCount)
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}

	// Создаем начальное событие
	event := Event{UserID: "user1", Title: "Test", Date: "2023-12-31"}
	eventID, _ := calendar.CreateEvent(event)

	// Запускаем горутины для чтения и записи
	done := make(chan bool, 20)

	// 10 горутин для чтения
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = calendar.GetEventsForDay("2023-12-31", "user1")
			done <- true
		}()
	}

	// 10 горутин для обновления
	for i := 0; i < 10; i++ {
		go func(id int) {
			updatedEvent := Event{
				ID:     eventID,
				UserID: "user1",
				Title:  "Updated",
				Date:   "2023-12-31",
			}
			_ = calendar.UpdateEvent(updatedEvent)
			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 20; i++ {
		<-done
	}

	// Проверяем, что событие все еще существует
	calendar.mu.RLock()
	_, exists := calendar.events[eventID]
	calendar.mu.RUnlock()

	if !exists {
		t.Fatal("Event was lost during concurrent access")
	}
}
