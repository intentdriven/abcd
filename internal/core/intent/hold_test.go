package intent

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/lint"
)

// TestHoldWritesTheKeyAndUnholdRemovesIt is the verb pair's contract
// (iss-2609200830076665): hold writes `held: "<reason>"` — quoted, trimmed,
// nothing else in the file touched — the loader reads it back, Plan refuses
// it, unhold removes the line and reports the reason it lifted, and the record
// then plans.
func TestHoldWritesTheKeyAndUnholdRemovesIt(t *testing.T) {
	root := t.TempDir()
	rel := draftsDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, draftWithAC("itd-10", "alpha"))
	before := readIntent(t, root, rel)

	res, err := Hold(root, "itd-10", "  awaiting the reading rethink  ")
	if err != nil {
		t.Fatal(err)
	}
	if res.IntentID != "itd-10" || res.Bucket != BucketDrafts || res.Reason != "awaiting the reading rethink" || res.Redacted != 0 {
		t.Fatalf("HoldResult = %+v", res)
	}
	after := readIntent(t, root, rel)
	if !containsLine(frontmatterLines(t, after), `held: "awaiting the reading rethink"`) {
		t.Fatalf("hold must write the quoted single-line key:\n%s", after)
	}
	// The body and every other key are untouched: removing the one line gives
	// the file back byte for byte, which is the write's whole footprint.
	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	it, _ := c.Lookup("itd-10")
	if it.Held != "awaiting the reading rethink" || it.HeldMalformed {
		t.Fatalf("loader must read the hold back: %+v", it)
	}
	if _, err := Plan(root, "itd-10", ""); err == nil || !strings.Contains(err.Error(), "intent unhold itd-10") {
		t.Fatalf("Plan must refuse the held draft naming the remedy: %v", err)
	}

	un, err := Unhold(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if un.Reason != "awaiting the reading rethink" || un.Bucket != BucketDrafts {
		t.Fatalf("UnholdResult = %+v", un)
	}
	if got := readIntent(t, root, rel); got != before {
		t.Fatalf("unhold must remove exactly the line hold wrote:\n--- before\n%s\n--- after\n%s", before, got)
	}
	if _, err := Plan(root, "itd-10", ""); err != nil {
		t.Fatalf("the record plans once the hold is lifted: %v", err)
	}
}

// TestHoldRefusesWhatItCannotMean: no reason, an already-held record (naming
// the standing reason), a terminal bucket, a multi-line reason, and an id that
// resolves to nothing — each refused with nothing written.
func TestHoldRefusesWhatItCannotMean(t *testing.T) {
	root := t.TempDir()
	draft := draftsDir + "/itd-10-alpha.md"
	writeFile(t, root, draft, draftWithAC("itd-10", "alpha"))
	writeFile(t, root, shippedDir+"/itd-3-done.md",
		"---\nid: itd-3\nslug: done\nspec_id: spc-1\nkind: standalone\nimpact: additive\n---\n# done\n")
	writeFile(t, root, ".abcd/development/intents/disciplines/itd-1-rule.md",
		"---\nid: itd-1\nslug: rule\nspec_id: null\nkind: discipline\n---\n# rule\n")
	writeFile(t, root, ".abcd/development/intents/superseded/itd-2-old.md",
		"---\nid: itd-2\nslug: old\nspec_id: null\nkind: standalone\nsuperseded_by: itd-3\n---\n# old\n")
	snapshot := func() map[string]string {
		out := map[string]string{}
		for _, rel := range []string{draft, shippedDir + "/itd-3-done.md", ".abcd/development/intents/disciplines/itd-1-rule.md", ".abcd/development/intents/superseded/itd-2-old.md"} {
			out[rel] = readIntent(t, root, rel)
		}
		return out
	}
	before := snapshot()

	for name, tc := range map[string]struct {
		id, reason, want string
	}{
		"empty reason":        {"itd-10", "   ", "--reason is required"},
		"multi-line reason":   {"itd-10", "one\ntwo", "single line"},
		"tab in reason":       {"itd-10", "one\ttwo", "single line"},
		"shipped record":      {"itd-3", "why", "shipped"},
		"discipline record":   {"itd-1", "why", "disciplines"},
		"superseded record":   {"itd-2", "why", "superseded"},
		"unknown record":      {"itd-99", "why", "not found"},
		"malformed id":        {"../x", "why", "must match"},
		"whitespace after id": {"itd-10 ", "why", "must match"},
	} {
		_, err := Hold(root, tc.id, tc.reason)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: want a refusal carrying %q, got %v", name, tc.want, err)
		}
	}
	if after := snapshot(); len(after) != len(before) {
		t.Fatal("snapshot size changed")
	} else {
		for rel, b := range before {
			if after[rel] != b {
				t.Errorf("%s was written by a refused hold", rel)
			}
		}
	}

	// Already held: refused naming the standing reason; the file is not rewritten.
	if _, err := Hold(root, "itd-10", "first reason"); err != nil {
		t.Fatal(err)
	}
	held := readIntent(t, root, draft)
	_, err := Hold(root, "itd-10", "second reason")
	if err == nil || !strings.Contains(err.Error(), "first reason") || !strings.Contains(err.Error(), "intent unhold itd-10") {
		t.Fatalf("a second hold is refused naming the standing reason and the lift: %v", err)
	}
	if readIntent(t, root, draft) != held {
		t.Fatal("a refused re-hold must not rewrite the record")
	}
}

// TestUnholdRefusesWhatItDidNotWrite: a record not held, a terminal record,
// and a `held` value in a shape no verb writes are each refused with nothing
// written — the verb removes only what it could have written, and the malformed
// case is sent to a hand repair by name.
func TestUnholdRefusesWhatItDidNotWrite(t *testing.T) {
	root := t.TempDir()
	free := draftsDir + "/itd-10-alpha.md"
	writeFile(t, root, free, draftWithAC("itd-10", "alpha"))
	bad := draftsDir + "/itd-11-bad.md"
	writeFile(t, root, bad, "---\nid: itd-11\nslug: bad\nspec_id: null\nkind: null\nheld: |\n  two\n  lines\n---\n# bad\n")
	shipped := shippedDir + "/itd-3-done.md"
	writeFile(t, root, shipped, "---\nid: itd-3\nslug: done\nspec_id: spc-1\nkind: standalone\nimpact: additive\nheld: \"forged by hand\"\n---\n# done\n")
	before := map[string]string{free: readIntent(t, root, free), bad: readIntent(t, root, bad), shipped: readIntent(t, root, shipped)}

	for name, tc := range map[string]struct{ id, want string }{
		"not held":       {"itd-10", "not held"},
		"malformed hold": {"itd-11", "record_provenance"},
		"shipped record": {"itd-3", "shipped"},
		"unknown":        {"itd-99", "not found"},
	} {
		_, err := Unhold(root, tc.id)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: want a refusal carrying %q, got %v", name, tc.want, err)
		}
	}
	for rel, b := range before {
		if readIntent(t, root, rel) != b {
			t.Errorf("%s was written by a refused unhold", rel)
		}
	}
	// A malformed hold stops hold and plan too — fail closed: the key's presence
	// is somebody's attempt at a hold, whatever its shape.
	if _, err := Hold(root, "itd-11", "why"); err == nil || !strings.Contains(err.Error(), "record_provenance") {
		t.Errorf("hold over a malformed value must send the caller to the line: %v", err)
	}
	if _, err := Plan(root, "itd-11", ""); err == nil || !strings.Contains(err.Error(), "shape no verb writes") {
		t.Errorf("plan over a malformed value must refuse, fail closed: %v", err)
	}
}

// TestHoldRedactsTheReason: a hold reason is durable committed prose, so it
// goes through the store's canonical redactor before the write and the result
// says so — a secret handed to `--reason` never reaches the record verbatim.
func TestHoldRedactsTheReason(t *testing.T) {
	root := t.TempDir()
	rel := draftsDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, draftWithAC("itd-10", "alpha"))
	secret := "ghp_" + strings.Repeat("A", 36)
	res, err := Hold(root, "itd-10", "blocked until "+secret+" is rotated")
	if err != nil {
		t.Fatal(err)
	}
	if res.Redacted == 0 {
		t.Fatalf("the redactor must report the span it rewrote: %+v", res)
	}
	after := readIntent(t, root, rel)
	if strings.Contains(after, secret) || strings.Contains(res.Reason, secret) {
		t.Fatalf("the secret reached the record or the result:\n%s\n%+v", after, res)
	}
	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if it, _ := c.Lookup("itd-10"); it.Held != res.Reason || it.Held == "" {
		t.Fatalf("the redacted reason is what the loader reads back: %+v vs %+v", it, res)
	}
}

// TestHoldWriteStaysLintValid: a held record — in drafts/ and in planned/ —
// passes the intent_lifecycle and record_provenance rules through the real
// lint engine, so `abcd intent hold` never leaves the tree red. The
// frontmatter round-trips through the shared codec so an escaped reason is read
// back as typed.
func TestHoldWriteStaysLintValid(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-10-alpha.md", draftWithAC("itd-10", "alpha"))
	writeFile(t, root, plannedDir+"/itd-2-beta.md",
		"---\nid: itd-2\nslug: beta\nspec_id: spc-1\nkind: standalone\n---\n# beta\n")
	if _, err := Hold(root, "itd-10", `say "why" \ and mean it`); err != nil {
		t.Fatal(err)
	}
	if _, err := Hold(root, "itd-2", "scope under review"); err != nil {
		t.Fatal(err)
	}
	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if it, _ := c.Lookup("itd-10"); it.Held != `say "why" \ and mean it` {
		t.Fatalf("escaped reason must read back as typed: %q", it.Held)
	}
	fields := frontmatter.Fields(strings.Split(readIntent(t, root, draftsDir+"/itd-10-alpha.md"), "\n"))
	if got := fields[HeldKey].Value; got != frontmatter.QuoteScalar(`say "why" \ and mean it`) {
		t.Fatalf("the written value is the shared encoder's: %q", got)
	}

	cfg := lint.Config{
		Roots: []string{".abcd/development"},
		Rules: map[string]lint.RuleConfig{
			"intent_lifecycle":  {Enabled: true, Severity: "blocker", IntentsDir: "intents"},
			"record_provenance": {Enabled: true, Severity: "blocker", RecordStores: map[string]string{"itd": ".abcd/development/intents"}},
		},
	}
	findings, err := lint.Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	for _, fnd := range findings {
		if fnd.RuleID == "intent_lifecycle" || fnd.RuleID == "record_provenance" {
			t.Fatalf("a held record must stay lint-valid: %s:%d %s", fnd.File, fnd.Line, fnd.Message)
		}
	}
}

// TestReconcileRefusesAHeldPlannedRecord: a hold blocks EVERY lifecycle move
// until `intent unhold`, the close included. `spec close` (Reconcile) on a held
// planned record refuses before anything moves — the spec stays open, the
// intent stays planned, both files byte-identical — naming the reason and the
// lift, and it never strips the key (fix round 1, R1).
func TestReconcileRefusesAHeldPlannedRecord(t *testing.T) {
	root := t.TempDir()
	intentRel := plannedDir + "/itd-10-alpha.md"
	specRel := specsOpen + "/spc-1-alpha.md"
	writeFile(t, root, intentRel, plannedLinked("itd-10", "alpha", "spc-1"))
	writeFile(t, root, specRel, specNaming("spc-1", "alpha", "itd-10"))
	if _, err := Hold(root, "itd-10", "scope under review"); err != nil {
		t.Fatal(err)
	}
	intentBefore := readIntent(t, root, intentRel)
	specBefore := readIntent(t, root, specRel)

	_, err := Reconcile(root, "spc-1", "", RemainderRequest{})
	if err == nil {
		t.Fatal("spec close must refuse a held planned record")
	}
	for _, want := range []string{"scope under review", "intent unhold itd-10", "spec close"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must carry %q: %v", want, err)
		}
	}
	if got := readIntent(t, root, intentRel); got != intentBefore {
		t.Fatalf("the held intent must be byte-identical after the refusal (and still planned):\n%s", got)
	}
	if got := readIntent(t, root, specRel); got != specBefore {
		t.Fatalf("the spec must be byte-identical after the refusal (and still open):\n%s", got)
	}
	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if it, _ := c.Lookup("itd-10"); it.Bucket != BucketPlanned || it.Held != "scope under review" {
		t.Fatalf("the record must stay planned and held: %+v", it)
	}
	// Lifted, the same close ships.
	if _, err := Unhold(root, "itd-10"); err != nil {
		t.Fatal(err)
	}
	if res, err := Reconcile(root, "spc-1", "", RemainderRequest{}); err != nil || !res.IntentMoved {
		t.Fatalf("the close ships once the hold is lifted: %+v %v", res, err)
	}
}

// TestUnholdRefusesAHandWrittenKeySpelling: a `held` line the reader accepts
// but the remover does not match — `held : "x"`, a space before the colon —
// is a spelling no verb writes. Unhold must not report a removal it did not
// make: it refuses naming the line as hand-written, the file is untouched,
// and the loader still reports the record held (fix round 1, R2).
func TestUnholdRefusesAHandWrittenKeySpelling(t *testing.T) {
	root := t.TempDir()
	rel := draftsDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, "---\nid: itd-10\nslug: alpha\nspec_id: null\nkind: null\nheld : \"by hand\"\n---\n# alpha\n\n## Acceptance Criteria\n\n- ok\n")
	before := readIntent(t, root, rel)

	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if it, _ := c.Lookup("itd-10"); it.Held != "by hand" {
		t.Fatalf("precondition: the reader accepts the spelling as a hold: %+v", it)
	}
	_, err = Unhold(root, "itd-10")
	if err == nil {
		t.Fatal("unhold must refuse when the removal would change nothing")
	}
	for _, want := range []string{"hand", "held : ", "nothing written"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must carry %q: %v", want, err)
		}
	}
	if got := readIntent(t, root, rel); got != before {
		t.Fatalf("a refused unhold must leave the file untouched:\n%s", got)
	}
	c, err = Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if it, _ := c.Lookup("itd-10"); it.Held != "by hand" {
		t.Fatalf("the record is still held after the refusal: %+v", it)
	}
	// Plan still refuses it: the hold stands however it was spelled.
	if _, err := Plan(root, "itd-10", ""); err == nil || !strings.Contains(err.Error(), "by hand") {
		t.Fatalf("plan must still refuse the held record: %v", err)
	}
}

// TestPlanRefusesAHoldOnTheBytesItReReads: Plan judges the hold on the corpus
// it loads AND on the bytes it re-reads before its first write, so a hold
// written between the load and the re-read is refused rather than planned
// past. Plan's load and re-read are one call with no seam between them, so
// the re-read half is exercised through the reader Plan calls: given a file
// carrying a hold, it refuses naming the reason and the lift; given one
// without, it returns the bytes.
func TestPlanRefusesAHoldOnTheBytesItReReads(t *testing.T) {
	root := t.TempDir()
	rel := draftsDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, draftWithAC("itd-10", "alpha"))
	abs := root + "/" + rel

	content, err := readIntentRefusingHold(abs, rel, "itd-10", "plan")
	if err != nil {
		t.Fatalf("an unheld draft reads: %v", err)
	}
	if content != draftWithAC("itd-10", "alpha") {
		t.Fatalf("the bytes read are the file's:\n%s", content)
	}
	// The hold lands after the load Plan would have done and before the write.
	if _, err := Hold(root, "itd-10", "landed late"); err != nil {
		t.Fatal(err)
	}
	_, err = readIntentRefusingHold(abs, rel, "itd-10", "plan")
	if err == nil {
		t.Fatal("a hold on the re-read bytes must refuse")
	}
	for _, want := range []string{"landed late", "intent unhold itd-10", "plan refuses"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must carry %q: %v", want, err)
		}
	}
	// And Plan as a whole still refuses the held draft with nothing moved.
	before := readIntent(t, root, rel)
	if _, err := Plan(root, "itd-10", ""); err == nil || !strings.Contains(err.Error(), "landed late") {
		t.Fatalf("Plan must refuse: %v", err)
	}
	if readIntent(t, root, rel) != before {
		t.Fatal("a refused plan wrote the draft")
	}
}
