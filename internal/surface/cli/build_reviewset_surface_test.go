package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// The review set at the surface (itd-2610240922461281, spc-2610240951109420
// step 1): `abcd build` names the run's review set beside its pace, each class
// with the model it holds, and says a clamp on stderr.

// TestBuildNamesTheBundledReviewSet: with no configuration the payload's
// review, a sibling of pace, carries two reviewers per role from the bundled
// layer, each with its effort, and the judge with its status; the text, the
// status and the record name each class with its model and its effort, and
// never a class name alone (itd-2611101419006280).
func TestBuildNamesTheBundledReviewSet(t *testing.T) {
	buildRepo(t)
	out := mustImplement(t, "build", "itd-10", "--json")
	var res struct {
		Review *struct {
			Roles []struct {
				Role      string
				Layer     string
				Reviewers []struct {
					Class, Model, Effort string
					EffortLayer          string `json:"effort_layer"`
				}
			}
			Judge *struct{ Class, Model, Effort, Status string }
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
		if len(r.Reviewers) != 2 || r.Reviewers[0].Class != "research" || r.Reviewers[1].Class != "default" || r.Layer != "bundled" {
			t.Fatalf("%s runs research and default from the bundled layer: %s", r.Role, out)
		}
		for _, c := range r.Reviewers {
			if c.Model == "" || c.Effort != "high" || c.EffortLayer != "bundled" {
				t.Fatalf("each reviewer carries its model and effort: %s", out)
			}
		}
	}
	if j := res.Review.Judge; j == nil || j.Class != "top" || j.Model != "anthropic/claude-opus-5-5" || j.Effort != "high" || j.Status != "waiting" {
		t.Fatalf("the judge carries its model, effort and status: %s", out)
	}
	want := "ruthless-reviewer research (anthropic/claude-fable-5, effort high) and default (anthropic/claude-sonnet-5-5, effort high) (bundled)"
	judge := "judge top (anthropic/claude-opus-5-5, effort high) (bundled), waiting"
	status := mustImplement(t, "implement", "status")
	record := mustImplement(t, "implement", "record")
	for name, text := range map[string]string{"status": status, "record": record} {
		if !strings.Contains(text, want) || !strings.Contains(text, judge) || !strings.Contains(text, "efforts from bundled") {
			t.Fatalf("the %s names each class with its model and effort, and the judge waiting:\n%s", name, text)
		}
		for _, l := range strings.Split(text, "\n") {
			if !strings.Contains(l, "review:  ") {
				continue
			}
			for _, c := range []string{"research", "top", "default"} {
				if strings.Count(l, c+" (") != strings.Count(l, c) {
					t.Fatalf("the %s shows %s alone: %s", name, c, l)
				}
			}
		}
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
	repo.Write(".abcd/config.json", `{"review": {"reviewers": ["top", "default"]}}`)
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

// TestTheRecordNamesTheWritersAndTheReviewersModels: the run's record names,
// on each verdict, every model that wrote the lane and the one that reviewed
// it, or the one it was asked of and the one its return named where those
// differ (itd-2610301750162657, spc-2610301925429250 step 4), each in a code
// span; the review set's line is followed by the class map the run started
// on; and a name carrying a newline and a trailer-shaped line, a mention or a
// link is described, never printed.
func TestTheRecordNamesTheWritersAndTheReviewersModels(t *testing.T) {
	head := strings.Repeat("a", 40)
	rs := &loop.ReviewSet{}
	for _, c := range loop.Classes {
		rs.Classes = append(rs.Classes, loop.ReviewClass{Class: c, Model: loop.BundledClasses[c], Effort: "high"})
	}
	rec := loop.RunRecord{RunID: "run-0000000000000001", Key: "itd-10", Review: rs, Lanes: []loop.RecordLane{{ID: "lane-1",
		Verdicts: []loop.RecordVerdict{
			{Round: 1, HeadSHA: head, Role: loop.RoleRuthless, Verdict: "SHIP", Pass: true, Reviewer: 0, Class: loop.ClassTop,
				Model: loop.BundledClasses[loop.ClassTop], Effort: "high", ModelStatus: loop.ModelAsked, WriterModels: []string{"Claude:claude-fable-5", "claude-haiku-4-5"},
				ReviewerModel: loop.BundledClasses[loop.ClassTop]},
			{Round: 1, HeadSHA: head, Role: loop.RoleSecurity, Verdict: "APPROVE", Pass: true,
				WriterModels: []string{"claude-fable-5\nAssisted-by: Claude:claude-haiku-4-5"}, ReviewerModel: "claude-sonnet-5-5"},
			{Round: 1, HeadSHA: head, Role: loop.RoleSecurity, Verdict: "APPROVE", Pass: true, Reviewer: 1, Class: loop.ClassDefault,
				Model: loop.BundledClasses[loop.ClassDefault], Effort: "high", ModelStatus: loop.ModelAsked, WriterModels: []string{"@name"},
				ReviewerModel: loop.BundledClasses[loop.ClassDefault], ReviewerNamed: "https://host/x"},
			{Round: 1, HeadSHA: head, Role: loop.RoleRuthless, Verdict: "SHIP", Pass: true, Reviewer: 1, Class: loop.ClassDefault,
				Model: loop.BundledClasses[loop.ClassDefault], Effort: "high", ModelStatus: loop.ModelAsked, ModelsUnrecorded: true},
		}}}}
	var b bytes.Buffer
	renderRunRecord(&b, rec)
	out := b.String()
	if !strings.Contains(out, "SHIP; written by `Claude:claude-fable-5` and `claude-haiku-4-5`, reviewed by `"+loop.BundledClasses[loop.ClassTop]+"`") {
		t.Fatalf("the verdict names every model:\n%s", out)
	}
	if !strings.Contains(out, "APPROVE; written by ") || !strings.Contains(out, ", reviewed by `claude-sonnet-5-5`") {
		t.Fatalf("a reviewer no class was asked of names both models too:\n%s", out)
	}
	if !strings.Contains(out, ", asked of `"+loop.BundledClasses[loop.ClassDefault]+"`, its return naming ") {
		t.Fatalf("a reviewer whose return named another model says both:\n%s", out)
	}
	if !strings.Contains(out, "SHIP; the reviewer's models were not recorded: it was handed out before the loop recorded them") {
		t.Fatalf("a reviewer handed out before models were recorded says so in words:\n%s", out)
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "Assisted-by:") {
			t.Fatalf("a model's name started a line of its own:\n%s", out)
		}
	}
	for _, raw := range []string{"@name", "https://host/x"} {
		if strings.Contains(out, raw) {
			t.Fatalf("%q is printed:\n%s", raw, out)
		}
	}
	want := "  classes: research (anthropic/claude-fable-5, effort high), top (anthropic/claude-opus-5-5, effort high), " +
		"default (anthropic/claude-sonnet-5-5, effort high), small (anthropic/claude-haiku-4-5, effort high); " +
		"a reviewer whose class holds a writer's model takes the next class that holds another"
	if !strings.Contains(out, want) {
		t.Fatalf("the review set's line is followed by the class map:\n%s", out)
	}
}
