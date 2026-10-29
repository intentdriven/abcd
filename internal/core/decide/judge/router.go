package judge

// router.go is the per-type switch (spc-2609221011151661 scopes 3, 5 and 6).
// Each judgement type is routed as oracle.judgements.<type> says, read and
// checked against the provider's allowlist when the configuration is loaded,
// before any call (adr-2609221009491186, criterion 4):
//
//   - host, or nothing configured: the host decides, and nothing else runs.
//   - shadow:<provider>/<model>: the host decides; the adapter answers beside
//     it and decides nothing; the pair is recorded in the newest lab home
//     (criterion 1). A state larger than the model's window is not sent: the
//     pair is recorded as skipped with its size (criterion 5).
//   - <provider>/<model>: the adapter decides and the receipt names the
//     route; an adapter that cannot answer — unreachable, failed, an answer
//     outside the question's choices, a state over its window — leaves the
//     host to decide, with a receipt saying why (criterion 3).
//
// The caller hands Decide its host judgement and nothing else changes when a
// type is switched (criterion 6).

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// DefaultWindow is the decision model's context, in tokens: the first
// candidate's 32K (itd-2609221009495079 scope condition 1).
const DefaultWindow = 32 << 10

// The routes a receipt names.
const (
	RouteHost    = "host"
	RouteAdapter = "adapter"
)

// maxDetail bounds an error's account in a pair or a receipt.
const maxDetail = 300

// Receipt is what a judgement's run record carries: the type and its mode, the
// route that decided, the adapter's target and the model it reported, the
// probability the deciding side gave, the state's hash and size, why a type
// turned on fell back to the host, and where a shadow pair went or why it was
// not kept.
type Receipt struct {
	Type          Type    `json:"type"`
	Mode          string  `json:"mode"`
	Route         string  `json:"route"`
	Target        string  `json:"target,omitempty"`
	ModelReported string  `json:"model_reported,omitempty"`
	Probability   float64 `json:"probability"`
	StateSHA256   string  `json:"state_sha256"`
	StateTokens   int     `json:"state_tokens"`
	Fallback      string  `json:"fallback,omitempty"`
	Pair          string  `json:"pair,omitempty"`
	PairSkipped   string  `json:"pair_skipped,omitempty"`
	Note          string  `json:"note,omitempty"`
}

// Recorder keeps a shadow pair and returns the lab and the file it went to.
type Recorder func(lab.Pair) (labID, where string, err error)

// Router routes each judgement type to the host, to the host with the adapter
// in shadow, or to the adapter.
type Router struct {
	routes  map[Type]oracle.Target
	adapter func(oracle.Target) Judge
	record  Recorder
	// Window is the adapter's window in tokens; DefaultWindow when zero.
	Window int
	// Diagnostics are the configuration's non-fatal reports, for a front
	// door to print: the oracle reader's, and a route naming a type outside
	// the registry.
	Diagnostics []string
}

// newRouter is a Router over routes (a type absent from it stays on the
// host), the adapter each target is judged by, and where shadow pairs are
// recorded.
func newRouter(routes map[Type]oracle.Target, adapter func(oracle.Target) Judge, record Recorder) *Router {
	r := &Router{routes: map[Type]oracle.Target{}, adapter: adapter, record: record}
	for t, tgt := range routes {
		r.routes[t] = tgt
	}
	return r
}

// NewForTest is newRouter, the seam a caller's tests route a type with
// through a fake adapter and recorder.
func NewForTest(routes map[Type]oracle.Target, adapter func(oracle.Target) Judge, record Recorder) *Router {
	return newRouter(routes, adapter, record)
}

// Load reads oracle.judgements.<type> through the oracle reader, which refuses
// a model not on its provider's allowlist (or one oracle.denylist matches)
// before any call, and returns the Router the repository at repoRoot judges
// with: the adapter is the API adapter with the key resolved by name from the
// account's credential store, and shadow pairs go to the newest lab minted for
// the repository. A route to a type outside the registry is a diagnostic.
func Load(roots layered.Roots, repoRoot string) (*Router, error) {
	cfg, err := oracle.LoadAPI(roots)
	if err != nil {
		return nil, err
	}
	routes := map[Type]oracle.Target{}
	var diags []string
	diags = append(diags, cfg.Diagnostics...)
	for _, pr := range cfg.Routes() {
		if pr.Kind != "judgement" {
			continue
		}
		if !known(pr.Name) {
			diags = append(diags, fmt.Sprintf("decision adapter: oracle.judgements.%s in %s is not a judgement type (%s); it is skipped",
				termsafe.Sanitize(clip(pr.Name)), termsafe.Sanitize(pr.Target.Origin), typeNames()))
			continue
		}
		routes[Type(pr.Name)] = pr.Target
	}
	creds := credential.Store(roots.Home)
	r := newRouter(routes, func(t oracle.Target) Judge { return Adapter{Calls: cfg, Creds: creds, Target: t} },
		func(p lab.Pair) (string, string, error) { return lab.RecordPair(repoRoot, p) })
	r.Diagnostics = diags
	return r, nil
}

// Mode is how t is routed: oracle.JudgementHost, JudgementShadow or
// JudgementOn.
func (r *Router) Mode(t Type) string {
	if tgt, ok := r.routes[t]; ok && (tgt.Mode == oracle.JudgementShadow || tgt.Mode == oracle.JudgementOn) {
		return tgt.Mode
	}
	return oracle.JudgementHost
}

func (r *Router) window() int {
	if r.Window > 0 {
		return r.Window
	}
	return DefaultWindow
}

// Decide answers q over s on the route t's configuration names, with host as
// the host's judgement. Its error is the host's (ErrHostPending among them)
// or the question's refusal; an adapter's failure is never an error: in shadow
// it is recorded in the pair, and turned on it is a fallback to host.
func (r *Router) Decide(ctx context.Context, s State, q Question, host Judge) (Answer, Receipt, error) {
	if err := q.Check(); err != nil {
		return Answer{}, Receipt{}, err
	}
	mode := r.Mode(q.Type)
	size := tokens(render(s, q))
	rc := Receipt{Type: q.Type, Mode: mode, Route: RouteHost, StateSHA256: s.SHA256(), StateTokens: size}
	tgt := r.routes[q.Type]
	if mode != oracle.JudgementHost {
		rc.Target = tgt.String()
	}
	switch mode {
	case oracle.JudgementOn:
		if size > r.window() {
			rc.Fallback = fmt.Sprintf("the state is about %d tokens, more than %s's window of %d, so it was not sent and the host decided",
				size, rc.Target, r.window())
			return r.byHost(ctx, s, q, host, rc)
		}
		a, err := r.ask(ctx, tgt, s, q)
		if err != nil {
			rc.Fallback = fmt.Sprintf("%s did not answer (%s), so the host decided", rc.Target, detail(err))
			return r.byHost(ctx, s, q, host, rc)
		}
		rc.Route, rc.ModelReported, rc.Probability = RouteAdapter, a.Model, a.Probability
		return a, rc, nil
	case oracle.JudgementShadow:
		a, rc, err := r.byHost(ctx, s, q, host, rc)
		if err != nil {
			return a, rc, err
		}
		pair := lab.Pair{Type: string(q.Type), Kind: string(q.Kind), StateSHA256: rc.StateSHA256, StateTokens: size,
			Host: a.Value, Target: rc.Target}
		if size > r.window() {
			pair.Skipped = lab.SkippedWindow
			pair.Detail = fmt.Sprintf("the state is about %d tokens, more than the window of %d; the host decided alone", size, r.window())
		} else if sa, err := r.ask(ctx, tgt, s, q); err != nil {
			pair.Skipped, pair.Detail = lab.SkippedFailed, detail(err)
		} else {
			p := sa.Probability
			pair.Adapter, pair.Probability, pair.ModelReported = sa.Value, &p, sa.Model
		}
		rc.PairSkipped = pair.Skipped
		if r.record == nil {
			rc.Note = "the shadow pair was not kept: no recorder"
			return a, rc, nil
		}
		if _, where, err := r.record(pair); err != nil {
			rc.Note = "the shadow pair was not kept: " + detail(err)
		} else {
			rc.Pair = where
		}
		return a, rc, nil
	default:
		return r.byHost(ctx, s, q, host, rc)
	}
}

// byHost is the host's decision, on rc.
func (r *Router) byHost(ctx context.Context, s State, q Question, host Judge, rc Receipt) (Answer, Receipt, error) {
	if host == nil {
		return Answer{}, rc, ErrHostPending
	}
	a, err := host.Judge(ctx, s, q)
	if err != nil {
		return Answer{}, rc, err
	}
	rc.Route, rc.Probability = RouteHost, a.Probability
	return a, rc, nil
}

// ask is the adapter's answer, held to the question: a value among its
// choices and a probability from 0 to 1, whatever implementation answered.
func (r *Router) ask(ctx context.Context, tgt oracle.Target, s State, q Question) (Answer, error) {
	if r.adapter == nil {
		return Answer{}, errors.New("no adapter")
	}
	j := r.adapter(tgt)
	if j == nil {
		return Answer{}, errors.New("no adapter")
	}
	a, err := j.Judge(ctx, s, q)
	if err != nil {
		return Answer{}, err
	}
	v, ok := q.match(a.Value)
	if !ok {
		return Answer{}, fmt.Errorf("its answer %q is not one of %s", termsafe.Sanitize(clip(a.Value)), strings.Join(q.Choices(), ", "))
	}
	if math.IsNaN(a.Probability) || a.Probability < 0 || a.Probability > 1 {
		return Answer{}, fmt.Errorf("its probability %v is not from 0 to 1", a.Probability)
	}
	a.Value = v
	return a, nil
}

// detail is an error's account for a pair or a receipt: terminal-safe, one
// line, bounded.
func detail(err error) string {
	s := strings.Join(strings.Fields(termsafe.Sanitize(err.Error())), " ")
	if len(s) > maxDetail {
		s = s[:maxDetail] + "…"
	}
	return s
}
