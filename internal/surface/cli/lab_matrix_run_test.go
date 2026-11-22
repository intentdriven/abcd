package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// matrixRunLab mints a lab of a throwaway repository, passes its preflight,
// lays the machine's configuration (an OpenRouter block with a pin, the
// opencode runner and a spending cap) and writes a declaration of a code task
// a writing task and a check, built by a host cell and an OpenRouter cell and reviewed
// by a host reviewer, three copies each in the pilot.
func matrixRunLab(t *testing.T) (home, id, decl string) {
	t.Helper()
	home, id, _ = matrixLabCLI(t, "builder=runner:opencode@openrouter/vendor/model", true)
	decl = filepath.Join(t.TempDir(), "matrix.json")
	body := `{"schema_version":1,"name":"routing check","tasks":[` +
		`{"id":"feature-a","kind":"code","intent":"itd-2610161733316518"},` +
		`{"id":"doc-a","kind":"writing","source":["README.md"],"criteria":"accurate against the source, then clear"},` +
		`{"id":"check-a","kind":"check","material":["README.md"]}],` +
		`"builders":[{"id":"host-big","route":"host"},{"id":"or-cheap","route":"builder=runner:opencode@openrouter/vendor/model","cost_per_copy":{"amount":0.05,"unit":"usd"}}],` +
		`"reviewers":[{"id":"host-rev","route":"host"}],` +
		`"repeats":{"pilot":3,"max":10},"concurrency":2,"judging":{"guests_may_judge":false}}`
	if err := os.WriteFile(decl, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, id, decl
}

// matrixHandOff is one copy's hand-off as `lab matrix --approve` lists it.
type matrixHandOff struct {
	Key struct {
		Task   string `json:"task"`
		Cell   string `json:"cell"`
		Repeat int    `json:"repeat"`
	} `json:"key"`
	State string `json:"state"`
	Clone string `json:"clone"`
	Brief string `json:"brief"`
}

func (h matrixHandOff) key() string {
	return h.Key.Task + "/" + h.Key.Cell + "/" + itoa(h.Key.Repeat)
}

// writeJSONFile writes v as JSON at a fresh path and returns it.
func writeJSONFile(t *testing.T, name string, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// labState is the path of a record under a lab's state/matrix/.
func labState(t *testing.T, home, id string, rel ...string) string {
	t.Helper()
	dirs, _ := filepath.Glob(filepath.Join(abcdhome.Path(home, "lab"), "*", id, "state", "matrix"))
	if len(dirs) != 1 {
		t.Fatalf("the lab %s has no state folder", id)
	}
	return filepath.Join(append([]string{dirs[0]}, rel...)...)
}

// A consented run is started from the CLI and goes the whole way through it:
// every pilot copy's clone is made under a random alias outside every working
// tree, the host's copies are handed back, checked for compliance and written
// as trials, a copy on a runner cell is never taken from the host, the host
// reviewer's blinded requests are drawn from the copies' clones and its answers
// recorded, a writing task's judging is drawn with every copy's signature
// withheld and its verdicts recorded, and the report reads the finished run.
func TestAConsentedMatrixRunGoesFromItsClonesToItsReportThroughTheCLI(t *testing.T) {
	home, id, decl := matrixRunLab(t)
	_, out, _ := runLab(t, "lab", "--json", "matrix", id, decl)
	var plan struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatalf("the plan: %s (%v)", out, err)
	}
	code, out, errOut := runLab(t, "lab", "--json", "matrix", id, decl, "--approve", plan.Digest)
	var started struct {
		RunID    string          `json:"run_id"`
		Launched bool            `json:"launched"`
		Copies   []matrixHandOff `json:"copies"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &started) != nil || started.RunID == "" {
		t.Fatalf("approve (exit %d): %s%s", code, out, errOut)
	}
	run := started.RunID
	if !started.Launched || len(started.Copies) != 18 {
		t.Fatalf("the run handed off %d copies (launched %v); want 3 tasks x 2 builders x 3 pilot repeats", len(started.Copies), started.Launched)
	}
	handed := map[string]matrixHandOff{}
	for _, h := range started.Copies {
		handed[h.key()] = h
		want := "handed_off"
		if h.Key.Cell == "or-cheap" {
			want = "awaiting_runner"
		}
		if h.State != want {
			t.Errorf("copy %s is %s; want %s", h.key(), h.State, want)
		}
		clone := strings.Replace(h.Clone, "~", home, 1)
		if fi, err := os.Stat(filepath.Join(clone, "README.md")); err != nil || fi.IsDir() {
			t.Errorf("copy %s has no clone of the frozen commit at %s", h.key(), h.Clone)
		}
		if b := filepath.Base(clone); !strings.HasPrefix(b, "copy_") || strings.Contains(clone, h.Key.Cell) || strings.Contains(clone, h.Key.Task) {
			t.Errorf("copy %s's clone %s names its task or cell", h.key(), h.Clone)
		}
		if strings.Contains(h.Brief, h.Key.Cell) || strings.Contains(h.Brief, h.Key.Task) {
			t.Errorf("copy %s's brief names its task or cell: %s", h.key(), h.Brief)
		}
	}
	if strings.Contains(out, home) {
		t.Errorf("the approval's output carries the home path")
	}

	// The host makes each host copy in its clone and hands it back.
	now := time.Now().UTC().Truncate(time.Second)
	for r, passed := range map[int]int{1: 4, 2: 4, 3: 2} {
		h := handed["feature-a/host-big/"+itoa(r)]
		clone := strings.Replace(h.Clone, "~", home, 1)
		if err := os.WriteFile(filepath.Join(clone, "feature.go"), []byte("package feature\n\n// Answer is the feature.\nfunc Answer() int { return "+itoa(r)+" }\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		receipt := writeJSONFile(t, "receipt.json", map[string]any{"schema_version": 1, "outcome": "made",
			"started_at": now, "finished_at": now.Add(time.Minute), "model_time_ms": 50000, "gate_wait_ms": 10000,
			"score": map[string]int{"passed": passed, "total": 4}})
		transcript := filepath.Join(t.TempDir(), "transcript.txt")
		if err := os.WriteFile(transcript, []byte("edited feature.go in "+clone+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := runLab(t, "lab", "--json", "matrix", "copy", id, run, h.key(), "--receipt", receipt, "--transcript", transcript)
		var trial struct {
			Status     string `json:"status"`
			Compliance *struct {
				Checks []struct{ Rule, State string } `json:"checks"`
			} `json:"compliance"`
			Unpinned []struct{ Field, Reason string } `json:"unpinned"`
		}
		if code != 0 || json.Unmarshal([]byte(out), &trial) != nil || trial.Compliance == nil || len(trial.Compliance.Checks) != 4 {
			t.Fatalf("handing back %s (exit %d): %s%s", h.key(), code, out, errOut)
		}
		want := "passed"
		if passed < 4 {
			want = "failed"
		}
		if trial.Status != want {
			t.Errorf("copy %s is %s; want %s", h.key(), trial.Status, want)
		}
		// Its hidden tests' result is the host's word, and its trial says so.
		marked := false
		for _, u := range trial.Unpinned {
			marked = marked || u.Field == "trial.score" && u.Reason == "self_reported"
		}
		if !marked {
			t.Errorf("copy %s's trial does not mark its score as the host's report: %+v", h.key(), trial.Unpinned)
		}
		if _, err := os.Stat(labState(t, home, id, run, "trials", "feature-a", "host-big", itoa(r)+".json")); err != nil {
			t.Errorf("copy %s's trial is not written: %v", h.key(), err)
		}
	}
	// A copy is handed back once.
	h1 := handed["feature-a/host-big/1"]
	receipt := writeJSONFile(t, "receipt.json", map[string]any{"schema_version": 1, "outcome": "failed", "reason": "gave up"})
	if code, _, errOut := runLab(t, "lab", "matrix", "copy", id, run, h1.key(), "--receipt", receipt); code != 2 || !strings.Contains(errOut, "already") {
		t.Errorf("a second hand-back: exit %d, %s; want 2", code, errOut)
	}
	// A copy on a runner cell is the runner's: the host never makes it.
	if code, _, errOut := runLab(t, "lab", "matrix", "copy", id, run, "feature-a/or-cheap/1", "--receipt", receipt); code != 2 || !strings.Contains(errOut, "runner") {
		t.Errorf("a host hand-back of a runner cell's copy: exit %d, %s; want 2", code, errOut)
	}
	// Without a receipt the verb shows the copy's hand-off and writes nothing.
	if code, out, errOut := runLab(t, "lab", "matrix", "copy", id, run, "feature-a/or-cheap/2"); code != 0 || !strings.Contains(out, "awaiting_runner") {
		t.Errorf("showing a hand-off: exit %d, %s%s", code, out, errOut)
	}

	// A request passes the secret screen before it is stored or handed out:
	// a copy whose change carries a credential refuses the draw, and nothing
	// is written, so the draw can be made once the copy is clean.
	token := "ghp_" + testsecret.Synthetic(31, 36)
	feature1 := filepath.Join(strings.Replace(h1.Clone, "~", home, 1), "feature.go")
	clean, err := os.ReadFile(feature1)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(feature1, append(append([]byte{}, clean...), []byte("\n// key "+token+"\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, errOut := runLab(t, "lab", "matrix", "review", id, run, "feature-a"); code != 2 || strings.Contains(out+errOut, token) {
		t.Errorf("a review of a copy carrying a credential: exit %d, %s%s; want 2, the value never shown", code, out, errOut)
	}
	if _, err := os.Stat(labState(t, home, id, run, "review", "feature-a", "asked.json")); !os.IsNotExist(err) {
		t.Errorf("a refused review wrote its requests: %v", err)
	}
	if err := os.WriteFile(feature1, clean, 0o600); err != nil {
		t.Fatal(err)
	}

	// The host reviewer's requests are drawn from the copies' clones, blind.
	code, out, errOut = runLab(t, "lab", "--json", "matrix", "review", id, run, "feature-a")
	var asked struct {
		Requests []struct {
			Reviewer string `json:"reviewer"`
			Label    string `json:"label"`
			State    string `json:"state"`
			Request  string `json:"request"`
		} `json:"requests"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &asked) != nil || len(asked.Requests) != 3 {
		t.Fatalf("review requests (exit %d): %s%s", code, out, errOut)
	}
	answers := t.TempDir()
	for i, rq := range asked.Requests {
		if rq.Reviewer != "host-rev" || rq.State != "handed_off" || !strings.Contains(rq.Request, "feature.go") {
			t.Errorf("request %+v is not the host reviewer's blinded request of the copy", rq)
		}
		for _, leak := range []string{"host-big", "feature-a", run, home} {
			if strings.Contains(rq.Request, leak) {
				t.Errorf("request %s carries %q", rq.Label, leak)
			}
		}
		verdict := "clean"
		if i == 0 {
			verdict = "defect"
		}
		if err := os.MkdirAll(filepath.Join(answers, "host-rev"), 0o700); err != nil {
			t.Fatal(err)
		}
		ans, _ := json.Marshal(map[string]any{"label": rq.Label, "verdict": verdict, "findings": []any{}, "author_guess": ""})
		if err := os.WriteFile(filepath.Join(answers, "host-rev", rq.Label+".json"), ans, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if code, out, errOut := runLab(t, "lab", "matrix", "review", id, run, "feature-a", "--answers", answers); code != 0 {
		t.Fatalf("recording the reviews (exit %d): %s%s", code, out, errOut)
	}

	// The host's writing copies are handed back with their text, and the
	// judging drawn from them withholds what would sign each copy.
	for r := 1; r <= 3; r++ {
		h := handed["doc-a/host-big/"+itoa(r)]
		clone := strings.Replace(h.Clone, "~", home, 1)
		answer := filepath.Join(t.TempDir(), "doc.md")
		text := "The root of the repository holds a README.\nWritten by host-big in " + clone + " for doc-a, copy " + filepath.Base(clone) + ".\n"
		if err := os.WriteFile(answer, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		receipt := writeJSONFile(t, "receipt.json", map[string]any{"schema_version": 1, "outcome": "made", "started_at": now, "finished_at": now.Add(time.Minute)})
		transcript := filepath.Join(t.TempDir(), "transcript.txt")
		if err := os.WriteFile(transcript, []byte("read README.md\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if code, out, errOut := runLab(t, "lab", "matrix", "copy", id, run, h.key(), "--receipt", receipt, "--transcript", transcript, "--answer", answer); code != 0 {
			t.Fatalf("handing back %s (exit %d): %s%s", h.key(), code, out, errOut)
		}
	}
	good := filepath.Join(t.TempDir(), "good.md")
	bad := filepath.Join(t.TempDir(), "bad.md")
	if os.WriteFile(good, []byte("The README says root.\n"), 0o600) != nil || os.WriteFile(bad, []byte("The README is a novel.\n"), 0o600) != nil {
		t.Fatal("calibration examples")
	}
	// The packet passes the secret screen before it is written.
	keyed := filepath.Join(t.TempDir(), "keyed.md")
	if err := os.WriteFile(keyed, []byte("The README says root, keyed "+token+".\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, errOut := runLab(t, "lab", "matrix", "judge", id, run, "doc-a", "--good", keyed, "--bad", bad, "--judges", "1"); code != 2 || strings.Contains(out+errOut, token) {
		t.Errorf("a judging whose calibration carries a credential: exit %d, %s%s; want 2, the value never shown", code, out, errOut)
	}
	if _, err := os.Stat(labState(t, home, id, run, "judging", "doc-a", "packet.json")); !os.IsNotExist(err) {
		t.Errorf("a refused judging wrote its packet: %v", err)
	}
	if code, out, errOut := runLab(t, "lab", "matrix", "judge", id, run, "doc-a", "--good", good, "--bad", bad, "--judges", "1"); code != 0 {
		t.Fatalf("drawing the judging (exit %d): %s%s", code, out, errOut)
	}
	packetRaw, err := os.ReadFile(labState(t, home, id, run, "judging", "doc-a", "packet.json"))
	if err != nil {
		t.Fatalf("the judging packet is not written: %v", err)
	}
	for _, leak := range []string{"host-big", "doc-a", run, home, "copy_"} {
		if strings.Contains(string(packetRaw), leak) {
			t.Errorf("the judging packet carries %q", leak)
		}
	}
	var packet struct {
		Slots []struct {
			ID     string `json:"id"`
			Copies []struct {
				Code string `json:"code"`
			} `json:"copies"`
		} `json:"slots"`
	}
	if err := json.Unmarshal(packetRaw, &packet); err != nil || len(packet.Slots) != 1 {
		t.Fatalf("the packet: %v", err)
	}
	var key struct {
		Packet string `json:"packet"`
	}
	keyRaw, _ := os.ReadFile(labState(t, home, id, run, "judging", "doc-a", "key.json"))
	if err := json.Unmarshal(keyRaw, &key); err != nil {
		t.Fatal(err)
	}
	slot := packet.Slots[0]
	var ranking []string
	marks := map[string]bool{}
	for i, c := range slot.Copies {
		ranking = append(ranking, c.Code)
		marks[c.Code] = i == 0
	}
	verdicts := writeJSONFile(t, "verdicts.json", map[string]any{"schema_version": 1, "packet": key.Packet, "verdicts": []any{
		map[string]any{"judge": map[string]string{"github": "judge-one"}, "slot": slot.ID, "calibrated": true,
			"ranking": ranking, "good_enough": marks, "given_at": now}}})
	if code, out, errOut := runLab(t, "lab", "matrix", "judge", id, run, "doc-a", "--verdicts", verdicts); code != 0 {
		t.Fatalf("recording the verdicts (exit %d): %s%s", code, out, errOut)
	}

	// A check copy's findings are recorded before its clone is deleted.
	hc := handed["check-a/host-big/1"]
	checkClone := strings.Replace(hc.Clone, "~", home, 1)
	answer := writeJSONFile(t, "answer.json", map[string]any{"schema_version": 1, "findings": []any{
		map[string]any{"class": "contradiction", "summary": "the README says root twice",
			"ends": []any{map[string]string{"path": "README.md", "quote": "root"}, map[string]string{"path": "README.md", "quote": "root"}}}}})
	receipt = writeJSONFile(t, "receipt.json", map[string]any{"schema_version": 1, "outcome": "made", "started_at": now, "finished_at": now.Add(time.Minute)})
	transcript := filepath.Join(t.TempDir(), "transcript.txt")
	if err := os.WriteFile(transcript, []byte("read README.md\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runLab(t, "lab", "matrix", "copy", id, run, hc.key(), "--receipt", receipt, "--transcript", transcript, "--answer", answer); code != 2 || !strings.Contains(errOut, "form") {
		t.Errorf("a check copy without its form: exit %d, %s; want 2", code, errOut)
	}
	if code, out, errOut := runLab(t, "lab", "matrix", "copy", id, run, hc.key(), "--receipt", receipt, "--transcript", transcript, "--answer", answer, "--form", "consistency"); code != 0 {
		t.Fatalf("handing back %s (exit %d): %s%s", hc.key(), code, out, errOut)
	}
	if _, err := os.Stat(labState(t, home, id, run, "check", "check-a", "copies", "host-big", "1.json")); err != nil {
		t.Errorf("the check copy's findings are not recorded: %v", err)
	}
	if _, err := os.Stat(checkClone); !os.IsNotExist(err) {
		t.Errorf("the check copy's clone is not deleted once its findings are recorded: %v", err)
	}

	// A check copy's clone is deleted only once its trial is written: a
	// receipt the secret screen refuses leaves the copy as it was, its clone
	// in place, and a clean hand-back then records it as the failure it is.
	hc2 := handed["check-a/host-big/2"]
	checkClone2 := strings.Replace(hc2.Clone, "~", home, 1)
	keyedReceipt := writeJSONFile(t, "receipt.json", map[string]any{"schema_version": 1, "outcome": "failed", "reason": "gave up at " + token})
	if code, out, errOut := runLab(t, "lab", "matrix", "copy", id, run, hc2.key(), "--receipt", keyedReceipt, "--transcript", transcript, "--form", "consistency"); code != 2 || strings.Contains(out+errOut, token) {
		t.Errorf("a check copy whose receipt carries a credential: exit %d, %s%s; want 2, the value never shown", code, out, errOut)
	}
	if _, err := os.Stat(checkClone2); err != nil {
		t.Fatalf("a refused hand-back deleted the check copy's clone: %v", err)
	}
	failed := writeJSONFile(t, "receipt.json", map[string]any{"schema_version": 1, "outcome": "failed", "reason": "gave up"})
	code, out, errOut = runLab(t, "lab", "--json", "matrix", "copy", id, run, hc2.key(), "--receipt", failed, "--transcript", transcript, "--form", "consistency")
	var failedTrial struct {
		Status string `json:"status"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &failedTrial) != nil || failedTrial.Status != "failed" {
		t.Fatalf("handing %s back clean (exit %d): %s%s; want it recorded failed", hc2.key(), code, out, errOut)
	}
	if _, err := os.Stat(checkClone2); !os.IsNotExist(err) {
		t.Errorf("the check copy's clone is not deleted once its trial is written: %v", err)
	}

	// The report reads the finished run end to end.
	code, out, errOut = runLab(t, "lab", "--json", "matrix", "report", id, run)
	var rep struct {
		Code []struct {
			Cells []struct {
				Cell    string `json:"cell"`
				PassAt1 struct {
					Passed   int `json:"passed"`
					Attempts int `json:"attempts"`
				} `json:"pass_at_1"`
			} `json:"cells"`
			Reviewers []struct {
				Reviewer string `json:"reviewer"`
			} `json:"reviewers"`
		} `json:"code"`
		Writing []struct {
			Judged bool `json:"judged"`
		} `json:"writing"`
		Trials     int `json:"trials"`
		Unreturned []struct {
			Task    string `json:"task"`
			Cell    string `json:"cell"`
			State   string `json:"state"`
			Repeats []int  `json:"repeats"`
		} `json:"unreturned"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &rep) != nil || len(rep.Code) != 1 || len(rep.Writing) != 1 {
		t.Fatalf("the report (exit %d): %s%s", code, out, errOut)
	}
	if rep.Trials != 8 || !rep.Writing[0].Judged || len(rep.Code[0].Reviewers) != 1 {
		t.Errorf("the report holds %d trials, writing judged %v, %d reviewers; want 8, true, 1", rep.Trials, rep.Writing[0].Judged, len(rep.Code[0].Reviewers))
	}
	// Every copy handed off and not back is counted: the host's check copy
	// never handed back, and every runner cell's copy waiting for its runner.
	accounted := map[string]int{}
	for _, u := range rep.Unreturned {
		accounted[u.Task+"/"+u.Cell+" "+u.State] += len(u.Repeats)
	}
	for k, n := range map[string]int{"check-a/host-big not_handed_back": 1, "feature-a/or-cheap awaiting_runner": 3,
		"doc-a/or-cheap awaiting_runner": 3, "check-a/or-cheap awaiting_runner": 3} {
		if accounted[k] != n {
			t.Errorf("the report accounts %d copies as %s; want %d (all: %+v)", accounted[k], k, n, rep.Unreturned)
		}
	}
	if len(accounted) != 4 {
		t.Errorf("the report accounts %+v; want those four alone", accounted)
	}
	if code, out, _ := runLab(t, "lab", "matrix", "report", id, run); code != 0 || !strings.Contains(out, "awaiting its runner") || !strings.Contains(out, "never handed back") {
		t.Errorf("the printed report does not say which copies wait for a runner or were never handed back:\n%s", out)
	} else if !strings.Contains(out, "3 the host's report") || !strings.Contains(out, "are the host's report") {
		t.Errorf("the printed report does not say host-big's results are the host's report:\n%s", out)
	}
	for _, c := range rep.Code[0].Cells {
		if c.Cell == "host-big" && (c.PassAt1.Passed != 2 || c.PassAt1.Attempts != 3) {
			t.Errorf("host-big passed %d of %d; want 2 of 3", c.PassAt1.Passed, c.PassAt1.Attempts)
		}
	}
}
