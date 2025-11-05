package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
}

func writeResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func writeErrorResponse(w http.ResponseWriter, status int, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	response := Response{Error: errorMsg}
	json.NewEncoder(w).Encode(response)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	// Загружаем .env файл (если существует)
	_ = godotenv.Load()

	// Получаем порт из переменной окружения или флага
	portFlag := flag.String("port", "", "Порт для запуска сервера")
	flag.Parse()

	port := os.Getenv("PORT")
	if *portFlag != "" {
		port = *portFlag
	}
	if port == "" {
		port = "8080"
	}

	r := mux.NewRouter()
	r.Use(loggingMiddleware)
	calendar := &Calendar{
		events: make(map[int]*Event),
		nextID: 1,
	}
	r.HandleFunc("/create_event", createEventHandler(calendar)).Methods("POST")
	r.HandleFunc("/update_event", updateEventHandler(calendar)).Methods("POST")
	r.HandleFunc("/delete_event", deleteEventHandler(calendar)).Methods("POST")
	r.HandleFunc("/events_for_day", eventsForDayHandler(calendar)).Methods("GET")
	r.HandleFunc("/events_for_week", eventsForWeekHandler(calendar)).Methods("GET")
	r.HandleFunc("/events_for_month", eventsForMonthHandler(calendar)).Methods("GET")

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Сервер запущен на порту %s", port)
	log.Fatal(http.ListenAndServe(addr, r))
}

func createEventHandler(calendar *Calendar) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to read request body")
			return
		}
		defer r.Body.Close()

		var data Event
		err = json.Unmarshal(body, &data)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format")
			return
		}

		// Валидация события
		if err := ValidateEvent(data); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		id, err := calendar.CreateEvent(data)
		if err != nil {
			writeErrorResponse(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		log.Println("Event created with id:", id)
		writeResponse(w, http.StatusOK, Response{Message: "Event created", Data: map[string]interface{}{"id": id}})
	}
}

func updateEventHandler(calendar *Calendar) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to read request body")
			return
		}
		defer r.Body.Close()

		var data Event
		err = json.Unmarshal(body, &data)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format")
			return
		}

		// Валидация события
		if err := ValidateEvent(data); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		err = calendar.UpdateEvent(data)
		if err != nil {
			writeErrorResponse(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		log.Println("Event updated with id:", data.ID)
		writeResponse(w, http.StatusOK, Response{Message: "Event updated", Data: map[string]interface{}{"id": data.ID}})
	}
}

func deleteEventHandler(calendar *Calendar) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to read request body")
			return
		}
		defer r.Body.Close()

		var data struct {
			EventID int `json:"event_id"`
		}
		err = json.Unmarshal(body, &data)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format")
			return
		}

		if data.EventID == 0 {
			writeErrorResponse(w, http.StatusBadRequest, "event_id is required")
			return
		}

		err = calendar.DeleteEvent(data.EventID)
		if err != nil {
			writeErrorResponse(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		log.Println("Event deleted with id:", data.EventID)
		writeResponse(w, http.StatusOK, Response{Message: "Event deleted", Data: map[string]interface{}{"id": data.EventID}})
	}
}

func eventsForDayHandler(calendar *Calendar) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		date := r.URL.Query().Get("date")
		if date == "" {
			http.Error(w, "Date is required", http.StatusBadRequest)
			return
		}

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Валидация даты
		if _, err := time.Parse("2006-01-02", date); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}

		events, err := calendar.GetEventsForDay(date, userID)
		if err != nil {
			writeErrorResponse(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		writeResponse(w, http.StatusOK, Response{Message: "Events for day", Data: events})
	}
}

func eventsForWeekHandler(calendar *Calendar) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")
		if date == "" {
			http.Error(w, "Date is required", http.StatusBadRequest)
			return
		}

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Валидация даты
		if _, err := time.Parse("2006-01-02", date); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}

		events, err := calendar.GetEventsForWeek(date, userID)
		if err != nil {
			writeErrorResponse(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		writeResponse(w, http.StatusOK, Response{Message: "Events for week", Data: events})
	}
}

func eventsForMonthHandler(calendar *Calendar) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")
		if date == "" {
			http.Error(w, "Date is required", http.StatusBadRequest)
			return
		}

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		// Валидация даты
		if _, err := time.Parse("2006-01-02", date); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
			return
		}

		events, err := calendar.GetEventsForMonth(date, userID)
		if err != nil {
			writeErrorResponse(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		writeResponse(w, http.StatusOK, Response{Message: "Events for month", Data: events})
	}
}
