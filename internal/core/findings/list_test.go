package findings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

const testRoot = "0123456789abcdef0123456789abcdef01234567"

var t0 = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

// newTestStore is a store under a fresh home that HOME also names, so the
// home path a text carries is the one the redactor rewrites.
func newTestStore(t *testing.T) (Store, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	s, err := NewStore(home, testRoot)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s, home
}

// cheapRedactor stands in for the ledger's scanner where a test puts many
// entries: building one runs git, and what is judged there is not redaction.
func cheapRedactor(t *testing.T) {
	t.Helper()
	prev := newRedactor
	newRedactor = func(context.Context, string) (textRedactor, error) { return passThrough{}, nil }
	t.Cleanup(func() { newRedactor = prev })
}

type passThrough struct{}

func (passThrough) Redact(text string) string { return text }

var bg = context.Background()

// put1 puts one entry.
func put1(s Store, e Entry) (Entry, error) {
	got, err := s.Put(bg, e)
	if len(got) == 1 {
		return got[0], err
	}
	return Entry{}, err
}

func entry(kind Kind, session, text string) Entry {
	return Entry{Kind: kind, Session: session, Checkout: "/checkout", Text: text, At: t0}
}

func mustPut(t *testing.T, s Store, e Entry) Entry {
	t.Helper()
	got, err := put1(s, e)
	if err != nil {
		t.Fatalf("Put(%+v): %v", e, err)
	}
	return got
}

func mustRead(t *testing.T, s Store) View {
	t.Helper()
	v, err := s.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return v
}

func storeDir(home, sub string) string {
	return abcdhome.Path(home, "findings", testRoot, sub)
}

func filesIn(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		out = append(out, e.Name())
	}
	return out
}

func TestSaidTwiceInOneSessionIsOneFileAndInTwoSessionsTwo(t *testing.T) {
	s, home := newTestStore(t)
	cheapRedactor(t)
	a := mustPut(t, s, entry(KindUntested, "s1", "the  Gate skips a FILE"))
	b := mustPut(t, s, entry(KindUntested, "s1", "the gate skips a file"))
	if a.Name != b.Name {
		t.Fatalf("one finding said twice in one session got two names: %s and %s", a.Name, b.Name)
	}
	if n := len(filesIn(t, storeDir(home, "open"))); n != 1 {
		t.Fatalf("said twice in one session left %d files, want 1", n)
	}
	c := mustPut(t, s, entry(KindUntested, "s2", "the gate skips a file"))
	if c.Name == a.Name {
		t.Fatal("another session's saying of a finding shares the first session's name")
	}
	if n := len(filesIn(t, storeDir(home, "open"))); n != 2 {
		t.Fatalf("said in two sessions left %d files, want 2", n)
	}
	if !strings.HasPrefix(a.Handle(), "f-") || len(a.Handle()) != 10 {
		t.Fatalf("handle %q is not f- and eight characters", a.Handle())
	}
}

func TestAGateEntryIsNamedByTheHead(t *testing.T) {
	s, _ := newTestStore(t)
	cheapRedactor(t)
	gate := func(session, head string) Entry {
		return Entry{Kind: KindCheck, Session: session, Checkout: "/lane", Place: "/lane", Head: head, Text: "preflight failed", At: t0}
	}
	a := mustPut(t, s, gate("s1", "aaaa"))
	b := mustPut(t, s, gate("s2", "aaaa"))
	if a.Name != b.Name {
		t.Fatal("a gate's failure at one head got two names in two sessions; a gate's entry is named by the place and the head")
	}
	if c := mustPut(t, s, gate("s1", "bbbb")); c.Name == a.Name {
		t.Fatal("a gate's failure at a new head reused the old head's name")
	}
}

func TestASettledEntryStaysSettledAndAReceiptSettledCheckDoesNot(t *testing.T) {
	s, home := newTestStore(t)
	cheapRedactor(t)
	e := mustPut(t, s, entry(KindLine, "s1", "left for later"))
	if _, err := s.Settle(bg, SettleRequest{Handles: []string{e.Handle()}, Act: ActNotAFinding, Reason: "prose", At: t0}); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	again := mustPut(t, s, entry(KindLine, "s1", "left for later"))
	if again.Settled == nil {
		t.Fatal("an entry put again after it was settled came back unsettled")
	}
	if v := mustRead(t, s); len(v.Open) != 0 {
		t.Fatalf("a settled entry put again is open again: %+v", v.Open)
	}

	c := mustPut(t, s, entry(KindCheck, "s1", "TestX still fails"))
	if _, err := s.SettleByReceipt(bg, []string{c.Handle()}, "receipt at cafe", t0); err != nil {
		t.Fatalf("Settle by receipt: %v", err)
	}
	if v := mustRead(t, s); len(v.Open) != 0 {
		t.Fatalf("a receipt did not settle the check: %+v", v.Open)
	}
	if back := mustPut(t, s, entry(KindCheck, "s1", "TestX still fails")); back.Settled != nil {
		t.Fatal("a check a receipt settled stayed settled when it was put again")
	}
	if v := mustRead(t, s); len(v.Open) != 1 {
		t.Fatalf("a receipt-settled check put again: %d open, want 1", len(v.Open))
	}
	for _, f := range filesIn(t, storeDir(home, "settled")) {
		if strings.HasPrefix(f, c.Name) {
			t.Fatalf("a check put again after its receipt kept its settled file %s, which would win over it", f)
		}
	}
}

func TestACrashInsideASettleLeavesTheSettledOneWinning(t *testing.T) {
	s, home := newTestStore(t)
	cheapRedactor(t)
	e := mustPut(t, s, entry(KindUntested, "s1", "the cache is never cleared"))
	open := filepath.Join(storeDir(home, "open"), e.Name+".json")
	raw, err := os.ReadFile(open)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Settle(bg, SettleRequest{Handles: []string{e.Handle()}, Act: ActRefuted, Reason: "go test ./x passes on the base", At: t0}); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	// What a crash between the settled write and the open removal leaves.
	if err := os.WriteFile(open, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if v := mustRead(t, s); len(v.Open) != 0 {
		t.Fatalf("with both files standing the open one won: %+v", v.Open)
	}
	_, err = s.Settle(bg, SettleRequest{Handles: []string{e.Handle()}, Act: ActRefuted, Reason: "again", At: t0})
	if !errors.Is(err, ErrNoEntry) {
		t.Fatalf("settling a handle settled/ already holds = %v; want ErrNoEntry, as the list reads it", err)
	}
	if _, err := os.Lstat(open); !os.IsNotExist(err) {
		t.Fatalf("the stale open twin was left in place (%v)", err)
	}
}

func TestACrashInsideAMoveToPrivateLeavesThePrivateOneWinning(t *testing.T) {
	s, home := newTestStore(t)
	cheapRedactor(t)
	e := mustPut(t, s, entry(KindUntested, "s1", "the token route"))
	open := filepath.Join(storeDir(home, "open"), e.Name+".json")
	raw, err := os.ReadFile(open)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Settle(bg, SettleRequest{Handles: []string{e.Handle()}, Act: ActPrivate, At: t0}); err != nil {
		t.Fatalf("Settle --private: %v", err)
	}
	if err := os.WriteFile(open, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if v := mustRead(t, s); len(v.Open) != 0 || len(v.Private) != 1 {
		t.Fatalf("with both files standing: open %+v, private %+v; want the private one alone", v.Open, v.Private)
	}
	if _, err := s.Settle(bg, SettleRequest{Handles: []string{e.Handle()}, Act: ActRefuted, Reason: "ran it", Person: true, At: t0}); err != nil {
		t.Fatalf("settling the private one with its open twin standing: %v", err)
	}
	if _, err := os.Lstat(open); !os.IsNotExist(err) {
		t.Fatalf("the stale open twin was left in place (%v)", err)
	}
}

func TestNothingRemovesAnEntryByAgeOrByItsWorktreeGoing(t *testing.T) {
	s, _ := newTestStore(t)
	cheapRedactor(t)
	wt := t.TempDir()
	e := entry(KindUntested, "s1", "the page names the wrong flag")
	e.Checkout, e.Place = wt, wt
	e.At = t0.AddDate(-1, 0, 0)
	mustPut(t, s, e)
	if err := os.RemoveAll(wt); err != nil {
		t.Fatal(err)
	}
	other, err := NewStore(s.home, testRoot)
	if err != nil {
		t.Fatal(err)
	}
	v := mustRead(t, other)
	if len(v.Open) != 1 || !v.Open[0].At.Equal(e.At) {
		t.Fatalf("a year-old entry whose worktree is gone, read by a new store: %+v", v.Open)
	}
}

func TestAnUnsafeStoreIsRefusedAndNamed(t *testing.T) {
	t.Run("a linked directory", func(t *testing.T) {
		s, home := newTestStore(t)
		elsewhere := t.TempDir()
		if err := os.MkdirAll(abcdhome.Path(home, "findings"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(elsewhere, abcdhome.Path(home, "findings", testRoot)); err != nil {
			t.Fatal(err)
		}
		_, err := put1(s, entry(KindUntested, "s1", "x"))
		var link *fsutil.HomeScopeLinkError
		if !errors.As(err, &link) || !strings.Contains(err.Error(), testRoot) {
			t.Fatalf("Put through a linked store = %v; want the link refused and named", err)
		}
		if _, err := s.Read(); !errors.As(err, &link) {
			t.Fatalf("Read through a linked store = %v; want the link refused", err)
		}
	})
	t.Run("a loose directory", func(t *testing.T) {
		s, home := newTestStore(t)
		cheapRedactor(t)
		mustPut(t, s, entry(KindUntested, "s1", "x"))
		if err := os.Chmod(storeDir(home, "open"), 0o777); err != nil {
			t.Fatal(err)
		}
		_, err := s.Read()
		if !errors.Is(err, fsutil.ErrHomeScopeExposed) || !strings.Contains(err.Error(), "open") {
			t.Fatalf("Read of a world-writable open/ = %v; want it refused and named", err)
		}
	})
	t.Run("a loose file, a linked file and another owner's file", func(t *testing.T) {
		s, home := newTestStore(t)
		cheapRedactor(t)
		loose := mustPut(t, s, entry(KindUntested, "s1", "loose"))
		linked := mustPut(t, s, entry(KindUntested, "s1", "linked"))
		foreign := mustPut(t, s, entry(KindUntested, "s1", "foreign"))
		dir := storeDir(home, "open")
		if err := os.Chmod(filepath.Join(dir, loose.Name+".json"), 0o644); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(t.TempDir(), "elsewhere.json")
		if err := os.Rename(filepath.Join(dir, linked.Name+".json"), target); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(dir, linked.Name+".json")); err != nil {
			t.Fatal(err)
		}
		foreignPath := filepath.Join(dir, foreign.Name+".json")
		restore := fsutil.SwapOwnerUIDForTest(func(p string) (uint32, error) {
			if p == foreignPath {
				return uint32(os.Getuid()) + 1, nil
			}
			return fsutil.OwnerUID(p)
		})
		defer restore()
		v := mustRead(t, s)
		if len(v.Open) != 0 {
			t.Fatalf("an unsafe file was read as an entry: %+v", v.Open)
		}
		named := map[string]bool{}
		for _, u := range v.Unreadable {
			named[u.Handle] = u.Reason != ""
		}
		for _, e := range []Entry{loose, linked, foreign} {
			if !named[e.Handle()] {
				t.Errorf("%s was not named as unreadable with a reason: %+v", e.Handle(), v.Unreadable)
			}
		}
	})
}

func TestTheStoreIsPrivate(t *testing.T) {
	s, home := newTestStore(t)
	cheapRedactor(t)
	e := mustPut(t, s, entry(KindUntested, "s1", "x"))
	for _, p := range []string{abcdhome.Path(home, "findings"), abcdhome.Path(home, "findings", testRoot), storeDir(home, "open")} {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o700 {
			t.Errorf("%s is %04o, want 0700", p, fi.Mode().Perm())
		}
	}
	fi, err := os.Stat(filepath.Join(storeDir(home, "open"), e.Name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("an entry's file is %04o, want 0600", fi.Mode().Perm())
	}
}

func TestReadingAnAbsentStoreMakesNothing(t *testing.T) {
	s, home := newTestStore(t)
	v := mustRead(t, s)
	if len(v.Open)+len(v.Private)+len(v.Newer)+len(v.Unreadable) != 0 || v.Full {
		t.Fatalf("an absent store read as %+v", v)
	}
	if _, err := os.Lstat(abcdhome.Path(home)); !os.IsNotExist(err) {
		t.Fatalf("reading an absent store made abcd's home (%v)", err)
	}
	if got, bad, err := s.SettledView(50); err != nil || len(got)+len(bad) != 0 {
		t.Fatalf("the settled view of an absent store = %v, %v", got, err)
	}
}

func TestTheBoundsHold(t *testing.T) {
	t.Run("one file", func(t *testing.T) {
		s, home := newTestStore(t)
		cheapRedactor(t)
		e := mustPut(t, s, entry(KindUntested, "s1", "x"))
		big := `{"schema_version":1,"kind":"untested","text":"` + strings.Repeat("y", maxFileBytes) + `"}`
		if err := os.WriteFile(filepath.Join(storeDir(home, "open"), e.Name+".json"), []byte(big), 0o600); err != nil {
			t.Fatal(err)
		}
		v := mustRead(t, s)
		if len(v.Open) != 0 || len(v.Unreadable) != 1 || v.Unreadable[0].Handle != e.Handle() {
			t.Fatalf("a file past %d bytes: open %d, unreadable %+v; want it refused and named", maxFileBytes, len(v.Open), v.Unreadable)
		}
	})
	t.Run("entry text", func(t *testing.T) {
		s, _ := newTestStore(t)
		cheapRedactor(t)
		e := mustPut(t, s, entry(KindUntested, "s1", strings.Repeat("é", MaxTextBytes)))
		if len(e.Text) > MaxTextBytes || !strings.HasPrefix(strings.Repeat("é", MaxTextBytes), e.Text) || e.Text == "" {
			t.Fatalf("a long text was kept as %d bytes; want it cut to at most %d on a rune", len(e.Text), MaxTextBytes)
		}
	})
	t.Run("open entries for a root", func(t *testing.T) {
		s, _ := newTestStore(t)
		cheapRedactor(t)
		for i := 0; i < MaxOpen; i++ {
			plant(t, s, "open", fmt.Sprintf("%064x", i), `{"schema_version":1,"kind":"untested","text":"x","at":"2026-11-25T12:00:00Z"}`)
		}
		_, err := put1(s, entry(KindUntested, "s1", "one more"))
		if !errors.Is(err, ErrFull) {
			t.Fatalf("a put past %d open entries = %v; want ErrFull", MaxOpen, err)
		}
		if v := mustRead(t, s); !v.Full {
			t.Fatal("a list at its bound does not say it is full")
		}
	})
	t.Run("files read", func(t *testing.T) {
		s, _ := newTestStore(t)
		for i := 0; i <= maxFilesRead; i++ {
			plant(t, s, "private", fmt.Sprintf("%064x", i), `{"schema_version":1,"kind":"private","text":"x","at":"2026-11-25T12:00:00Z"}`)
		}
		if _, err := s.Read(); !errors.Is(err, ErrTooManyFiles) {
			t.Fatalf("a read past %d files = %v; want ErrTooManyFiles", maxFilesRead, err)
		}
	})
}

// plant writes one raw entry file the way the store lays them, for a test
// that needs a file Put would not write.
func plant(t *testing.T, s Store, sub, name, body string) {
	t.Helper()
	dir := abcdhome.Path(s.home, "findings", testRoot, sub)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{abcdhome.Path(s.home), abcdhome.Path(s.home, "findings"), abcdhome.Path(s.home, "findings", testRoot), dir} {
		if err := os.Chmod(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestALaterSchemaAndAnUnknownKindAreANewerAbcds(t *testing.T) {
	s, _ := newTestStore(t)
	later := strings.Repeat("a", 64)
	unknown := strings.Repeat("b", 64)
	plant(t, s, "open", later, `{"schema_version":2,"kind":"untested","text":"x"}`)
	plant(t, s, "open", unknown, `{"schema_version":1,"kind":"omen","text":"x"}`)
	v := mustRead(t, s)
	if len(v.Open) != 0 || len(v.Newer) != 2 {
		t.Fatalf("a later schema and an unknown kind read as open %+v, newer %v; want both a newer abcd's", v.Open, v.Newer)
	}
	for _, h := range []string{"f-aaaaaaaa", "f-bbbbbbbb"} {
		if _, err := s.Settle(bg, SettleRequest{Handles: []string{h}, Act: ActCaptured, Record: "iss-1", Records: holding{}, At: t0}); !errors.Is(err, ErrNewer) {
			t.Errorf("settling a newer abcd's %s = %v; want ErrNewer", h, err)
		}
	}
	if _, err := put1(s, entry(Kind("omen"), "s1", "x")); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("putting a kind this build does not know = %v; want ErrUnknownKind", err)
	}
}

func TestTwoWritersAtOnceWriteOneFile(t *testing.T) {
	s, home := newTestStore(t)
	cheapRedactor(t)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := put1(s, entry(KindHandedUp, "s1", "the same finding")); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("a concurrent Put: %v", err)
	}
	if got := filesIn(t, storeDir(home, "open")); len(got) != 1 || !strings.HasSuffix(got[0], ".json") {
		t.Fatalf("sixteen writers of one finding left %v, want one entry file", got)
	}
}

func TestTextIsRedactedAndWithheldWhenTheScannerCannotBeMade(t *testing.T) {
	s, home := newTestStore(t)
	repo := gittest.NewRepo(t).Root()
	token := "ghp_" + strings.Repeat("A1", 18)
	planted := entry(KindUntested, "s1", "the token "+token+" is in "+filepath.Join(home, "x.txt"))
	planted.Checkout = repo
	e := mustPut(t, s, planted)
	if strings.Contains(e.Text, token) || strings.Contains(e.Text, home) {
		t.Fatalf("an entry kept the planted token or the home path: %q", e.Text)
	}
	raw, err := os.ReadFile(filepath.Join(storeDir(home, "open"), e.Name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), token) || strings.Contains(string(raw), home) {
		t.Fatalf("the entry's file kept the planted token or the home path:\n%s", raw)
	}

	prev := newRedactor
	newRedactor = func(context.Context, string) (textRedactor, error) { return nil, capture.ErrScannerUnavailable }
	defer func() { newRedactor = prev }()
	w := mustPut(t, s, entry(KindUntested, "s1", "a second finding "+token))
	if !w.Withheld || w.Text != "" {
		t.Fatalf("with no scanner the entry was put as %+v; want its text withheld", w)
	}
}

// TestTheKindTableIsWhatClearsAnEntry walks every cell of the spec's table.
func TestTheKindTableIsWhatClearsAnEntry(t *testing.T) {
	yes := map[Kind][]Act{
		KindUntested: {ActCaptured, ActSettled, ActRefuted, ActPrivate},
		KindHandedUp: {ActCaptured, ActSettled, ActRefuted, ActPrivate},
		KindDoubt:    {ActCaptured, ActSettled},
		KindKnown:    {ActCaptured, ActSettled, ActRefuted, ActPrivate},
		KindLine:     {ActCaptured, ActSettled, ActRefuted, ActNotAFinding, ActPrivate},
		KindCheck:    {ActCaptured, ActSettled, ActOwn, ActPrivate},
		KindPrivate:  {ActCaptured, ActSettled, ActRefuted},
	}
	acts := []Act{ActCaptured, ActSettled, ActRefuted, ActOwn, ActNotAFinding, ActPrivate}
	for kind, allowed := range yes {
		for _, act := range acts {
			want := false
			for _, a := range allowed {
				want = want || a == act
			}
			for _, person := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/person=%v", kind, act, person), func(t *testing.T) {
					s, home := newTestStore(t)
					cheapRedactor(t)
					e := mustPut(t, s, entry(kind, "s1", "a finding"))
					req := SettleRequest{Handles: []string{e.Handle()}, Act: act, Person: person, Records: holding{}, At: t0}
					switch act {
					case ActCaptured, ActSettled:
						req.Record = "iss-1"
					case ActRefuted:
						req.Reason = "go test ./x on the base passes"
					case ActOwn:
						req.Reason = "run 2611251200000000"
					case ActNotAFinding:
						req.Reason = "ordinary prose"
					}
					_, err := s.Settle(bg, req)
					cell := want && (kind != KindPrivate || person)
					if cell && err != nil {
						t.Fatalf("the table allows it and Settle refused: %v", err)
					}
					if !cell {
						if !errors.Is(err, ErrNotCleared) {
							t.Fatalf("the table does not allow it and Settle = %v; want ErrNotCleared", err)
						}
						if v := mustRead(t, s); len(v.Open)+len(v.Private) != 1 {
							t.Fatalf("a refused settle changed the list: %+v", v)
						}
						return
					}
					v := mustRead(t, s)
					if act == ActPrivate {
						if len(v.Open) != 0 || len(v.Private) != 1 || v.Private[0].Kind != KindPrivate {
							t.Fatalf("--private left open %+v, private %+v; want the entry on the private list", v.Open, v.Private)
						}
						return
					}
					if len(v.Open)+len(v.Private) != 0 {
						t.Fatalf("a settle the table allows left the entry on the list: %+v", v)
					}
					got, _, err := s.SettledView(50)
					if err != nil || len(got) != 1 || got[0].Settled == nil || got[0].Settled.Act != act {
						t.Fatalf("the settled view after %s = %+v, %v", act, got, err)
					}
					if req.Reason != "" && got[0].Settled.Reason != req.Reason {
						t.Fatalf("the settled view kept reason %q, want %q", got[0].Settled.Reason, req.Reason)
					}
					_ = home
				})
			}
		}
	}
}

// holding answers every record as holding, for the kind table's walk, which
// judges the table and not the records.
type holding struct{}

func (holding) Answer(Act, string) (Answer, error) { return Answer{Holds: true, State: "open"}, nil }

type refusing struct{}

func (refusing) Answer(Act, string) (Answer, error) {
	return Answer{State: "resolved", Where: "the default branch"}, nil
}

func TestASettleWritesNothingWhenAnyPartOfItIsRefused(t *testing.T) {
	s, _ := newTestStore(t)
	cheapRedactor(t)
	a := mustPut(t, s, entry(KindUntested, "s1", "one"))
	b := mustPut(t, s, entry(KindDoubt, "s1", "two"))
	for name, req := range map[string]SettleRequest{
		"a record that does not hold": {Handles: []string{a.Handle()}, Act: ActCaptured, Record: "iss-1", Records: refusing{}},
		"no record":                   {Handles: []string{a.Handle()}, Act: ActCaptured, Records: holding{}},
		"no reason":                   {Handles: []string{a.Handle()}, Act: ActRefuted},
		"one handle of two refused":   {Handles: []string{a.Handle(), b.Handle()}, Act: ActRefuted, Reason: "ran it"},
		"an unknown handle":           {Handles: []string{a.Handle(), "f-ffffffff"}, Act: ActRefuted, Reason: "ran it"},
		"a handle of no shape":        {Handles: []string{"iss-1"}, Act: ActRefuted, Reason: "ran it"},
	} {
		t.Run(name, func(t *testing.T) {
			req.At = t0
			if _, err := s.Settle(bg, req); err == nil {
				t.Fatal("Settle accepted it")
			}
			if v := mustRead(t, s); len(v.Open) != 2 {
				t.Fatalf("a refused settle changed the list: %+v", v.Open)
			}
		})
	}
	_, err := s.Settle(bg, SettleRequest{Handles: []string{a.Handle()}, Act: ActCaptured, Record: "iss-1", Records: refusing{}, At: t0})
	if !strings.Contains(fmt.Sprint(err), "resolved") || !strings.Contains(fmt.Sprint(err), "the default branch") {
		t.Fatalf("a record that does not hold was refused as %v; want the state found named", err)
	}
}

func TestAnAmbiguousHandleIsRefused(t *testing.T) {
	s, _ := newTestStore(t)
	plant(t, s, "open", "abcdef01"+strings.Repeat("1", 56), `{"schema_version":1,"kind":"untested","text":"x","at":"2026-11-25T12:00:00Z"}`)
	plant(t, s, "open", "abcdef01"+strings.Repeat("2", 56), `{"schema_version":1,"kind":"untested","text":"y","at":"2026-11-25T12:00:00Z"}`)
	_, err := s.Settle(bg, SettleRequest{Handles: []string{"f-abcdef01"}, Act: ActRefuted, Reason: "ran it", At: t0})
	if !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("an ambiguous handle = %v; want ErrAmbiguous", err)
	}
}

func TestTheOpenViewPutsThisCheckoutFirstAndTheSettledViewKeepsTheLastFifty(t *testing.T) {
	s, _ := newTestStore(t)
	cheapRedactor(t)
	for i, co := range []string{"/other", "/here", "/other", "/here"} {
		e := entry(KindUntested, "s1", fmt.Sprintf("finding %d", i))
		e.Checkout = co
		e.At = t0.Add(time.Duration(i) * time.Minute)
		mustPut(t, s, e)
	}
	got := mustRead(t, s).Ordered("/here")
	var order []string
	for _, e := range got {
		order = append(order, e.Checkout+":"+e.Text)
	}
	want := "/here:finding 1 /here:finding 3 /other:finding 0 /other:finding 2"
	if strings.Join(order, " ") != want {
		t.Fatalf("ordered = %v, want %s", order, want)
	}

	for i := 0; i < 55; i++ {
		e := mustPut(t, s, entry(KindLine, "s2", fmt.Sprintf("line %d", i)))
		if _, err := s.Settle(bg, SettleRequest{Handles: []string{e.Handle()}, Act: ActNotAFinding, Reason: fmt.Sprintf("why %d", i), At: t0.Add(time.Duration(i) * time.Second)}); err != nil {
			t.Fatal(err)
		}
	}
	last, _, err := s.SettledView(50)
	if err != nil {
		t.Fatal(err)
	}
	if len(last) != 50 || last[0].Settled.Reason != "why 54" || last[49].Settled.Reason != "why 5" {
		t.Fatalf("the settled view held %d, first %q, last %q; want the last fifty, newest first", len(last), last[0].Settled.Reason, last[len(last)-1].Settled.Reason)
	}
}

func TestAnEntryFileCarriesSchemaVersionOne(t *testing.T) {
	s, home := newTestStore(t)
	cheapRedactor(t)
	e := mustPut(t, s, entry(KindDoubt, "s1", "x"))
	raw, err := os.ReadFile(filepath.Join(storeDir(home, "open"), e.Name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var head struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &head); err != nil || head.SchemaVersion != 1 {
		t.Fatalf("the entry's file reads schema_version %d (%v):\n%s", head.SchemaVersion, err, raw)
	}
}

func TestAPrivateEntryIsPutOnThePrivateList(t *testing.T) {
	s, home := newTestStore(t)
	cheapRedactor(t)
	e := mustPut(t, s, entry(KindPrivate, "s1", "the token route"))
	if got := filesIn(t, storeDir(home, "private")); len(got) != 1 || got[0] != e.Name+".json" {
		t.Fatalf("a private entry landed as %v in private/", got)
	}
	v := mustRead(t, s)
	if len(v.Open) != 0 || len(v.Private) != 1 {
		t.Fatalf("a private entry read as open %+v, private %+v", v.Open, v.Private)
	}
}

// TestWhatTheStoreWritesFitsItsBound is review MUST 1: text and a reason that
// JSON would escape, and a field past its bound, still make a file the store
// reads back.
func TestWhatTheStoreWritesFitsItsBound(t *testing.T) {
	s, _ := newTestStore(t)
	cheapRedactor(t)
	e := mustPut(t, s, entry(KindLine, "s1", strings.Repeat("<", MaxTextBytes)))
	reason := strings.Repeat("&", MaxTextBytes)
	if _, err := s.Settle(bg, SettleRequest{Handles: []string{e.Handle()}, Act: ActNotAFinding, Reason: reason, At: t0}); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	got, bad, err := s.SettledView(50)
	if err != nil || len(got) != 1 || got[0].Text != strings.Repeat("<", MaxTextBytes) || got[0].Settled.Reason != reason {
		t.Fatalf("a 600-byte text of < and reason of & read back as %d entries, %+v unreadable, %v", len(got), bad, err)
	}

	long := entry(KindUntested, strings.Repeat("s", 5000), "a long session")
	if _, err := put1(s, long); err != nil {
		t.Fatalf("a 5,000-byte session: %v", err)
	}
	v := mustRead(t, s)
	if len(v.Open) != 1 || len(v.Unreadable) != 0 || len(v.Open[0].Session) > maxIDBytes {
		t.Fatalf("a 5,000-byte session read back as open %d, unreadable %+v", len(v.Open), v.Unreadable)
	}
}

func TestTheSettledViewNamesAFileItCannotRead(t *testing.T) {
	s, _ := newTestStore(t)
	name := strings.Repeat("c", 64)
	plant(t, s, "settled", name, `{"schema_version":1,"kind":"untested","text":"`+strings.Repeat("y", maxFileBytes)+`"}`)
	_, bad, err := s.SettledView(50)
	if err != nil || len(bad) != 1 || bad[0].Handle != "f-cccccccc" {
		t.Fatalf("an oversized settled file: unreadable %+v, %v; want it named", bad, err)
	}
}

// writePII gives a checkout a pattern file of its own.
func writePII(t *testing.T, repo, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config", "pii.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTextTheFullScanCannotClearIsWithheld is review MUST 2: a scanner that
// runs degraded clears nothing, so the text is withheld and the entry says so.
func TestTextTheFullScanCannotClearIsWithheld(t *testing.T) {
	s, _ := newTestStore(t)
	repo := gittest.NewRepo(t).Root()
	writePII(t, repo, `{`)
	e := entry(KindUntested, "s1", "a finding the scan cannot clear")
	e.Checkout = repo
	got := mustPut(t, s, e)
	if !got.Withheld || got.Text != "" {
		t.Fatalf("under a degraded scan the entry was put as %+v; want its text withheld", got)
	}
	// A withheld entry is named at random: its name and handle are no
	// function of the text, so a repeat is its own entry.
	again := mustPut(t, s, e)
	if again.Name == got.Name || again.Handle() == got.Handle() {
		t.Fatalf("two puts of one withheld text share the name %s; want two entries", got.Name)
	}
	for _, n := range []string{got.Name, again.Name} {
		if n == nameOf(e, e.Text) || !entryFileRe.MatchString(n+".json") {
			t.Fatalf("a withheld entry is named %q, the hash of its text or no name of the store's shape", n)
		}
	}
	if v := mustRead(t, s); len(v.Open) != 2 {
		t.Fatalf("the list holds %d withheld entries; want 2", len(v.Open))
	}
}

func TestASettleReasonIsRedactedWithItsEntrysPatterns(t *testing.T) {
	s, _ := newTestStore(t)
	repo := gittest.NewRepo(t).Root()
	writePII(t, repo, `{"patterns":{"house_token":{"regex":"HOUSE-[0-9]{6}","kind":"house_token","label":"house token","severity":"hard_fail"}}}`)
	e := entry(KindUntested, "s1", "a finding")
	e.Checkout = repo
	got := mustPut(t, s, e)
	if _, err := s.Settle(bg, SettleRequest{Handles: []string{got.Handle()}, Act: ActRefuted, Reason: "ran it with HOUSE-123456", At: t0}); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	last, _, err := s.SettledView(50)
	if err != nil || len(last) != 1 || strings.Contains(last[0].Settled.Reason, "HOUSE-123456") || last[0].Settled.Reason == "" {
		t.Fatalf("the settled reason reads %+v (%v); want the repository's pattern redacted", last, err)
	}
}

func TestAPutRacingASettleLeavesItSettled(t *testing.T) {
	s, _ := newTestStore(t)
	cheapRedactor(t)
	e := mustPut(t, s, entry(KindUntested, "s1", "raced"))
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_, _ = put1(s, entry(KindUntested, "s1", "raced"))
		}
	}()
	var settleErr error
	go func() {
		defer wg.Done()
		_, settleErr = s.Settle(bg, SettleRequest{Handles: []string{e.Handle()}, Act: ActRefuted, Reason: "ran it", At: t0})
	}()
	wg.Wait()
	if settleErr != nil {
		t.Fatalf("Settle beside a Put: %v", settleErr)
	}
	if v := mustRead(t, s); len(v.Open) != 0 {
		t.Fatalf("a settled entry read as open after a racing Put: %+v", v.Open)
	}
	if again := mustPut(t, s, entry(KindUntested, "s1", "raced")); again.Settled == nil {
		t.Fatal("a Put after the race came back unsettled")
	}
}

func TestThePrivateListHasItsOwnBoundAndTwinsCountOnce(t *testing.T) {
	s, _ := newTestStore(t)
	cheapRedactor(t)
	for i := 0; i < MaxPrivate; i++ {
		plant(t, s, "private", fmt.Sprintf("%064x", i), `{"schema_version":1,"kind":"private","text":"x","at":"2026-11-25T12:00:00Z"}`)
	}
	if _, err := put1(s, entry(KindPrivate, "s1", "one more")); !errors.Is(err, ErrFull) {
		t.Fatalf("a private put past %d = %v; want ErrFull", MaxPrivate, err)
	}

	s2, _ := newTestStore(t)
	for i := 0; i < MaxOpen; i++ {
		plant(t, s2, "open", fmt.Sprintf("%064x", i), `{"schema_version":1,"kind":"untested","text":"x","at":"2026-11-25T12:00:00Z"}`)
	}
	plant(t, s2, "settled", fmt.Sprintf("%064x", 0), `{"schema_version":1,"kind":"untested","text":"x","at":"2026-11-25T12:00:00Z","settled":{"act":"refuted","reason":"r","at":"2026-11-25T12:00:00Z"}}`)
	if _, err := put1(s2, entry(KindUntested, "s1", "one more")); err != nil {
		t.Fatalf("a put with %d open files, one of them settled already: %v; want it counted once", MaxOpen, err)
	}
}

func TestTheCommandReadsPastTheHooksBudget(t *testing.T) {
	s, _ := newTestStore(t)
	for i := 0; i <= maxFilesRead; i++ {
		plant(t, s, "open", fmt.Sprintf("%064x", i), `{"schema_version":1,"kind":"untested","text":"x","at":"2026-11-25T12:00:00Z"}`)
	}
	v, err := s.ReadWithin(2 * maxFilesRead)
	if err != nil || len(v.Open) != maxFilesRead+1 {
		t.Fatalf("a read within %d files = %d open, %v", 2*maxFilesRead, len(v.Open), err)
	}
}

func TestPrivateTextReachesOnlyItsOneAccessor(t *testing.T) {
	s, _ := newTestStore(t)
	cheapRedactor(t)
	const plantText = "PLANTED-PRIVATE-TEXT"
	e := mustPut(t, s, entry(KindUntested, "s1", plantText))
	moved, err := s.Settle(bg, SettleRequest{Handles: []string{e.Handle()}, Act: ActPrivate, At: t0})
	if err != nil {
		t.Fatal(err)
	}
	direct := mustPut(t, s, entry(KindPrivate, "s1", plantText+" by command"))
	again := mustPut(t, s, entry(KindUntested, "s1", plantText))
	v := mustRead(t, s)
	view, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	for what, out := range map[string]string{
		"the moved entry": fmt.Sprint(moved), "the private put": direct.Text,
		"a put the private list holds": again.Text, "the view": string(view),
	} {
		if strings.Contains(out, plantText) {
			t.Errorf("%s carries the private text: %s", what, out)
		}
	}
	if got, err := s.PrivateText(e.Handle()); err != nil || got != plantText {
		t.Fatalf("PrivateText = %q, %v; want the text", got, err)
	}
	if _, err := s.Settle(bg, SettleRequest{Handles: []string{e.Handle()}, Act: ActRefuted, Reason: "ran it", Person: true, At: t0}); err != nil {
		t.Fatal(err)
	}
	last, _, err := s.SettledView(50)
	if err != nil || len(last) != 1 || strings.Contains(fmt.Sprint(last), plantText) {
		t.Fatalf("the settled view carries the private text: %+v, %v", last, err)
	}
}

func TestOnlyTheReceiptEntryPointSettlesByReceipt(t *testing.T) {
	s, _ := newTestStore(t)
	cheapRedactor(t)
	c := mustPut(t, s, entry(KindCheck, "s1", "TestX still fails"))
	if _, err := s.Settle(bg, SettleRequest{Handles: []string{c.Handle()}, Act: ActReceipt, Reason: "receipt", At: t0}); !errors.Is(err, ErrNotCleared) {
		t.Fatalf("Settle with ActReceipt = %v; want ErrNotCleared", err)
	}
	if v := mustRead(t, s); len(v.Open) != 1 {
		t.Fatal("a refused receipt settle changed the list")
	}
}

func TestARefusalQuotesTheRecordAndNamesNoHomePath(t *testing.T) {
	s, home := newTestStore(t)
	cheapRedactor(t)
	a := mustPut(t, s, entry(KindUntested, "s1", "one"))
	_, err := s.Settle(bg, SettleRequest{Handles: []string{a.Handle()}, Act: ActCaptured, Record: "iss-1", Records: refusing{}, At: t0})
	if !strings.Contains(fmt.Sprint(err), `"iss-1"`) {
		t.Fatalf("the refusal names the record as %v; want it quoted", err)
	}
	if err := os.Chmod(filepath.Join(storeDir(home, "open"), a.Name+".json"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := mustRead(t, s)
	if len(v.Unreadable) != 1 || strings.Contains(v.Unreadable[0].Reason, home) {
		t.Fatalf("the unreadable reason names the home path: %+v", v.Unreadable)
	}
}

// TestANameIsTheRedactedText is the coordinator's ruling: two sayings that
// differ only in what the redactor removes are one entry.
func TestANameIsTheRedactedText(t *testing.T) {
	s, _ := newTestStore(t)
	repo := gittest.NewRepo(t).Root()
	// The redactor keeps a token's family and last two characters.
	a := entry(KindUntested, "s1", "the token ghp_"+strings.Repeat("A1", 17)+"Z9 leaked")
	b := entry(KindUntested, "s1", "the token ghp_"+strings.Repeat("B2", 17)+"Z9 leaked")
	a.Checkout, b.Checkout = repo, repo
	if x, y := mustPut(t, s, a), mustPut(t, s, b); x.Name != y.Name {
		t.Fatalf("two sayings differing only in a redacted token got two names (%q, %q)", x.Text, y.Text)
	}
}
