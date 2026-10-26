package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The review set at the surface (itd-2610240922461281, spc-2610240951109420
// step 1): `abcd build` names the run's review set beside its pace, each class
// with the model it holds, and says a clamp on stderr.

// TestBuildNamesTheBundledReviewSet: with no configuration the payload's
// review, a sibling of pace, carries two reviewers per role from the bundled
// layer, and the text and the status name each class with its model.
func TestBuildNamesTheBundledReviewSet(t *testing.T) {
	buildRepo(t)
	out := mustImplement(t, "build", "itd-10", "--json")
	var res struct {
		Review *struct {
			Roles []struct {
				Role      string
				Layer     string
				Reviewers []struct{ Class, Model string }
			}
		}
		Diagnostics any
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if res.Review == nil || len(res.Review.Roles) != 2 || res.Diagnostics != nil {
		t.Fatalf("the payload carries the review set and no diagnostics: %s", out)
	}
	for _, r := range res.Review.Roles {
		if len(r.Reviewers) != 2 || r.Reviewers[0].Class != "middle" || r.Reviewers[1].Class != "below" || r.Layer != "bundled" {
			t.Fatalf("%s runs middle and below from the bundled layer: %s", r.Role, out)
		}
	}
	status := mustImplement(t, "implement", "status")
	if !strings.Contains(status, "review:  ruthless-reviewer middle (anthropic/claude-opus-5-5) and below (anthropic/claude-sonnet-5-5) (bundled)") {
		t.Fatalf("the status names each class with its model:\n%s", status)
	}
}

// TestBuildSaysAClampedReviewSetOnStderr: a repository list longer than the
// machine's ceiling is clamped with one line on stderr naming both files.
func TestBuildSaysAClampedReviewSetOnStderr(t *testing.T) {
	repo := buildRepo(t)
	home := os.Getenv("HOME")
	if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abcdhome.Path(home, "config.json"), []byte(`{"review": {"max_reviewers": 1}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	repo.Write(".abcd/config.json", `{"review": {"reviewers": ["middle", "below"]}}`)
	repo.Commit("review set")
	code, out, errOut := implementCLI(t, "build", "itd-10")
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errOut)
	}
	var lines []string
	for _, l := range strings.Split(errOut, "\n") {
		if strings.Contains(l, "review.max_reviewers") {
			lines = append(lines, l)
		}
	}
	if len(lines) != 1 || !strings.Contains(lines[0], ".abcd/config.json") || !strings.Contains(lines[0], abcdhome.Display("config.json")) {
		t.Fatalf("one stderr line names both files:\n%s", errOut)
	}
	if !strings.Contains(out, "clamped from 2") {
		t.Fatalf("the text names the clamp:\n%s", out)
	}
}
