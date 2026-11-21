package matrix

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/docfidelity"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/record/findingkey"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The check task's fixture: one brief chapter the copies check, frozen at the
// lab's pin.
const (
	guardChapter = docfidelity.ChaptersDir + "/17-guard.md"
	guardText    = "# The guard\n\n" +
		"The guard reads its registry from the plugin root.\n" +
		"It refuses a hazard\n" +
		"before the command runs.\n" +
		"The guard never logs a refusal.\n"
	seededLine = "The guard never logs a refusal."
	checkTask  = "fidelity"
)

var checkSeed = [32]byte{7, 7, 7}

// dfAnswer is a doc-fidelity copy's answer: a HOLD naming each brief sentence
// of the guard chapter, and each public sentence as "public:<path>:<text>".
func dfAnswer(t *testing.T, sentences ...string) []byte {
	t.Helper()
	return dfAnswerIn(t, "17-guard.md", sentences...)
}

// dfAnswerIn is dfAnswer with each brief sentence in chapter.
func dfAnswerIn(t *testing.T, chapter string, sentences ...string) []byte {
	t.Helper()
	type failing struct {
		Disposition string `json:"disposition"`
		Doc         string `json:"doc"`
		Chapter     string `json:"chapter"`
		Sentence    string `json:"sentence"`
		Evidence    string `json:"evidence"`
		Replacement string `json:"replacement,omitempty"`
	}
	var fs []failing
	for _, s := range sentences {
		f := failing{Disposition: "confirmed", Doc: "brief", Chapter: chapter, Sentence: s,
			Evidence: "internal/core/guard/registry.go:40 reads it elsewhere", Replacement: s + " (corrected)"}
		if rest, ok := strings.CutPrefix(s, "public:"); ok {
			p, text, _ := strings.Cut(rest, ":")
			f.Doc, f.Chapter, f.Sentence, f.Replacement = "public", p, text, ""
		}
		fs = append(fs, f)
	}
	data, err := json.Marshal(map[string]any{"schema_version": 1, "verificationResult": "HOLD",
		"judgeModel": "example-model-2026-10-01", "tier": "full", "failing": fs})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// fidelityCopy reads one doc-fidelity copy's answer against the fixture's
// material, as the run reads it.
func fidelityCopy(t *testing.T, cell string, repeat int, sentences ...string) CheckCopy {
	t.Helper()
	return readCheckCopy(checkCopyInputs{
		RepoRoot: "/work/checkout", Store: "/work/store", Run: "mx-261023012949-ab12",
		Key: TrialKey{Task: checkTask, Cell: cell, Repeat: repeat}, Form: CheckDocFidelity,
		Material: map[string]string{guardChapter: guardText}, Answer: dfAnswer(t, sentences...),
	})
}

// rowOf is the row holding copy key's finding n.
func rowOf(t *testing.T, rows []CheckRow, key TrialKey, n int) int {
	t.Helper()
	for i, r := range rows {
		for _, m := range r.Members {
			if m.Copy == key && m.Finding == n {
				return i
			}
		}
	}
	t.Fatalf("no row holds %s finding %d: %+v", key, n, rows)
	return -1
}

// Every finding of a check copy is the copy's model's output and is read as
// untrusted input: bounded, decoded strictly, its text made one inert line,
// every mention of where the copy ran withheld, and its position computed from
// the frozen material, never taken from the copy. An answer that fails is the
// copy's failure, with nothing of it kept.
func TestACheckCopysAnswerIsReadAsUntrustedInput(t *testing.T) {
	c := fidelityCopy(t, "big", 1, "It refuses a hazard before   the command runs.")
	if !c.Answered || len(c.Findings) != 1 {
		t.Fatalf("copy = %+v; want one finding answered", c)
	}
	f := c.Findings[0]
	if f.Doc != docfidelity.DocBrief || len(f.Ends) != 1 || f.Ends[0].Path != guardChapter ||
		f.Ends[0].Quote != "It refuses a hazard before the command runs." {
		t.Fatalf("finding = %+v; want the chapter's path and the quote collapsed", f)
	}
	if l := f.Ends[0].Lines; l == nil || *l != (LineSpan{Start: 4, End: 5}) {
		t.Fatalf("lines = %+v; want 4-5, found in the frozen material", f.Ends[0].Lines)
	}
	if !findingkey.Valid(f.Key) {
		t.Fatalf("key %q is not the one derivation's shape", f.Key)
	}

	// Where the copy ran is withheld, and counted.
	in := checkCopyInputs{RepoRoot: "/work/checkout", Store: "/work/store", Run: "mx-261023012949-ab12",
		Key: TrialKey{Task: checkTask, Cell: "big", Repeat: 2}, Form: CheckDocFidelity,
		Material: map[string]string{guardChapter: guardText}}
	raw := strings.Replace(string(dfAnswer(t, "It refuses a hazard")), "internal/core/guard/registry.go:40",
		"/work/store/mx-261023012949-ab12/fidelity/big/2/x.go:1 \\u001b[31m", 1)
	in.Answer = []byte(raw)
	c = readCheckCopy(in)
	if !c.Answered || c.Redacted == 0 || strings.Contains(c.Findings[0].Evidence, "/work/store") ||
		strings.Contains(c.Findings[0].Evidence, "mx-261023012949-ab12") || strings.ContainsRune(c.Findings[0].Evidence, 0x1b) {
		t.Fatalf("copy = %+v; want the store, the run and the control withheld", c)
	}

	// A quote the frozen material does not hold has no position: the copy
	// cannot claim one.
	c = fidelityCopy(t, "big", 3, "The guard is written in Rust.")
	if !c.Answered || c.Findings[0].Ends[0].Lines != nil {
		t.Fatalf("copy = %+v; want the unlocated quote kept without a position", c)
	}

	// A consistency copy's finding is keyed by the consistency finding key,
	// whichever order its ends come in.
	cons := func(a, b string) CheckCopy {
		ans := `{"schema_version":1,"findings":[{"class":"contradiction","summary":"the two disagree","ends":[` + a + `,` + b + `]}]}`
		return readCheckCopy(checkCopyInputs{RepoRoot: "/work/checkout", Store: "/work/store", Run: "mx-261023012949-ab12",
			Key: TrialKey{Task: checkTask, Cell: "big", Repeat: 1}, Form: CheckConsistency,
			Material: map[string]string{guardChapter: guardText}, Answer: []byte(ans)})
	}
	endA := `{"path":"` + guardChapter + `","quote":"It refuses a hazard"}`
	endB := `{"path":"docs/guide.md","quote":"It never refuses."}`
	c1, c2 := cons(endA, endB), cons(endB, endA)
	want := findingkey.Derive("contradiction", findingkey.End{Path: guardChapter, Quote: "It refuses a hazard"},
		findingkey.End{Path: "docs/guide.md", Quote: "It never refuses."})
	if !c1.Answered || !c2.Answered || c1.Findings[0].Key != want || c2.Findings[0].Key != want {
		t.Fatalf("consistency keys = %+v / %+v; want %s for both orders", c1, c2, want)
	}

	good := string(dfAnswer(t, "It refuses a hazard"))
	for name, ans := range map[string]string{
		"not JSON":           "{",
		"an unknown key":     strings.Replace(good, `"tier"`, `"model":"x","tier"`, 1),
		"trailing data":      good + "{}",
		"too large":          good + strings.Repeat(" ", maxCheckAnswerBytes),
		"a traversal":        strings.Replace(good, `"17-guard.md"`, `"../17-guard.md"`, 1),
		"a sentence too big": strings.Replace(good, `It refuses a hazard"`, strings.Repeat("x", 3000)+`"`, 1),
	} {
		in.Answer = []byte(ans)
		in.Key.Repeat = 4
		c := readCheckCopy(in)
		if c.Answered || len(c.Findings) != 0 || c.Reason == "" {
			t.Errorf("%s: copy = %+v; want it the copy's failure, nothing kept", name, c)
		}
	}
	// A runner that failed has no answer to read.
	in.Answer, in.Failure = []byte(good), "the runner exited 1"
	if c := readCheckCopy(in); c.Answered || len(c.Findings) != 0 || !strings.Contains(c.Reason, "exited 1") {
		t.Fatalf("a failed runner's copy = %+v; want it unanswered", c)
	}
}

// checkLab mints a lab whose pin holds the guard chapter, writes a run record
// at its pin, each edit made to the run's inputs first, and returns the source,
// the home, the lab, the record and a clone store outside every working tree.
func checkLab(t *testing.T, edits ...func(*runInputs)) (*gittest.Repo, string, lab.Minted, RunRecord, string) {
	t.Helper()
	return checkLabWith(t, nil, edits...)
}

// checkLabWith is checkLab with files, by path, frozen beside the guard
// chapter.
func checkLabWith(t *testing.T, files map[string]string, edits ...func(*runInputs)) (*gittest.Repo, string, lab.Minted, RunRecord, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	r := gittest.NewRepo(t)
	r.Write("README.md", "root\n")
	r.Write(guardChapter, guardText)
	for p, text := range files {
		r.Write(p, text)
	}
	r.Commit("root")
	t.Cleanup(lab.SetAbcdRootCommitForTest(strings.Repeat("e", 40)))
	m, err := lab.Mint(r.Root(), "does a cheaper model check as well?", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lab.Preflight(r.Root(), m.ID); err != nil {
		t.Fatal(err)
	}
	in := inputs(t)
	in.Lab, in.RootSHA, in.FrozenCommit = m.ID, m.RootSHA, m.Pin
	for _, edit := range edits {
		edit(&in)
	}
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(r.Root(), rec, in.Declaration); err != nil {
		t.Fatal(err)
	}
	return r, home, m, rec, filepath.Join(t.TempDir(), "store")
}

// collect snapshots what the release gates read, makes copy key's clone and
// collects its doc-fidelity answer, as the run does once the copy's runner is
// done.
func collect(t *testing.T, r *gittest.Repo, m lab.Minted, rec RunRecord, store string, key TrialKey, answer []byte) (string, CheckCopy, error) {
	t.Helper()
	return collectForm(t, r, m, rec, store, key, CheckDocFidelity, answer)
}

func collectForm(t *testing.T, r *gittest.Repo, m lab.Minted, rec RunRecord, store string, key TrialKey, form CheckForm, answer []byte) (string, CheckCopy, error) {
	t.Helper()
	gates, err := snapshotGates(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	clone, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key)
	if err != nil {
		t.Fatal(err)
	}
	c, err := collectCheckCopy(m.ID, checkInputs(t, r, rec, store, key, form, gates, answer))
	return clone, c, err
}

// collectCheckCopy is a check copy's collection as a hand-back makes it, short
// of its trial: its findings recorded (recordCheckCopy), then its clone
// deleted. CollectCopy is the production path, which writes the trial between
// the two.
func collectCheckCopy(labID string, in checkCopyInputs) (CheckCopy, error) {
	c, clone, err := recordCheckCopy(labID, in)
	if err != nil {
		return CheckCopy{}, err
	}
	if err := os.RemoveAll(clone); err != nil {
		return CheckCopy{}, fmt.Errorf("copy %s's findings are recorded, and its clone could not be deleted: %w", in.Key, err)
	}
	return c, nil
}

// checkInputs is copy key's inputs to collectCheckCopy, the gates snapshotted
// before its copy began.
func checkInputs(t *testing.T, r *gittest.Repo, rec RunRecord, store string, key TrialKey, form CheckForm, gates gateSnapshot, answer []byte) checkCopyInputs {
	t.Helper()
	material, err := readFrozenMaterial(context.Background(), r.Root(), rec.FrozenCommit, []string{docfidelity.ChaptersDir})
	if err != nil {
		t.Fatal(err)
	}
	return checkCopyInputs{RepoRoot: r.Root(), Store: store, Run: rec.RunID, Frozen: rec.FrozenCommit,
		Key: key, Form: form, Material: material, Answer: answer, Gates: gates}
}

// A check copy's findings are in the run's record before its clone is
// deleted: a copy whose findings cannot be recorded keeps its clone, so
// nothing it found is lost with it.
func TestACheckCopysFindingsAreReadBeforeItsCloneIsDeleted(t *testing.T) {
	r, home, m, rec, store := checkLab(t)
	key := TrialKey{Task: checkTask, Cell: "host-big", Repeat: 1}
	clone, c, err := collect(t, r, m, rec, store, key, dfAnswer(t, "It refuses a hazard"))
	if err != nil {
		t.Fatalf("collectCheckCopy: %v", err)
	}
	if _, err := os.Stat(clone); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the clone is still there once its findings were recorded: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(runDir(home, m, rec.RunID), "check", checkTask, "copies", "host-big", "1.json"))
	if err != nil {
		t.Fatalf("the copy's findings are not in the run's record: %v", err)
	}
	var back CheckCopy
	if err := json.Unmarshal(data, &back); err != nil || !back.Answered || len(back.Findings) != 1 || back.Findings[0].Key != c.Findings[0].Key {
		t.Fatalf("recorded = %s (%v); want the copy's one finding", data, err)
	}

	// A second collection of the same copy cannot be recorded, so its clone
	// stays where it is.
	key2 := TrialKey{Task: checkTask, Cell: "host-big", Repeat: 2}
	clone2, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clone2, "kept.txt"), []byte("the copy's work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.WriteState(checkCopyRel(rec.RunID, key2), []byte("{}")); err != nil {
		t.Fatal(err)
	}
	p.Close()
	gates, err := snapshotGates(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	_, err = collectCheckCopy(m.ID, checkInputs(t, r, rec, store, key2, CheckDocFidelity, gates, dfAnswer(t, "It refuses a hazard")))
	if err == nil {
		t.Fatal("a copy whose findings could not be recorded was collected")
	}
	if _, err := os.Stat(filepath.Join(clone2, "kept.txt")); err != nil {
		t.Fatalf("a copy whose findings were not recorded lost its clone: %v", err)
	}
}

// treeDigest is every entry under root's rel, by path, with each regular
// file's bytes hashed: the test's own reading, apart from the run's.
func treeDigest(t *testing.T, root string, rels ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, rel := range rels {
		err := filepath.WalkDir(filepath.Join(root, rel), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			k, _ := filepath.Rel(root, p)
			if d.Type().IsRegular() {
				data, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				sum := sha256.Sum256(data)
				out[k] = hex.EncodeToString(sum[:])
				return nil
			}
			out[k] = d.Type().String()
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// A whole check run, copies to the overlap table, leaves the doc-fidelity
// receipts, the review flags, the issue ledger and the checked material as
// they were: a copy's verdict is validated without being recorded where the
// release gates read, nothing it proposed is applied, and no finding is
// filed (decisions 22 and 30).
func TestTheDocFidelityReceiptsAndTheIssueLedgerAreUnchangedByARun(t *testing.T) {
	r, home, m, rec, store := checkLab(t)
	head := r.Git("rev-parse", "HEAD")
	receipt := filepath.Join(docfidelity.ReceiptsDir, head, docfidelity.GateName+".json")
	for rel, body := range map[string]string{
		receipt:                                  `{"verificationResult":"PROMOTE"}` + "\n",
		recordid.IssuesRelDir + "/open/iss-1.md": "an open issue\n",
		recordid.IssuesRelDir + "/resolved/iss-2.md": "a resolved issue\n",
		docfidelity.FlagsPath:                        `{"schema_version":1,"flags":[]}` + "\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(r.Root(), rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(r.Root(), rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gates := []string{docfidelity.ReceiptsDir, recordid.IssuesRelDir, docfidelity.FlagsPath, guardChapter}
	before := treeDigest(t, r.Root(), gates...)

	// Every copy's verdict is a HOLD drafting a correction, the kind
	// docfidelity.Record would save as a receipt and --apply would write.
	var copies []CheckCopy
	for i, cell := range []string{"host-big", "or-cheap"} {
		_, c, err := collect(t, r, m, rec, store, TrialKey{Task: checkTask, Cell: cell, Repeat: 1},
			dfAnswer(t, "The guard reads its registry from the plugin root.", []string{"It refuses a hazard", seededLine}[i]))
		if err != nil {
			t.Fatalf("collectCheckCopy %s: %v", cell, err)
		}
		copies = append(copies, c)
	}
	rows := matchExact(copies).rows()
	p, k, err := drawCheckPacket(rec.RunID, checkTask, copies, rows, checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeCheckJudging(r.Root(), m.ID, p, k); err != nil {
		t.Fatal(err)
	}
	if _, err := recordCheckVerdicts(r.Root(), m.ID, rec.RunID, checkTask, checkVerdicts(t, p, k, true, nil)); err != nil {
		t.Fatal(err)
	}
	if _, err := recordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, checkTableInputs{}); err != nil {
		t.Fatal(err)
	}
	if after := treeDigest(t, r.Root(), gates...); !mapsEqual(before, after) {
		t.Fatalf("a run changed the release gates' receipts, the ledger or the material:\nbefore %v\nafter  %v", before, after)
	}
	if _, err := os.Stat(filepath.Join(runDir(home, m, rec.RunID), "check", checkTask, "table.json")); err != nil {
		t.Fatalf("the run wrote no table: %v", err)
	}

	// The run's own guard sees a gate written while it ran.
	snap, err := snapshotGates(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err := snap.unchanged(r.Root()); err != nil {
		t.Fatalf("an untouched checkout reads as changed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(r.Root(), recordid.IssuesRelDir, "open", "iss-3.md"), []byte("filed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := snap.unchanged(r.Root()); err == nil || !strings.Contains(err.Error(), "iss-3.md") {
		t.Fatalf("a finding filed while the run ran = %v; want a refusal naming it", err)
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// Matching takes exact keys first, the document, chapter and quoted sentence,
// then overlapping lines in the frozen material, and only what neither joins
// to another copy's finding goes to the matcher model, which names no copy or
// model.
func TestExactKeyMatchingComesFirstAndTheMatcherTakesOnlyTheLeftovers(t *testing.T) {
	a1 := fidelityCopy(t, "big", 1, "The guard reads its registry from the plugin root.", "public:docs/guide.md:Run ahoy first.")
	b1 := fidelityCopy(t, "small", 1, "The guard  reads its registry\tfrom the plugin root.", "public:docs/guide.md:Always run ahoy before anything else.")
	c1 := fidelityCopy(t, "local", 1, "It refuses a hazard", "public:docs/other.md:Nothing here matches.")
	a2 := fidelityCopy(t, "big", 2, "It refuses a hazard before the command runs.")
	copies := []CheckCopy{a1, b1, c1, a2}
	m := matchExact(copies)
	rows := m.rows()
	if rowOf(t, rows, a1.Key, 0) != rowOf(t, rows, b1.Key, 0) || rows[rowOf(t, rows, a1.Key, 0)].MatchedBy != MatchedByKey {
		t.Fatalf("the same sentence quoted with other spacing is not one row by key: %+v", rows)
	}
	if rowOf(t, rows, c1.Key, 0) != rowOf(t, rows, a2.Key, 0) || rows[rowOf(t, rows, c1.Key, 0)].MatchedBy != MatchedByLines {
		t.Fatalf("two quotes over overlapping lines are not one row by lines: %+v", rows)
	}

	req, ids, err := m.matcherRequest(checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Items) != 3 || len(ids) != 3 {
		t.Fatalf("matcher request = %+v; want the three unjoined public findings and nothing else", req)
	}
	js, _ := json.Marshal(req)
	for _, leak := range []string{"big", "small", "local", "mx-261023012949-ab12", checkTask, "registry from the plugin root", "hazard"} {
		if bytes.Contains(js, []byte(leak)) {
			t.Errorf("the matcher's request carries %q: %s", leak, js)
		}
	}
	idOf := func(quote string) string {
		for _, it := range req.Items {
			if it.Ends[0].Quote == quote {
				return it.ID
			}
		}
		t.Fatalf("no item quotes %q", quote)
		return ""
	}
	x, y := idOf("Run ahoy first."), idOf("Always run ahoy before anything else.")
	answer := []byte(`{"schema_version":1,"same":[["` + x + `","` + y + `"]]}`)
	pairs, err := parseMatcherAnswer(answer, req)
	if err != nil {
		t.Fatalf("parseMatcherAnswer: %v", err)
	}
	m.applyMatcher(ids, pairs)
	rows = m.rows()
	if rowOf(t, rows, a1.Key, 1) != rowOf(t, rows, b1.Key, 1) || rows[rowOf(t, rows, a1.Key, 1)].MatchedBy != MatchedByMatcher {
		t.Fatalf("the matcher's pair is not one row: %+v", rows)
	}
	if rows[rowOf(t, rows, c1.Key, 1)].MatchedBy != MatchedAlone || len(rows) != 4 {
		t.Fatalf("rows = %+v; want four, the unmatched one alone", rows)
	}

	for name, ans := range map[string]string{
		"not JSON":      `{`,
		"an unknown id": `{"schema_version":1,"same":[["` + x + `","m-zzzz"]]}`,
		"a self pair":   `{"schema_version":1,"same":[["` + x + `","` + x + `"]]}`,
		"an extra key":  `{"schema_version":1,"same":[],"cells":["big"]}`,
		"three in pair": `{"schema_version":1,"same":[["` + x + `","` + y + `","` + x + `"]]}`,
	} {
		if _, err := parseMatcherAnswer([]byte(ans), req); err == nil {
			t.Errorf("%s: the matcher's answer was accepted", name)
		}
	}
}

// The matcher is checked against people: its decisions on a labelled sample
// of the leftover pairs are reported as its agreement, never assumed.
func TestTheMatchersAgreementOnALabelledSampleIsReported(t *testing.T) {
	// A HOLD names a brief sentence, so each copy names the one all three
	// share by key, which is no leftover, beside its public sentence.
	const shared = "The guard reads its registry from the plugin root."
	copies := []CheckCopy{
		fidelityCopy(t, "big", 1, shared, "public:docs/guide.md:Run ahoy first."),
		fidelityCopy(t, "small", 1, shared, "public:docs/guide.md:Always run ahoy before anything else."),
		fidelityCopy(t, "local", 1, shared, "public:docs/guide.md:Ahoy is optional."),
	}
	m := matchExact(copies)
	req, _, err := m.matcherRequest(checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	id := map[string]string{}
	for _, it := range req.Items {
		id[it.Ends[0].Quote] = it.ID
	}
	x, y, z := id["Run ahoy first."], id["Always run ahoy before anything else."], id["Ahoy is optional."]
	pairs, err := parseMatcherAnswer([]byte(`{"schema_version":1,"same":[["`+x+`","`+y+`"]]}`), req)
	if err != nil {
		t.Fatal(err)
	}
	if got := matcherAgreement(req, pairs, nil); got.Agreed.Estimable() {
		t.Fatalf("agreement with no labels = %+v; want not estimable", got)
	}
	sample := drawMatcherSample(req, pairs, 3, checkSeed)
	if len(sample) != 3 || !slices.ContainsFunc(sample, func(p [2]string) bool { return p == [2]string{x, y} || p == [2]string{y, x} }) {
		t.Fatalf("sample = %v; want three pairs, the matcher's own among them", sample)
	}
	labels, err := parseMatcherLabels([]byte(`{"schema_version":1,"labels":[`+
		`{"a":"`+x+`","b":"`+y+`","same":true},`+
		`{"a":"`+x+`","b":"`+z+`","same":false},`+
		`{"a":"`+y+`","b":"`+z+`","same":true}]}`), req)
	if err != nil {
		t.Fatalf("parseMatcherLabels: %v", err)
	}
	got := matcherAgreement(req, pairs, labels)
	if got.Agreed != (Rate{Hits: 2, Of: 3}) || got.Leftovers != 3 || got.Joined != 1 {
		t.Fatalf("agreement = %+v; want 2 of 3 over 3 leftovers, 1 pair joined", got)
	}
	for name, l := range map[string]string{
		"an unknown id": `{"schema_version":1,"labels":[{"a":"` + x + `","b":"m-zzzz","same":true}]}`,
		"a pair twice":  `{"schema_version":1,"labels":[{"a":"` + x + `","b":"` + y + `","same":true},{"a":"` + y + `","b":"` + x + `","same":false}]}`,
		"a self pair":   `{"schema_version":1,"labels":[{"a":"` + x + `","b":"` + x + `","same":true}]}`,
	} {
		if _, err := parseMatcherLabels([]byte(l), req); err == nil {
			t.Errorf("%s: the labels were accepted", name)
		}
	}
}

// checkVerdicts is the person's verdict file on every item of p: valid unless
// invalid names its code.
func checkVerdicts(t *testing.T, p CheckPacket, k CheckKey, locked bool, invalid map[string]bool) []byte {
	t.Helper()
	f := CheckVerdictFile{SchemaVersion: CheckJudgingSchema, Packet: k.Packet, Locked: locked}
	for _, it := range p.Items {
		f.Verdicts = append(f.Verdicts, CheckVerdict{Code: it.Code, Valid: !invalid[it.Code]})
	}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// The person judges the findings first, shuffled, with no model named and no
// proposed verdict; the proposed verdict and the overlap table appear only
// once their verdicts are locked (decision 27).
func TestTheOverlapTableAppearsOnlyAfterThePersonsVerdictsAreLocked(t *testing.T) {
	r, home, m, rec, store := checkLab(t)
	var copies []CheckCopy
	for _, cell := range []string{"host-big", "or-cheap"} {
		_, c, err := collect(t, r, m, rec, store, TrialKey{Task: checkTask, Cell: cell, Repeat: 1},
			dfAnswer(t, "The guard reads its registry from the plugin root.", "It refuses a hazard"))
		if err != nil {
			t.Fatal(err)
		}
		copies = append(copies, c)
	}
	p, k, err := drawCheckPacket(rec.RunID, checkTask, copies, matchExact(copies).rows(), checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 2 {
		t.Fatalf("packet = %+v; want the two distinct findings", p)
	}
	// Phase one names no cell, model, run, task, path or proposed verdict,
	// and nothing of the overlap.
	js, _ := json.Marshal(p)
	for _, leak := range []string{"host-big", "or-cheap", "openrouter", rec.RunID, `"` + checkTask + `"`, docfidelity.ChaptersDir,
		r.Root(), home, store, "valid", "proposed", "raised", "matched", "members"} {
		if bytes.Contains(js, []byte(leak)) {
			t.Errorf("the phase-one packet carries %q: %s", leak, js)
		}
	}
	if err := writeCheckJudging(r.Root(), m.ID, p, k); err != nil {
		t.Fatal(err)
	}
	tablePath := filepath.Join(runDir(home, m, rec.RunID), "check", checkTask, "table.json")
	proposal := func(valid bool) []byte {
		f := CheckProposalFile{SchemaVersion: CheckJudgingSchema, Packet: k.Packet}
		for _, it := range p.Items {
			f.Proposed = append(f.Proposed, CheckProposal{Code: it.Code, Valid: valid, Reason: "the code agrees"})
		}
		data, _ := json.Marshal(f)
		return data
	}
	if _, err := recordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, checkTableInputs{Proposals: proposal(true)}); err == nil ||
		!strings.Contains(err.Error(), "locked") {
		t.Fatalf("the table before the verdicts = %v; want a refusal saying they are not locked", err)
	}
	if _, err := os.Stat(tablePath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a table was written before the verdicts were locked: %v", err)
	}
	// Verdicts the session has not locked are not taken.
	if _, err := recordCheckVerdicts(r.Root(), m.ID, rec.RunID, checkTask, checkVerdicts(t, p, k, false, nil)); err == nil {
		t.Fatal("unlocked verdicts were recorded")
	}
	invalid := map[string]bool{p.Items[0].Code: true}
	if _, err := recordCheckVerdicts(r.Root(), m.ID, rec.RunID, checkTask, checkVerdicts(t, p, k, true, invalid)); err != nil {
		t.Fatalf("recordCheckVerdicts: %v", err)
	}
	// Locked means locked: a second file is never taken over the first.
	if _, err := recordCheckVerdicts(r.Root(), m.ID, rec.RunID, checkTask, checkVerdicts(t, p, k, true, nil)); err == nil {
		t.Fatal("a second verdict file replaced the locked one")
	}
	table, err := recordCheckTable(r.Root(), m.ID, rec.RunID, checkTask, checkTableInputs{Proposals: proposal(true)})
	if err != nil {
		t.Fatalf("recordCheckTable: %v", err)
	}
	if len(table.Rows) != 2 || !slices.Equal(table.Cells, []string{"host-big", "or-cheap"}) {
		t.Fatalf("table = %+v; want two rows and a column a cell", table)
	}
	for _, row := range table.Rows {
		if row.Proposed == nil || !row.Proposed.Valid || row.Valid == invalid[row.Code] || row.RaisedBy["host-big"] != 1 || row.RaisedBy["or-cheap"] != 1 {
			t.Errorf("row = %+v; want the person's verdict, the proposal and both cells", row)
		}
	}
	if _, err := os.Stat(tablePath); err != nil {
		t.Fatalf("the table is not recorded: %v", err)
	}
}

// A verdict or proposal file is untrusted input, refused whole when any part
// of it does not validate.
func TestACheckVerdictOrProposalFileIsRefusedWhole(t *testing.T) {
	copies := []CheckCopy{fidelityCopy(t, "big", 1, "It refuses a hazard", "The guard reads its registry from the plugin root.")}
	p, k, err := drawCheckPacket("mx-261023012949-ab12", checkTask, copies, matchExact(copies).rows(), checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	good := string(checkVerdicts(t, p, k, true, nil))
	if _, err := lockCheckVerdicts(p, k, []byte(good)); err != nil {
		t.Fatalf("a good file is refused: %v", err)
	}
	c0, c1 := p.Items[0].Code, p.Items[1].Code
	for name, raw := range map[string]string{
		"not JSON":       "{",
		"unlocked":       strings.Replace(good, `"locked":true`, `"locked":false`, 1),
		"an unknown key": strings.Replace(good, `"packet"`, `"model":"x","packet"`, 1),
		"another packet": strings.Replace(good, k.Packet, strings.Repeat("0", 64), 1),
		"a code missing": strings.Replace(good, `{"code":"`+c1+`","valid":true}`, ``, 1),
		"a code twice":   strings.Replace(good, `"code":"`+c1+`"`, `"code":"`+c0+`"`, 1),
		"a foreign code": strings.Replace(good, `"code":"`+c1+`"`, `"code":"ZZZZZ"`, 1),
		"too large":      good + strings.Repeat(" ", maxCheckVerdictBytes),
		"another schema": strings.Replace(good, `"schema_version":1`, `"schema_version":2`, 1),
	} {
		if _, err := lockCheckVerdicts(p, k, []byte(raw)); err == nil {
			t.Errorf("%s: the verdict file was accepted", name)
		}
	}
	prop := `{"schema_version":1,"packet":"` + k.Packet + `","proposed":[{"code":"` + c0 + `","valid":true,"reason":"ok"},{"code":"` + c1 + `","valid":false,"reason":"no"}]}`
	if _, err := parseCheckProposals(p, k, []byte(prop)); err != nil {
		t.Fatalf("a good proposal is refused: %v", err)
	}
	for name, raw := range map[string]string{
		"a code missing": strings.Replace(prop, `,{"code":"`+c1+`","valid":false,"reason":"no"}`, ``, 1),
		"a long reason":  strings.Replace(prop, `"reason":"ok"`, `"reason":"`+strings.Repeat("x", maxFindingBytes+1)+`"`, 1),
		"another packet": strings.Replace(prop, k.Packet, strings.Repeat("0", 64), 1),
	} {
		if _, err := parseCheckProposals(p, k, []byte(raw)); err == nil {
			t.Errorf("%s: the proposal was accepted", name)
		}
	}
}

// Each model's column: precision over what it raised, unique valid findings
// counted only for a model that was repeated, recall on the falsehoods seeded
// into the frozen material and against the findings judged valid, and how
// much each pair of models overlaps.
func TestEachModelIsScoredOnRecallOfSeededFalsehoodsAndAgainstTheValidFindings(t *testing.T) {
	const (
		registry = "The guard reads its registry from the plugin root."
		hazard   = "It refuses a hazard"
	)
	copies := []CheckCopy{
		fidelityCopy(t, "big", 1, registry, seededLine),
		fidelityCopy(t, "big", 2, registry, "The guard never logs"), // the seeded line, quoted in part
		fidelityCopy(t, "small", 1, hazard),
		fidelityCopy(t, "local", 1, registry, "public:docs/guide.md:Run ahoy first."),
		fidelityCopy(t, "local", 2, registry),
	}
	rows := matchExact(copies).rows()
	p, k, err := drawCheckPacket("mx-261023012949-ab12", checkTask, copies, rows, checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	codeOf := func(key TrialKey, n int) string {
		for _, kr := range k.Rows {
			for _, mem := range kr.Row.Members {
				if mem.Copy == key && mem.Finding == n {
					return kr.Code
				}
			}
		}
		t.Fatalf("no code for %s %d", key, n)
		return ""
	}
	// The registry and the seeded line are valid; the hazard and the guide's
	// line are not.
	invalid := map[string]bool{codeOf(copies[2].Key, 0): true, codeOf(copies[3].Key, 1): true}
	v, err := lockCheckVerdicts(p, k, checkVerdicts(t, p, k, true, invalid))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compareCheck(k, copies, LockedCheckVerdicts{}, nil, nil, nil); err == nil {
		t.Fatal("a table was compared with no locked verdicts")
	}
	seeded, err := locateSeeded(map[string]string{guardChapter: guardText}, []FindingEnd{{Path: guardChapter, Quote: seededLine}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := locateSeeded(map[string]string{guardChapter: guardText}, []FindingEnd{{Path: guardChapter, Quote: "not in the chapter"}}); err == nil {
		t.Fatal("a seeded falsehood the frozen material does not hold was accepted")
	}
	table, err := compareCheck(k, copies, v, nil, seeded, nil)
	if err != nil {
		t.Fatal(err)
	}
	score := map[string]CheckModelScore{}
	for _, s := range table.Scores {
		score[s.Cell] = s
	}
	big, small, local := score["big"], score["small"], score["local"]
	if big.RecallSeeded != (Rate{Hits: 2, Of: 2}) || small.RecallSeeded != (Rate{Hits: 0, Of: 1}) || local.RecallSeeded != (Rate{Hits: 0, Of: 2}) {
		t.Errorf("recall on the seeded falsehoods = %+v %+v %+v", big.RecallSeeded, small.RecallSeeded, local.RecallSeeded)
	}
	// Two valid rows: the registry and the seeded line.
	if big.RecallValid != (Rate{Hits: 4, Of: 4}) || small.RecallValid != (Rate{Hits: 0, Of: 2}) || local.RecallValid != (Rate{Hits: 2, Of: 4}) {
		t.Errorf("recall against the valid findings = %+v %+v %+v", big.RecallValid, small.RecallValid, local.RecallValid)
	}
	if big.Precision != (Rate{Hits: 4, Of: 4}) || small.Precision != (Rate{Hits: 0, Of: 1}) || local.Precision != (Rate{Hits: 2, Of: 3}) {
		t.Errorf("precision = %+v %+v %+v", big.Precision, small.Precision, local.Precision)
	}
	// Only big raised the seeded line, and big was repeated, so it counts.
	if big.UniqueValid != 1 || big.UniqueUncounted != 0 || small.UniqueValid != 0 || local.UniqueValid != 0 {
		t.Errorf("unique valid = %+v %+v %+v", big, small, local)
	}
	only := 0
	for _, row := range table.Rows {
		if row.OnlyModel != "" {
			only++
		}
	}
	if only != 3 {
		t.Errorf("rows only one model raised = %d; want 3 (the seeded line, the hazard, the guide's line): %+v", only, table.Rows)
	}
	overlap := map[[2]string]CheckOverlap{}
	for _, o := range table.Overlap {
		overlap[[2]string{o.A, o.B}] = o
	}
	if o := overlap[[2]string{"big", "local"}]; o.Both != 1 || o.Either != 3 {
		t.Errorf("big and local overlap = %+v; want 1 of 3", o)
	}
	if o := overlap[[2]string{"big", "small"}]; o.Both != 0 || o.Either != 3 {
		t.Errorf("big and small overlap = %+v; want 0 of 3", o)
	}

	// A model run once cannot have a unique finding counted: its one copy
	// may be noise.
	once := []CheckCopy{fidelityCopy(t, "big", 1, seededLine), fidelityCopy(t, "small", 1, registry)}
	p1, k1, err := drawCheckPacket("mx-261023012949-ab12", checkTask, once, matchExact(once).rows(), checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	v1, err := lockCheckVerdicts(p1, k1, checkVerdicts(t, p1, k1, true, nil))
	if err != nil {
		t.Fatal(err)
	}
	t1, err := compareCheck(k1, once, v1, nil, seeded, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range t1.Scores {
		if s.UniqueValid != 0 || s.UniqueUncounted != 1 {
			t.Errorf("%s run once: unique %d, uncounted %d; want 0 and 1", s.Cell, s.UniqueValid, s.UniqueUncounted)
		}
	}
}

// What the release gates read is snapshotted before a copy runs, not when its
// answer is collected, so a receipt, a review flag or a ledger entry the copy
// changes while it runs is seen: the copy keeps its clone, and the refusal
// names what changed. A collection with no snapshot is refused.
func TestTheGatesSnapshotBracketsTheCopysWholeRun(t *testing.T) {
	r, _, m, rec, store := checkLab(t)
	head := r.Git("rev-parse", "HEAD")
	receipt := filepath.Join(docfidelity.ReceiptsDir, head, docfidelity.GateName+".json")
	ledger := recordid.IssuesRelDir + "/open/iss-1.md"
	for rel, body := range map[string]string{
		receipt:               `{"verificationResult":"PROMOTE"}` + "\n",
		ledger:                "an open issue\n",
		docfidelity.FlagsPath: `{"schema_version":1,"flags":[]}` + "\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(r.Root(), rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(r.Root(), rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for i, rel := range []string{receipt, docfidelity.FlagsPath, ledger} {
		key := TrialKey{Task: checkTask, Cell: "host-big", Repeat: i + 1}
		// The run snapshots the gates, then starts the copy.
		gates, err := snapshotGates(r.Root())
		if err != nil {
			t.Fatal(err)
		}
		clone, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key)
		if err != nil {
			t.Fatal(err)
		}
		// While the copy runs, something rewrites what a gate reads.
		if err := os.WriteFile(filepath.Join(r.Root(), rel), []byte("changed while the copy ran\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err = collectCheckCopy(m.ID, checkInputs(t, r, rec, store, key, CheckDocFidelity, gates, dfAnswer(t, "It refuses a hazard")))
		if err == nil || !strings.Contains(err.Error(), filepath.ToSlash(rel)) {
			t.Errorf("%s changed while the copy ran: collectCheckCopy = %v; want a refusal naming it", rel, err)
		}
		if _, err := os.Stat(clone); err != nil {
			t.Errorf("%s changed while the copy ran, and the copy lost its clone: %v", rel, err)
		}
	}
	key := TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 1}
	if _, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key); err != nil {
		t.Fatal(err)
	}
	if _, err := collectCheckCopy(m.ID, checkInputs(t, r, rec, store, key, CheckDocFidelity, nil, dfAnswer(t, "It refuses a hazard"))); err == nil ||
		!strings.Contains(err.Error(), "snapshot") {
		t.Errorf("a collection with no snapshot taken before its copy ran = %v; want a refusal", err)
	}
}

// A copy cannot sign what it found: a model's id, its maker's name or the
// harness's, in a finding's class, summary, evidence or a quote the frozen
// material does not hold, is withheld before the record, the person's packet
// or the matcher's request carries it. The names are the run's own pinned
// models and the canonical lists (modelFamilies, developers, the AI names the
// attribution gate refuses), and a path the material does not hold that names
// one refuses the answer.
func TestACheckCopyCannotNameItsOwnModelToItsJudges(t *testing.T) {
	r, home, m, rec, store := checkLab(t, func(in *runInputs) { remodel(in, 1, "acme/zephyrine-7") })
	const (
		ownID  = "claude-opus-4-8"
		pinned = "zephyrine-7"
	)
	// Each name fused to a letter or a digit, or in another case, too.
	const fused = " xclaude Claudex OPUSx xopus anthropicx gpt5x xzephyrine-7 ZEPHYRINE-72"
	cons := `{"schema_version":1,"findings":[` +
		`{"class":"x` + ownID + `","summary":"found by ` + ownID + ` (Claude, Anthropic)` + fused + `","ends":[` +
		`{"path":"` + guardChapter + `","quote":"It refuses a hazard"},{"path":"docs/guide.md","quote":"signed ` + ownID + fused + `"}]},` +
		`{"class":"` + pinned + `","summary":"cell or-cheap on openrouter/acme/` + pinned + ` says so","ends":[` +
		`{"path":"` + guardChapter + `","quote":"The guard never logs a refusal."},{"path":"docs/guide.md","quote":"by ` + pinned + `"}]}]}`
	key := TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 1}
	_, c, err := collectForm(t, r, m, rec, store, key, CheckConsistency, []byte(cons))
	if err != nil {
		t.Fatalf("collectCheckCopy: %v", err)
	}
	if !c.Answered || len(c.Findings) != 2 {
		t.Fatalf("copy = %+v; want both findings answered", c)
	}
	// A quote the frozen material holds is the material's, kept as it is.
	if c.Findings[0].Ends[0].Quote != "It refuses a hazard" || c.Findings[0].Ends[0].Lines == nil {
		t.Errorf("a located quote = %+v; want it kept with its position", c.Findings[0].Ends[0])
	}
	df := strings.Replace(string(dfAnswer(t, "It refuses a hazard")), "internal/core/guard/registry.go:40 reads it elsewhere",
		ownID+" and gpt-5 and "+pinned+" read it elsewhere", 1)
	dfKey := TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 2}
	_, dc, err := collect(t, r, m, rec, store, dfKey, []byte(df))
	if err != nil {
		t.Fatalf("collectCheckCopy: %v", err)
	}
	copies := []CheckCopy{c, dc}
	p, _, err := drawCheckPacket(rec.RunID, checkTask, copies, matchExact(copies).rows(), checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	req, _, err := matchExact(copies).matcherRequest(checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	record, err := os.ReadFile(filepath.Join(runDir(home, m, rec.RunID), "check", checkTask, "copies", "or-cheap", "1.json"))
	if err != nil {
		t.Fatal(err)
	}
	pj, _ := json.Marshal(p)
	mj, _ := json.Marshal(req)
	dj, _ := json.Marshal(dc)
	for what, js := range map[string][]byte{"the record": record, "the packet": pj, "the matcher's request": mj, "the doc-fidelity copy": dj} {
		low := strings.ToLower(string(js))
		for _, leak := range []string{"claude", "opus", "anthropic", "zephyrine", "gpt-5", "gpt5"} {
			if strings.Contains(low, leak) {
				t.Errorf("%s names %q: %s", what, leak, js)
			}
		}
	}
	if c.Redacted == 0 || dc.Redacted == 0 {
		t.Errorf("redacted = %d, %d; want the withheld names counted", c.Redacted, dc.Redacted)
	}

	// A path the frozen material does not hold, named for a model, refuses
	// the answer: a path is the finding's place, never withheld in part.
	signed := `{"schema_version":1,"findings":[{"class":"contradiction","summary":"the two disagree","ends":[` +
		`{"path":"` + guardChapter + `","quote":"It refuses a hazard"},{"path":"notes/` + ownID + `.md","quote":"It never refuses."}]}]}`
	_, sc, err := collectForm(t, r, m, rec, store, TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 3}, CheckConsistency, []byte(signed))
	if err != nil {
		t.Fatalf("collectCheckCopy: %v", err)
	}
	if sc.Answered || len(sc.Findings) != 0 || strings.Contains(strings.ToLower(sc.Reason), "claude") {
		t.Errorf("a copy naming its model in a path = %+v; want it the copy's failure, the name not kept", sc)
	}
}

// The matcher is shown what a person labelling its pairs may see: each end's
// document by its file name, never its path in the repository.
func TestTheMatcherIsShownNoFullPath(t *testing.T) {
	// A HOLD names a brief sentence, so both copies name the one they share
	// by key, which is no leftover.
	const shared = "The guard reads its registry from the plugin root."
	copies := []CheckCopy{
		fidelityCopy(t, "big", 1, shared, "public:docs/guide.md:Run ahoy first."),
		fidelityCopy(t, "small", 1, shared, "It refuses a hazard"),
	}
	req, _, err := matchExact(copies).matcherRequest(checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Items) != 2 {
		t.Fatalf("matcher request = %+v; want both findings", req)
	}
	js, _ := json.Marshal(req)
	for _, leak := range []string{"docs/guide.md", docfidelity.ChaptersDir} {
		if bytes.Contains(js, []byte(leak)) {
			t.Errorf("the matcher's request carries %q: %s", leak, js)
		}
	}
	names := map[string]bool{}
	for _, it := range req.Items {
		for _, e := range it.Ends {
			if strings.Contains(e.Document, "/") {
				t.Errorf("the matcher is shown the path %q", e.Document)
			}
			names[e.Document] = true
		}
	}
	if !names["guide.md"] || !names[path.Base(guardChapter)] {
		t.Errorf("the matcher's documents = %v; want guide.md and %s", names, path.Base(guardChapter))
	}
}

// The run's own chapter of the frozen material: it names the grammar's words
// (host, local, provider), a route's vendor (openrouter), a harness version
// and a cell, so a quote of it that the material holds carries them, the same
// for every copy; and, on its fifth line, models and their makers.
const (
	routesChapter = docfidelity.ChaptersDir + "/18-routes.md"
	routesLine    = "The host reads the local registry before a provider or openrouter route runs, as harness 1.2.3 does for or-cheap."
	modelsLine    = "The guide was drafted with Claude Opus 5.5 and checked by GPT-4o from OpenAI."
	routesText    = "# Routes\n\n" + routesLine + "\n\n" + modelsLine + "\n"
)

// identityLab is a run of three cells, two of them named for their routes
// (host-big, or-cheap) and one by a two-byte id (lr), over the guard and
// routes chapters.
func identityLab(t *testing.T) (*gittest.Repo, string, lab.Minted, RunRecord, string) {
	t.Helper()
	return checkLabWith(t, map[string]string{routesChapter: routesText}, func(in *runInputs) {
		in.Declaration = []byte(strings.Replace(string(in.Declaration), `"id": "local-rev"`, `"id": "lr"`, 1))
		in.Cells[2].Cell = "lr"
	})
}

// cloneAlias is the alias copy key's clone was made under, read from the
// run's state as the test's own reading.
func cloneAlias(t *testing.T, home string, m lab.Minted, rec RunRecord, key TrialKey) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(runDir(home, m, rec.RunID), "aliases", key.Task, key.Cell, strconv.Itoa(key.Repeat)+".json"))
	if err != nil {
		t.Fatalf("copy %s's alias is not in the run's state: %v", key, err)
	}
	var a struct {
		Alias string `json:"alias"`
	}
	if err := json.Unmarshal(data, &a); err != nil || a.Alias == "" {
		t.Fatalf("copy %s's alias record = %s (%v)", key, data, err)
	}
	return a.Alias
}

// shownPath is each segment of the path a copy's clone is at under the
// store: what the copy is shown of where it runs, past the store every copy
// of the run shares.
func shownPath(t *testing.T, store, clone string) []string {
	t.Helper()
	rel, err := filepath.Rel(store, clone)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(filepath.ToSlash(rel), "/")
}

// leakShapes is name in every shape an ordinary id is joined in: fused to a
// digit or a letter on either side, joined by -, ., :, + or _, and in upper
// case.
func leakShapes(name string) []string {
	return []string{name + "1", "x" + name, name + "-2", name + "2", name + ".x", name + ":3", "+" + name, "_" + name + "_", strings.ToUpper(name)}
}

// signedWith is text a copy writes to sign what it found with each name, in
// every leak shape.
func signedWith(names ...string) string {
	var b strings.Builder
	for _, n := range names {
		for _, s := range leakShapes(n) {
			b.WriteString(s)
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// runNames is every name of the run a copy could hold of three bytes or
// more: the long cell ids, the task's and the other tasks', the run's, the
// routes' runner and provider, the pins' names, the cells' sources and the
// harness version.
var runNames = []string{"host-big", "or-cheap", "fidelity", "doc-page", "feature-a", "mx-261023012949-ab12",
	"opencode", "openrouter", "provider-a", "llama-server", "q4_k_m", "b6000", "host", "local", "qwen", "1.2.3"}

// noIdentity fails t where js names, in any case, any of the run's names or
// any of extra.
func noIdentity(t *testing.T, what string, js []byte, extra ...string) {
	t.Helper()
	low := strings.ToLower(string(js))
	for _, leak := range append(slices.Clone(runNames), extra...) {
		if strings.Contains(low, strings.ToLower(leak)) {
			t.Errorf("%s names %q: %s", what, leak, js)
		}
	}
}

// consistencyJSON is a consistency answer of the given findings.
func consistencyJSON(t *testing.T, findings ...map[string]any) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{"schema_version": 1, "findings": findings})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func consistencyRow(class, summary string, ends ...[2]string) map[string]any {
	var es []map[string]string
	for _, e := range ends {
		es = append(es, map[string]string{"path": e[0], "quote": e[1]})
	}
	return map[string]any{"class": class, "summary": summary, "ends": es}
}

// A check copy is never shown its cell's or its task's id: its clone is made
// under an alias minted for it at random when it starts and recorded in the
// run's state, so the path it works in, past the store every copy of the run
// shares, is the run's id and the alias alone. The alias is never derived
// from the copy's key, so echoing it tells a judge nothing; a copy whose
// clone was not made takes the alias recorded for it again.
func TestACheckCopyIsShownNoCellOrTaskID(t *testing.T) {
	r, home, m, rec, store := identityLab(t)
	seen := map[string]TrialKey{}
	for _, cell := range []string{"or-cheap", "lr", "host-big"} {
		for repeat := 1; repeat <= 2; repeat++ {
			key := TrialKey{Task: checkTask, Cell: cell, Repeat: repeat}
			clone, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key)
			if err != nil {
				t.Fatalf("makeCopyClone(%s): %v", key, err)
			}
			alias := cloneAlias(t, home, m, rec, key)
			if !regexp.MustCompile(`^copy_[0-9a-f]{32}$`).MatchString(alias) {
				t.Errorf("copy %s's alias is %q; want an opaque token", key, alias)
			}
			if got := shownPath(t, store, clone); !slices.Equal(got, []string{rec.RunID, alias}) {
				t.Errorf("copy %s works at %v under the store; want the run's id and its alias alone", key, got)
			}
			for _, id := range []string{cell, checkTask, strconv.Itoa(repeat) + "/"} {
				if strings.Contains(alias, id) {
					t.Errorf("copy %s's alias %q holds %q", key, alias, id)
				}
			}
			if other, dup := seen[alias]; dup {
				t.Errorf("copies %s and %s share the alias %q", other, key, alias)
			}
			seen[alias] = key
		}
	}

	// A clone that was not made is made again under the alias its copy was
	// given; one that is there is never made twice.
	key := TrialKey{Task: checkTask, Cell: "lr", Repeat: 1}
	alias := cloneAlias(t, home, m, rec, key)
	if _, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key); err == nil {
		t.Error("a copy's clone was made twice")
	}
	if err := os.RemoveAll(filepath.Join(store, rec.RunID, alias)); err != nil {
		t.Fatal(err)
	}
	again, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key)
	if err != nil || again != filepath.Join(store, rec.RunID, alias) {
		t.Errorf("the clone made again = %q, %v; want it under the alias %s", again, err, alias)
	}
}

// A copy cannot sign what it found with where it runs or what it is: neither
// what it is shown of its clone's path (the run's id and its alias, whole, in
// part or in another case) nor any long name of the run that could still
// reach it (a cell's id, a task's, a route's runner or provider, a pin's
// name), fused to a letter or a digit or joined by a punctuation mark, in a
// consistency finding's class, summary or quote the frozen material does not
// hold, reaches the record, the person's packet or the matcher's request. A
// class naming one is withheld; a path the frozen material does not hold that
// names one refuses the answer.
func TestACheckCopyCannotNameItsCellTaskOrRouteToItsJudges(t *testing.T) {
	r, _, m, rec, store := identityLab(t)
	gates, err := snapshotGates(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	var copies []CheckCopy
	var shown []string
	for i, cell := range []string{"or-cheap", "lr", "host-big"} {
		key := TrialKey{Task: checkTask, Cell: cell, Repeat: 1}
		clone, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key)
		if err != nil {
			t.Fatal(err)
		}
		parts := shownPath(t, store, clone)
		// Each copy words its quotes its own way, so the matcher is shown
		// them.
		own := []string{"first", "second", "third"}[i]
		rel := strings.Join(parts, "/")
		last := parts[len(parts)-1]
		shown = append(shown, parts...)
		sign := signedWith(append(slices.Clone(runNames), parts...)...) +
			"wrote " + rel + "/docs/x.md, " + last + "/docs, " + strings.ToUpper(rel) + " and " + clone + "/notes.md"
		// A quote is bounded tighter than a summary, so it is signed with the
		// clone's path and the cells' ids alone.
		quote := own + " signed " + signedWith(append(slices.Clone(parts), "or-cheap", "host-big")...) + rel
		ans := consistencyJSON(t,
			consistencyRow("xor-cheap1", sign, [2]string{guardChapter, "It refuses a hazard"}, [2]string{"docs/guide.md", quote}),
			consistencyRow("x"+asciiLower(last)+"-2", "it drifts: "+sign,
				[2]string{guardChapter, "The guard reads its registry from the plugin root."}, [2]string{"docs/guide.md", own + " " + rel + "/docs/x.md says so"}),
			consistencyRow("fidelity2", "it disagrees", [2]string{guardChapter, "The guard never logs a refusal."},
				[2]string{guardChapter, "before the command runs."}),
		)
		c, err := collectCheckCopy(m.ID, checkInputs(t, r, rec, store, key, CheckConsistency, gates, ans))
		if err != nil {
			t.Fatalf("collectCheckCopy(%s): %v", key, err)
		}
		if !c.Answered || len(c.Findings) != 3 {
			t.Fatalf("copy %s = %+v; want its three findings answered", key, c)
		}
		for i, f := range c.Findings {
			if f.Class != withheldClass {
				t.Errorf("copy %s's finding %d's class is %q; want it withheld", key, i+1, f.Class)
			}
		}
		if c.Redacted == 0 {
			t.Errorf("copy %s withheld nothing", key)
		}
		copies = append(copies, c)
	}
	p, _, err := drawCheckPacket(rec.RunID, checkTask, copies, matchExact(copies).rows(), checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	req, _, err := matchExact(copies).matcherRequest(checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) == 0 || len(req.Items) == 0 {
		t.Fatalf("packet %d items, matcher %d; want both shown findings", len(p.Items), len(req.Items))
	}
	pj, _ := json.Marshal(p)
	mj, _ := json.Marshal(req)
	noIdentity(t, "the packet", pj, shown...)
	noIdentity(t, "the matcher's request", mj, shown...)
	for _, cc := range copies {
		fj, _ := json.Marshal(cc.Findings)
		noIdentity(t, "copy "+cc.Key.String()+"'s findings", fj, shown...)
	}

	// A path the frozen material does not hold that names the copy's clone,
	// one of the run's ids or a pin, fused or not, refuses the answer, and the
	// refusal names none of them.
	for i, sign := range []func(run, alias string) string{
		func(run, alias string) string { return run + "/" + alias + "/docs/x.md" },
		func(_, alias string) string { return "docs/" + strings.ToUpper(alias) + ".md" },
		func(string, string) string { return "notes/xhost-big2.md" },
		func(string, string) string { return "docs/opencode1.md" },
	} {
		key := TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 2 + i}
		clone, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key)
		if err != nil {
			t.Fatal(err)
		}
		parts := shownPath(t, store, clone)
		signed := sign(parts[0], parts[len(parts)-1])
		shown = append(shown, parts...)
		ans := consistencyJSON(t, consistencyRow("contradiction", "the two disagree",
			[2]string{guardChapter, "It refuses a hazard"}, [2]string{signed, "It never refuses."}))
		sc, err := collectCheckCopy(m.ID, checkInputs(t, r, rec, store, key, CheckConsistency, gates, ans))
		if err != nil {
			t.Fatalf("collectCheckCopy: %v", err)
		}
		if sc.Answered || len(sc.Findings) != 0 {
			t.Errorf("a copy naming %s = %+v; want it the copy's failure", signed, sc)
		}
		noIdentity(t, "the refusal of "+signed, []byte(sc.Reason), shown...)
	}
}

// A quote the frozen material holds is located and verified against the
// material as written, and shown with every model and maker withheld, as the
// copy's own words are (dcn-2611211626560528 corrects dcn-2611210903322839 in
// part): a copy chooses which line it quotes, so a quote of a line that names
// a model is a signature by choice. The rest of the line is kept: the
// grammar's words (host, local, provider), a harness version and even a
// cell's id stay in it, in the record, the person's packet and the matcher's
// request, at the lines the material holds it; a route's vendor is withheld.
func TestALocatedQuoteWithholdsModelsAndMakersAndStaysLocated(t *testing.T) {
	r, _, m, rec, store := identityLab(t)
	wantRoutes := strings.Replace(routesLine, "openrouter", withheldMark, 1)
	wantModels := "The guide was drafted with " + withheldMark + " " + withheldMark + " and checked by " + withheldMark + " from " + withheldMark + "."
	var copies []CheckCopy
	for _, x := range []struct{ cell, class string }{{"or-cheap", "contradiction"}, {"lr", "drift"}} {
		cons := consistencyJSON(t, consistencyRow(x.class, "the routes chapter disagrees with itself",
			[2]string{routesChapter, routesLine}, [2]string{routesChapter, modelsLine}))
		_, c, err := collectForm(t, r, m, rec, store, TrialKey{Task: checkTask, Cell: x.cell, Repeat: 1}, CheckConsistency, cons)
		if err != nil || !c.Answered {
			t.Fatalf("collectCheckCopy = %+v, %v", c, err)
		}
		copies = append(copies, c)
	}
	_, dc, err := collect(t, r, m, rec, store, TrialKey{Task: checkTask, Cell: "host-big", Repeat: 1},
		dfAnswerIn(t, path.Base(routesChapter), routesLine, modelsLine))
	if err != nil || !dc.Answered {
		t.Fatalf("collectCheckCopy = %+v, %v", dc, err)
	}
	shownAs := func(what string, quote string, lines *LineSpan) {
		t.Helper()
		want, line := wantRoutes, 3
		if strings.Contains(quote, "guide") {
			want, line = wantModels, 5
		}
		if quote != want {
			t.Errorf("%s quotes %q; want %q", what, quote, want)
		}
		if lines != nil && (lines.Start != line || lines.End != line) {
			t.Errorf("%s is at %+v; want line %d", what, lines, line)
		}
	}
	for _, c := range append(slices.Clone(copies), dc) {
		var ends []FindingEnd
		for _, f := range c.Findings {
			ends = append(ends, f.Ends...)
		}
		if len(ends) != 2 {
			t.Fatalf("copy %s's ends = %+v; want two", c.Key, ends)
		}
		for _, e := range ends {
			if e.Lines == nil {
				t.Errorf("copy %s's quote %q of the routes chapter is not located", c.Key, e.Quote)
			}
			shownAs("copy "+c.Key.String()+"'s record", e.Quote, e.Lines)
		}
	}
	for _, set := range [][]CheckCopy{copies, {dc}} {
		p, _, err := drawCheckPacket(rec.RunID, checkTask, set, matchExact(set).rows(), checkSeed)
		if err != nil {
			t.Fatal(err)
		}
		req, _, err := matchExact(set).matcherRequest(checkSeed)
		if err != nil {
			t.Fatal(err)
		}
		shown := 0
		for _, it := range p.Items {
			for _, e := range it.Ends {
				if e.Document == path.Base(routesChapter) {
					shownAs("the packet", e.Quote, e.Lines)
					shown++
				}
			}
		}
		for _, it := range req.Items {
			for _, e := range it.Ends {
				if e.Document == path.Base(routesChapter) {
					shownAs("the matcher's request", e.Quote, nil)
					shown++
				}
			}
		}
		if shown < 2 {
			t.Errorf("the routes chapter's quotes are shown %d times in the packet and the matcher's request; want both", shown)
		}
		js, _ := json.Marshal(p)
		for _, leak := range []string{"Claude", "Opus", "GPT", "OpenAI", "openrouter"} {
			if bytes.Contains(js, []byte(leak)) {
				t.Errorf("the packet carries %q: %s", leak, js)
			}
		}
	}
}

// The same holds for every other field of a copy's output: a doc-fidelity
// copy's quoted sentence the material does not hold and its evidence, the
// chapter it names, and the reason a copy failed.
func TestADocFidelityCopyOrAFailedCopyCannotNameItsCellTaskOrRoute(t *testing.T) {
	r, _, m, rec, store := identityLab(t)
	gates, err := snapshotGates(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	key := TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 1}
	clone, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key)
	if err != nil {
		t.Fatal(err)
	}
	parts := shownPath(t, store, clone)
	sign := signedWith(append(slices.Clone(runNames), parts...)...) + "at " + strings.Join(parts, "/") + "/docs/guide.md"
	// A quoted sentence is bounded tighter than evidence, so it is signed with
	// the clone's path and the cells' ids alone.
	sentence := signedWith(append(slices.Clone(parts), "or-cheap", "host-big")...) + strings.Join(parts, "/")
	df := strings.Replace(string(dfAnswer(t, "It refuses a hazard", sentence)),
		"internal/core/guard/registry.go:40 reads it elsewhere", sign, -1)
	dc, err := collectCheckCopy(m.ID, checkInputs(t, r, rec, store, key, CheckDocFidelity, gates, []byte(df)))
	if err != nil {
		t.Fatalf("collectCheckCopy: %v", err)
	}
	if !dc.Answered || len(dc.Findings) != 2 {
		t.Fatalf("copy = %+v; want its two findings answered", dc)
	}
	_, dc2, err := collect(t, r, m, rec, store, TrialKey{Task: checkTask, Cell: "host-big", Repeat: 1},
		dfAnswer(t, "The guard never logs a refusal."))
	if err != nil {
		t.Fatal(err)
	}
	copies := []CheckCopy{dc, dc2}
	p, _, err := drawCheckPacket(rec.RunID, checkTask, copies, matchExact(copies).rows(), checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	req, _, err := matchExact(copies).matcherRequest(checkSeed)
	if err != nil {
		t.Fatal(err)
	}
	if len(req.Items) == 0 {
		t.Fatal("the matcher is shown nothing; want the leftovers")
	}
	pj, _ := json.Marshal(p)
	mj, _ := json.Marshal(req)
	fj, _ := json.Marshal(dc.Findings)
	noIdentity(t, "the packet", pj, parts...)
	noIdentity(t, "the matcher's request", mj, parts...)
	noIdentity(t, "the copy's findings", fj, parts...)

	// A chapter the material does not hold, named for the copy's clone.
	public := dfAnswer(t, "It refuses a hazard", "public:"+strings.Join(parts, "/")+"/docs/guide.md:Run ahoy first.")
	_, pc, err := collect(t, r, m, rec, store, TrialKey{Task: checkTask, Cell: "or-cheap", Repeat: 2}, public)
	if err != nil {
		t.Fatal(err)
	}
	if pc.Answered || !strings.Contains(pc.Reason, "does not hold names") {
		t.Errorf("a copy naming its clone's path as a chapter = %+v; want it the copy's failure", pc)
	}
	noIdentity(t, "the refusal of a chapter", []byte(pc.Reason), parts...)

	// A copy whose runner failed is recorded with the runner's words, which
	// name nothing of the run either.
	fkey := TrialKey{Task: checkTask, Cell: "lr", Repeat: 1}
	fclone, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, fkey)
	if err != nil {
		t.Fatal(err)
	}
	fparts := shownPath(t, store, fclone)
	in := checkInputs(t, r, rec, store, fkey, CheckDocFidelity, gates, nil)
	in.Failure = signedWith(append(slices.Clone(runNames), fparts...)...) + " in " + fclone
	fc, err := collectCheckCopy(m.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if fc.Answered || fc.Reason == "" {
		t.Fatalf("failed copy = %+v; want its failure's reason", fc)
	}
	noIdentity(t, "a failed copy's reason", []byte(fc.Reason), fparts...)
}

// modelListDoc is a material file whose lines differ only in the model they
// name, so a quote of each is shown alike once the model is withheld.
const (
	modelListDoc  = "docs/reference/models.md"
	modelListText = "# Models\n\n- opus: reads the brief\n- sonnet: reads the brief\n"
)

// modelListCopy reads one consistency copy on cell quoting quote of the
// model list against its first line.
func modelListCopy(t *testing.T, cell, quote string) CheckCopy {
	t.Helper()
	return readCheckCopy(checkCopyInputs{
		RepoRoot: "/work/checkout", Store: "/work/store", Run: "mx-261023012949-ab12",
		Key: TrialKey{Task: checkTask, Cell: cell, Repeat: 1}, Form: CheckConsistency,
		Material: map[string]string{modelListDoc: modelListText},
		Answer: consistencyJSON(t, consistencyRow("drift", "the list disagrees with itself",
			[2]string{modelListDoc, quote}, [2]string{modelListDoc, "# Models"})),
	})
}

// A located quote is keyed on the line as the material holds it, and shown
// with every model and maker withheld: two copies quoting two lines that
// differ only in the model they name are two findings, never joined by their
// key as the copies agreeing, though both are shown alike and the path and
// line span still name each line. A quote the material does not hold is the
// copy's own words, keyed as it is shown.
func TestLocatedQuotesThatDifferOnlyInAModelKeepTheirOwnKeys(t *testing.T) {
	a := modelListCopy(t, "big", "- opus: reads the brief")
	b := modelListCopy(t, "cheap", "- sonnet: reads the brief")
	shown := "- " + withheldMark + ": reads the brief"
	for i, c := range []CheckCopy{a, b} {
		if !c.Answered || len(c.Findings) != 1 {
			t.Fatalf("copy %s = %+v; want one finding", c.Key, c)
		}
		var end FindingEnd
		for _, e := range c.Findings[0].Ends {
			if e.Quote != "# Models" {
				end = e
			}
		}
		if end.Quote != shown {
			t.Errorf("copy %s shows %q; want %q", c.Key, end.Quote, shown)
		}
		if want := 3 + i; end.Lines == nil || end.Lines.Start != want || end.Lines.End != want {
			t.Errorf("copy %s's quote is at %+v; want line %d", c.Key, end.Lines, want)
		}
	}
	if a.Findings[0].Key == b.Findings[0].Key {
		t.Errorf("two located lines that differ only in their model share the key %s", a.Findings[0].Key)
	}
	if rows := matchExact([]CheckCopy{a, b}).rows(); len(rows) != 2 {
		t.Errorf("exact matching joined two different lines: %+v; want two rows", rows)
	}
	same := modelListCopy(t, "cheap", "- opus: reads the brief")
	if same.Findings[0].Key != a.Findings[0].Key {
		t.Errorf("two copies quoting the same line have keys %s and %s; want one", a.Findings[0].Key, same.Findings[0].Key)
	}
	ua := modelListCopy(t, "big", "written by opus")
	ub := modelListCopy(t, "cheap", "written by sonnet")
	if ua.Findings[0].Key != ub.Findings[0].Key {
		t.Errorf("two unlocated quotes shown alike have keys %s and %s; want them keyed as shown", ua.Findings[0].Key, ub.Findings[0].Key)
	}
}

// A check copy's words are scrubbed before their white space is collapsed to
// one line, so a name broken across lines, by a line feed or a carriage
// return and line feed, is withheld from a check copy's summary and from a
// quote of its own, as it is from a writing copy's text.
func TestANameBrokenAcrossLinesIsWithheldFromACheckCopysWords(t *testing.T) {
	for _, nl := range []string{"\n", "\r\n"} {
		signed := "Found by Clau" + nl + "de in the list."
		want := "Found by " + withheldMark + " in the list."
		c := readCheckCopy(checkCopyInputs{
			RepoRoot: "/work/checkout", Store: "/work/store", Run: "mx-261023012949-ab12",
			Key: TrialKey{Task: checkTask, Cell: "big", Repeat: 1}, Form: CheckConsistency,
			Material: map[string]string{modelListDoc: modelListText},
			Answer: consistencyJSON(t, consistencyRow("drift", signed,
				[2]string{modelListDoc, signed}, [2]string{modelListDoc, "# Models"})),
		})
		if !c.Answered || len(c.Findings) != 1 {
			t.Fatalf("copy = %+v; want one finding", c)
		}
		if got := c.Findings[0].Evidence; got != want {
			t.Errorf("a summary broken by %q = %q; want %q", nl, got, want)
		}
		quoted := false
		for _, e := range c.Findings[0].Ends {
			if e.Lines == nil {
				quoted = true
				if e.Quote != want {
					t.Errorf("an unlocated quote broken by %q = %q; want %q", nl, e.Quote, want)
				}
			}
		}
		if !quoted {
			t.Errorf("the copy's own quote is located: %+v", c.Findings[0].Ends)
		}
		if got, _ := scrubFor(writingInputs(1, 1, 1)).scrub(signed); got != "Found by "+withheldMark+" in the list." {
			t.Errorf("a writing copy's text broken by %q = %q; want the name withheld", nl, got)
		}
	}
	// Prose wrapped between two short words stays prose either way.
	for _, nl := range []string{"\n", "\r\n"} {
		prose := "I met" + nl + "a friend."
		if got, n := scrubFor(writingInputs(1, 1, 1)).scrub(prose); n != 0 {
			t.Errorf("prose wrapped by %q = %q; want it kept", nl, got)
		}
	}
}
