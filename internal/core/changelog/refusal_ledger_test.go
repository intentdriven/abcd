package changelog

import (
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/refusal"
)

// refusal_ledger_test.go — the release guard of adr-2610301638558260: once a
// release has carried a refusal code, the code stays. The guard reads the
// ledger of codes at the last tag, as GuardSurface reads the surface snapshot,
// and refuses a cut in which a code that ledger lists is missing from HEAD's.

// ledgerOf builds a ledger from code[:retired] specs.
func ledgerOf(specs ...string) refusal.Ledger {
	l := refusal.Ledger{FormatVersion: refusal.LedgerFormatVersion}
	for _, s := range specs {
		code, retired, _ := strings.Cut(s, ":")
		l.Codes = append(l.Codes, refusal.LedgerEntry{Code: refusal.Code(code), Retired: retired})
	}
	return l
}

func writeLedger(r *fixtureRepo, l refusal.Ledger) {
	r.t.Helper()
	data, err := refusal.EncodeLedger(l)
	if err != nil {
		r.t.Fatal(err)
	}
	r.write(refusal.LedgerPath, string(data))
}

// ledgerRepo tags a release carrying atTag (no ledger when nil), then commits
// head on top (the ledger removed when nil).
func ledgerRepo(t *testing.T, atTag, head *refusal.Ledger) *fixtureRepo {
	t.Helper()
	r := newFixtureRepo(t)
	r.write("README.md", "seed\n")
	if atTag != nil {
		writeLedger(r, *atTag)
	}
	r.commit("the release")
	r.git("tag", "v0.4.0")
	if head != nil {
		writeLedger(r, *head)
	} else if atTag != nil {
		r.remove(refusal.LedgerPath)
	}
	r.write("CHANGES", "after the release\n")
	r.commit("work after the release")
	return r
}

func ptr(l refusal.Ledger) *refusal.Ledger { return &l }

func TestGuardRefusalCodes(t *testing.T) {
	tests := []struct {
		name        string
		atTag, head *refusal.Ledger
		wantStatus  RefusalCodeGuardStatus
		wantMissing []refusal.Code
		wantRevived []refusal.Code
	}{
		{
			name:       "a tag that carries no ledger has no code to keep",
			head:       ptr(ledgerOf("capture/missing-remedy")),
			wantStatus: RefusalCodeGuardPassed,
		},
		{
			name:       "the same codes pass",
			atTag:      ptr(ledgerOf("capture/missing-remedy", "usage/unknown-flag")),
			head:       ptr(ledgerOf("capture/missing-remedy", "usage/unknown-flag")),
			wantStatus: RefusalCodeGuardPassed,
		},
		{
			name:       "an added code passes",
			atTag:      ptr(ledgerOf("capture/missing-remedy")),
			head:       ptr(ledgerOf("capture/missing-remedy", "capture/unknown-issue")),
			wantStatus: RefusalCodeGuardPassed,
		},
		{
			name:       "a retired code is still present, so it passes",
			atTag:      ptr(ledgerOf("capture/missing-remedy", "usage/unknown-flag")),
			head:       ptr(ledgerOf("capture/missing-remedy:v0.5.0", "usage/unknown-flag")),
			wantStatus: RefusalCodeGuardPassed,
		},
		{
			name:        "a removed code is refused and named",
			atTag:       ptr(ledgerOf("capture/missing-remedy", "usage/unknown-flag")),
			head:        ptr(ledgerOf("usage/unknown-flag")),
			wantStatus:  RefusalCodeGuardFailed,
			wantMissing: []refusal.Code{"capture/missing-remedy"},
		},
		{
			name:        "a renamed code is a removal, and is refused",
			atTag:       ptr(ledgerOf("capture/missing-remedy")),
			head:        ptr(ledgerOf("capture/no-remedy")),
			wantStatus:  RefusalCodeGuardFailed,
			wantMissing: []refusal.Code{"capture/missing-remedy"},
		},
		{
			name:        "a ledger deleted since the tag misses every code",
			atTag:       ptr(ledgerOf("capture/missing-remedy", "usage/unknown-flag")),
			wantStatus:  RefusalCodeGuardFailed,
			wantMissing: []refusal.Code{"capture/missing-remedy", "usage/unknown-flag"},
		},
		{
			name:        "a code retired at the tag cannot be handed out again",
			atTag:       ptr(ledgerOf("capture/missing-remedy:v0.3.0")),
			head:        ptr(ledgerOf("capture/missing-remedy")),
			wantStatus:  RefusalCodeGuardFailed,
			wantRevived: []refusal.Code{"capture/missing-remedy"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := ledgerRepo(t, tc.atTag, tc.head)
			g, err := GuardRefusalCodes(r.root, "v0.4.0")
			if err != nil {
				t.Fatalf("GuardRefusalCodes: %v", err)
			}
			if g.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q (reason %q)", g.Status, tc.wantStatus, g.Reason)
			}
			if !slices.Equal(g.Missing, tc.wantMissing) {
				t.Errorf("missing = %q, want %q", g.Missing, tc.wantMissing)
			}
			if !slices.Equal(g.Revived, tc.wantRevived) {
				t.Errorf("revived = %q, want %q", g.Revived, tc.wantRevived)
			}
			if tc.wantStatus == RefusalCodeGuardFailed {
				for _, c := range append(slices.Clone(tc.wantMissing), tc.wantRevived...) {
					if !strings.Contains(g.Reason, string(c)) {
						t.Errorf("the reason does not name %s: %s", c, g.Reason)
					}
				}
			} else if g.Reason != "" {
				t.Errorf("a pass carries a reason: %s", g.Reason)
			}
		})
	}
}

// TestGuardRefusalCodesFailsOnACorruptLedgerAtTheTag: a ledger present at the
// tag that cannot be read is an error, never "no codes": read as empty, it
// would let every code the release carried go.
func TestGuardRefusalCodesFailsOnACorruptLedgerAtTheTag(t *testing.T) {
	r := newFixtureRepo(t)
	r.write(refusal.LedgerPath, "{not json")
	r.commit("the release")
	r.git("tag", "v0.4.0")
	if _, err := GuardRefusalCodes(r.root, "v0.4.0"); err == nil {
		t.Fatal("a corrupt ledger at the tag was read as passing")
	}
}
