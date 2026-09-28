package cli

import (
	"strings"
	"testing"
)

// The bare memory board tells a person what the --json envelope tells a parser.
// The drift list — the index or the contradictions register no longer matching
// what the store's pages render — is the one line on the board that asks the
// reader to do something, and the text render used to drop it, so the only
// reader who could act on it was the one who asked for machine output
// (iss-2609091647582259).

// TestMemoryBoardPrintsItsDriftOnTheTextRender: a store whose index is stale
// says so on the human board, in the words the JSON carries, and names the verb
// that heals it.
func TestMemoryBoardPrintsItsDriftOnTheTextRender(t *testing.T) {
	memoryStoreFixture(t) // one page, no index.md: the index is stale

	st := memoryBoard(t)
	if len(st.Drift) == 0 {
		t.Fatalf("fixture drift: a store with no index.md reports no drift in --json: %+v", st)
	}
	text := string(runCLI(t, "memory"))
	for _, line := range st.Drift {
		if !strings.Contains(text, line) {
			t.Errorf("the text board omits the drift line the JSON carries, %q:\n%s", line, text)
		}
		if !strings.Contains(line, "abcd memory ingest") {
			t.Errorf("drift line %q does not name the verb that heals it", line)
		}
	}
}

// TestMemoryBoardIsQuietAboutDriftWhenTheStoreIsCurrent is the anti-vacuity
// control: after an ingest re-renders the index and the register, neither
// surface reports drift, so the text line above is a fact about the store and
// not a fixed banner.
func TestMemoryBoardIsQuietAboutDriftWhenTheStoreIsCurrent(t *testing.T) {
	repo, _ := memoryStoreFixture(t)
	src, pages := ingestOperands(t, repo)
	if out, err := runCLIErr(t, "memory", "ingest", src, "--pages-json", pages); err != nil {
		t.Fatalf("memory ingest: %v\n%s", err, out)
	}

	st := memoryBoard(t)
	if len(st.Drift) != 0 {
		t.Fatalf("a store an ingest just re-rendered reports drift in --json: %q", st.Drift)
	}
	if text := string(runCLI(t, "memory")); strings.Contains(text, "stale") {
		t.Errorf("a current store's text board reports staleness:\n%s", text)
	}
}
