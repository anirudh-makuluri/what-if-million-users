package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/anirudh-makuluri/what-if-million-users/ticketmaster/internal/store"
)

type Handler struct {
	store *store.PostgresStore
}

type createEventRequest struct {
	Name         string `json:"name"`
	TotalTickets int    `json:"total_tickets"`
}

type bookTicketRequest struct {
	EventID  int64  `json:"event_id"`
	UserID   string `json:"user_id"`
	Quantity int    `json:"quantity"`
}

func NewHandler(store *store.PostgresStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/events", h.events)
	mux.HandleFunc("/events/", h.eventByID)
	mux.HandleFunc("/book", h.bookTicket)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) events(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createEvent(w, r)
	case http.MethodGet:
		h.listEvents(w)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) createEvent(w http.ResponseWriter, r *http.Request) {
	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.TotalTickets <= 0 {
		respondError(w, http.StatusBadRequest, "total_tickets must be greater than 0")
		return
	}

	e, err := h.store.CreateEvent(req.Name, req.TotalTickets)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create event")
		return
	}

	respondJSON(w, http.StatusCreated, e)
}

func (h *Handler) listEvents(w http.ResponseWriter) {
	events, err := h.store.GetEvents()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch events")
		return
	}

	respondJSON(w, http.StatusOK, events)
}

func (h *Handler) eventByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	eventIDStr := strings.TrimPrefix(r.URL.Path, "/events/")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		respondError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	event, err := h.store.GetEvent(eventID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}
	if event == nil {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	respondJSON(w, http.StatusOK, event)
}

func (h *Handler) bookTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req bookTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.UserID = strings.TrimSpace(req.UserID)
	if req.EventID <= 0 {
		respondError(w, http.StatusBadRequest, "event_id must be greater than 0")
		return
	}
	if req.UserID == "" {
		respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.Quantity <= 0 {
		respondError(w, http.StatusBadRequest, "quantity must be greater than 0")
		return
	}

	booking, err := h.store.BookTicket(req.EventID, req.UserID, req.Quantity)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "event not found") {
			respondError(w, http.StatusNotFound, "event not found")
			return
		}
		if strings.Contains(errMsg, "not enough tickets") {
			respondError(w, http.StatusConflict, "not enough tickets available")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to book ticket")
		return
	}

	respondJSON(w, http.StatusCreated, booking)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
