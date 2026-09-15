package reading

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// setMinter swaps the package's mint for the duration of one test.
func setMinter(t *testing.T, m recordid.Minter) {
	t.Helper()
	orig := minter
	minter = m
	t.Cleanup(func() { minter = orig })
}

// fixedMinter builds a mint with a pinned clock and pinned entropy, so a run
// identifier is an input a test can state rather than a wall-clock read.
func fixedMinter(stamp string, suffix uint16) recordid.Minter {
	at, err := time.Parse("0601021504", stamp)
	if err != nil {
		panic(err)
	}
	return recordid.Minter{
		Now:     func() time.Time { return at.UTC() },
		Entropy: repeatingReader([]byte{byte(suffix >> 8), byte(suffix)}),
	}
}

// repeatingReader feeds the same two entropy bytes for every draw, so a test
// may mint more than once under one pinned identifier.
func repeatingReader(b []byte) io.Reader {
	return &cycle{src: b}
}

type cycle struct {
	src []byte
	off int
}

func (c *cycle) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = c.src[c.off%len(c.src)]
		c.off++
	}
	return len(p), nil
}

// TestRunIDIsAdr45Native holds the run identifier's form: the family tag
// satisfies the mint's ^[a-z]+$ bound, and the mint reads no maximum, so two
// checkouts assembling in the same window cannot converge on one id.
func TestRunIDIsAdr45Native(t *testing.T) {
	setMinter(t, fixedMinter("2608301200", 789))
	id, err := mintRunID()
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	const want = "rdg-2608301200000789"
	if id != want {
		t.Errorf("run id is %q, want %q", id, want)
	}
	if !strings.HasPrefix(id, RunIDFamily+"-") {
		t.Errorf("run id %q does not carry the readings family tag", id)
	}
}

// TestRunIDReadsNoMaximum holds the reason the family is timestamp-minted: the
// mint consults no existing record, so a sibling worktree cannot be raced.
func TestRunIDReadsNoMaximum(t *testing.T) {
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC) },
		Entropy: bytes.NewReader([]byte{0x03, 0x15, 0x00, 0x2a}),
	})
	first, err := mintRunID()
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	second, err := mintRunID()
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if first == second {
		t.Error("two mints in one clock tick collided; the suffix is what separates them")
	}
}
