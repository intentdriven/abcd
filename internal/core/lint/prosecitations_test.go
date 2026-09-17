package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// proseCfg builds a one-rule config over a fixture tree. The stores are named
// exactly as the shipped configuration names them, so a fixture that passes here
// is a fixture `make record-lint` would also pass.
func proseCfg() Config {
	return Config{Rules: map[string]RuleConfig{
		ruleProseCitationResolves: {
			Enabled: true, Severity: severityBlocker,
			RecordStores: map[string]string{
				"adr": ".abcd/development/decisions/adrs",
				"itd": ".abcd/development/intents",
				"spc": ".abcd/development/specs",
				"iss": ".abcd/work/issues",
			},
		},
	}}
}

// writeProseIssue puts an issue in the ledger with the given body, so a test can
// aim prose at the rule without hand-building a frontmatter block each time.
func writeProseIssue(t *testing.T, root, status, name, body string) {
	t.Helper()
	writeFile(t, root, filepath.Join(".abcd", "work", "issues", status, name),
		"---\nid: \""+issueIDOf(name)+"\"\n---\n\n"+body+"\n")
}

// proseCorpus populates one live record in each store. Every test needs it: the
// rule fails closed on a corpus it cannot see, so an empty tree would make the
// guard, rather than the behaviour under test, the thing being measured.
func proseCorpus(t *testing.T, root string) {
	t.Helper()
	writeIssue(t, root, "open", "iss-42-record-orientation-currency.md")
	writeIntent(t, root, "planned", "itd-73-derived-versioning.md")
	writeSpec(t, root, "open", "spc-21-plugin-provisions-its-binary.md")
	writeADR(t, root, "0002-record-is-the-spec.md", "adr-2", "accepted", "null")
}

// TestProseCitationCatchesAnInventedID is the motivating instance
// (iss-2609100518527863): a session writing a record put an id into its body that
// names no record, and nothing read the body.
func TestProseCitationCatchesAnInventedID(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeSpec(t, root, "open", "spc-90-a-fixture-spec.md")
	writeFile(t, root, filepath.Join(".abcd", "development", "specs", "open", "spc-90-a-fixture-spec.md"),
		"---\nid: spc-90\n---\n\n# fixture\n\nThe renderer half is already fixed (iss-2608231243286557).\n")

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 1 {
		t.Fatalf("expected the invented id to fire once, got %d: %+v", n, fs)
	}
	rel := filepath.Join(".abcd", "development", "specs", "open", "spc-90-a-fixture-spec.md")
	if !hasFinding(fs, rel, ruleProseCitationResolves, 7) {
		t.Errorf("expected the finding anchored on the citing line; got %+v", fs)
	}
	// The refusal must TEACH the escape, or an author who is legitimately
	// illustrating has no way to learn what to write.
	if !messageContains(fs, "record-lint: illustrative") || !messageContains(fs, "record-lint: forward-looking") {
		t.Errorf("expected the message to name both escape spellings; got %+v", fs)
	}
}

// TestProseCitationStaysQuietOnLiveCitations is the precision half. A rule that
// fires on the corpus's ordinary, correct prose gets disabled, and then it gates
// nothing.
//
// Note what is NOT asserted here any more: that a slug-continued handle is
// exempt. It is not — a slug does not stop an id being an id — so the
// filename-shaped mentions in this fixture name LIVE records and are quiet
// because they resolve, not because the shape excused them. The failing case the
// old reading admitted is
// TestProseCitationReadsASlugContinuedHandleAsACitation.
func TestProseCitationStaysQuietOnLiveCitations(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeProseIssue(t, root, "open", "iss-91-live-citations.md", strings.Join([]string{
		"It builds on itd-73 and spc-21, and ADR-02 is the decision behind it.",
		"",
		"Padded and cased spellings are the same handle: Adr-0002, ITD-073.",
		"",
		"A placeholder written with a letter is not a citation: itd-N, spc-<id>, adr-NNNN.",
		"",
		"A handle continued by a slug is still that handle, so a LIVE one stays quiet:",
		"itd-73-derived-versioning.md and spc-21-a-spec.",
	}, "\n"))

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 0 {
		t.Fatalf("expected silence on live prose, got %d findings: %+v", n, fs)
	}
}

// TestProseCitationHonoursTheLineEscapes covers the ruling's own exemption: an id
// the author marks as illustrative or forward-looking is not required to resolve.
func TestProseCitationHonoursTheLineEscapes(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeProseIssue(t, root, "open", "iss-92-marked-mentions.md", strings.Join([]string{
		"A fixture introducing `supersedes: adr-999` is refused. <!-- record-lint: illustrative -->",
		"",
		"The options rule lands as itd-900. <!-- record-lint: forward-looking — minted when itd-901 is planned -->",
		"",
		"Spelling and spacing are folded: adr-998 <!--record-lint:ILLUSTRATIVE-->",
	}, "\n"))

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 0 {
		t.Fatalf("expected the marked mentions to be exempt, got %d: %+v", n, fs)
	}
}

// TestProseCitationEscapeIsScopedToItsOwnLine pins the escape's blast radius. A
// marker that carried past its line would silence a whole document from one
// sentence, which is a disarmed gate wearing the shape of an exemption.
func TestProseCitationEscapeIsScopedToItsOwnLine(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeProseIssue(t, root, "open", "iss-93-escape-scope.md", strings.Join([]string{
		"A marked mention: adr-999. <!-- record-lint: illustrative -->",
		"An unmarked one on the next line: adr-998.",
	}, "\n"))

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 1 {
		t.Fatalf("expected only the unmarked mention to fire, got %d: %+v", n, fs)
	}
	if !messageContains(fs, "adr-998") {
		t.Errorf("expected the finding to name adr-998; got %+v", fs)
	}
}

// TestProseCitationSkipsTypedFrontmatterAndFencedCode keeps the rule off the two
// regions that are not free prose: the TYPED cross-reference fields, whose
// targets record_schema already resolves (a second report of one dangling
// `supersedes` would be two findings for one defect), and fenced code, where a
// handle is sample text rather than a claim.
//
// Every OTHER frontmatter key IS read — see TestProseCitationReadsFreeTextFrontmatter.
func TestProseCitationSkipsTypedFrontmatterAndFencedCode(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeFile(t, root, filepath.Join(".abcd", "work", "issues", "open", "iss-94-regions.md"),
		strings.Join([]string{
			"---",
			"id: \"iss-94\"",
			"supersedes: adr-997",
			"related_intents:",
			"  - itd-9997",
			"blocked_by: [iss-9996]",
			"---",
			"",
			"Body prose naming nothing unresolved.",
			"",
			"```",
			"$ abcd intent ready itd-9998",
			"```",
			"",
		}, "\n"))

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 0 {
		t.Fatalf("expected typed frontmatter and fenced code to be out of scope, got %d: %+v", n, fs)
	}
}

// TestProseCitationBaselineGrandfathersOnlyWhatItNames is the ratchet: the
// corpus that predates the gate is carried, and a NEW unresolvable id in the very
// same file still fails.
func TestProseCitationBaselineGrandfathersOnlyWhatItNames(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeProseIssue(t, root, "open", "iss-95-ratchet.md", strings.Join([]string{
		"The pruned decision adr-996 is narrated here.",
		"A newly invented id is not: spc-995.",
	}, "\n"))
	writeFile(t, root, DefaultProseBaselinePath,
		`{"schema_version":1,"ids":[{"id":"adr-996","class":"pruned","note":"pruned by a successor; narrated historically"}]}`)

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 1 {
		t.Fatalf("expected only the un-baselined id to fire, got %d: %+v", n, fs)
	}
	if !messageContains(fs, "spc-995") {
		t.Errorf("expected the finding to name spc-995; got %+v", fs)
	}
}

// TestProseCitationBaselineInvitesItsOwnShrink is the ratchet's other direction.
// An entry nothing needs any more — the id now resolves, or no prose cites it —
// is reported so the backlog is visible as it drains, and never as a failure.
func TestProseCitationBaselineInvitesItsOwnShrink(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeProseIssue(t, root, "open", "iss-96-drained.md", "Nothing unresolved here; itd-73 is live.")
	writeFile(t, root, DefaultProseBaselinePath,
		`{"schema_version":1,"ids":[`+
			`{"id":"adr-996","class":"pruned","note":"nothing cites it any more"},`+
			`{"id":"itd-73","class":"never-minted","note":"this one has since been minted"}]}`)

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 0 {
		t.Fatalf("a shrink invitation is never a failure of the gate itself: %+v", fs)
	}
	stale := findingsFor(fs, ruleProseCitationBaselineStale)
	if len(stale) != 2 {
		t.Fatalf("expected both spent entries to invite their removal, got %d: %+v", len(stale), fs)
	}
	for _, f := range stale {
		if f.Severity != severityInfo {
			t.Errorf("a shrink invitation must never block: %+v", f)
		}
	}
}

// TestProseCitationFailsClosedOnAnEmptyCorpus covers the one way an armed gate
// could check nothing and still report clean: configured stores that hold no
// record files at all.
func TestProseCitationFailsClosedOnAnEmptyCorpus(t *testing.T) {
	root := t.TempDir()
	if _, err := Lint(proseCfg(), root); err == nil {
		t.Fatal("expected an armed rule over an empty corpus to fail closed")
	}
}

// TestProseCitationBaselineRefusesAMalformedRecord pins the loader's refusals.
// The baseline is a committed RULING — which ids are carried and why — so an
// entry that does not say both is refused at load rather than carried as a
// nameless exemption.
func TestProseCitationBaselineRefusesAMalformedRecord(t *testing.T) {
	cases := map[string]string{
		"wrong schema version": `{"schema_version":2,"ids":[]}`,
		"unknown field":        `{"schema_version":1,"ids":[{"id":"adr-4","class":"pruned","note":"n","why":"x"}]}`,
		"id is not a handle":   `{"schema_version":1,"ids":[{"id":"adr-4-slug","class":"pruned","note":"n"}]}`,
		"id is not canonical":  `{"schema_version":1,"ids":[{"id":"ADR-04","class":"pruned","note":"n"}]}`,
		"unknown class":        `{"schema_version":1,"ids":[{"id":"adr-4","class":"legacy","note":"n"}]}`,
		"empty note":           `{"schema_version":1,"ids":[{"id":"adr-4","class":"pruned","note":"  "}]}`,
		"duplicate id":         `{"schema_version":1,"ids":[{"id":"adr-4","class":"pruned","note":"n"},{"id":"adr-4","class":"pruned","note":"n"}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			proseCorpus(t, root)
			writeFile(t, root, DefaultProseBaselinePath, body)
			if _, err := Lint(proseCfg(), root); err == nil {
				t.Fatalf("expected %s to be refused at load", name)
			}
		})
	}
}

// TestProseCitationsAreCleanOnTheLiveRecord is the corpus assertion: the shipped
// configuration, run over this repository, reports no blocker. It is what makes
// the baseline decision falsifiable — a seeded entry that does not actually cover
// the corpus shows up here rather than at the next author's push.
func TestProseCitationsAreCleanOnTheLiveRecord(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(repoRootFromPackage, ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	rc, ok := cfg.Rules[ruleProseCitationResolves]
	if !ok || !rc.Enabled {
		t.Fatal("the shipped configuration must arm prose_citation_resolves")
	}
	fs, err := checkProseCitations(repoRootFromPackage, rc)
	if err != nil {
		t.Fatal(err)
	}
	var blockers []Finding
	for _, f := range fs {
		if f.Severity == severityBlocker {
			blockers = append(blockers, f)
		}
	}
	if len(blockers) != 0 {
		t.Fatalf("the live record must be green under the seeded baseline; %d blocker(s): %+v", len(blockers), blockers)
	}
}

// ---- fix round: the four reds ----

// F2: a slug does not stop an id being an id.
func TestProseCitationReadsASlugContinuedHandleAsACitation(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeProseIssue(t, root, "open", "iss-97-slug-continued.md",
		"The renderer half is fixed (iss-2608231243286557-the-renderer-half.md).")

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 1 {
		t.Fatalf("expected the slug-continued invented id to fire once, got %d: %+v", n, fs)
	}
}

// F3: free-text frontmatter values carry prose and must be read.
func TestProseCitationReadsFreeTextFrontmatter(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeFile(t, root, filepath.Join(".abcd", "work", "issues", "open", "iss-98-frontmatter-prose.md"),
		strings.Join([]string{
			"---",
			"id: \"iss-98\"",
			"deferral_reason: \"the renderer half is fixed (iss-2608231243286557)\"",
			"---",
			"",
			"Body prose naming nothing unresolved.",
			"",
		}, "\n"))

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 1 {
		t.Fatalf("expected the invented id in deferral_reason to fire once, got %d: %+v", n, fs)
	}
}

// F4: a hyphen on the LEFT does not make the handle part of a longer token.
func TestProseCitationReadsAHandleAfterAHyphen(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeProseIssue(t, root, "open", "iss-99-left-hyphen.md",
		"A pre-adr-999 mention and a foo-adr-998 one.")

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 2 {
		t.Fatalf("expected both hyphen-prefixed handles to fire, got %d: %+v", n, fs)
	}
}

// F6: a 0-byte baseline is refused with a message naming the remedy.
func TestProseCitationEmptyBaselineNamesTheRemedy(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeFile(t, root, DefaultProseBaselinePath, "")
	_, err := Lint(proseCfg(), root)
	if err == nil {
		t.Fatal("expected an empty baseline to be refused")
	}
	if !strings.Contains(err.Error(), `{"schema_version": 1, "ids": []}`) {
		t.Fatalf("the refusal must name the minimal valid document; got %v", err)
	}
}
