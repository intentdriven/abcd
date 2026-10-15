package memnote

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/memory"
)

// gate.go — the one path into the machine layer judges every note before
// anything is written (spc-2610111627213138, "The write gate").
//
//  1. The shared scanner, through the memory-page writer's own redactor
//     (memory.StoreRedactor): a degraded scanner refuses, then each field is
//     scanned and any span the page writer would have to rewrite refuses —
//     every blocking finding and the literal-home backstop. The gate never
//     redacts and writes: a lesson with a span cut out of it may say something
//     wrong, so the agent rewrites the note in its own words.
//  2. The private-names banlist, through banlist.MatchPrivate, against this
//     checkout's store and, in a linked worktree, the primary checkout's
//     (banlist.PrimaryWorktreeRoot), the layer the committed guard enforces
//     there. A store that cannot be read is a match, as MatchPrivate's
//     contract requires.
//
// The stamps written_by and session come from the caller and are judged like
// any other field; written_at is the clock's and is not the caller's to judge.

// GateError is the gate's refusal of one field: the field and the kind of span
// it carries, never the span.
type GateError struct {
	Field string
	Kind  string
}

func (e *GateError) Error() string {
	return fmt.Sprintf("the note's %s carries %s, which a note never holds; rewrite the note in your own words", e.Field, e.Kind)
}

// Seams the tests replace; production reads the shared primitives.
var (
	openRedactor = memory.OpenStoreRedactor
	matchPrivate = banlist.MatchPrivate
	primaryRoot  = banlist.PrimaryWorktreeRoot
)

// gateField is one judged field of a note: its name in a refusal and its text.
type gateField struct {
	name string
	text string
}

// gateFields lists every field of n the caller supplied.
func gateFields(n Note) []gateField {
	fs := []gateField{{"title", n.Title}, {"body", n.Body}}
	for i, w := range n.Recall {
		fs = append(fs, gateField{fmt.Sprintf("recall[%d]", i), w})
	}
	fs = append(fs, gateField{"written_by", n.WrittenBy})
	if n.Session != nil {
		fs = append(fs, gateField{"session", *n.Session})
	}
	return fs
}

// judge runs the gate over n for a write from repoRoot. nil admits it.
func judge(repoRoot string, n Note) error {
	red, err := openRedactor(repoRoot)
	if err != nil {
		return fmt.Errorf("the note is not judged, so it is not written: %v", err)
	}
	fields := gateFields(n)
	for _, f := range fields {
		kinds, err := red.BlockingKinds(f.text, f.name)
		if err != nil {
			return fmt.Errorf("the note is not judged, so it is not written: %v", err)
		}
		if len(kinds) > 0 {
			return &GateError{Field: f.name, Kind: "a span the shared scanner blocks [" + strings.Join(kinds, ", ") + "]"}
		}
	}
	roots := []string{repoRoot}
	if p, ok := primaryRoot(repoRoot); ok {
		roots = append(roots, p)
	}
	var lines []string
	var owner []string
	for _, f := range fields {
		for _, l := range strings.Split(f.text, "\n") {
			lines = append(lines, l)
			owner = append(owner, f.name)
		}
	}
	for _, root := range roots {
		match, err := matchPrivate(root, lines)
		if err != nil {
			// An unreadable store, a pattern the engine refuses, or no engine
			// at all: every name is treated as matched.
			return fmt.Errorf("the private-names banlist could not judge the note, which counts as a match, so it is not written: %v", err)
		}
		for i, m := range match {
			if m {
				return &GateError{Field: owner[i], Kind: "a name the private-names banlist holds"}
			}
		}
	}
	return nil
}
