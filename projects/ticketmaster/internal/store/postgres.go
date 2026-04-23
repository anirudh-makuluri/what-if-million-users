package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

type Event struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	TotalTickets     int    `json:"total_tickets"`
	AvailableTickets int    `json:"available_tickets"`
}

type Booking struct {
	ID       int64  `json:"id"`
	EventID  int64  `json:"event_id"`
	UserID   string `json:"user_id"`
	Quantity int    `json:"quantity"`
}

func NewPostgresStore(connString string) (*PostgresStore, error) {
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) InitSchema() error {
	const schema = `
	CREATE TABLE IF NOT EXISTS events (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		total_tickets INTEGER NOT NULL CHECK (total_tickets > 0),
		available_tickets INTEGER NOT NULL CHECK (available_tickets >= 0),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS bookings (
		id BIGSERIAL PRIMARY KEY,
		event_id BIGINT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL,
		quantity INTEGER NOT NULL CHECK (quantity > 0),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	`

	_, err := s.pool.Exec(context.Background(), schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

func (s *PostgresStore) CreateEvent(name string, totalTickets int) (*Event, error) {
	if totalTickets <= 0 {
		return nil, fmt.Errorf("total_tickets must be greater than 0")
	}

	var e Event
	err := s.pool.QueryRow(
		context.Background(),
		`INSERT INTO events (name, total_tickets, available_tickets)
		 VALUES ($1, $2, $2)
		 RETURNING id, name, total_tickets, available_tickets`,
		name,
		totalTickets,
	).Scan(&e.ID, &e.Name, &e.TotalTickets, &e.AvailableTickets)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return &e, nil
}

func (s *PostgresStore) GetEvent(eventID int64) (*Event, error) {
	var e Event
	err := s.pool.QueryRow(
		context.Background(),
		`SELECT id, name, total_tickets, available_tickets
		 FROM events
		 WHERE id = $1`,
		eventID,
	).Scan(&e.ID, &e.Name, &e.TotalTickets, &e.AvailableTickets)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return &e, nil
}

func (s *PostgresStore) GetEvents() ([]Event, error) {
	var events []Event
	rows, err := s.pool.Query(
		context.Background(),
		`SELECT id, name, total_tickets, available_tickets
		 FROM events
		 ORDER BY id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Name, &e.TotalTickets, &e.AvailableTickets); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return events, nil
}

func (s *PostgresStore) BookTicket(eventID int64, userID string, quantity int) (*Booking, error) {
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be greater than 0")
	}

	tx, err := s.pool.Begin(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	var availableTickets int
	err = tx.QueryRow(
		context.Background(),
		`SELECT available_tickets
		 FROM events
		 WHERE id = $1
		 FOR UPDATE`,
		eventID,
	).Scan(&availableTickets)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("event not found")
		}
		return nil, fmt.Errorf("failed to lock event row: %w", err)
	}

	if availableTickets < quantity {
		return nil, fmt.Errorf("not enough tickets available")
	}

	_, err = tx.Exec(
		context.Background(),
		`UPDATE events
		 SET available_tickets = available_tickets - $1
		 WHERE id = $2`,
		quantity,
		eventID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to decrement tickets: %w", err)
	}

	var b Booking
	err = tx.QueryRow(
		context.Background(),
		`INSERT INTO bookings (event_id, user_id, quantity)
		 VALUES ($1, $2, $3)
		 RETURNING id, event_id, user_id, quantity`,
		eventID,
		userID,
		quantity,
	).Scan(&b.ID, &b.EventID, &b.UserID, &b.Quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to commit booking transaction: %w", err)
	}

	return &b, nil
}
