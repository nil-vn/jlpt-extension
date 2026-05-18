package study

import (
	"context"
	"database/sql"
	"fmt"
)

type OrderMode string

const (
	OrderModeSequential OrderMode = "sequential"
	OrderModeRandom     OrderMode = "random"
)

type State struct {
	CurrentCardID *string
	OrderMode     OrderMode
	RevealAnswers bool
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context) (State, error) {
	state, found, err := r.find(ctx)
	if err != nil {
		return State{}, err
	}
	if found {
		return state, nil
	}
	state = State{OrderMode: OrderModeSequential, RevealAnswers: false}
	if err := r.Save(ctx, state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (r *Repository) Save(ctx context.Context, state State) error {
	reveal := 0
	if state.RevealAnswers {
		reveal = 1
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO study_state (id, current_card_id, order_mode, reveal_answers)
VALUES (1, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  current_card_id = excluded.current_card_id,
  order_mode = excluded.order_mode,
  reveal_answers = excluded.reveal_answers,
  updated_at = CURRENT_TIMESTAMP`, state.CurrentCardID, state.OrderMode, reveal)
	if err != nil {
		return fmt.Errorf("save study state: %w", err)
	}
	return nil
}

func (r *Repository) find(ctx context.Context) (State, bool, error) {
	var state State
	var currentCard sql.NullString
	var reveal int
	err := r.db.QueryRowContext(ctx, `SELECT current_card_id, order_mode, reveal_answers FROM study_state WHERE id = 1`).Scan(&currentCard, &state.OrderMode, &reveal)
	if err == sql.ErrNoRows {
		return State{}, false, nil
	}
	if err != nil {
		return State{}, false, fmt.Errorf("get study state: %w", err)
	}
	if currentCard.Valid {
		state.CurrentCardID = &currentCard.String
	}
	state.RevealAnswers = reveal != 0
	return state, true, nil
}
