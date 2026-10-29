package judge

import (
	"context"
	"errors"
	"fmt"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// ErrHostPending is a host judgement asked out of band: the caller has
// written the host's request (the drain's request file) and the answer comes
// back in a later move. A Router passes it through, with its receipt.
var ErrHostPending = errors.New("judge: the host's answer is awaited")

// Host is the host's own judgement through the one interface (adr-25): Ask is
// shown the rendered prompt, the same text the adapter is sent, and returns
// the host's answer, which is parsed against the question's choices at
// probability 1. A caller whose host has already answered uses Answered.
type Host struct {
	Ask func(ctx context.Context, prompt string) (string, error)
}

// Answered is a Host whose answer is already in hand: the drain's host has
// written its answer file, or a page's host has handed its judgement back.
func Answered(text string) Host {
	return Host{Ask: func(context.Context, string) (string, error) { return text, nil }}
}

// Pending is a Host asked out of band: its Judge returns ErrHostPending.
func Pending() Host {
	return Host{Ask: func(context.Context, string) (string, error) { return "", ErrHostPending }}
}

// Judge asks the host and parses its answer at probability 1.
func (h Host) Judge(ctx context.Context, s State, q Question) (Answer, error) {
	if err := q.Check(); err != nil {
		return Answer{}, err
	}
	if h.Ask == nil {
		return Answer{}, ErrHostPending
	}
	text, err := h.Ask(ctx, render(s, q))
	if err != nil {
		return Answer{}, err
	}
	v, ok := q.match(text)
	if !ok {
		return Answer{}, fmt.Errorf("judge: the host's answer %q to %s is not one of %v", termsafe.Sanitize(clip(text)), q.Type, q.Choices())
	}
	return Answer{Value: v, Probability: 1}, nil
}
