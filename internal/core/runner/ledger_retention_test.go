package runner

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

var retentionNow = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

// agedEntries lays one entry file per age, in days, and returns the ledger at
// retentionNow with its retention set, and each file's name by age.
func agedEntries(t *testing.T, r Retention, days ...int) (*Ledger, map[int]string) {
	t.Helper()
	home := t.TempDir()
	dir := abcdhome.Path(home, ledgerFolder)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	names := map[int]string{}
	for _, d := range days {
		id, err := newLaunchID(retentionNow.AddDate(0, 0, -d))
		if err != nil {
			t.Fatal(err)
		}
		names[d] = id + ledgerExt
		if err := os.WriteFile(filepath.Join(dir, names[d]), []byte(`{"schema_version":1,"id":"`+id+`"}`+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	l := NewLedger(home)
	l.retention = r
	l.now = func() time.Time { return retentionNow }
	return l, names
}

// survivors runs a launch, whose write prunes, and returns the ages still held.
func survivors(t *testing.T, l *Ledger, names map[int]string) []int {
	t.Helper()
	if _, err := RecordLaunch(context.Background(), l, ledgerStart(), func(context.Context) LaunchEnd { return LaunchEnd{} }); err != nil {
		t.Fatal(err)
	}
	var out []int
	for d, n := range names {
		if _, err := os.Stat(filepath.Join(abcdhome.Path(l.home, ledgerFolder), n)); err == nil {
			out = append(out, d)
		}
	}
	slices.Sort(out)
	return out
}

// The floor: nothing inside the longest cap period is pruned, whatever days and
// count are set. A calendar month plus a day is 32 days, so the shipped 30 days
// holds 31 as well and an entry 33 days old goes.
func TestPruningNeverDropsAnEntryInsideTheLongestCapPeriod(t *testing.T) {
	for name, r := range map[string]Retention{
		"the shipped default": DefaultRetention(),
		"one day":             {Days: 1, Count: DefaultRetentionCount},
		"one entry":           {Days: DefaultRetentionDays, Count: 1},
		"both at their least": {Days: 1, Count: 1},
	} {
		t.Run(name, func(t *testing.T) {
			l, names := agedEntries(t, r, 0, 1, 15, 28, 30, 31, 32, 33, 45)
			got := survivors(t, l, names)
			want := []int{0, 1, 15, 28, 30, 31, 32}
			if !slices.Equal(got, want) {
				t.Errorf("held %v, want %v", got, want)
			}
		})
	}
}

// Past the floor the person's days are what hold an entry, and the oldest whole
// files go first past the count.
func TestPruningRemovesTheOldestWholeFilesPastDaysAndCount(t *testing.T) {
	t.Run("days", func(t *testing.T) {
		l, names := agedEntries(t, Retention{Days: 60, Count: DefaultRetentionCount}, 40, 59, 61, 90)
		if got, want := survivors(t, l, names), []int{40, 59}; !slices.Equal(got, want) {
			t.Errorf("held %v, want %v", got, want)
		}
	})
	t.Run("count", func(t *testing.T) {
		l, names := agedEntries(t, Retention{Days: 365, Count: 3}, 40, 50, 60, 70, 80)
		if got, want := survivors(t, l, names), []int{40, 50}; !slices.Equal(got, want) {
			t.Errorf("held %v, want %v (the new launch is the third)", got, want)
		}
	})
}

// A file whose name is no launch's is not the pruner's to judge.
func TestPruningLeavesAFileItCannotDate(t *testing.T) {
	l, names := agedEntries(t, Retention{Days: 1, Count: 1}, 90)
	stray := filepath.Join(abcdhome.Path(l.home, ledgerFolder), "notes.json")
	if err := os.WriteFile(stray, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := survivors(t, l, names); len(got) != 0 {
		t.Errorf("held %v, want the dated file gone", got)
	}
	if _, err := os.Stat(stray); err != nil {
		t.Errorf("the undated file went: %v", err)
	}
}

// Decision 14: 30 days and 20,000 entries by default; a machine setting
// overrides them and a repository's lengthens them; a figure below the floor is
// read as the floor and named as such.
func TestRetentionIsReadFromTheConfiguration(t *testing.T) {
	if r := mustLoad(t, "", "").Ledger().retention; r != (Retention{Days: 32, Count: 20000}) {
		t.Errorf("default = %+v", r)
	}
	if r := mustLoad(t, `{"ledger":{"retention_days":90,"retention_count":500}}`, "").Ledger().retention; r != (Retention{Days: 90, Count: 500}) {
		t.Errorf("machine = %+v", r)
	}
	if r := mustLoad(t, `{"ledger":{"retention_days":90}}`, `{"ledger":{"retention_days":120,"retention_count":30000}}`).Ledger().retention; r != (Retention{Days: 120, Count: 30000}) {
		t.Errorf("repository over machine = %+v", r)
	}
	c := mustLoad(t, "", `{"ledger":{"retention_days":10}}`)
	if r := c.Ledger().retention; r.Days != RetentionFloorDays {
		t.Errorf("below the floor = %+v", r)
	}
	if !slices.ContainsFunc(c.Diagnostics, func(d string) bool {
		return strings.Contains(d, "ledger.retention_days") && strings.Contains(d, "32")
	}) {
		t.Errorf("the floor is not named: %q", c.Diagnostics)
	}
	for _, bad := range []string{
		`{"ledger":{"retention_days":0}}`,
		`{"ledger":{"retention_count":-1}}`,
		`{"ledger":{"retention_days":"a month"}}`,
		`{"ledger":{"keep":3}}`,
	} {
		if _, err := Load(roots(t, "", bad)); err == nil || !strings.Contains(err.Error(), "ledger") {
			t.Errorf("%s: err = %v", bad, err)
		}
	}
}
