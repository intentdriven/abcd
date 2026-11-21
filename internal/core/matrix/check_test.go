package matrix

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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
		f := failing{Disposition: "confirmed", Doc: "brief", Chapter: "17-guard.md", Sentence: s,
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
// at its pin, and returns the source, the home, the lab, the record and a clone
// store outside every working tree.
func checkLab(t *testing.T) (*gittest.Repo, string, lab.Minted, RunRecord, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	r := gittest.NewRepo(t)
	r.Write("README.md", "root\n")
	r.Write(guardChapter, guardText)
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
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(r.Root(), rec, in.Declaration); err != nil {
		t.Fatal(err)
	}
	return r, home, m, rec, filepath.Join(t.TempDir(), "store")
}

// collect makes copy key's clone and collects its answer, as the run does once
// the copy's runner is done.
func collect(t *testing.T, r *gittest.Repo, m lab.Minted, rec RunRecord, store string, key TrialKey, answer []byte) (string, CheckCopy, error) {
	t.Helper()
	clone, err := makeClone(r.Root(), store, rec.RunID, rec.FrozenCommit, key)
	if err != nil {
		t.Fatal(err)
	}
	material, err := readFrozenMaterial(context.Background(), r.Root(), rec.FrozenCommit, []string{docfidelity.ChaptersDir})
	if err != nil {
		t.Fatal(err)
	}
	c, err := collectCheckCopy(m.ID, checkCopyInputs{RepoRoot: r.Root(), Store: store, Run: rec.RunID, Frozen: rec.FrozenCommit,
		Key: key, Form: CheckDocFidelity, Material: material, Answer: answer})
	return clone, c, err
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
	clone2, err := makeClone(r.Root(), store, rec.RunID, rec.FrozenCommit, key2)
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
	material := map[string]string{guardChapter: guardText}
	_, err = collectCheckCopy(m.ID, checkCopyInputs{RepoRoot: r.Root(), Store: store, Run: rec.RunID, Frozen: rec.FrozenCommit,
		Key: key2, Form: CheckDocFidelity, Material: material, Answer: dfAnswer(t, "It refuses a hazard")})
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
