package notify

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/leotime/leotime/apps/api/internal/config"
	"github.com/leotime/leotime/apps/api/internal/outbox"
	"github.com/leotime/leotime/apps/api/internal/store"
)

type StillRunningNotifier struct {
	store         *store.Store
	outbox        *outbox.Store
	publicBaseURL string
	maxAttempts   int
	now           func() time.Time
}

func NewStillRunningNotifier(st *store.Store, outboxStore *outbox.Store, cfg config.Config) *StillRunningNotifier {
	return &StillRunningNotifier{
		store:         st,
		outbox:        outboxStore,
		publicBaseURL: cfg.PublicBaseURL,
		maxAttempts:   cfg.MailMaxAttempts,
		now:           time.Now,
	}
}

func (n *StillRunningNotifier) EnqueueDue(ctx context.Context) (int, error) {
	now := n.now()
	candidates, err := n.store.ListStillRunningNotificationCandidates(ctx, now, 100)
	if err != nil {
		return 0, err
	}

	enqueued := 0
	for _, candidate := range candidates {
		if err := n.enqueueCandidate(ctx, candidate, now); err != nil {
			if errors.Is(err, outbox.ErrDuplicate) {
				continue
			}
			return enqueued, fmt.Errorf("enqueue still running notification for %s: %w", candidate.TimeEntryID, err)
		}
		enqueued++
	}

	return enqueued, nil
}

func (n *StillRunningNotifier) enqueueCandidate(ctx context.Context, candidate store.StillRunningCandidate, now time.Time) error {
	_, err := n.outbox.Enqueue(ctx, outbox.EnqueueInput{
		UserID:      candidate.UserID,
		TimeEntryID: candidate.TimeEntryID,
		Kind:        outbox.KindTimerStillRunning,
		ToAddress:   candidate.UserEmail,
		Subject:     stillRunningSubject(candidate.Locale),
		BodyText:    stillRunningBody(candidate, n.publicBaseURL, now),
		MaxAttempts: n.maxAttempts,
	}, now)
	return err
}

func (n *StillRunningNotifier) HandleSent(ctx context.Context, email outbox.Email) error {
	if email.Kind != outbox.KindTimerStillRunning || email.TimeEntryID == "" {
		return nil
	}

	sentAt := time.Now().UTC()
	if err := n.store.MarkStillRunningEmailSent(ctx, email.TimeEntryID, sentAt); err != nil {
		if errors.Is(err, store.ErrTimeEntryNotFound) {
			log.Printf("still running email sent for closed timer %s", email.TimeEntryID)
			return nil
		}
		return err
	}
	return nil
}
