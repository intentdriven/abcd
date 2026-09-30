package drainrule

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/decide"
)

// The drained repository's own eligibility record (ruling BX2) and the floors
// it may loosen (ruling H11). Every test writes a decision store into a
// temporary checkout and reads it back through Load, the one reader the drain
// uses.

// strictFields is the baseline as a record states it.
const strictFields = "drain_categories: [tech-debt, documentation, inconsistency, drift, bug, ux]\n" +
	"drain_severities: [nitpick, minor]\n" +
	"drain_security: handback\n" +
	"drain_remedy: required\n"

// writeADR writes one decision record into repo's store.
func writeADR(t *testing.T, repo, name, id, status, extra string) string {
	t.Helper()
	dir := filepath.Join(repo, filepath.FromSlash(ADRsRelDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: " + id + "\nslug: s\nstatus: " + status + "\ndate: 2026-09-30\n" + extra + "---\n\n# ADR\n"
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestLoadRefusesARepositoryWithoutItsOwnRecord is BX2: the project must hold
// the decision in its own record, and the drain refuses until it does, naming
// how to add it. A store that decides other things is not a drain rule.
func TestLoadRefusesARepositoryWithoutItsOwnRecord(t *testing.T) {
	for name, setup := range map[string]func(repo string){
		"no store":         func(string) {},
		"no drain fields":  func(repo string) { writeADR(t, repo, "0001-other.md", "adr-1", "accepted", "") },
		"only proposed":    func(repo string) { writeADR(t, repo, "0002-rule.md", "adr-2", "proposed", strictFields) },
		"only superseded":  func(repo string) { writeADR(t, repo, "0003-rule.md", "adr-3", "superseded", strictFields) },
		"not a store file": func(repo string) { writeADR(t, repo, "README.md", "adr-4", "accepted", strictFields) },
	} {
		t.Run(name, func(t *testing.T) {
			repo := t.TempDir()
			setup(repo)
			_, err := Load(repo)
			if !errors.Is(err, ErrUnrecorded) {
				t.Fatalf("got %v, want ErrUnrecorded", err)
			}
			for _, want := range []string{"ahoy install", FieldCategories, "accepted decision record"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("the refusal does not name %q: %v", want, err)
				}
			}
		})
	}
	// A record carrying the fields but not accepted is named, so its author sees
	// why it does not count.
	repo := t.TempDir()
	writeADR(t, repo, "0002-rule.md", "adr-2", "proposed", strictFields)
	if _, err := Load(repo); err == nil || !strings.Contains(err.Error(), "adr-2") || !strings.Contains(err.Error(), "proposed") {
		t.Errorf("a proposed record is not named in the refusal: %v", err)
	}
}

// TestLoadReadsTheStrictRecordAsTheBaseline: a record stating the baseline
// loosens nothing, and the rule carries the record's id and path.
func TestLoadReadsTheStrictRecordAsTheBaseline(t *testing.T) {
	repo := t.TempDir()
	writeADR(t, repo, "2609291342092738-rule.md", "adr-2609291342092738", "accepted", strictFields)
	r, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if r.Record != "adr-2609291342092738" || r.Path != ADRsRelDir+"/2609291342092738-rule.md" {
		t.Errorf("record %q at %q", r.Record, r.Path)
	}
	if len(r.Loosened) != 0 {
		t.Errorf("the baseline loosens %v", r.Loosened)
	}
	b := Baseline()
	if !reflect.DeepEqual(r.Categories, b.Categories) || !reflect.DeepEqual(r.Severities, b.Severities) ||
		r.Security != SecurityHandBack || r.Remedy != RemedyRequired {
		t.Errorf("the strict record reads as %+v, want the baseline %+v", r, b)
	}
}

// TestARecordMayNarrowTheFixableSet: a project that takes less than the
// baseline is not loosening anything, and the drain order stays abcd's.
func TestARecordMayNarrowTheFixableSet(t *testing.T) {
	repo := t.TempDir()
	writeADR(t, repo, "0007-rule.md", "adr-7", "accepted",
		"drain_categories: [ux, documentation]\ndrain_severities: [nitpick]\ndrain_security: handback\ndrain_remedy: required\n")
	r, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.Categories, ",") != "documentation,ux" || strings.Join(r.Severities, ",") != "nitpick" {
		t.Errorf("narrowed rule = %+v", r)
	}
	if len(r.Loosened) != 0 {
		t.Errorf("a narrowed rule loosens %v", r.Loosened)
	}
	if r.TakesCategory("bug") || !r.TakesCategory("ux") || r.TakesSeverity("minor") {
		t.Errorf("the narrowed rule takes the wrong set: %+v", r)
	}
}

// TestALoosenedFloorIsNamed is H11: a project may let a drain take major and
// critical issues and security issues, and every floor it loosens is named,
// measured against abcd's baseline.
func TestALoosenedFloorIsNamed(t *testing.T) {
	repo := t.TempDir()
	writeADR(t, repo, "0008-rule.md", "adr-8", "accepted",
		"drain_categories: [bug]\ndrain_severities: [critical, minor, major]\ndrain_security: take\ndrain_remedy: required\n")
	r, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"severity major", "severity critical", "security"}; !reflect.DeepEqual(r.Loosened, want) {
		t.Errorf("loosened = %v, want %v", r.Loosened, want)
	}
	if strings.Join(r.Severities, ",") != "minor,major,critical" {
		t.Errorf("severities in drain order = %v", r.Severities)
	}
	if strings.Join(r.Categories, ",") != "bug,security" || !r.TakesCategory("security") || !r.TakesSeverity("critical") {
		t.Errorf("a record taking security: %+v", r)
	}
}

// TestAMalformedRecordRefuses: a partial or malformed record never falls back
// to a looser rule or to a stricter one without saying so. It refuses, naming
// the field and the record.
func TestAMalformedRecordRefuses(t *testing.T) {
	cases := map[string]struct{ fields, want string }{
		"missing remedy":      {strings.Replace(strictFields, "drain_remedy: required\n", "", 1), FieldRemedy},
		"missing severities":  {strings.Replace(strictFields, "drain_severities: [nitpick, minor]\n", "", 1), FieldSeverities},
		"misspelt key":        {strictFields + "drain_severity: [major]\n", "drain_severity"},
		"unknown severity":    {strings.Replace(strictFields, "[nitpick, minor]", "[nitpick, small]", 1), "small"},
		"decision category":   {strings.Replace(strictFields, "ux]", "ux, process]", 1), "process"},
		"security as a list":  {strings.Replace(strictFields, "ux]", "ux, security]", 1), FieldSecurity},
		"unknown category":    {strings.Replace(strictFields, "ux]", "ux, chores]", 1), "chores"},
		"empty categories":    {strings.Replace(strictFields, "[tech-debt, documentation, inconsistency, drift, bug, ux]", "[]", 1), FieldCategories},
		"block sequence":      {strings.Replace(strictFields, "[nitpick, minor]", "\n  - nitpick", 1), FieldSeverities},
		"security value":      {strings.Replace(strictFields, "handback", "sometimes", 1), "sometimes"},
		"remedy loosened":     {strings.Replace(strictFields, "drain_remedy: required", "drain_remedy: optional", 1), FieldRemedy},
		"duplicate key":       {strictFields + "drain_security: take\n", "more than once"},
		"quoted list members": {strings.Replace(strictFields, "[nitpick, minor]", `["nitpick", "major"]`, 1), ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			repo := t.TempDir()
			writeADR(t, repo, "0009-rule.md", "adr-9", "accepted", c.fields)
			r, err := Load(repo)
			if c.want == "" { // a well-formed spelling the reader must accept
				if err != nil {
					t.Fatalf("a quoted list refused: %v", err)
				}
				if strings.Join(r.Loosened, ",") != "severity major" {
					t.Errorf("loosened = %v", r.Loosened)
				}
				return
			}
			if !errors.Is(err, ErrMalformed) {
				t.Fatalf("got rule %+v, err %v; want ErrMalformed", r, err)
			}
			if !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "adr-9") {
				t.Errorf("the refusal does not name %q and the record: %v", c.want, err)
			}
		})
	}
}

// TestTwoAcceptedRecordsRefuse: which rule an unattended drain applies is never
// a choice the reader makes between two records.
func TestTwoAcceptedRecordsRefuse(t *testing.T) {
	repo := t.TempDir()
	writeADR(t, repo, "0010-a.md", "adr-10", "accepted", strictFields)
	writeADR(t, repo, "0011-b.md", "adr-11", "accepted", strings.Replace(strictFields, "handback", "take", 1))
	_, err := Load(repo)
	if !errors.Is(err, ErrAmbiguous) || !strings.Contains(err.Error(), "adr-10") || !strings.Contains(err.Error(), "adr-11") {
		t.Fatalf("got %v, want ErrAmbiguous naming both records", err)
	}
}

// TestLoadNeverReadsThroughALinkOutOfTheCheckout: the rule is committed history
// of the drained tree, so a store that is a symlink leaving the checkout is not
// read, and the drain refuses rather than applying a rule from elsewhere.
func TestLoadNeverReadsThroughALinkOutOfTheCheckout(t *testing.T) {
	outside := t.TempDir()
	writeADR(t, outside, "0012-rule.md", "adr-12", "accepted", strings.Replace(strictFields, "handback", "take", 1))
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "development", "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, filepath.FromSlash(ADRsRelDir)), filepath.Join(repo, filepath.FromSlash(ADRsRelDir))); err != nil {
		t.Fatal(err)
	}
	if r, err := Load(repo); !errors.Is(err, ErrUnreadable) {
		t.Fatalf("a rule was read through a link out of the checkout: %+v, %v; want ErrUnreadable", r, err)
	}
}

// TestTheProposalIsTheBaseline: what the setup offer writes reads back as the
// strict baseline, so the offer can never loosen a floor.
func TestTheProposalIsTheBaseline(t *testing.T) {
	repo := t.TempDir()
	writeADR(t, repo, "0013-rule.md", "adr-13", "accepted", ProposalFrontmatter())
	r, err := Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	b := Baseline()
	if len(r.Loosened) != 0 || !reflect.DeepEqual(r.Categories, b.Categories) || !reflect.DeepEqual(r.Severities, b.Severities) {
		t.Errorf("the proposal reads as %+v", r)
	}
	if !strings.Contains(ProposalBody(), "## Decision") || !strings.Contains(ProposalBody(), FieldSeverities) {
		t.Errorf("the proposal body does not state the decision and its fields:\n%s", ProposalBody())
	}
}

// TestTheStoreIsTheDecisionStore pins the store the rule is read from to the
// one `abcd decide` mints into, so the setup offer and the reader agree.
func TestTheStoreIsTheDecisionStore(t *testing.T) {
	if ADRsRelDir != decide.ADRsRelDir {
		t.Fatalf("drainrule reads %s, decide mints into %s", ADRsRelDir, decide.ADRsRelDir)
	}
}

// TestAnyDuplicateKeyInARuleRecordRefuses: the line scanner keeps a key's first
// value and a YAML reader its last, so a record stating any top-level key twice
// says two things, and which one an unattended drain obeys is never the
// reader's choice. `status: accepted` then `status: superseded` is the sharp
// case: first-wins admits a record every YAML reader calls superseded.
func TestAnyDuplicateKeyInARuleRecordRefuses(t *testing.T) {
	cases := map[string]struct{ status, extra, want string }{
		"status accepted then superseded": {"accepted", "status: superseded\n" + strictFields, "status"},
		"status superseded then accepted": {"superseded", "status: accepted\n" + strictFields, "status"},
		"id twice":                        {"accepted", "id: adr-99\n" + strictFields, "id"},
		"date twice":                      {"accepted", "date: 2026-10-01\n" + strictFields, "date"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			repo := t.TempDir()
			writeADR(t, repo, "0014-rule.md", "adr-14", c.status, c.extra)
			r, err := Load(repo)
			if !errors.Is(err, ErrMalformed) {
				t.Fatalf("got rule %+v, err %v; want ErrMalformed", r, err)
			}
			if !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "more than once") {
				t.Errorf("the refusal does not name the duplicated key %q: %v", c.want, err)
			}
		})
	}
}

// TestARuleRecordWhoseIdDisagreesWithItsFileNameRefuses: every surface names
// the rule by its record's id, so a record whose frontmatter claims another
// record's id (abcd's own strict one, say) while loosening floors would put
// that record's name on its own rule.
func TestARuleRecordWhoseIdDisagreesWithItsFileNameRefuses(t *testing.T) {
	repo := t.TempDir()
	writeADR(t, repo, "2609300000000001-x.md", "adr-2609291342092738", "accepted",
		strings.Replace(strictFields, "handback", "take", 1))
	r, err := Load(repo)
	if !errors.Is(err, ErrMalformed) {
		t.Fatalf("got rule %+v, err %v; want ErrMalformed", r, err)
	}
	for _, want := range []string{"adr-2609291342092738", "adr-2609300000000001"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %s: %v", want, err)
		}
	}
	// The same number written with its ordinal zeros is the same id.
	repo = t.TempDir()
	writeADR(t, repo, "0015-rule.md", "adr-0015", "accepted", strictFields)
	if r, err := Load(repo); err != nil || r.Record != "adr-0015" && r.Record != "adr-15" {
		t.Errorf("a zero-padded id refused or renamed: %+v %v", r, err)
	}
}

// TestAnOversizedRecordIsNeverRead: the store is repository-authored, so every
// record in it is read through the capped reader, and one past the cap refuses
// the load rather than being read whole.
func TestAnOversizedRecordIsNeverRead(t *testing.T) {
	repo := t.TempDir()
	p := writeADR(t, repo, "0016-rule.md", "adr-16", "accepted", strictFields)
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(strings.Repeat("padding line\n", (2<<20)/13)); err != nil {
		t.Fatal(err)
	}
	f.Close()
	r, err := Load(repo)
	if !errors.Is(err, ErrUnreadable) {
		t.Fatalf("got rule %+v, err %v; want ErrUnreadable", r, err)
	}
	if !strings.Contains(err.Error(), "0016-rule.md") || !strings.Contains(err.Error(), "size cap") {
		t.Errorf("the refusal does not name the record and the cap: %v", err)
	}
}

// TestASymlinkedRecordInsideTheStoreIsNeverFollowed: a record in the store
// that is a link, even one resolving inside the checkout, is refused rather
// than read as a live record.
func TestASymlinkedRecordInsideTheStoreIsNeverFollowed(t *testing.T) {
	repo := t.TempDir()
	loose := filepath.Join(repo, "docs", "loose.md")
	if err := os.MkdirAll(filepath.Dir(loose), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: adr-17\nstatus: accepted\n" + strings.Replace(strictFields, "handback", "take", 1) + "---\n"
	if err := os.WriteFile(loose, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(repo, filepath.FromSlash(ADRsRelDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Relative, so the link resolves inside the checkout.
	if err := os.Symlink(filepath.FromSlash("../../../../docs/loose.md"), filepath.Join(dir, "0017-rule.md")); err != nil {
		t.Fatal(err)
	}
	if r, err := Load(repo); !errors.Is(err, ErrUnreadable) {
		t.Fatalf("got rule %+v, err %v; want ErrUnreadable", r, err)
	}
}

// TestASymlinkedStoreIsRefusedWhereverItPoints: the store is read as the
// drained tree's own directory, so a store that is a link is refused the way a
// linked record is, whether it points inside the checkout or out of it, and so
// is a link at any directory above it: a store reached through a link is not a
// store the checkout holds at its own path.
func TestASymlinkedStoreIsRefusedWhereverItPoints(t *testing.T) {
	taking := strings.Replace(strictFields, "handback", "take", 1)
	for name, link := range map[string]func(t *testing.T, repo string){
		"store linked inside the checkout": func(t *testing.T, repo string) {
			writeADR(t, filepath.Join(repo, "elsewhere"), "0018-rule.md", "adr-18", "accepted", taking)
			if err := os.MkdirAll(filepath.Join(repo, ".abcd", "development", "decisions"), 0o755); err != nil {
				t.Fatal(err)
			}
			// Relative, so the link resolves inside the checkout.
			target := filepath.FromSlash("../../../elsewhere/" + ADRsRelDir)
			if err := os.Symlink(target, filepath.Join(repo, filepath.FromSlash(ADRsRelDir))); err != nil {
				t.Fatal(err)
			}
		},
		"store linked out of the checkout": func(t *testing.T, repo string) {
			outside := t.TempDir()
			writeADR(t, outside, "0018-rule.md", "adr-18", "accepted", taking)
			if err := os.MkdirAll(filepath.Join(repo, ".abcd", "development", "decisions"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(outside, filepath.FromSlash(ADRsRelDir)), filepath.Join(repo, filepath.FromSlash(ADRsRelDir))); err != nil {
				t.Fatal(err)
			}
		},
		"a directory above the store linked inside the checkout": func(t *testing.T, repo string) {
			writeADR(t, filepath.Join(repo, "elsewhere"), "0018-rule.md", "adr-18", "accepted", taking)
			if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.FromSlash("../elsewhere/.abcd/development"), filepath.Join(repo, ".abcd", "development")); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo := t.TempDir()
			link(t, repo)
			r, err := Load(repo)
			if !errors.Is(err, ErrUnreadable) {
				t.Fatalf("a rule was read through a linked store: %+v, %v; want ErrUnreadable", r, err)
			}
			if !strings.Contains(err.Error(), "is not a regular directory") {
				t.Errorf("the refusal does not say the store is not a regular directory: %v", err)
			}
		})
	}
}
