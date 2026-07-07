package outbox

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/leotime/leotime/apps/api/internal/mail"
)

type Processor struct {
	store       *Store
	sender      mail.Sender
	retryPolicy RetryPolicy
	sendTimeout time.Duration
	batchLimit  int
	rng         *rand.Rand
	now         func() time.Time
	onSent      func(ctx context.Context, email Email) error
}

type ProcessorOptions struct {
	RetryPolicy RetryPolicy
	SendTimeout time.Duration
	BatchLimit  int
	RNG         *rand.Rand
	Now         func() time.Time
	OnSent      func(ctx context.Context, email Email) error
}

type ProcessResult struct {
	Sent    int
	Retried int
	Dead    int
}

func NewProcessor(store *Store, sender mail.Sender, opts ProcessorOptions) *Processor {
	if opts.SendTimeout <= 0 {
		opts.SendTimeout = 30 * time.Second
	}
	if opts.BatchLimit <= 0 {
		opts.BatchLimit = 20
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.RNG == nil {
		opts.RNG = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	return &Processor{
		store:       store,
		sender:      sender,
		retryPolicy: opts.RetryPolicy,
		sendTimeout: opts.SendTimeout,
		batchLimit:  opts.BatchLimit,
		rng:         opts.RNG,
		now:         opts.Now,
		onSent:      opts.OnSent,
	}
}

func (p *Processor) ProcessOnce(ctx context.Context) (ProcessResult, error) {
	emails, err := p.store.ListDuePending(ctx, p.batchLimit, p.now())
	if err != nil {
		return ProcessResult{}, err
	}

	var result ProcessResult
	for _, email := range emails {
		outcome, err := p.processEmail(ctx, email)
		if err != nil {
			return result, err
		}
		switch outcome {
		case outcomeSent:
			result.Sent++
		case outcomeRetried:
			result.Retried++
		case outcomeDead:
			result.Dead++
		}
	}

	return result, nil
}

type processOutcome int

const (
	outcomeSent processOutcome = iota
	outcomeRetried
	outcomeDead
)

func (p *Processor) processEmail(ctx context.Context, email Email) (processOutcome, error) {
	sendCtx, cancel := context.WithTimeout(ctx, p.sendTimeout)
	defer cancel()

	err := p.sender.Send(sendCtx, mail.Message{
		To:      email.ToAddress,
		Subject: email.Subject,
		Body:    email.BodyText,
	})
	if err == nil {
		if err := p.store.MarkSent(ctx, email.ID, p.now()); err != nil {
			return 0, fmt.Errorf("mark sent %s: %w", email.ID, err)
		}
		if p.onSent != nil {
			if err := p.onSent(ctx, email); err != nil {
				return 0, fmt.Errorf("on sent hook %s: %w", email.ID, err)
			}
		}
		return outcomeSent, nil
	}

	attempts := email.Attempts + 1
	lastError := err.Error()
	now := p.now()

	if mail.IsPermanent(err) || attempts >= email.MaxAttempts {
		if err := p.store.MarkDead(ctx, email.ID, attempts, lastError, now); err != nil {
			return 0, fmt.Errorf("mark dead %s: %w", email.ID, err)
		}
		return outcomeDead, nil
	}

	nextRetryAt := NextRetryAt(p.retryPolicy, attempts, now, p.rng)
	if err := p.store.ScheduleRetry(ctx, email.ID, attempts, nextRetryAt, lastError); err != nil {
		return 0, fmt.Errorf("schedule retry %s: %w", email.ID, err)
	}
	return outcomeRetried, nil
}
