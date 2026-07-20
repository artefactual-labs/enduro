package collection

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	StatusHistoryAvailable   = "available"
	StatusHistoryPartial     = "partial"
	StatusHistoryUnavailable = "unavailable"
)

type StatusTransition struct {
	ID             uint64         `db:"id"`
	CollectionID   uint           `db:"collection_id"`
	WorkflowID     string         `db:"workflow_id"`
	RunID          string         `db:"run_id"`
	PreviousStatus sql.NullInt64  `db:"previous_status"`
	Status         Status         `db:"status"`
	OccurredAt     time.Time      `db:"occurred_at"`
	IsRunStart     bool           `db:"is_run_start"`
	Reason         sql.NullString `db:"reason"`
}

type collectionStatusState struct {
	WorkflowID string `db:"workflow_id"`
	RunID      string `db:"run_id"`
	Status     Status `db:"status"`
}

func (svc *collectionImpl) updateWithStatusTransition(
	ctx context.Context,
	ID uint,
	next collectionStatusState,
	update func(*sqlx.Tx) error,
) error {
	tx, err := svc.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error beginning collection update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	previous := collectionStatusState{}
	query := `SELECT workflow_id, run_id, status FROM collection WHERE id = (?) FOR UPDATE`
	if err := tx.GetContext(ctx, &previous, tx.Rebind(query), ID); err != nil {
		return fmt.Errorf("error reading collection status: %w", err)
	}
	if next.WorkflowID == "" {
		next.WorkflowID = previous.WorkflowID
	}
	if next.RunID == "" {
		next.RunID = previous.RunID
	}

	if err := update(tx); err != nil {
		return err
	}

	runChanged := previous.WorkflowID != next.WorkflowID || previous.RunID != next.RunID
	if previous.Status != next.Status || runChanged {
		if err := insertStatusTransition(ctx, tx, ID, &previous.Status, next, runChanged, transitionReason(previous.Status, next.Status, runChanged)); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing collection update: %w", err)
	}

	return nil
}

func insertStatusTransition(
	ctx context.Context,
	tx *sqlx.Tx,
	collectionID uint,
	previousStatus *Status,
	next collectionStatusState,
	isRunStart bool,
	reason string,
) error {
	query := `INSERT INTO collection_status_transition (collection_id, workflow_id, run_id, previous_status, status, is_run_start, reason) VALUES ((?), (?), (?), (?), (?), (?), (?))`
	args := []any{
		collectionID,
		next.WorkflowID,
		next.RunID,
		previousStatus,
		next.Status,
		isRunStart,
		reason,
	}
	if _, err := tx.ExecContext(ctx, tx.Rebind(query), args...); err != nil {
		return fmt.Errorf("error inserting collection status transition: %w", err)
	}

	return nil
}

func transitionReason(previous, next Status, runStart bool) string {
	if runStart {
		return "workflow_retried"
	}

	switch next {
	case StatusQueued:
		return "workflow_queued"
	case StatusInProgress:
		if previous == StatusQueued {
			return "pipeline_acquired"
		}
		if previous == StatusPending {
			return "operator_decision_received"
		}
		return "processing_resumed"
	case StatusPending:
		return "operator_decision_required"
	case StatusDone:
		return "workflow_completed"
	case StatusError:
		return "workflow_failed"
	case StatusAbandoned:
		return "workflow_abandoned"
	default:
		return "status_changed"
	}
}

func (svc *collectionImpl) readStatusTransitions(ctx context.Context, collectionID uint) ([]StatusTransition, error) {
	query := `SELECT id, collection_id, workflow_id, run_id, previous_status, status, CONVERT_TZ(occurred_at, @@session.time_zone, '+00:00') AS occurred_at, is_run_start, reason FROM collection_status_transition WHERE collection_id = (?) ORDER BY occurred_at ASC, id ASC`
	transitions := []StatusTransition{}
	if err := svc.db.SelectContext(ctx, &transitions, svc.db.Rebind(query), collectionID); err != nil {
		return nil, fmt.Errorf("error reading collection status transitions: %w", err)
	}

	return transitions, nil
}

func statusHistoryAvailability(transitions []StatusTransition, currentRunID string) string {
	for _, transition := range transitions {
		if transition.RunID != currentRunID {
			continue
		}
		if transition.IsRunStart {
			return StatusHistoryAvailable
		}
		return StatusHistoryPartial
	}

	return StatusHistoryUnavailable
}
