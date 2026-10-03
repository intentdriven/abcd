package openaiapi

// stream.go reads a streamed chat completion (iss-2610030931521214): the
// protocol's server-sent events, one chunk per data line and data: [DONE] at
// the end, assembled into the answer a single body would have carried, under
// the call's three limits.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"strings"
	"sync/atomic"
	"time"
)

// The causes a call's limits cancel it with, read back by callError.
var (
	errFirstByte = errors.New("first-byte limit")
	errIdle      = errors.New("idle limit")
	errTotal     = errors.New("total limit")
	// errStreamTooLarge ends a read past maxStreamBytes.
	errStreamTooLarge = errors.New("stream too large")
)

// limits is one call's bounds, applied through its request's context: the
// total cap as the context's deadline, and one timer that first waits for the
// answer's first byte and then, re-armed by every read that returns one,
// bounds the silence between reads. Whichever fires cancels the request, and
// net/http then closes the connection, so the server sees the client go.
type limits struct {
	parent context.Context
	ctx    context.Context
	cancel context.CancelCauseFunc
	total  context.CancelFunc
	timer  *time.Timer
	idle   time.Duration
	begun  atomic.Bool
}

func (c *Client) limit(parent context.Context) *limits {
	l := &limits{parent: parent, idle: c.idle}
	ctx, cancel := context.WithCancelCause(parent)
	l.ctx, l.total = context.WithTimeoutCause(ctx, c.timeout, errTotal)
	l.cancel = cancel
	l.timer = time.AfterFunc(c.firstByte, func() {
		if l.begun.Load() {
			cancel(errIdle)
			return
		}
		cancel(errFirstByte)
	})
	return l
}

// alive records that the server sent something, and re-arms the idle limit.
func (l *limits) alive() {
	l.begun.Store(true)
	l.timer.Reset(l.idle)
}

// stop releases the call's limits, cancelling its request if it is still open.
func (l *limits) stop() {
	l.timer.Stop()
	l.total()
	l.cancel(nil)
}

// liveBody is the answer's body, each read that returns a byte counted as the
// server being alive: a chunk, or a keep-alive comment between chunks.
type liveBody struct {
	r io.Reader
	l *limits
}

func (b *liveBody) Read(p []byte) (int, error) {
	n, err := b.r.Read(p)
	if n > 0 {
		b.l.alive()
	}
	return n, err
}

// capped refuses to read past its allowance.
type capped struct {
	r    io.Reader
	left int64
}

func (c *capped) Read(p []byte) (int, error) {
	if c.left <= 0 {
		return 0, errStreamTooLarge
	}
	if int64(len(p)) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= int64(n)
	return n, err
}

// eventStream reports whether a response's content type is an event stream.
// Anything else is read as one chat-completion body, which a server may answer
// a streaming request with.
func eventStream(contentType string) bool {
	mt, _, err := mime.ParseMediaType(contentType)
	return err == nil && mt == "text/event-stream"
}

// chunk is the part of a chat-completion chunk the adapter reads. A delta's
// other members (the role, a reasoning model's reasoning) are not the answer
// and are not read.
type chunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content *string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage          `json:"usage"`
	Error json.RawMessage `json:"error"`
}

// readStream assembles an event stream into the answer and admits it. Each
// data line is one event (the servers that speak the protocol send one chunk
// per line); a comment (a keep-alive), a blank line and any other field are
// read past. data: [DONE] ends the answer. A stream that closes before it
// still holds a complete answer when a chunk gave a finish reason; one with
// neither was cut off and is refused. One event, the assembled answer and the
// whole stream are each bounded.
func (c *Client) readStream(l *limits, body io.Reader, asked string, contract func([]byte) error) (Result, error) {
	sc := bufio.NewScanner(&capped{r: body, left: maxStreamBytes})
	sc.Buffer(make([]byte, 0, 64<<10), MaxResponseBytes)
	var (
		a      assembled
		chosen bool // a chunk carried the answer's choice
		done   bool
	)
	for !done && sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " ")
		if data == "[DONE]" {
			done = true
			continue
		}
		var ch chunk
		if err := json.Unmarshal([]byte(data), &ch); err != nil {
			// The decoder's message can quote the event; it is dropped.
			return Result{}, c.fail(c.host + " answered with a stream event that is not a chat completion chunk (not a JSON object of the protocol's shape)")
		}
		if isError(ch.Error) {
			return Result{}, c.reportedError([]byte(data))
		}
		switch {
		case ch.Model == "":
		case a.model == "":
			a.model = ch.Model
		case ch.Model != a.model:
			// The denylist judges the model reported; an answer from two
			// models cannot show which one answered.
			return Result{}, c.fail(c.host + " reported more than one model in one answer, so what answered cannot be shown; the answer is refused rather than used")
		}
		if ch.Usage != nil {
			a.usage = ch.Usage
		}
		for _, choice := range ch.Choices {
			if choice.Index != 0 {
				continue
			}
			chosen = true
			if p := choice.Delta.Content; p != nil {
				if a.content.Len()+len(*p) > MaxResponseBytes {
					return Result{}, c.fail(fmt.Sprintf("%s streamed an answer larger than %d bytes, so it is refused", c.host, MaxResponseBytes))
				}
				a.content.WriteString(*p)
			}
			if p := choice.FinishReason; p != nil && *p != "" {
				a.finish = *p
			}
		}
	}
	if err := sc.Err(); err != nil {
		switch {
		case errors.Is(err, bufio.ErrTooLong):
			return Result{}, c.fail(fmt.Sprintf("%s streamed one event larger than %d bytes, so it is refused unread", c.host, MaxResponseBytes))
		case errors.Is(err, errStreamTooLarge):
			return Result{}, c.fail(fmt.Sprintf("%s answered with a stream larger than %d bytes, so it is refused", c.host, maxStreamBytes))
		}
		return Result{}, c.callError(l, err)
	}
	if !done && a.finish == "" {
		return Result{}, c.fail(c.host + "'s stream ended before the answer was complete (no finish reason and no [DONE]), so it is refused rather than used")
	}
	if !chosen {
		return Result{}, c.noChoice()
	}
	return c.admit(&a, asked, contract)
}
