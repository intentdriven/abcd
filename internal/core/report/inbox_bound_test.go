package report

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// plantOlder writes n copies of the one filed report in the inbox under home,
// each received a second apart in the day before it, and returns their ids
// oldest first.
func plantOlder(t *testing.T, home string, n int) []string {
	t.Helper()
	dir := abcdhome.Path(home, "inbox")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var filed string
	for _, e := range entries {
		if fileNameRe.MatchString(e.Name()) {
			filed = e.Name()
		}
	}
	if filed == "" {
		t.Fatal("no filed report to copy")
	}
	data, err := os.ReadFile(filepath.Join(dir, filed))
	if err != nil {
		t.Fatal(err)
	}
	key := fileNameRe.FindStringSubmatch(filed)[2]
	base := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	ids := make([]string, 0, n)
	for i := range n {
		stamp := base.Add(time.Duration(i)*time.Second).Format("060102150405") + "0000"
		if err := os.WriteFile(filepath.Join(dir, stamp+"-"+key+".md"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, idFamily+"-"+stamp)
	}
	return ids
}

// TestListingReadsTheNewestAndCountsTheRest: a listing reads whole only the
// newest ListLimit waiting reports, says how many older ones it left out, and
// tallies every one that waits (iss-2611150658164454).
func TestListingReadsTheNewestAndCountsTheRest(t *testing.T) {
	home := sandbox(t, time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC))
	f, err := File(mustParse(t, filled(t)), Sender{Key: strings.Repeat("a", 40), Name: "alpha"})
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	older := plantOlder(t, home, ListLimit+4)

	l, err := Waiting()
	if err != nil {
		t.Fatalf("Waiting: %v", err)
	}
	if len(l.Reports) != ListLimit || l.Left != 5 {
		t.Fatalf("Waiting listed %d and left %d; want %d and 5", len(l.Reports), l.Left, ListLimit)
	}
	if l.Tally.Reports != ListLimit+5 || l.Tally.Senders != 1 {
		t.Errorf("tally = %+v; want %d reports from 1 sender", l.Tally, ListLimit+5)
	}
	if l.Reports[0].ID != f.ID {
		t.Errorf("first listed %s; want the newest, %s", l.Reports[0].ID, f.ID)
	}
	listed := make([]string, 0, len(l.Reports))
	for _, e := range l.Reports {
		listed = append(listed, e.ID)
	}
	for _, id := range older[:5] {
		if slices.Contains(listed, id) {
			t.Errorf("%s, among the five oldest, is listed", id)
		}
	}
	list, err := List()
	if err != nil || len(list) != ListLimit {
		t.Errorf("List = %d entries, %v; want %d", len(list), err, ListLimit)
	}
}

// TestNewestNamesKeepsTheWholeCutoffSecond: the names a bounded listing reads
// include every report received in the same second as the last one it keeps,
// since newestFirst orders those by when each was written.
func TestNewestNamesKeepsTheWholeCutoffSecond(t *testing.T) {
	key := strings.Repeat("b", 40)
	n := func(stamp string) string { return stamp + "-" + key + ".md" }
	names := []string{n("2609230900010001"), n("2609230900020001"), n("2609230900020002"), n("2609230900020003"), n("2609230900000001")}
	got := newestNames(names, 2)
	want := []string{n("2609230900020003"), n("2609230900020002"), n("2609230900020001")}
	if !slices.Equal(got, want) {
		t.Errorf("newestNames(_, 2) = %v; want %v", got, want)
	}
	if got := newestNames(names, 9); len(got) != len(names) || got[0] != n("2609230900020003") {
		t.Errorf("newestNames(_, 9) = %v; want all five, newest first", got)
	}
}
