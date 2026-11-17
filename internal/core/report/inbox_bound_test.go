package report

import (
	"fmt"
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

	// Only the listed reports are read whole, and show reads only the one it
	// is asked for.
	read := countReads(t)
	if _, err := Waiting(); err != nil {
		t.Fatalf("Waiting: %v", err)
	}
	if len(*read) != ListLimit {
		t.Errorf("a listing read %d reports whole; want %d", len(*read), ListLimit)
	}
	for _, name := range *read {
		for _, id := range older[:5] {
			if strings.HasPrefix(name, strings.TrimPrefix(id, idFamily+"-")) {
				t.Errorf("a listing read %s, among the five oldest", name)
			}
		}
	}
	*read = nil
	if e, err := Show(older[0]); err != nil || e.ID != older[0] {
		t.Fatalf("Show(%s) = %s, %v", older[0], e.ID, err)
	}
	if len(*read) != 1 {
		t.Errorf("show read %d reports whole; want 1: %v", len(*read), *read)
	}
}

// countReads records the file name of every report read whole until the test
// ends.
func countReads(t *testing.T) *[]string {
	t.Helper()
	var names []string
	prev := readReport
	readReport = func(root *os.Root, rel, name, state string) Entry {
		names = append(names, name)
		return prev(root, rel, name, state)
	}
	t.Cleanup(func() { readReport = prev })
	return &names
}

// TestNewestNamesCutsTheCutoffSecondByWriteTime: where reports share the
// oldest second a bounded listing reaches, they are ordered by when each file
// was written and cut to the limit before any is read whole, however many
// share it.
func TestNewestNamesCutsTheCutoffSecondByWriteTime(t *testing.T) {
	dir := t.TempDir()
	key := strings.Repeat("b", 40)
	n := func(stamp string) string { return stamp + "-" + key + ".md" }
	base := time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC)
	plant := func(name string, written time.Time) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, written, written); err != nil {
			t.Fatal(err)
		}
	}
	// One report in the newest second, then four in the second before it,
	// written in an order their stamps do not follow, then one older still.
	plant(n("2609230900030001"), base.Add(3*time.Second))
	plant(n("2609230900020009"), base.Add(2*time.Second+100*time.Millisecond))
	plant(n("2609230900020001"), base.Add(2*time.Second+400*time.Millisecond))
	plant(n("2609230900020005"), base.Add(2*time.Second+300*time.Millisecond))
	plant(n("2609230900020003"), base.Add(2*time.Second+200*time.Millisecond))
	plant(n("2609230900010001"), base.Add(time.Second))
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	got := newestNames(root, names, 3)
	want := []string{n("2609230900030001"), n("2609230900020001"), n("2609230900020005")}
	if !slices.Equal(got, want) {
		t.Errorf("newestNames(_, 3) = %v; want %v", got, want)
	}
	if got := newestNames(root, names, 9); len(got) != len(names) || got[0] != n("2609230900030001") {
		t.Errorf("newestNames(_, 9) = %v; want all six, newest first", got)
	}
	if got := newestNames(root, names, 1); !slices.Equal(got, want[:1]) {
		t.Errorf("newestNames(_, 1) = %v; want %v", got, want[:1])
	}
}

// TestAListingReadsNoMoreThanTheLimitFromOneSecond: a burst of reports in one
// second past the limit is read whole only up to the limit.
func TestAListingReadsNoMoreThanTheLimitFromOneSecond(t *testing.T) {
	home := sandbox(t, time.Date(2026, 9, 23, 9, 0, 0, 0, time.UTC))
	if _, err := File(mustParse(t, filled(t)), Sender{Key: strings.Repeat("a", 40), Name: "alpha"}); err != nil {
		t.Fatalf("File: %v", err)
	}
	dir := abcdhome.Path(home, "inbox")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	key := fileNameRe.FindStringSubmatch(entries[0].Name())[2]
	for i := range ListLimit + 50 {
		name := "260922000000" + fmt.Sprintf("%04d", i) + "-" + key + ".md"
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	read := countReads(t)
	l, err := Waiting()
	if err != nil {
		t.Fatalf("Waiting: %v", err)
	}
	if len(*read) != ListLimit || len(l.Reports) != ListLimit || l.Left != 51 {
		t.Errorf("read %d whole, listed %d, left %d; want %d, %d, 51", len(*read), len(l.Reports), l.Left, ListLimit, ListLimit)
	}
}
