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
// that a host payload is secret-scanned before it is written into the lifeboat:
// the real scanner refuses a GitHub-token-shaped
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

// TestDisembarkSynthesisVerbsRefuseAPayloadCarryingASecret is the same proof
// for the other three verbs that write a host payload into the lifeboat, each
// through the real scanner: exit 2, the token neither echoed nor written
// anywhere in the lifeboat.
func TestDisembarkSynthesisVerbsRefuseAPayloadCarryingASecret(t *testing.T) {
	// Built at run time, so no secret-shaped literal is committed.
	token := "ghp_" + testsecret.Synthetic(2610291345, 36)
	cases := []struct {
		name string
		opts synthOpts
		args func(dir string) []string
		body string
	}{
		{"principles", synthOpts{}, func(dir string) []string {
			return []string{"disembark", "principles", dir, "--principles-json"}
		}, `{"schema_version":2,"mode":"delegated","prompt_version":"0.2.0",` +
			`"principles":[{"id":"prn-carrier","principle":"the deploy used ` + token + `","confidence":"high",` +
			`"claim_type":null,"reference":null,"comparison":null,"evidence":["adr-12"]}]}`},
		{"press-release", synthOpts{briefPR: true}, func(dir string) []string {
			return []string{"disembark", "press-release", dir, "--press-release-json"}
		}, `{"schema_version":1,"mode":"delegated","prompt_version":"0.1.0",` +
			`"headline":"H","body":"the deploy used ` + token + `","evidence":["brief/01-product/01-press-release.md"]}`},
		{"review", synthOpts{sourceName: "src", coverage: &lifeboat.Summary{Grounded: 7, Blank: 3}}, func(dir string) []string {
			return []string{"disembark", "review", dir, realSrcDir(t, "src"), "--review-json"}
		}, `{"schema_version":1,"mode":"delegated","prompt_version":"0.1.0","verdict":"SHIP",` +
			`"findings":[{"id":"fnd-carrier","finding":"the deploy used ` + token + `","evidence":["coverage.json"]}]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := buildSynthLifeboat(t, c.opts)
			payload := synthPayloadFile(t, c.body)
			var stdout, stderr bytes.Buffer
			code := Run(append(c.args(dir), payload), &stdout, &stderr)
			if code != 2 {
				t.Fatalf("exit = %d, want 2\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stderr.String(), "hard-fail secret") {
				t.Errorf("the refusal must name the secret scan:\n%s", stderr.String())
			}
			if out := stdout.String() + stderr.String(); strings.Contains(out, token) || strings.Contains(out, token[4:16]) {
				t.Errorf("the refusal echoed the token:\n%s", out)
			}
			err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				data, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				if bytes.Contains(data, []byte(token)) {
					t.Errorf("the token reached the lifeboat in %s", p)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// writeSourcePiiConfig gives a review's source repository a scanner config.
func writeSourcePiiConfig(t *testing.T, src, cfg string) {
	t.Helper()
	dir := filepath.Join(src, ".abcd", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pii.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
}

// reviewCarrying runs a delegated review whose one finding carries text, over a
// fresh lifeboat and the source src, and returns the lifeboat, the exit code and
// both streams.
func reviewCarrying(t *testing.T, src, text string) (dir string, code int, stdout, stderr string) {
	t.Helper()
	dir = buildSynthLifeboat(t, synthOpts{sourceName: "src", coverage: &lifeboat.Summary{Grounded: 7, Blank: 3}})
	payload := synthPayloadFile(t, `{"schema_version":1,"mode":"delegated","prompt_version":"0.1.0","verdict":"SHIP",`+
		`"findings":[{"id":"fnd-carrier","finding":"the deploy used `+text+`","evidence":["coverage.json"]}]}`)
	var out, errb bytes.Buffer
	code = Run([]string{"disembark", "review", dir, src, "--review-json", payload}, &out, &errb)
	return dir, code, out.String(), errb.String()
}

// assertReviewRefusedTheSecret checks a review refused for a hard-fail secret,
// echoed nothing of it and wrote no review into the lifeboat.
func assertReviewRefusedTheSecret(t *testing.T, dir string, code int, stdout, stderr, secret string) {
	t.Helper()
	if code != 2 {
		t.Fatalf("exit = %d, want 2\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "hard-fail secret") {
		t.Errorf("the refusal must name the secret scan:\n%s", stderr)
	}
	if strings.Contains(stdout+stderr, secret) {
		t.Errorf("the refusal echoed the secret:\n%s", stdout+stderr)
	}
	if _, err := os.Lstat(filepath.Join(dir, "review")); err == nil {
		t.Error("review/ was written by a refused review")
	}
}

// TestDisembarkReviewScansWithTheSourcesOwnPatterns: review is handed the
// source repository, so a pattern that repository added catches a secret
// abcd's built-in rules do not know.
func TestDisembarkReviewScansWithTheSourcesOwnPatterns(t *testing.T) {
	secret := "ACME-WIDGET-" + strings.Repeat("7", 6)
	src := realSrcDir(t, "src")
	writeSourcePiiConfig(t, src,
		`{"patterns":{"acme_secret":{"regex":"ACME-WIDGET-[0-9]{6}","kind":"token","label":"acme secret","severity":"hard_fail"}}}`)
	dir, code, stdout, stderr := reviewCarrying(t, src, secret)
	assertReviewRefusedTheSecret(t, dir, code, stdout, stderr, secret)
}

// TestDisembarkReviewSourceSettingsCannotHideABuiltInHit: the source's
// settings add to the built-in rules and remove none. A config that excludes,
// skips every path and extension of what review writes and redefines the
// built-in GitHub token pattern as a never-matching warning still has review
// refuse that token.
func TestDisembarkReviewSourceSettingsCannotHideABuiltInHit(t *testing.T) {
	// Built at run time, so no secret-shaped literal is committed.
	token := "ghp_" + testsecret.Synthetic(2610300212, 36)
	src := realSrcDir(t, "src")
	writeSourcePiiConfig(t, src, `{"skip_dirs":["review"],"skip_path_fragments":["review"],`+
		`"exclude_path_fragments":[{"fragment":"review","reason":"generated audit"}],`+
		`"skip_extensions":[".json",".md"],"skip_filenames":["audit.json","audit.md"],`+
		`"patterns":{"github_pat":{"regex":"a^","severity":"warn"}}}`)
	dir, code, stdout, stderr := reviewCarrying(t, src, token)
	assertReviewRefusedTheSecret(t, dir, code, stdout, stderr, token[4:16])
}

// TestDisembarkReviewRefusesAnUnreadableSourceConfig: a source scanner config
// that cannot be read refuses the review, as it refuses a pack.
func TestDisembarkReviewRefusesAnUnreadableSourceConfig(t *testing.T) {
	src := realSrcDir(t, "src")
	writeSourcePiiConfig(t, src, "{ this is not json")
	dir, code, _, stderr := reviewCarrying(t, src, "nothing secret")
	if code != 2 || !strings.Contains(stderr, "secret scanner unavailable") {
		t.Fatalf("exit = %d, want 2 naming the scanner\nstderr: %s", code, stderr)
	}
	if _, err := os.Lstat(filepath.Join(dir, "review")); err == nil {
		t.Error("review/ was written by a refused review")
	}
}
