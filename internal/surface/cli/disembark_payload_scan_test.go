package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lifeboat"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// TestDisembarkGraveyardRefusesAPayloadCarryingASecret is the end-to-end proof
// that a host payload is secret-scanned before it is written into the lifeboat,
// as pack's planned bytes are: the real scanner refuses a GitHub-token-shaped
// string in a lesson's prose, the verb exits 2 without echoing it, and the
// lifeboat is left as it was (the prior lessons.json is not cleared).
func TestDisembarkGraveyardRefusesAPayloadCarryingASecret(t *testing.T) {
	dir := buildGraveyardLifeboat(t)
	prior := lessonsPayloadFile(t, t.TempDir(),
		lifeboat.Lesson{ID: "les-prior", Lesson: "the prior lesson", Confidence: lifeboat.ConfidenceHigh,
			Evidence: []string{"rev-9f3a1c2d4e5b"}})
	runCLI(t, "disembark", "graveyard", dir, "--lessons-json", prior)
	lessonsPath := filepath.Join(dir, "graveyard", "lessons.json")
	before, err := os.ReadFile(lessonsPath)
	if err != nil {
		t.Fatalf("seed ingest wrote no lessons.json: %v", err)
	}

	// Built at run time, so no secret-shaped literal is committed.
	token := "ghp_" + testsecret.Synthetic(2610290907, 36)
	carrier := lessonsPayloadFile(t, t.TempDir(),
		lifeboat.Lesson{ID: "les-carrier", Lesson: "the deploy used " + token + " until it was rotated",
			Confidence: lifeboat.ConfidenceHigh, Evidence: []string{"adr-12"}})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"disembark", "graveyard", dir, "--lessons-json", carrier}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit = %d, want 2\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "hard-fail secret") {
		t.Errorf("the refusal must name the secret scan:\n%s", stderr.String())
	}
	if out := stdout.String() + stderr.String(); strings.Contains(out, token) || strings.Contains(out, token[4:16]) {
		t.Errorf("the refusal echoed the token:\n%s", out)
	}
	after, err := os.ReadFile(lessonsPath)
	if err != nil {
		t.Fatalf("the refusal removed the prior lessons.json: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("the refusal changed lessons.json:\nbefore: %s\nafter: %s", before, after)
	}
	if bytes.Contains(after, []byte(token)) {
		t.Error("the token reached the lifeboat")
	}
}
