package intent

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/grounds"
)

// mustGrounds builds a validated Grounds for a test, where the vocabulary and
// the floor are not what is under test.
func mustGrounds(t *testing.T, tok grounds.Token, text string) grounds.Grounds {
	t.Helper()
	g, err := grounds.New(tok, text)
	if err != nil {
		t.Fatalf("grounds.New(%q, %q): %v", tok, text, err)
	}
	return g
}

// readIntent reads a record back off disk after a write.
func readIntent(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(data)
}

// TestRecordGroundsWritesEntry: the entry lands under the record's existing
// `## Grounds` heading as one top-level bullet in the shared grammar, so the
// record reads as prose and the gate reads it as data with no second parser.
func TestRecordGroundsWritesEntry(t *testing.T) {
	root := t.TempDir()
	const rel = plannedDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, plannedUnlinked("itd-10", "alpha")+"\n## Grounds\n\n- pursued: an earlier conjecture already recorded here\n")

	g := mustGrounds(t, grounds.Pursued, "we expect a stamped identity to survive rewording")
	res, err := RecordGrounds(root, "itd-10", g)
	if err != nil {
		t.Fatalf("RecordGrounds: %v", err)
	}
	if res.IntentID != "itd-10" || res.Path != rel {
		t.Fatalf("result = %+v, want itd-10 at %s", res, rel)
	}
	body := readIntent(t, root, rel)
	if !strings.Contains(body, g.Bullet()) {
		t.Fatalf("record does not carry %q:\n%s", g.Bullet(), body)
	}
	if got := ParseGrounds(body); len(got) != 2 {
		t.Fatalf("ParseGrounds read %d entries, want 2:\n%s", len(got), body)
	}
	if res.Entries != 2 {
		t.Fatalf("result Entries = %d, want 2", res.Entries)
	}
}

// TestRecordGroundsCreatesSectionWhenAbsent: a record that has never carried
// grounds gains the section rather than losing the entry.
func TestRecordGroundsCreatesSectionWhenAbsent(t *testing.T) {
	root := t.TempDir()
	const rel = plannedDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, plannedUnlinked("itd-10", "alpha"))

	g := mustGrounds(t, grounds.Pursued, "we expect the gate to make the reasoning survive the session")
	if _, err := RecordGrounds(root, "itd-10", g); err != nil {
		t.Fatalf("RecordGrounds: %v", err)
	}
	body := readIntent(t, root, rel)
	if !strings.Contains(body, "## "+GroundsHeading) {
		t.Fatalf("the section was not created:\n%s", body)
	}
	got := ParseGrounds(body)
	if len(got) != 1 || got[0] != g {
		t.Fatalf("ParseGrounds = %+v, want exactly %+v", got, g)
	}
}

// TestRecordGroundsAppendsSecondEntry: recording is append-only. A second gate
// decision on one record adds an entry; it never rewrites the first, because the
// earlier conjecture is what a later reader checks the outcome against.
func TestRecordGroundsAppendsSecondEntry(t *testing.T) {
	root := t.TempDir()
	const rel = plannedDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, plannedUnlinked("itd-10", "alpha"))

	first := mustGrounds(t, grounds.Deferred, "deferred while the identity it keys on has no consumer")
	second := mustGrounds(t, grounds.Pursued, "pursued now that the disposition finally has something to key on")
	if _, err := RecordGrounds(root, "itd-10", first); err != nil {
		t.Fatalf("first RecordGrounds: %v", err)
	}
	res, err := RecordGrounds(root, "itd-10", second)
	if err != nil {
		t.Fatalf("second RecordGrounds: %v", err)
	}
	if res.Entries != 2 {
		t.Fatalf("Entries = %d, want 2", res.Entries)
	}
	got := ParseGrounds(readIntent(t, root, rel))
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("ParseGrounds = %+v, want [%+v %+v] in that order", got, first, second)
	}
}

// TestRecordGroundsRedactsText: a grounds text is operator prose landing in a
// committed record, so it goes through the same detector every other durable
// free text does — before the write, never after. The spans below are FAKE
// shapes, matched only by shape, never real credentials.
func TestRecordGroundsRedactsText(t *testing.T) {
	root := t.TempDir()
	const rel = plannedDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, plannedUnlinked("itd-10", "alpha"))

	const fakeHome = "/Users/alice/.ssh/id_rsa"
	g := grounds.Grounds{Token: grounds.Pursued,
		Text: "we expect the receipt at " + fakeHome + " to be the thing that proves it"}
	res, err := RecordGrounds(root, "itd-10", g)
	if err != nil {
		t.Fatalf("RecordGrounds: %v", err)
	}
	body := readIntent(t, root, rel)
	if strings.Contains(body, "/Users/alice") {
		t.Fatalf("the record persisted the raw home path verbatim:\n%s", body)
	}
	if res.Redacted == 0 {
		t.Fatal("Redacted = 0; a rewritten text must be reported, never silently altered")
	}
}

// TestRecordGroundsRefusesUnknownIntent: an id naming no record is a structural
// fault, and nothing is written anywhere.
func TestRecordGroundsRefusesUnknownIntent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedUnlinked("itd-10", "alpha"))

	g := mustGrounds(t, grounds.Pursued, "we expect an unknown id to be refused before any write")
	if _, err := RecordGrounds(root, "itd-999", g); err == nil {
		t.Fatal("RecordGrounds on an unknown intent = nil error, want a refusal")
	}
	if _, err := RecordGrounds(root, "nonsense", g); err == nil {
		t.Fatal("RecordGrounds on a malformed id = nil error, want a refusal")
	}
	if body := readIntent(t, root, plannedDir+"/itd-10-alpha.md"); strings.Contains(body, "## "+GroundsHeading) {
		t.Fatalf("a refused call wrote to an unrelated record:\n%s", body)
	}
}

// TestRecordGroundsRefusesDegenerateTextAfterRedaction: validation runs on the
// text that will actually be written. Redacting after validation would let a
// rewritten span reach a field the validator had already passed.
func TestRecordGroundsRefusesDegenerateTextAfterRedaction(t *testing.T) {
	root := t.TempDir()
	const rel = plannedDir + "/itd-10-alpha.md"
	writeFile(t, root, rel, plannedUnlinked("itd-10", "alpha"))

	if _, err := RecordGrounds(root, "itd-10", grounds.Grounds{Token: grounds.Pursued, Text: "because"}); err == nil {
		t.Fatal("a text below the floor = nil error, want a refusal")
	}
	if _, err := RecordGrounds(root, "itd-10", grounds.Grounds{Token: "planned", Text: "a perfectly good conjecture about identity"}); err == nil {
		t.Fatal("an out-of-vocabulary token = nil error, want a refusal")
	}
	if body := readIntent(t, root, rel); strings.Contains(body, "## "+GroundsHeading) {
		t.Fatalf("a refused call still wrote the section:\n%s", body)
	}
}

// TestParseGroundsIgnoresFencedAndCommentedBullets: an example of the grammar in
// a fenced block, or a bullet parked in an HTML comment, is not a recorded
// ground — the same rule the scope-conditions reader already holds.
func TestParseGroundsIgnoresFencedAndCommentedBullets(t *testing.T) {
	content := plannedUnlinked("itd-10", "alpha") + "\n## Grounds\n\n" +
		"```\n- pursued: this is an example of the grammar, not a recorded ground\n```\n\n" +
		"<!--\n- deferred: this one was parked and is not live either\n-->\n\n" +
		"- pursued: the only live entry on this record names a real conjecture\n"
	got := ParseGrounds(content)
	if len(got) != 1 {
		t.Fatalf("ParseGrounds read %d entries, want 1: %+v", len(got), got)
	}
	if !strings.HasPrefix(got[0].Text, "the only live entry") {
		t.Fatalf("ParseGrounds read the wrong bullet: %+v", got)
	}
}

// TestRecordGroundsRefusesTextThatMasksTheRecord is
// iss-2608300927577980: a grounds text carrying an unclosed HTML comment opener
// was written, and the comment mask then hid that entry and every line after it
// from the grounds reader AND the claims readers — while the result reported
// success and the surface said nothing. A write that makes the record less
// readable than it was is refused before it happens.
func TestRecordGroundsRefusesTextThatMasksTheRecord(t *testing.T) {
	root := t.TempDir()
	const rel = plannedDir + "/itd-10-alpha.md"
	before := plannedUnlinked("itd-10", "alpha") +
		"\n## Grounds\n\n- pursued: an earlier conjecture that must stay readable\n"
	writeFile(t, root, rel, before)

	masking := grounds.Grounds{Token: grounds.Pursued,
		Text: "we expect the mask to swallow this <!-- and everything after it"}
	if _, err := RecordGrounds(root, "itd-10", masking); err == nil {
		t.Fatal("a text leaving a comment open = nil error, want a refusal")
	}
	if got := readIntent(t, root, rel); got != before {
		t.Fatalf("a refused write still changed the record:\n%s", got)
	}
	// The record still reads exactly as it did: one entry, and the claim sections
	// the mask would have swallowed.
	if n := len(ParseGrounds(readIntent(t, root, rel))); n != 1 {
		t.Fatalf("entries after the refusal = %d, want 1", n)
	}
	if c := ParseClaims(readIntent(t, root, rel)); c.ConditionsState != ClaimNullity {
		t.Fatalf("the scope-conditions claim reads as %q, want the nullity it carried", c.ConditionsState)
	}
}

// TestParseGroundsSkipsSubFloorEntry: the readiness gate claims a substance
// floor, so the reader has to apply it. A hand-typed `- pursued: yes` is not an
// entry — it clears the grammar and records nothing — and admitting it let the
// seventh check pass on a record that answers nothing (iss-2608300930057882).
func TestParseGroundsSkipsSubFloorEntry(t *testing.T) {
	content := plannedUnlinked("itd-10", "alpha") + "\n## Grounds\n\n- pursued: yes\n"
	if got := ParseGrounds(content); len(got) != 0 {
		t.Fatalf("ParseGrounds read %d entries from a sub-floor bullet, want 0: %+v", len(got), got)
	}
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", content)
	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if g := checkByName(t, res, CheckGrounds); g.OK {
		t.Fatalf("grounds check = %+v, want a refusal on a sub-floor entry", g)
	}
}

// TestRecordGroundsRefusesTerminalBuckets: population is forward-only, and the
// readiness gate says so by exempting shipped/ and superseded/ records from the
// grounds check. A writer that backfills them anyway makes that exemption a lie
// (iss-2608300930057882).
func TestRecordGroundsRefusesTerminalBuckets(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-10-alpha.md", plannedUnlinked("itd-10", "alpha"))
	writeFile(t, root, supersededDir+"/itd-11-beta.md", plannedUnlinked("itd-11", "beta"))
	writeFile(t, root, plannedDir+"/itd-12-gamma.md", plannedUnlinked("itd-12", "gamma"))

	g := mustGrounds(t, grounds.Pursued, "we expect a forward-only rule to be enforced by the writer, not only claimed by the gate")
	for _, id := range []string{"itd-10", "itd-11"} {
		if _, err := RecordGrounds(root, id, g); err == nil {
			t.Fatalf("RecordGrounds on a terminal-bucket record (%s) = nil error, want a refusal", id)
		}
	}
	for _, rel := range []string{shippedDir + "/itd-10-alpha.md", supersededDir + "/itd-11-beta.md"} {
		if strings.Contains(readIntent(t, root, rel), "## "+GroundsHeading) {
			t.Fatalf("a refused call still wrote to %s", rel)
		}
	}
	// A planned record is unaffected: the rule is about backfilling, not about
	// recording.
	if _, err := RecordGrounds(root, "itd-12", g); err != nil {
		t.Fatalf("RecordGrounds on a planned record = %v, want it accepted", err)
	}
}

// TestRecordGroundsConcurrentAppendsBothLand reproduces iss-2608301206036067:
// RecordGrounds is a read-modify-write over a record two sessions can reach at
// once, and the entry is APPEND-ONLY by contract. Two concurrent calls that each
// read the same bytes and each write their own snapshot lose one entry, and both
// return clean — the per-writer read-back check compares that writer's own
// before/after pair, which stays consistent. Twenty trials, because the loss is
// a race and one trial proves nothing.
func TestRecordGroundsConcurrentAppendsBothLand(t *testing.T) {
	const trials = 20
	for trial := 0; trial < trials; trial++ {
		root := t.TempDir()
		const rel = plannedDir + "/itd-10-alpha.md"
		writeFile(t, root, rel, plannedUnlinked("itd-10", "alpha")+"\n## Grounds\n\n")

		first := mustGrounds(t, grounds.Pursued, "we expect the first conjecture to survive a concurrent write")
		second := mustGrounds(t, grounds.Deferred, "we expect the second conjecture to survive a concurrent write")

		var wg sync.WaitGroup
		errs := make([]error, 2)
		start := make(chan struct{})
		for i, g := range []grounds.Grounds{first, second} {
			wg.Add(1)
			go func(i int, g grounds.Grounds) {
				defer wg.Done()
				<-start
				_, errs[i] = RecordGrounds(root, "itd-10", g)
			}(i, g)
		}
		close(start)
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("trial %d: writer %d: %v", trial, i, err)
			}
		}
		body := readIntent(t, root, rel)
		got := ParseGrounds(body)
		if len(got) != 2 {
			t.Fatalf("trial %d: %d entries survived two clean appends, want 2 — one write was discarded:\n%s",
				trial, len(got), body)
		}
	}
}

// TestRecordGroundsHoldsTheMintLock is the deterministic half: the read, the
// append and the write are one critical section under the same advisory lock
// every other writer in this package takes, so a concurrent holder blocks it.
func TestRecordGroundsHoldsTheMintLock(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", plannedUnlinked("itd-10", "alpha"))

	held := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- withIntentMintLock(root, func() error {
			close(held)
			// Hold the lock long enough that an unlocked write would finish inside it.
			time.Sleep(150 * time.Millisecond)
			return nil
		})
	}()
	<-held
	start := time.Now()
	g := mustGrounds(t, grounds.Pursued, "we expect the grounds write to serialize with every other writer")
	if _, err := RecordGrounds(root, "itd-10", g); err != nil {
		t.Fatal(err)
	}
	waited := time.Since(start)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if waited < 100*time.Millisecond {
		t.Fatalf("the grounds write completed in %v while the mint lock was held — it took no lock", waited)
	}
}

// TestRecordGroundsRefusesAWriteTheRecordSwallows is the read-back count check's
// own fixture (iss-2608301212423956). A record whose body ends inside an
// unclosed fence masks everything appended below it: the new `## Grounds`
// section and its entry land in the file and are invisible to every reader of
// it, including the readiness gate the entry was written to satisfy. The
// comment question does not fire — the text opens no comment — so the count
// question is the only thing standing between the caller and a write that
// reports success and records nothing.
func TestRecordGroundsRefusesAWriteTheRecordSwallows(t *testing.T) {
	root := t.TempDir()
	const rel = plannedDir + "/itd-10-alpha.md"
	body := plannedUnlinked("itd-10", "alpha") + "\n## Notes\n\n```go\nfunc main() {}\n"
	writeFile(t, root, rel, body)

	g := mustGrounds(t, grounds.Pursued, "we expect the entry to be refused rather than written where nothing can read it")
	_, err := RecordGrounds(root, "itd-10", g)
	if err == nil {
		t.Fatal("RecordGrounds into an unclosed fence = nil error, want the read-back refusal")
	}
	if !strings.Contains(err.Error(), "does not read back") {
		t.Fatalf("RecordGrounds = %v, want the read-back refusal", err)
	}
	if got := readIntent(t, root, rel); got != body {
		t.Fatalf("the refused write altered the record:\n--- got\n%s\n--- want\n%s", got, body)
	}
}

// TestParseGroundsReadsAScriptWithoutWordBreaks is the reader half of
// iss-2608301301044588. The floor counted letter-runs, so a Japanese bullet was
// dropped by ParseGrounds and the readiness gate then reported "no recorded
// grounds" about a record whose `## Grounds` section visibly carried one — the
// gate/reader divergence this feature exists to close, re-entering through the
// floor rather than through the grammar.
func TestParseGroundsReadsAScriptWithoutWordBreaks(t *testing.T) {
	const text = "スタンプされた識別子は改名されても生き残ると予期している"
	content := plannedUnlinked("itd-10", "alpha") + "\n## Grounds\n\n- pursued: " + text + "\n"
	got := ParseGrounds(content)
	if len(got) != 1 {
		t.Fatalf("ParseGrounds read %d entries from a bullet in a script without word breaks, want 1: %+v", len(got), got)
	}
	if got[0].Text != text {
		t.Fatalf("entry text = %q, want %q", got[0].Text, text)
	}
	root := t.TempDir()
	writeFile(t, root, plannedDir+"/itd-10-alpha.md", content)
	res, err := Ready(root, "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	if g := checkByName(t, res, CheckGrounds); !g.OK {
		t.Fatalf("grounds check = %+v, want it satisfied by the recorded entry", g)
	}
}
