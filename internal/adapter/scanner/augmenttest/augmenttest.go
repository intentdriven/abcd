// Package augmenttest is the test double for the scanner's Augmenter seam
// (iss-2608291814575788): a fake augmenter that flags one fixed value the
// native pattern set does not see, a not-found augmenter, and Install, which
// registers either as the default every scanner.New picks up. It spawns no
// process and needs no gitleaks binary, so a consumer's test proves the
// consumer reports what an augmenter found, and behaves on the not-found gap,
// without any external tool.
package augmenttest

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// Value is the value the fake flags. It is lowercase words joined by hyphens,
// which no native pattern matches, so a finding for it can only have come from
// the augmenter; it is not token-shaped.
const Value = "plumbob-harvest-quiet-lantern"

// Kind is the kind the fake reports.
const Kind = "fake:augmented"

// Func is an Augmenter built from a scan function: Scan calls F, and an error
// F returns is kept and reported by Available from then on, the way the
// gitleaks augmenter keeps a failed run. Err, set before use, is Available's
// answer from the start (ErrNotFound for the gap).
type Func struct {
	F   func(text, file string) ([]scanner.Finding, error)
	Err error

	mu    sync.Mutex
	calls int
}

// Available reports Err, or the error of the last failed Scan.
func (f *Func) Available() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Err
}

// Scan runs F.
func (f *Func) Scan(text, file string) []scanner.Finding {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.F == nil {
		return nil
	}
	out, err := f.F(text, file)
	if err != nil {
		f.mu.Lock()
		f.Err = err
		f.mu.Unlock()
		return nil
	}
	return out
}

// Calls is how many times Scan ran.
func (f *Func) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// Locate returns a finding for every occurrence of value in text, the way a
// real augmenter must (every occurrence, so a write path can verify it by its
// bytes).
func Locate(text, file, value, kind string) []scanner.Finding {
	var out []scanner.Finding
	for i, ln := range strings.Split(text, "\n") {
		for from := 0; ; {
			c := strings.Index(ln[from:], value)
			if c < 0 {
				break
			}
			out = append(out, scanner.Finding{File: file, Line: i + 1, Column: from + c + 1, Kind: kind,
				Severity: scanner.SeverityHardFail, Matched: value})
			from += c + len(value)
		}
	}
	return out
}

// Fake returns an augmenter that flags every occurrence of Value.
func Fake() *Func {
	return &Func{F: func(text, file string) ([]scanner.Finding, error) {
		return Locate(text, file, Value, Kind), nil
	}}
}

// NotFound returns an augmenter the repository configured whose tool is not
// installed: Available matches scanner.ErrAugmenterNotFound, and it never
// scans.
func NotFound() *Func {
	return &Func{Err: fmt.Errorf("%w: fake augmenter not on PATH", scanner.ErrAugmenterNotFound)}
}

// Install registers a as the default augmenter every scanner.New wires for
// the rest of the test. A test that installs one must not run in parallel with
// another that builds a scanner.
func Install(t testing.TB, a scanner.Augmenter) {
	t.Helper()
	restore := scanner.SetDefaultAugmenter(func(string) scanner.Augmenter { return a })
	t.Cleanup(restore)
}
