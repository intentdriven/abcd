package landing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/ghcontract"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// R4-G1: every field PRStateQuery selects is a field of its type in the
// forge's schema as ghcontract pins it, so a renamed or mistyped field fails
// here rather than on the real forge.
func TestPRStateQuerySelectsOnlyTheForgesFields(t *testing.T) {
	if err := ghcontract.CheckGraphQL(PRStateQuery); err != nil {
		t.Fatalf("PRStateQuery: %v", err)
	}
}

// R4-G2: the forge client's stderr is scrubbed of credential URLs BEFORE it
// is cut to maxStderr, so no part of a token reaches the error.
func TestGHScrubsStderrBeforeCuttingIt(t *testing.T) {
	token := testsecret.Synthetic(6, 64)
	const into = 40
	lead := "https://x-access-token:"
	stderr := strings.Repeat("x", maxStderr-len(lead)-into) + " " + lead + token + "@example.com/o/r.git\n"
	bin := t.TempDir()
	msg := filepath.Join(t.TempDir(), "stderr")
	if err := os.WriteFile(msg, []byte(stderr), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\ncat '"+msg+"' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err := GH(t.TempDir(), "pr", "list")
	if err == nil {
		t.Fatal("the failing forge client did not fail the call")
	}
	if strings.Contains(err.Error(), token[:8]) {
		t.Fatalf("the error carries the token's start:\n%s", err)
	}
	if !strings.Contains(err.Error(), "https://example.com/o/r.git") || len(err.Error()) > maxStderr+256 {
		t.Fatalf("the error is %d bytes, want the scrubbed URL kept and stderr cut to the cap:\n%s", len(err.Error()), err)
	}
}

// R4-G3: an answer carrying a GraphQL errors array is refused, naming the
// forge's message sanitised, even when it has data and the client exits 0.
func TestAQueueStateAnswerWithErrorsIsRefused(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(f.head())
	f.forge.answer = `{"data":{"repository":{"pullRequest":` + viewQueued + `}},` +
		`"errors":[{"type":"FORBIDDEN","path":["repository","pullRequest","mergeQueueEntry"],"message":"Resource not accessible\u001b[31m by integration"}]}`
	res, err := f.land(f.options())
	wantRefusal(t, err, "Resource not accessible")
	wantRefusal(t, err, "by integration")
	if strings.Contains(err.Error(), "\x1b") {
		t.Fatalf("the refusal carries the forge's escape byte: %q", err)
	}
	if f.forge.called("pr", "merge") != 0 || res.PR != 7 {
		t.Fatalf("forge calls %v, result %+v: want nothing armed", f.forge.calls, res)
	}
}
