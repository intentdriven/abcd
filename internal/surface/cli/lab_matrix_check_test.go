package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// matrixCheckRun records a consented run of one check task over the
// throwaway repository's README, built by the host cell alone with extra
// declaration fields after its builders, and hands back each host copy with
// the class of its one finding ("" hands the copy back failed). It returns the
// home, the lab and the run.
func matrixCheckRun(t *testing.T, extra string, classes ...string) (home, id, run string) {
	t.Helper()
	home, id, _ = matrixLabCLI(t, "builder=runner:opencode@openrouter/vendor/model", true)
	decl := filepath.Join(t.TempDir(), "matrix.json")
	body := `{"schema_version":1,"name":"check judging","tasks":[{"id":"check-a","kind":"check","material":["README.md"]}],` +
		`"builders":[{"id":"host-big","route":"host"}]` + extra + `,` +
		`"repeats":{"pilot":3,"max":10},"concurrency":2,"judging":{"guests_may_judge":false}}`
	if err := os.WriteFile(decl, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, out, _ := runLab(t, "lab", "--json", "matrix", id, decl)
	var plan struct {
		Digest string `json:"digest"`
		Cells  []struct {
			Cell string `json:"cell"`
			Role string `json:"role"`
		} `json:"cells"`
	}
	if err := json.Unmarshal([]byte(out), &plan); err != nil || plan.Digest == "" {
		t.Fatalf("the plan: %s (%v)", out, err)
	}
	code, out, errOut := runLab(t, "lab", "--json", "matrix", id, decl, "--approve", plan.Digest)
	var started struct {
		RunID  string          `json:"run_id"`
		Copies []matrixHandOff `json:"copies"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &started) != nil || len(started.Copies) != 3 {
		t.Fatalf("approve (exit %d): %s%s", code, out, errOut)
	}
	now := time.Now().UTC().Truncate(time.Second)
	transcript := filepath.Join(t.TempDir(), "transcript.txt")
	if err := os.WriteFile(transcript, []byte("read README.md\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for i, h := range started.Copies {
		args := []string{"lab", "matrix", "copy", id, started.RunID, h.key(), "--transcript", transcript, "--form", "consistency"}
		if class := classes[i]; class != "" {
			answer := writeJSONFile(t, "answer.json", map[string]any{"schema_version": 1, "findings": []any{
				map[string]any{"class": class, "summary": "the README says root twice",
					"ends": []any{map[string]string{"path": "README.md", "quote": "root"}, map[string]string{"path": "README.md", "quote": "root"}}}}})
			receipt := writeJSONFile(t, "receipt.json", map[string]any{"schema_version": 1, "outcome": "made", "started_at": now, "finished_at": now.Add(time.Minute)})
			args = append(args, "--receipt", receipt, "--answer", answer)
		} else {
			args = append(args, "--receipt", writeJSONFile(t, "receipt.json", map[string]any{"schema_version": 1, "outcome": "failed", "reason": "gave up"}))
		}
		if code, out, errOut := runLab(t, args...); code != 0 {
			t.Fatalf("handing back %s (exit %d): %s%s", h.key(), code, out, errOut)
		}
	}
	return home, id, started.RunID
}

// checkJudged is a check's judging as `lab matrix judge --json` prints it.
type checkJudged struct {
	Stage     string `json:"stage"`
	Copies    int    `json:"copies"`
	Leftovers int    `json:"leftovers"`
	Matcher   *struct {
		Cell           string `json:"cell"`
		AwaitingRunner bool   `json:"awaiting_runner"`
	} `json:"matcher"`
	MatcherRequest *struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	} `json:"matcher_request"`
	Packet *struct {
		Items []struct {
			Code string `json:"code"`
		} `json:"items"`
	} `json:"packet"`
	PacketDigest string `json:"packet_digest"`
}

// A check task is judged from the command line in two phases: the packet is
// drawn with no model named and no verdict proposed, the findings no exact key
// joins stay rows of their own with no matcher declared, the person's verdicts
// lock once, the table is refused until they do, and every refusal reaches the
// terminal through the sanitiser (spec step 9).
func TestACheckIsJudgedThroughTheCLIInTwoPhases(t *testing.T) {
	home, id, run := matrixCheckRun(t, "", "contradiction", "contradiction", "drift")
	if code, _, errOut := runLab(t, "lab", "matrix", "judge", id, run, "check-a", "--good", "good.md"); code != 2 || !strings.Contains(errOut, "writing task") {
		t.Fatalf("a writing flag on a check: exit %d, %s; want 2", code, errOut)
	}
	if code, _, errOut := runLab(t, "lab", "matrix", "judge", id, run, "check-a", "--verdicts", "v.json", "--table"); code != 2 || !strings.Contains(errOut, "one at a time") {
		t.Fatalf("two steps at once: exit %d, %s; want 2", code, errOut)
	}
	code, out, errOut := runLab(t, "lab", "--json", "matrix", "judge", id, run, "check-a")
	var j checkJudged
	if code != 0 || json.Unmarshal([]byte(out), &j) != nil {
		t.Fatalf("drawing the check's judging (exit %d): %s%s", code, out, errOut)
	}
	if j.Stage != "awaiting_verdicts" || j.Copies != 3 || j.Leftovers != 1 || j.Packet == nil || len(j.Packet.Items) != 2 || j.Matcher != nil {
		t.Fatalf("judging = %s; want the packet of two findings, one left apart, no matcher", out)
	}
	for _, leak := range []string{"host-big", home, run, "proposed", "valid"} {
		if strings.Contains(out, leak) {
			t.Errorf("the first phase carries %q: %s", leak, out)
		}
	}
	if code, _, errOut := runLab(t, "lab", "matrix", "judge", id, run, "check-a", "--table"); code != 2 || !strings.Contains(errOut, "locked") {
		t.Fatalf("the table before the verdicts: exit %d, %s; want 2", code, errOut)
	}
	// A verdict file is untrusted input, and its refusal is printed escaped.
	hostile := writeJSONFile(t, "verdicts.json", map[string]any{"schema_version": 1, "packet": j.PacketDigest, "locked": true,
		"verdicts": []any{map[string]any{"code": "\x1b]0;owned\x07", "valid": true}}})
	if code, out, errOut := runLab(t, "lab", "matrix", "judge", id, run, "check-a", "--verdicts", hostile); code != 2 ||
		strings.ContainsAny(out+errOut, "\x1b\x07") || !strings.Contains(errOut, "nothing of it is recorded") {
		t.Fatalf("a hostile verdict file: exit %d, %q%q; want 2, printed escaped", code, out, errOut)
	}
	var given []any
	for i, it := range j.Packet.Items {
		given = append(given, map[string]any{"code": it.Code, "valid": i == 0})
	}
	verdicts := writeJSONFile(t, "verdicts.json", map[string]any{"schema_version": 1, "packet": j.PacketDigest, "locked": true, "verdicts": given})
	if code, out, errOut := runLab(t, "lab", "matrix", "judge", id, run, "check-a", "--verdicts", verdicts); code != 0 || !strings.Contains(out, "locked 2 verdict(s), 1 valid") {
		t.Fatalf("locking the verdicts (exit %d): %s%s", code, out, errOut)
	}
	if code, _, errOut := runLab(t, "lab", "matrix", "judge", id, run, "check-a", "--verdicts", verdicts); code != 2 || !strings.Contains(errOut, "locked already") {
		t.Fatalf("a second verdict file: exit %d, %s; want 2", code, errOut)
	}
	if code, _, errOut := runLab(t, "lab", "matrix", "judge", id, run, "check-a", "--table", "--proposals", verdicts); code != 2 || !strings.Contains(errOut, "no proposer") {
		t.Fatalf("proposals with no proposer declared: exit %d, %s; want 2", code, errOut)
	}
	code, out, errOut = runLab(t, "lab", "--json", "matrix", "judge", id, run, "check-a", "--table")
	var table struct {
		Rows []struct {
			MatchedBy string `json:"matched_by"`
		} `json:"rows"`
		Matcher struct {
			NotEstimable string `json:"not_estimable"`
		} `json:"matcher"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &table) != nil || len(table.Rows) != 2 || !strings.Contains(table.Matcher.NotEstimable, "no matcher") {
		t.Fatalf("the table (exit %d): %s%s; want two rows, the matcher's agreement not estimable", code, out, errOut)
	}
	if code, out, _ := runLab(t, "lab", "matrix", "judge", id, run, "check-a"); code != 0 || !strings.Contains(out, "tabled") {
		t.Fatalf("the judging once tabled (exit %d): %s", code, out)
	}
	code, out, _ = runLab(t, "lab", "--json", "matrix", "report", id, run)
	var rep struct {
		Check []struct {
			Table *json.RawMessage `json:"table"`
		} `json:"check"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &rep) != nil || len(rep.Check) != 1 || rep.Check[0].Table == nil {
		t.Fatalf("the report (exit %d): %s; want the check's table", code, out)
	}
}

// A matcher declared on a runner route is planned in its own role, its request
// about the findings no exact key joins is drawn and waits for its launcher,
// and the host never answers for it.
func TestARunnerMatcherWaitsForItsLauncherThroughTheCLI(t *testing.T) {
	_, id, run := matrixCheckRun(t, `,"matcher":{"id":"match-or","route":"matcher=runner:opencode@openrouter/vendor/model"}`,
		"contradiction", "drift", "gap")
	code, out, errOut := runLab(t, "lab", "--json", "matrix", "judge", id, run, "check-a")
	var j checkJudged
	if code != 0 || json.Unmarshal([]byte(out), &j) != nil {
		t.Fatalf("drawing the check's judging (exit %d): %s%s", code, out, errOut)
	}
	if j.Stage != "awaiting_matcher" || j.Matcher == nil || !j.Matcher.AwaitingRunner || j.MatcherRequest == nil || len(j.MatcherRequest.Items) != 3 {
		t.Fatalf("judging = %s; want the request about three leftovers, waiting for the runner matcher", out)
	}
	answer := writeJSONFile(t, "answer.json", map[string]any{"schema_version": 1,
		"same": [][]string{{j.MatcherRequest.Items[0].ID, j.MatcherRequest.Items[1].ID}}})
	if code, _, errOut := runLab(t, "lab", "matrix", "judge", id, run, "check-a", "--matcher", answer); code != 2 || !strings.Contains(errOut, "runner cell") {
		t.Fatalf("a host answer for a runner matcher: exit %d, %s; want 2", code, errOut)
	}
	if code, out, _ := runLab(t, "lab", "matrix", "judge", id, run, "check-a"); code != 0 || !strings.Contains(out, "waits for a launcher") {
		t.Fatalf("the printed judging (exit %d): %s; want it to say the matcher waits", code, out)
	}
}
