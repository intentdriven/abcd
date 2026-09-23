package implement

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/reading"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// testSHA is a well-formed root-commit SHA for a run no repository owns.
const testSHA = "0123456789abcdef0123456789abcdef01234567"

// clock is a settable test clock.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// newRun opens a run under a temporary HOME — never the real ~/.abcd — with a
// settable clock.
func newRun(t *testing.T) (*Run, *clock) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	r, err := Open(testSHA)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	c := &clock{t: time.Date(2026, 9, 23, 6, 0, 0, 0, time.UTC)}
	r.Now = c.now
	return r, c
}

// join joins a session or fails the test.
func join(t *testing.T, r *Run, id string, role Role) {
	t.Helper()
	if _, err := r.Join(id, role, "", "", 0); err != nil {
		t.Fatalf("join %s: %v", id, err)
	}
}

// events reads the log and returns the event names in order.
func eventNames(t *testing.T, r *Run) []string {
	t.Helper()
	evs, bad, err := r.ReadLog()
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) > 0 {
		t.Fatalf("the verbs wrote lines their own reader cannot parse: %+v", bad)
	}
	var out []string
	for _, e := range evs {
		out = append(out, e.Event)
	}
	return out
}

// lastEvent returns the last event with the given name.
func lastEvent(t *testing.T, r *Run, name string) Event {
	t.Helper()
	evs, _, err := r.ReadLog()
	if err != nil {
		t.Fatal(err)
	}
	for i := len(evs) - 1; i >= 0; i-- {
		if evs[i].Event == name {
			return evs[i]
		}
	}
	t.Fatalf("no %s line in the log (have %v)", name, eventNames(t, r))
	return Event{}
}

// claimExists reports whether a claim file exists for record.
func claimExists(t *testing.T, r *Run, record string) bool {
	t.Helper()
	ok, err := fsutil.Exists(filepath.Join(r.Dir, claimsDirName, record+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

// TestTheClaimFileExcludesTwoWritersByItself holds the exclusion at its root:
// with no lock in the way, many goroutines creating the same claim file through
// the canonical exclusive create leave exactly one winner.
func TestTheClaimFileExcludesTwoWritersByItself(t *testing.T) {
	r, _ := newRun(t)
	const n = 16
	var wg sync.WaitGroup
	wins := make(chan int, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			root, err := os.OpenRoot(r.Dir)
			if err != nil {
				t.Error(err)
				return
			}
			defer root.Close()
			<-start
			err = fsutil.CreateExclusiveIn(root, claimRel("itd-1"), []byte(fmt.Sprintf(`{"n":%d}`, i)), fileMode)
			switch {
			case err == nil:
				wins <- i
			case !errors.Is(err, os.ErrExist):
				t.Errorf("writer %d: %v, want ErrExist", i, err)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(wins)
	if got := len(wins); got != 1 {
		t.Fatalf("%d writers took the claim; exactly one must", got)
	}
}

// TestConcurrentClaimsOfOneRecordGrantExactlyOne drives the whole claim path
// from many sessions at once: one is granted, every other is told who holds it,
// and the log carries one claim and a denial per loser.
func TestConcurrentClaimsOfOneRecordGrantExactlyOne(t *testing.T) {
	r, _ := newRun(t)
	const n = 8
	for i := 0; i < n; i++ {
		join(t, r, fmt.Sprintf("s%d", i), RoleFirst)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	granted, denied := 0, 0
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := r.Claim(ClaimRequest{Session: fmt.Sprintf("s%d", i), Record: "itd-7", Lane: "lane"})
			mu.Lock()
			defer mu.Unlock()
			var held *HeldError
			switch {
			case err == nil:
				granted++
			case errors.As(err, &held) && errors.Is(err, ErrContention):
				denied++
			default:
				t.Errorf("session s%d: %v", i, err)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if granted != 1 || denied != n-1 {
		t.Fatalf("granted %d, denied %d; want 1 and %d", granted, denied, n-1)
	}
	count := map[string]int{}
	for _, e := range eventNames(t, r) {
		count[e]++
	}
	if count[EventClaim] != 1 || count[EventClaimDenied] != n-1 {
		t.Fatalf("log carries %d claim and %d claim_denied lines; want 1 and %d", count[EventClaim], count[EventClaimDenied], n-1)
	}
}

// TestAHeldClaimIsDeniedNamingTheHolder is the refusal: another session's live
// claim is not taken, the denial names the holder in the error and in the log,
// and the holder's file is untouched. The ok case beside it: the holder claiming
// again renews its own lease.
func TestAHeldClaimIsDeniedNamingTheHolder(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleSecond)
	if _, err := r.Claim(ClaimRequest{Session: "alpha", Record: "itd-1", Lane: "one"}); err != nil {
		t.Fatal(err)
	}
	_, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-1", Lane: "two"})
	var held *HeldError
	if !errors.As(err, &held) || held.Holder.Session != "alpha" || !strings.Contains(err.Error(), "alpha") {
		t.Fatalf("second claim = %v; want a HeldError naming alpha", err)
	}
	d := lastEvent(t, r, EventClaimDenied)
	if d.Session != "beta" || d.String("holder") != "alpha" || d.String("record") != "itd-1" {
		t.Fatalf("claim_denied line = %+v", d.Fields)
	}
	// The second session backs off, and says why and for how long.
	b := lastEvent(t, r, EventBackoff)
	if b.Session != "beta" || !strings.Contains(b.String("reason"), "alpha") {
		t.Fatalf("backoff line = %+v", b.Fields)
	}
	if _, ok := b.Number("minutes"); !ok {
		t.Fatalf("backoff line carries no minutes: %+v", b.Fields)
	}

	c.advance(30 * time.Minute)
	res, err := r.Claim(ClaimRequest{Session: "alpha", Record: "itd-1", Lane: "one"})
	if err != nil || !res.Renewed {
		t.Fatalf("holder's re-claim = %+v, %v; want a renewal", res, err)
	}
	if !res.Claim.ExpiresAt.Equal(c.now().Add(DefaultLease)) {
		t.Fatalf("renewed lease ends %s, want %s", res.Claim.ExpiresAt, c.now().Add(DefaultLease))
	}
}

// TestALapsedLeaseIsClaimableAndTheLapseIsLogged: a session that stops holding a
// claim strands nothing once its lease has passed.
func TestALapsedLeaseIsClaimableAndTheLapseIsLogged(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleFirst)
	if _, err := r.Claim(ClaimRequest{Session: "alpha", Record: "iss-5", Lane: "fix", Lease: 10 * time.Minute}); err != nil {
		t.Fatal(err)
	}
	// One second before the lapse the claim still holds.
	c.advance(10*time.Minute - time.Second)
	if _, err := r.Claim(ClaimRequest{Session: "beta", Record: "iss-5", Lane: "fix"}); !errors.Is(err, ErrContention) {
		t.Fatalf("claim before the lapse = %v, want contention", err)
	}
	c.advance(time.Second)
	res, err := r.Claim(ClaimRequest{Session: "beta", Record: "iss-5", Lane: "fix"})
	if err != nil {
		t.Fatalf("claim after the lapse: %v", err)
	}
	if res.Lapsed == nil || res.Lapsed.Session != "alpha" || res.Claim.Session != "beta" {
		t.Fatalf("result = %+v; want beta's claim replacing alpha's lapsed one", res)
	}
	l := lastEvent(t, r, EventClaimLapsed)
	if l.Session != "beta" || l.String("holder") != "alpha" || l.String("record") != "iss-5" {
		t.Fatalf("claim_lapsed line = %+v", l.Fields)
	}
	states, err := r.Claims()
	if err != nil || len(states) != 1 || states[0].Session != "beta" || !states[0].Live {
		t.Fatalf("claims = %+v, %v", states, err)
	}
}

// TestTheSecondSessionHoldsAtMostOneLane is the cap, with the first session's
// two claims beside it as the ok case.
func TestTheSecondSessionHoldsAtMostOneLane(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleSecond)
	for _, rec := range []string{"itd-1", "itd-2"} {
		if _, err := r.Claim(ClaimRequest{Session: "alpha", Record: rec, Lane: rec}); err != nil {
			t.Fatalf("first session's claim on %s: %v", rec, err)
		}
	}
	if _, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-3", Lane: "three", Lease: 5 * time.Minute}); err != nil {
		t.Fatal(err)
	}
	_, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-4", Lane: "four"})
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "itd-3") {
		t.Fatalf("second claim by the second session = %v; want a refusal naming itd-3", err)
	}
	if claimExists(t, r, "itd-4") {
		t.Fatal("a refused claim left a claim file")
	}
	if e := lastEvent(t, r, EventRefusal); e.String("condition") != "second_session_lane_cap" {
		t.Fatalf("refusal line = %+v", e.Fields)
	}
	// Once its first claim lapses, the cap no longer binds.
	c.advance(5 * time.Minute)
	if _, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-4", Lane: "four"}); err != nil {
		t.Fatalf("claim after the first lapsed: %v", err)
	}
}

// withPresets gives the run a checkout carrying this repository's own preset
// file, committed, so the reading corpus is derived exactly as the live run
// derives it.
func withPresets(t *testing.T, r *Run) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(reading.PresetConfigPath)))
	if err != nil {
		t.Fatal(err)
	}
	repo := gittest.NewRepo(t)
	repo.Write(reading.PresetConfigPath, string(data))
	repo.Commit("presets")
	r.RepoRoot = repo.Root()
}

// TestTheSecondSessionIsRefusedAReadingCorpusLane refuses a claim whose declared
// paths reach the reading corpus, and grants one whose paths do not.
func TestTheSecondSessionIsRefusedAReadingCorpusLane(t *testing.T) {
	r, _ := newRun(t)
	withPresets(t, r)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleSecond)
	_, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-1", Lane: "one",
		Paths: []string{"docs/x.md", "internal/core/intent/store.go"}})
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "internal/core/intent/store.go") {
		t.Fatalf("corpus claim = %v; want a refusal naming the path", err)
	}
	if claimExists(t, r, "itd-1") {
		t.Fatal("a refused claim left a claim file")
	}
	if _, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-1", Lane: "one",
		Paths: []string{"internal/core/implement/claim.go"}}); err != nil {
		t.Fatalf("non-corpus claim: %v", err)
	}
	// The first session takes a corpus lane freely.
	if _, err := r.Claim(ClaimRequest{Session: "alpha", Record: "itd-2", Lane: "two",
		Paths: []string{".abcd/config/reading-presets.json"}}); err != nil {
		t.Fatalf("first session's corpus claim: %v", err)
	}
}

// TestTheReadingCorpusIsThePresetsObjectPaths: the corpus is the union of every
// position's object.paths in the committed preset file, plus that file — so a
// file the presets name (internal/surface/cli/reading.go) is refused to the
// second session, a directory they name covers what is under it and nothing
// beside it, and a path outside the union is open.
func TestTheReadingCorpusIsThePresetsObjectPaths(t *testing.T) {
	r, _ := newRun(t)
	withPresets(t, r)
	join(t, r, "beta", RoleSecond)
	for _, p := range []string{
		"internal/surface/cli/reading.go",
		".abcd/config/reading-presets.json",
		"internal/core/lint/rules.go",
		"commands/reading.md",
	} {
		if _, err := r.Check("beta", StepLane, []string{p}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), p) {
			t.Errorf("second session, lane touching %s = %v; want a refusal naming it", p, err)
		}
	}
	for _, p := range []string{
		"internal/surface/cli/implement.go",
		"internal/core/lintx/a.go",
		"internal/core/implement/bounds.go",
		"commands/implement.md",
	} {
		if v, err := r.Check("beta", StepLane, []string{p}); err != nil || !v.Allowed {
			t.Errorf("second session, lane touching %s = %+v, %v; want it allowed", p, v, err)
		}
	}
	corpus, err := ReadingCorpus(r.RepoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(corpus, "internal/surface/cli/reading.go") || !slices.Contains(corpus, reading.PresetConfigPath) {
		t.Fatalf("corpus = %v", corpus)
	}
}

// TestAnUnreadablePresetFileFailsClosedForTheSecondSession: when the corpus
// cannot be derived — no preset file, one that does not parse, or no checkout
// to read it from — a second session's lane with declared paths is refused and
// the refusal logged, since nothing can say the lane stays clear of the corpus.
// A lane with no declared paths asks no corpus question, and the first session
// is never bounded.
func TestAnUnreadablePresetFileFailsClosedForTheSecondSession(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T, r *Run){
		"no checkout": func(t *testing.T, r *Run) { r.RepoRoot = "" },
		"absent": func(t *testing.T, r *Run) {
			repo := gittest.NewRepo(t)
			repo.Write("README.md", "x\n")
			repo.Commit("init")
			r.RepoRoot = repo.Root()
		},
		"unparseable": func(t *testing.T, r *Run) {
			repo := gittest.NewRepo(t)
			repo.Write(reading.PresetConfigPath, "{not json")
			repo.Commit("init")
			r.RepoRoot = repo.Root()
		},
	} {
		t.Run(name, func(t *testing.T) {
			r, _ := newRun(t)
			setup(t, r)
			join(t, r, "alpha", RoleFirst)
			join(t, r, "beta", RoleSecond)
			if _, err := r.Check("beta", StepLane, []string{"internal/core/implement/bounds.go"}); !errors.Is(err, ErrRefused) {
				t.Fatalf("check with no corpus = %v; want a refusal", err)
			}
			if e := lastEvent(t, r, EventRefusal); e.String("condition") != "reading_corpus_unknown" {
				t.Fatalf("refusal line = %+v", e.Fields)
			}
			if _, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-1", Lane: "one",
				Paths: []string{"internal/core/implement/bounds.go"}}); !errors.Is(err, ErrRefused) {
				t.Fatalf("claim with no corpus = %v; want a refusal", err)
			}
			if _, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-1", Lane: "one"}); err != nil {
				t.Fatalf("claim with no declared paths: %v", err)
			}
			if v, err := r.Check("alpha", StepLane, []string{"internal/core/implement/bounds.go"}); err != nil || !v.Allowed {
				t.Fatalf("first session with no corpus = %+v, %v", v, err)
			}
		})
	}
}

// TestTheSecondSessionOpensNoLaneInASplitRolesWindow: in split-roles the second
// reviews, audits and lands.
func TestTheSecondSessionOpensNoLaneInASplitRolesWindow(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleSecond)
	if _, err := r.SetMode("alpha", ModeClaim, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-1", Lane: "one"}); err != nil {
		t.Fatalf("claim in a claim window: %v", err)
	}
	if _, err := r.Release("beta", "itd-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.SetMode("alpha", ModeSplitRoles, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-1", Lane: "one"}); !errors.Is(err, ErrRefused) {
		t.Fatalf("claim in a split-roles window = %v; want a refusal", err)
	}
	if _, err := r.Claim(ClaimRequest{Session: "alpha", Record: "itd-1", Lane: "one"}); err != nil {
		t.Fatalf("first session's claim in a split-roles window: %v", err)
	}
}

// TestOnlyTheHolderReleases, and leaving releases everything the session holds.
func TestOnlyTheHolderReleases(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleFirst)
	for _, rec := range []string{"itd-1", "itd-2"} {
		if _, err := r.Claim(ClaimRequest{Session: "alpha", Record: rec, Lane: "l"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.Release("beta", "itd-1"); !errors.Is(err, ErrRefused) {
		t.Fatalf("release by a non-holder = %v; want a refusal", err)
	}
	if !claimExists(t, r, "itd-1") {
		t.Fatal("a refused release removed the claim")
	}
	if _, err := r.Release("alpha", "itd-9"); !errors.Is(err, ErrRefused) {
		t.Fatalf("release of an unclaimed record = %v; want a refusal", err)
	}
	if _, err := r.Release("alpha", "itd-1"); err != nil {
		t.Fatal(err)
	}
	res, err := r.Leave("alpha", "window")
	if err != nil || len(res.Released) != 1 || res.Released[0].Record != "itd-2" {
		t.Fatalf("leave = %+v, %v; want itd-2 released", res, err)
	}
	states, _ := r.Claims()
	if len(states) != 0 {
		t.Fatalf("claims after leave: %+v", states)
	}
	names := eventNames(t, r)
	if names[len(names)-1] != EventSessionClose {
		t.Fatalf("log does not end with session_close: %v", names)
	}
	if _, err := r.Claim(ClaimRequest{Session: "alpha", Record: "itd-3", Lane: "l"}); !errors.Is(err, ErrRefused) {
		t.Fatalf("claim after leaving = %v; want a refusal (not joined)", err)
	}
}

// TestMalformedClaimsWriteNothing: every input the claim does not recognise is
// refused before anything reaches the run state.
func TestMalformedClaimsWriteNothing(t *testing.T) {
	r, _ := newRun(t)
	join(t, r, "alpha", RoleFirst)
	before := eventNames(t, r)
	for name, req := range map[string]ClaimRequest{
		"not a record":    {Session: "alpha", Record: "itd-1/../../x", Lane: "l"},
		"bare word":       {Session: "alpha", Record: "routing", Lane: "l"},
		"bad lane":        {Session: "alpha", Record: "itd-1", Lane: "../x"},
		"lease too long":  {Session: "alpha", Record: "itd-1", Lane: "l", Lease: 25 * time.Hour},
		"lease too short": {Session: "alpha", Record: "itd-1", Lane: "l", Lease: time.Second},
		"absolute path":   {Session: "alpha", Record: "itd-1", Lane: "l", Paths: []string{"/etc/passwd"}},
		"unjoined":        {Session: "gamma", Record: "itd-1", Lane: "l"},
	} {
		if _, err := r.Claim(req); !errors.Is(err, ErrRefused) {
			t.Errorf("%s: %v; want a refusal", name, err)
		}
	}
	if claimExists(t, r, "itd-1") {
		t.Fatal("a refused claim left a claim file")
	}
	if after := eventNames(t, r); len(after) != len(before) {
		t.Fatalf("refused malformed claims logged %v", after[len(before):])
	}
}

// TestAnUnreadableClaimLapsesAfterAGrace: a claim file nobody can parse — a
// session killed between the exclusive create and the write leaves an empty
// one — blocks nothing else: the status, other claims and leave read past it.
// Its own record is contention for a short grace after it was written, naming
// the file's full path, and then lapses: the next claim logs claim_lapsed with
// reason unparseable and takes it.
func TestAnUnreadableClaimLapsesAfterAGrace(t *testing.T) {
	r, c := newRun(t)
	join(t, r, "alpha", RoleFirst)
	join(t, r, "beta", RoleSecond)
	path := filepath.Join(r.Dir, claimsDirName, "itd-1.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, c.now(), c.now()); err != nil {
		t.Fatal(err)
	}

	states, err := r.Claims()
	if err != nil || len(states) != 1 || !states[0].Unreadable || !states[0].Live || states[0].Record != "itd-1" {
		t.Fatalf("status over an empty claim = %+v, %v", states, err)
	}
	if _, err := r.Claim(ClaimRequest{Session: "beta", Record: "itd-2", Lane: "two"}); err != nil {
		t.Fatalf("another record's claim over an empty claim file: %v", err)
	}
	_, err = r.Claim(ClaimRequest{Session: "alpha", Record: "itd-1", Lane: "one"})
	if !errors.Is(err, ErrContention) || !strings.Contains(err.Error(), path) {
		t.Fatalf("claim within the grace = %v; want contention naming %s", err, path)
	}
	if _, err := r.Release("alpha", "itd-1"); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), path) {
		t.Fatalf("release of an unreadable claim = %v; want a refusal naming %s", err, path)
	}
	if res, err := r.Leave("beta", "window"); err != nil || len(res.Released) != 1 {
		t.Fatalf("leave over an empty claim file = %+v, %v", res, err)
	}

	c.advance(UnreadableClaimGrace)
	res, err := r.Claim(ClaimRequest{Session: "alpha", Record: "itd-1", Lane: "one"})
	if err != nil || res.Claim.Session != "alpha" {
		t.Fatalf("claim after the grace = %+v, %v", res, err)
	}
	e := lastEvent(t, r, EventClaimLapsed)
	if e.String("reason") != "unparseable" || e.String("record") != "itd-1" || e.String("path") != "claims/itd-1.json" {
		t.Fatalf("claim_lapsed = %+v", e.Fields)
	}
	if states, err := r.Claims(); err != nil || len(states) != 1 || states[0].Unreadable || states[0].Session != "alpha" {
		t.Fatalf("claims after the lapse = %+v, %v", states, err)
	}
}
