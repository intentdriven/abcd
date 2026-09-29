package capture

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

// duplicatesFixture is provenanceFixture (one open issue, itd-7 and spc-3
// planted) plus a second, unrelated issue the first can be a duplicate of.
func duplicatesFixture(t *testing.T) (repo, ir, subject, original string) {
	t.Helper()
	repo, ir, subject = provenanceFixture(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: "zebra quartz vellum xylophone", Severity: SeverityMinor,
		Category: "bug", Source: "manual-test", Slug: "the-original", FoundDuring: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	return repo, ir, subject, res.ID
}

// TestWontfixWritesTheTypedDuplicatesLink: a record closed as a duplicate
// carries the relation in the field the ledger defines, not only in the
// reason's prose (iss-2609291118049254), and an issue and an intent are both
// legal targets.
func TestWontfixWritesTheTypedDuplicatesLink(t *testing.T) {
	repo, ir, subject, original := duplicatesFixture(t)
	res, err := Wontfix(WontfixRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: subject,
		Reason:     "a duplicate of the original and of the planned intent",
		Duplicates: []string{original, "itd-7", original},
	})
	if err != nil {
		t.Fatalf("Wontfix: %v", err)
	}
	if res.ToStatus != StateWontfix {
		t.Fatalf("ToStatus = %q, want wontfix", res.ToStatus)
	}
	got := readIssue(t, ir, subject)
	if want := []string{original, "itd-7"}; !slices.Equal(got.Duplicates, want) {
		t.Fatalf("duplicates = %v, want %v", got.Duplicates, want)
	}
}

// TestWontfixDuplicatesKeepsTheFilingTimeLinks: a record the filing-time match
// already linked keeps those links; the closure adds to them.
func TestWontfixDuplicatesKeepsTheFilingTimeLinks(t *testing.T) {
	repo, ir, subject, original := duplicatesFixture(t)
	path, _, err := findIssue(ir, subject)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	linked, err := setListField(string(data), "duplicates", []string{"itd-7"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(linked), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Wontfix(WontfixRequest{
		RepoRoot: repo, IssuesRoot: ir, ID: subject, Reason: "a duplicate",
		Duplicates: []string{original, "itd-7"},
	}); err != nil {
		t.Fatalf("Wontfix: %v", err)
	}
	if got, want := readIssue(t, ir, subject).Duplicates, []string{"itd-7", original}; !slices.Equal(got, want) {
		t.Fatalf("duplicates = %v, want %v", got, want)
	}
}

// TestWontfixDuplicatesRefusesBeforeWriting: a target the record-lint gate
// would refuse is refused by the verb, and the record stays open and
// untouched.
func TestWontfixDuplicatesRefusesBeforeWriting(t *testing.T) {
	for name, tc := range map[string]struct {
		target func(subject string) string
		want   string
	}{
		"the record itself": {func(s string) string { return s }, "names the record itself"},
		"an unknown issue":  {func(string) string { return "iss-999" }, "not found"},
		"an unknown intent": {func(string) string { return "itd-99" }, "not found"},
		"a spec":            {func(string) string { return "spc-3" }, "iss-N or itd-N"},
		"a malformed id":    {func(string) string { return "iss-" }, "iss-N or itd-N"},
	} {
		t.Run(name, func(t *testing.T) {
			repo, ir, subject, _ := duplicatesFixture(t)
			path, _, err := findIssue(ir, subject)
			if err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Wontfix(WontfixRequest{
				RepoRoot: repo, IssuesRoot: ir, ID: subject, Reason: "a duplicate",
				Duplicates: []string{tc.target(subject)},
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want one naming %q", err, tc.want)
			}
			if !errors.Is(err, ErrRequestRefused) {
				t.Fatalf("err = %v, want a refusal", err)
			}
			after, rerr := os.ReadFile(path)
			if rerr != nil {
				t.Fatalf("the record left open/: %v", rerr)
			}
			if string(after) != string(before) {
				t.Fatalf("the refused wontfix rewrote the record:\n%s", after)
			}
		})
	}
}
