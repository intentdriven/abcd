package gitleaks

import (
	"context"
	"sync"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// Augmenter is the gitleaks adapter in the scanner's Augmenter shape, so every
// consumer that builds a scanner inherits the repository's opt-in rather than
// the transcript store alone (iss-2608291814575788). The scanner declares the
// interface and never imports this package; the composition root (cmd/abcd)
// registers NewAugmenter with scanner.SetDefaultAugmenter.
//
// Available is the binary's state, or the last run's: a run that fails, a
// report that cannot be parsed, or a finding gitleaks placed nowhere in the
// text (ErrFindingNotLocated) is kept and returned from then on, which the
// scanner reads as a degrade. A binary the repository configured and nobody
// installed is ErrConfiguredNotFound, which matches scanner.ErrAugmenterNotFound
// and is carried as a coverage gap instead.
type Augmenter struct {
	adapter  *Adapter
	repoRoot string
	cfg      Config

	mu  sync.Mutex
	err error
}

// NewAugmenter is the production factory: the default adapter over the
// repository's own .abcd/config/gitleaks.json. It returns nil when the
// repository did not opt in, so a repository that did nothing pays nothing.
func NewAugmenter(repoRoot string) scanner.Augmenter {
	return NewDefault().AugmenterFor(repoRoot)
}

// AugmenterFor builds the augmenter for repoRoot from this adapter's lookup and
// runner. An absent config is nil (not opted in); a present but broken config
// is an augmenter that is never available, so the repository that tried to arm
// gitleaks and got it wrong is told rather than silently left on the native
// scanner. The binary is resolved once here, to answer Available, and again
// on every run, so what is executed is judged when it is executed.
func (a *Adapter) AugmenterFor(repoRoot string) scanner.Augmenter {
	cfg, err := LoadConfig(repoRoot)
	if err != nil {
		return &Augmenter{err: err}
	}
	if !cfg.Enabled {
		return nil
	}
	g := &Augmenter{adapter: a, repoRoot: repoRoot, cfg: cfg}
	if _, err := a.resolveBinary(repoRoot, cfg); err != nil {
		g.err = err
	}
	return g
}

// Available reports whether the augmenter can run: nil, or the error that
// stops it.
func (g *Augmenter) Available() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}

// Scan runs gitleaks over text and returns its findings located in text. A
// failure returns nothing and is kept for Available to report.
func (g *Augmenter) Scan(text, file string) []scanner.Finding {
	if g.Available() != nil {
		return nil
	}
	fs, err := g.adapter.Augment(context.Background(), g.repoRoot, g.cfg, text, file)
	if err != nil {
		g.mu.Lock()
		if g.err == nil {
			g.err = err
		}
		g.mu.Unlock()
		return nil
	}
	return fs
}
