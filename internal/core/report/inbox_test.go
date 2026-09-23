package report

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// sandbox points HOME at a fresh directory, so the inbox the test sees is its
// own, and pins the clock. It returns the home.
func sandbox(t *testing.T, at time.Time) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	setClock(t, at)
	return home
}

// setClock pins the package clock the id and the received stamp read.
func setClock(t *testing.T, at time.Time) {
	t.Helper()
	prev := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = prev })
}

// committedRepo is a throwaway repository with one commit, so it has a root
// commit to be keyed on.
func committedRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Git("-c", "user.name=abcd test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "root")
	return r
}

func mustParse(t *testing.T, s string) Report {
	t.Helper()
	r, err := Parse([]byte(s))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return r
}

var reportNameRe = regexp.MustCompile(`^[0-9]{16}-[0-9a-f]{40,64}\.md$`)

// TestFileLandsInTheMachineStoreOnly: criteria 1 and 2 — the report is filed
// in the user account's inbox, named by abcd, and the sending repository's tree
// is untouched.
func TestFileLandsInTheMachineStoreOnly(t *testing.T) {
	home := sandbox(t, time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC))
	repo := committedRepo(t)
	sender, err := SenderOf(repo.Root())
	if err != nil {
		t.Fatalf("SenderOf: %v", err)
	}
	if len(sender.Key) != 40 || sender.Name == "" {
		t.Fatalf("sender = %+v, want a full root-commit key and a name", sender)
	}
	filedRep, err := File(mustParse(t, filled(t)), sender)
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if !strings.HasPrefix(filedRep.ID, "rpt-260923100000") {
		t.Errorf("id = %q, want rpt-<received stamp>", filedRep.ID)
	}
	entries, err := os.ReadDir(filepath.Join(home, ".abcd", "inbox"))
	if err != nil {
		t.Fatalf("inbox: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	if len(names) != 1 || !reportNameRe.MatchString(names[0]) || !strings.Contains(names[0], sender.Key) {
		t.Fatalf("inbox holds %q, want one <stamp>-<sender-key>.md", names)
	}
	info, err := os.Stat(filepath.Join(home, ".abcd", "inbox", names[0]))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("report mode = %v (%v), want 0600", info.Mode().Perm(), err)
	}
	if st := repo.Git("status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Errorf("the sending repository's tree changed:\n%s", st)
	}
	if strings.Contains(filedRep.Path, home) {
		t.Errorf("the reported path %q carries the home directory", filedRep.Path)
	}
}

// TestListNewestFirstNamingTheSender: criteria 3 and 4 — the inbox renders
// newest first with the sender named, and the tally counts reports and
// distinct senders. Reading files nothing.
func TestListNewestFirstNamingTheSender(t *testing.T) {
	sandbox(t, time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC))
	a := Sender{Key: strings.Repeat("a", 40), Name: "alpha"}
	b := Sender{Key: strings.Repeat("b", 40), Name: "beta"}
	for i, s := range []Sender{a, b, a} {
		setClock(t, time.Date(2026, 9, 23, 9, i, 0, 0, time.UTC))
		if _, err := File(mustParse(t, filled(t)), s); err != nil {
			t.Fatalf("File: %v", err)
		}
	}
	list, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("List has %d entries, want 3", len(list))
	}
	if list[0].SenderName != "alpha" || list[1].SenderName != "beta" || list[2].SenderName != "alpha" {
		t.Errorf("order = %s, %s, %s; want newest first", list[0].SenderName, list[1].SenderName, list[2].SenderName)
	}
	if list[0].ID <= list[1].ID {
		t.Errorf("ids %s then %s are not newest first", list[0].ID, list[1].ID)
	}
	if list[0].State != StateWaiting || list[0].Title == "" {
		t.Errorf("entry = %+v", list[0])
	}
	tally, err := Count()
	if err != nil || tally.Reports != 3 || tally.Senders != 2 {
		t.Errorf("Count = %+v, %v; want 3 reports from 2 senders", tally, err)
	}
}

// TestCountOnAnAbsentInboxIsZero: a machine that never received a report has
// no inbox, and reading it creates none.
func TestCountOnAnAbsentInboxIsZero(t *testing.T) {
	home := sandbox(t, time.Now())
	tally, err := Count()
	if err != nil || tally.Reports != 0 || tally.Senders != 0 {
		t.Fatalf("Count = %+v, %v", tally, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd", "inbox")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("reading created the inbox (%v)", err)
	}
}

// TestUnknownVersionIsListedUnreadable: criterion 6 — a report written to a
// later template is listed, naming its version, and is not dropped.
func TestUnknownVersionIsListedUnreadable(t *testing.T) {
	home := sandbox(t, time.Date(2026, 9, 23, 11, 0, 0, 0, time.UTC))
	dir := filepath.Join(home, ".abcd", "inbox")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	name := "2609231100001234-" + strings.Repeat("c", 40) + ".md"
	body := "---\nschema_version: 7\nsomething_new: yes\n---\n\nprose\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	list, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].State != StateUnreadable || !strings.Contains(list[0].Unreadable, "7") {
		t.Fatalf("List = %+v, want one unreadable entry naming version 7", list)
	}
	if list[0].ID != "rpt-2609231100001234" || list[0].SenderKey != strings.Repeat("c", 40) {
		t.Errorf("entry = %+v", list[0])
	}
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Errorf("the unreadable report was dropped: %v", err)
	}
	if tally, _ := Count(); tally.Reports != 1 {
		t.Errorf("Count = %+v, want the unreadable report counted", tally)
	}
	if _, err := Promote(t.TempDir(), list[0].ID); !errors.Is(err, ErrRefused) {
		t.Errorf("Promote(unreadable) = %v, want a refusal", err)
	}
}

// TestShowRefusesWhatIsNotAReportID: the id is the only thing a caller names,
// and it never becomes a path.
func TestShowRefusesWhatIsNotAReportID(t *testing.T) {
	sandbox(t, time.Now())
	for _, id := range []string{"../../etc/passwd", "rpt-1", "iss-2609221656361680", "rpt-2609231100001234/x"} {
		if _, err := Show(id); !errors.Is(err, ErrRefused) {
			t.Errorf("Show(%q) = %v, want a refusal", id, err)
		}
	}
	if _, err := Show("rpt-2609231100001234"); !errors.Is(err, ErrRefused) {
		t.Errorf("Show(absent) = %v, want a refusal", err)
	}
}

// TestPromoteFingerprintsAndNeverNamesTheSender: criteria 5 and 7 — nothing is
// filed until promote runs; the capture carries the root-commit key, a generic
// description and the report id, and the sender's name is absent from every
// file abcd writes into the repository.
func TestPromoteFingerprintsAndNeverNamesTheSender(t *testing.T) {
	sandbox(t, time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC))
	ledger := committedRepo(t)
	const name = "Zanzibar-Quartz"
	sender := Sender{Key: strings.Repeat("d", 40), Name: name}
	in := strings.Replace(filled(t), "was refused.", "was refused while working in zanzibar-quartz.", 1)
	in = strings.Replace(in, `title: "capture refuses`, `title: "Zanzibar-Quartz: capture refuses`, 1)
	f, err := File(mustParse(t, in), sender)
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	if st := ledger.Git("status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("a report filed itself before anyone acted:\n%s", st)
	}

	p, err := Promote(ledger.Root(), f.ID)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if !strings.HasPrefix(p.Capture, "iss-") || p.Report != f.ID {
		t.Fatalf("Promote = %+v", p)
	}
	body, err := os.ReadFile(filepath.Join(ledger.Root(), filepath.FromSlash(p.Path)))
	if err != nil {
		t.Fatalf("read capture: %v", err)
	}
	text := string(body)
	for _, want := range []string{sender.Key, GenericSender, f.ID, `source: "managed-repo"`, "capture refuses a slug"} {
		if !strings.Contains(text, want) {
			t.Errorf("the capture lacks %q:\n%s", want, text)
		}
	}
	// Criterion 7: the name is absent from everything the repository gained.
	status := ledger.Git("status", "--porcelain", "--untracked-files=all")
	if status == "" {
		t.Fatal("promote wrote nothing into the repository")
	}
	for _, line := range strings.Split(status, "\n") {
		rel := strings.TrimSpace(line[2:])
		if strings.Contains(strings.ToLower(rel), "zanzibar") {
			t.Errorf("a file name carries the sender's name: %s", rel)
		}
		data, err := os.ReadFile(filepath.Join(ledger.Root(), filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(data)), "zanzibar") {
			t.Errorf("%s carries the sender's name:\n%s", rel, data)
		}
	}

	// The report is kept, marked, and no longer waits.
	if tally, _ := Count(); tally.Reports != 0 {
		t.Errorf("Count after promote = %+v, want 0 waiting", tally)
	}
	e, err := Show(f.ID)
	if err != nil || e.State != StatePromoted || e.PromotedTo != p.Capture {
		t.Errorf("Show after promote = %+v, %v; want promoted to %s", e, err, p.Capture)
	}
	if e.SenderName != name {
		t.Errorf("the inbox stopped naming the sender: %+v", e)
	}
	if _, err := Promote(ledger.Root(), f.ID); !errors.Is(err, ErrRefused) {
		t.Errorf("a second promote = %v, want a refusal", err)
	}
}

// TestNameScrubberReplacesTheNameAsAWord: the sender's name is replaced
// wherever it stands as a word, in any case, and a word that merely contains
// it is left alone.
func TestNameScrubberReplacesTheNameAsAWord(t *testing.T) {
	scrub := nameScrubber("cap")
	got := scrub("CAP broke capture in cap-v2 and (cap).")
	want := GenericSender + " broke capture in " + GenericSender + "-v2 and (" + GenericSender + ")."
	if got != want {
		t.Errorf("scrub = %q, want %q", got, want)
	}
}

// TestAReportAtTheBoundReadsBack: a report accepted at the size bound stays
// readable once abcd has stamped its envelope and escaped its quotes, so a
// report filed is never a report the inbox calls unreadable.
func TestAReportAtTheBoundReadsBack(t *testing.T) {
	sandbox(t, time.Date(2026, 9, 23, 13, 0, 0, 0, time.UTC))
	in := strings.Replace(filled(t), `remedy: "accept a leading digit"`, `remedy: '`+strings.Repeat(`\"`, 1000)+`'`, 1)
	in += strings.Repeat("p", MaxBytes-len(in))
	r := mustParse(t, in)
	if _, err := File(r, Sender{Key: strings.Repeat("e", 40), Name: "edge"}); err != nil {
		t.Fatalf("File: %v", err)
	}
	list, err := List()
	if err != nil || len(list) != 1 || list[0].State != StateWaiting {
		t.Fatalf("List = %+v, %v; want the report waiting and readable", list, err)
	}
}

// TestNameScrubberCatchesSpellingVariants: a directory name reaches prose
// spelt many ways. Each separator variant, a trailing digit, a camel-cased
// directory written with separators, and a forge address's owner segment are
// all replaced, so none of them leaks the sender into a public ledger.
func TestNameScrubberCatchesSpellingVariants(t *testing.T) {
	scrub := nameScrubber("acme-secret")
	for _, in := range []string{
		"acme-secret", "acme_secret", "acme secret", "acmesecret", "ACME.Secret",
		"acme-secret2", "AcmeSecret", "see github.com/acme/acme-secret for it",
		"cloned git@github.com:acme/acme_secret.git today", "https://gitlab.example.com/acme/acmesecret/-/issues/4",
	} {
		got := scrub(in)
		if strings.Contains(strings.ToLower(got), "acme") || strings.Contains(strings.ToLower(got), "secret") {
			t.Errorf("scrub(%q) = %q; the name survives", in, got)
		}
		if !strings.Contains(got, GenericSender) {
			t.Errorf("scrub(%q) = %q; nothing stands in for the name", in, got)
		}
	}
	// A camel-cased directory name is caught when the prose separates it.
	if got := nameScrubber("AcmeSecret")("the acme-secret repo"); strings.Contains(got, "acme") {
		t.Errorf("camel-cased name: scrub = %q", got)
	}
	// Words that merely contain the name's parts are left alone.
	for _, in := range []string{"a secretary at acmes", "acme alone", "secret alone"} {
		if got := scrub(in); got != in {
			t.Errorf("scrub(%q) = %q; an ordinary phrase was rewritten", in, got)
		}
	}
}

// TestPromoteRetryAfterAFailedMoveFilesOneCapture: a promotion whose move into
// the promoted folder fails has already filed its capture; the retry completes
// the move and names that capture, and never files a second one.
func TestPromoteRetryAfterAFailedMoveFilesOneCapture(t *testing.T) {
	home := sandbox(t, time.Date(2026, 9, 23, 14, 0, 0, 0, time.UTC))
	ledger := committedRepo(t)
	f, err := File(mustParse(t, filled(t)), Sender{Key: strings.Repeat("c", 40), Name: "retry"})
	if err != nil {
		t.Fatal(err)
	}
	// Occupy the move's destination with a non-empty directory, so the rename fails.
	name := filepath.Base(f.Path)
	blocker := filepath.Join(home, ".abcd", "inbox", "promoted", name)
	if err := os.MkdirAll(filepath.Join(blocker, "x"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Promote(ledger.Root(), f.ID); err == nil {
		t.Fatal("Promote succeeded with its destination occupied")
	}
	if err := os.RemoveAll(blocker); err != nil {
		t.Fatal(err)
	}
	captures := func() []string {
		out, _ := filepath.Glob(filepath.Join(ledger.Root(), ".abcd", "work", "issues", "open", "iss-*.md"))
		return out
	}
	first := captures()
	if len(first) != 1 {
		t.Fatalf("the failed promotion filed %d captures, want 1", len(first))
	}

	p, err := Promote(ledger.Root(), f.ID)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if got := captures(); len(got) != 1 {
		t.Fatalf("the retry filed a second capture: %v", got)
	}
	if !strings.Contains(filepath.Base(first[0]), p.Capture) || !p.Resumed {
		t.Errorf("retry = %+v, want the first capture %s, resumed", p, first[0])
	}
	if e, err := Show(f.ID); err != nil || e.State != StatePromoted || e.PromotedTo != p.Capture {
		t.Errorf("Show after retry = %+v, %v", e, err)
	}
	if _, err := Promote(ledger.Root(), f.ID); !errors.Is(err, ErrRefused) {
		t.Errorf("a third promote = %v, want a refusal", err)
	}
}
