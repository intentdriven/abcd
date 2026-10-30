package openaiapi

// logprobs.go reads the log-probabilities of an answer's first token, for a
// caller that asks for them (WithTopLogprobs): the decision adapter's second
// candidate, a small text model whose closed options are scored by the
// probability it gives each option's answer token (itd-2609221009495079).
//
// It is opt-in per call. A client built without the option sends no
// logprobs field and reads none, so every other call is exactly as before;
// the protocol's logprobs member is kept raw on every answer and decoded only
// for a caller that asked, so a provider's variation there never refuses an
// answer that did not ask for it.

import (
	"encoding/json"
	"math"
)

// MaxTopLogprobs bounds how many alternatives a call asks for at a token: the
// protocol's own ceiling.
const MaxTopLogprobs = 20

// maxLogprobToken bounds one alternative's token text, which is the
// provider's own text: an answer token is a few bytes.
const maxLogprobToken = 64

// TokenLogprob is one alternative at the answer's first token: the token as
// the provider spelled it, scrubbed and bounded, and its natural-log
// probability.
type TokenLogprob struct {
	// Token is the provider's own text, scrubbed of the key and at most
	// maxLogprobToken bytes, and NOT safe for a terminal. It is kept byte for
	// byte so a caller can compare it with its labels: termsafe.Sanitize
	// would turn a newline inside it into '?' and an invalid UTF-8 byte into
	// U+FFFD, and the comparison would no longer read what the model said. A
	// caller that prints it sanitises it where it prints, as the decision
	// adapter's error does (strconv.Quote, then the router's termsafe pass).
	Token   string
	Logprob float64
}

// LogprobSink receives the first answer token's alternatives once an answer
// is admitted: the alternatives that read (the top list, or the token chosen
// alone when the provider sent no list), and whether the answer carried any
// log-probabilities at all.
//
// It runs when the client admits the answer, before Complete returns, so a
// layer above the client can still refuse the call after it ran: oracle's
// Call discards an answer whose reported model its denylist refuses. The
// alternatives count only when the whole call returns no error. A sink keeps
// them for its caller to read after the call, and never acts on them itself.
type LogprobSink func(top []TokenLogprob, carried bool)

// WithTopLogprobs asks the call for the log-probabilities of its answer's
// tokens, n alternatives at each (bounded to 1 through MaxTopLogprobs), and
// hands the first token's to sink. A nil sink, or n below 1, asks nothing.
func WithTopLogprobs(n int, sink LogprobSink) Option {
	return func(c *Client) {
		if sink == nil || n < 1 {
			return
		}
		c.topLogprobs, c.logprobSink = min(n, MaxTopLogprobs), sink
	}
}

// logprobFields are the request fields a call that asks adds.
func (c *Client) logprobFields(fields map[string]any) {
	if c.logprobSink == nil {
		return
	}
	fields["logprobs"] = true
	fields["top_logprobs"] = c.topLogprobs
}

// choiceLogprobs is the protocol's logprobs member on a choice, as far as the
// first token's alternatives.
type choiceLogprobs struct {
	Content []struct {
		Token       string   `json:"token"`
		Logprob     *float64 `json:"logprob"`
		TopLogprobs []struct {
			Token   string   `json:"token"`
			Logprob *float64 `json:"logprob"`
		} `json:"top_logprobs"`
	} `json:"content"`
}

// readLogprobs reads a choice's raw logprobs member into a, for a client that
// asked: the first token's alternatives, once, from the first member that
// carries a token. A member that is absent, null or not of the protocol's
// shape reads as none; it never refuses the answer.
func (c *Client) readLogprobs(raw json.RawMessage, a *assembled) {
	if c.logprobSink == nil || a.logprobsRead || len(raw) == 0 || string(raw) == "null" {
		return
	}
	var lp choiceLogprobs
	if json.Unmarshal(raw, &lp) != nil || len(lp.Content) == 0 {
		return
	}
	first := lp.Content[0]
	a.logprobsRead = true
	for _, t := range first.TopLogprobs {
		if len(a.logprobs) == c.topLogprobs {
			break
		}
		c.keepLogprob(a, t.Token, t.Logprob)
	}
	if len(first.TopLogprobs) == 0 {
		c.keepLogprob(a, first.Token, first.Logprob)
	}
}

// keepLogprob keeps one alternative that reads: a finite log-probability no
// greater than zero, and token text bounded and scrubbed of the key.
func (c *Client) keepLogprob(a *assembled, token string, lp *float64) {
	if lp == nil || math.IsNaN(*lp) || math.IsInf(*lp, 0) || *lp > 0 || len(token) > maxLogprobToken {
		return
	}
	a.logprobs = append(a.logprobs, TokenLogprob{Token: c.scrub(token), Logprob: *lp})
}

// handLogprobs gives an admitted answer's alternatives to the sink that asked.
func (c *Client) handLogprobs(a *assembled) {
	if c.logprobSink == nil {
		return
	}
	c.logprobSink(append([]TokenLogprob(nil), a.logprobs...), a.logprobsRead)
}
