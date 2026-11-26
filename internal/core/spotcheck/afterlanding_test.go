package spotcheck

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// originWithRemote is a checkout whose origin, a bare remote, holds main,
// fetched, so origin/main is tracked.
func originWithRemote(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	r := clone(t, bare)
	r.Git("push", "-q", "origin", "HEAD:refs/heads/main")
	r.Git("fetch", "-q", "origin")
	return r, bare
}

func encodeOrFail(t *testing.T, v any) string {
	t.Helper()
	b, err := encode(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// setTracking puts files on origin's branch, as another writer would, and
// fetches it to the tracking ref.
func setTracking(t *testing.T, r *gittest.Repo, files map[string]string) string {
	t.Helper()
	c := commitFilesAt(t, r, "", files)
	r.Git("push", "-q", "origin", c+":"+BranchRef)
	r.Git("update-ref", TrackingRef, c)
	return c
}

// drawCase is a checkout set up for AfterLanding: an empty home, the beacon
// serving the round-1000 vector and a landings reader answering rows.
type drawCase struct {
	t       *testing.T
	r       *gittest.Repo
	bare    string
	beacons int
	reads   int
	rows    []LandingRow
	beacon  func() ([]byte, error)
	readErr error
	from    string
	to      string
}

func newDrawCase(t *testing.T) *drawCase {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	r, bare := originWithRemote(t)
	c := &drawCase{t: t, r: r, bare: bare}
	c.beacon = func() ([]byte, error) { return beaconReply("1000", vectorSignature, vectorRandomness), nil }
	oldGet := beaconGet
	beaconGet = func(ctx context.Context, url string) ([]byte, error) {
		c.beacons++
		return c.beacon()
	}
	oldReader := registeredReader()
	RegisterLandingsReader(func(ctx context.Context, root, from, to string) ([]LandingRow, error) {
		c.reads++
		c.from, c.to = from, to
		return c.rows, c.readErr
	})
	t.Cleanup(func() { beaconGet = oldGet; RegisterLandingsReader(oldReader) })
	return c
}

func (c *drawCase) observe(at time.Time) string {
	c.t.Helper()
	return AfterLanding(c.t.Context(), Observed{Root: c.r.Root(), Default: "main", Site: SiteLane, Now: at})
}

// originTip is origin's store branch, "" for none.
func (c *drawCase) originTip() string {
	out, err := os.ReadFile(filepath.Join(c.bare, "refs", "heads", Branch))
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(gitIn(c.t, c.bare, "for-each-ref", "--format=%(objectname)", BranchRef))
}

// originFile is one file of origin's branch.
func (c *drawCase) originFile(p string) string {
	c.t.Helper()
	return gitIn(c.t, c.bare, "show", BranchRef+":"+p)
}

// land puts n first-parent commits on origin's main, committed inside the
// first window, and fetches it.
func (c *drawCase) land(n int) string {
	c.t.Helper()
	return c.landAt(n, w0.Add(time.Hour))
}

// closeWindow lands the commit, committed just after the first window's seven
// days, that closes it.
func (c *drawCase) closeWindow() {
	c.t.Helper()
	c.landAt(1, q1000.Add(time.Second))
}

func TestFirstCheckOpensTheWindow(t *testing.T) {
	c := newDrawCase(t)
	note := c.observe(at0)
	if !strings.Contains(note, "first window opened") {
		t.Fatalf("the first check's note is %q", note)
	}
	if files := remoteFiles(t, c.bare); !slices.Equal(files, []string{WindowOpenFile}) {
		t.Fatalf("origin's branch holds %v; want the window's opening alone", files)
	}
	var w WindowOpen
	if err := json.Unmarshal([]byte(c.originFile(WindowOpenFile)), &w); err != nil || w.Commit != c.r.Git("rev-parse", "origin/main") || !w.At.Equal(at0) {
		t.Fatalf("window-open.json = %+v, %v", w, err)
	}
	if c.beacons != 0 || c.reads != 0 {
		t.Fatalf("the first check asked the beacon %d and the reader %d times; it draws nothing", c.beacons, c.reads)
	}
}

// TestWindowNotPassedDrawsNothingAndFetchesNothing: with the window open and
// not passed, and nothing to carry, an observation reaches no network (origin
// cannot be reached at all), asks neither the beacon nor the reader, and
// says nothing. A record to carry is carried at the next observation.
func TestWindowNotPassedDrawsNothingAndFetchesNothing(t *testing.T) {
	c := newDrawCase(t)
	c.observe(w0)
	tip := c.originTip()
	c.r.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone.git"))
	if note := c.observe(w0.Add(6 * 24 * time.Hour)); note != "" {
		t.Fatalf("an observation inside the window said %q", note)
	}
	if c.beacons != 0 || c.reads != 0 {
		t.Fatalf("inside the window the beacon was asked %d and the reader %d times", c.beacons, c.reads)
	}
	write(t, c.r.Root(), hb(handBack("iss-1", w0.Add(time.Hour))))
	if note := c.observe(w0.Add(6 * 24 * time.Hour)); !strings.Contains(note, "not fetched") {
		t.Fatalf("a record to carry with origin gone: note %q", note)
	}
	c.r.Git("remote", "set-url", "origin", c.bare)
	if note := c.observe(w0.Add(6 * 24 * time.Hour)); !strings.Contains(note, "carried 1 record") {
		t.Fatalf("the record was not carried: %q", note)
	}
	if c.originTip() == tip || c.beacons != 0 {
		t.Fatalf("origin is at %s (was %s), beacon asked %d times", c.originTip(), tip, c.beacons)
	}
}

// TestADrawAtAPassedWindow: once the window has passed, the observation
// fetches the beacon, reads the window's landings, and pushes the cycle, its
// items and the records in one commit; the draw recomputes from the cycle
// record alone; no free text reaches origin; and the next window opens at the
// cycle's end.
func TestADrawAtAPassedWindow(t *testing.T) {
	c := newDrawCase(t)
	c.observe(w0)
	from := c.r.Git("rev-parse", "origin/main")
	to := c.land(16)
	line := strings.Fields(c.r.Git("rev-list", "--reverse", from+".."+to))
	for i := range 14 {
		c.rows = append(c.rows, row(line[i], ClassDrain))
	}
	c.rows = append(c.rows, row(line[14], ClassAutomatic), row(line[15], "person-approved"))
	h := handBack("iss-2", w0.Add(24*time.Hour))
	h.Reason = "the secret reason the drain wrote"
	d := gateDecision(4, sha('f'), true, w0.Add(25*time.Hour))
	write(t, c.r.Root(), Batch{HandBacks: []HandBack{h}, Decisions: []Decision{d}})
	c.closeWindow()
	at := q1000.Add(time.Hour)
	note := c.observe(at)
	if !strings.Contains(note, "drew cycle") || !strings.Contains(note, "13 item(s)") {
		t.Fatalf("the draw's note is %q", note)
	}
	if c.from != from || c.to != to {
		t.Fatalf("the reader read %s..%s; want %s..%s", c.from, c.to, from, to)
	}
	id := cycleID(q1000, to)
	var cy Cycle
	if err := json.Unmarshal([]byte(c.originFile(CyclesDir+"/"+id+".json")), &cy); err != nil {
		t.Fatal(err)
	}
	if cy.FromCommit != from || cy.ToCommit != to || cy.ClosedBy != ClosedByDays || cy.Site != SiteLane || cy.Beacon.Round != 1000 ||
		cy.Sample.Size != 10 || cy.Sample.Why != SizeFull || len(cy.Groups) != 4 {
		t.Fatalf("the cycle is %+v", cy)
	}
	randomness, _ := hex.DecodeString(cy.Beacon.Randomness)
	items := 0
	for _, g := range cy.Groups {
		if again := drawKeys(randomness, drawSeed(cy), g.Group, g.Population.Keys, cy.Sample.Size); !slices.Equal(again, g.Drawn) {
			t.Errorf("group %s does not recompute from the cycle record: %v vs %v", g.Group, again, g.Drawn)
		}
		items += len(g.Drawn)
	}
	want := map[string]int{GroupDrainLanded: 14, GroupAutomaticMerges: 1, GroupDrainHandBacks: 1, GroupFacilitatorDecisions: 1}
	for _, g := range cy.Groups {
		if g.Population.Count != want[g.Group] {
			t.Errorf("group %s has a population of %d; want %d", g.Group, g.Population.Count, want[g.Group])
		}
	}
	files := remoteFiles(t, c.bare)
	nItems := 0
	for _, f := range files {
		if strings.HasPrefix(f, ItemsDir+"/"+id+"/") {
			nItems++
		}
		if body := c.originFile(f); strings.Contains(body, "secret reason") || strings.Contains(body, "review receipt") {
			t.Errorf("origin's %s carries free text:\n%s", f, body)
		}
	}
	if nItems != items || items != 13 {
		t.Fatalf("origin holds %d items for %d drawn; want 13", nItems, items)
	}
	if note := c.observe(at.Add(time.Hour)); note != "" {
		t.Fatalf("an observation an hour after the draw said %q", note)
	}
}

// TestBeaconUnreachableWritesNothing and its kin: a draw the beacon refuses
// (unreachable, a signature that does not verify, another chain) writes
// nothing to origin, names the beacon, and leaves the window passed for the
// next observation, which draws.
func TestBeaconUnreachableWritesNothing(t *testing.T) {
	for name, reply := range map[string]func() ([]byte, error){
		"unreachable":   func() ([]byte, error) { return nil, errors.New("no route") },
		"bad signature": func() ([]byte, error) { return beaconReply("1001", vectorSignature, ""), nil },
		"other chain": func() ([]byte, error) {
			return []byte(`{"round":1000,"signature":"` + vectorSignature + `","chain_hash":"` + strings.Repeat("ab", 32) + `"}`), nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := newDrawCase(t)
			c.observe(w0)
			c.rows = []LandingRow{row(c.land(1), ClassDrain)}
			write(t, c.r.Root(), hb(handBack("iss-2", w0.Add(time.Hour))))
			c.closeWindow()
			tip := c.originTip()
			c.beacon = reply
			note := c.observe(q1000.Add(time.Hour))
			if !strings.Contains(note, "beacon") || !strings.Contains(note, "nothing was drawn") {
				t.Fatalf("the refused draw's note is %q", note)
			}
			if c.originTip() != tip {
				t.Fatal("a refused draw moved origin's branch")
			}
			c.beacon = func() ([]byte, error) { return beaconReply("1000", vectorSignature, ""), nil }
			if note := c.observe(q1000.Add(time.Hour + time.Minute)); !strings.Contains(note, "drew cycle") {
				t.Fatalf("the next observation did not draw: %q", note)
			}
		})
	}
}

// TestBeaconBadSignatureWritesNothing is the bad-signature case named alone,
// as the spec lists it.
func TestBeaconBadSignatureWritesNothing(t *testing.T) {
	c := newDrawCase(t)
	c.observe(w0)
	c.closeWindow()
	tip := c.originTip()
	c.beacon = func() ([]byte, error) { return beaconReply("999", vectorSignature, ""), nil }
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "round 999") || !strings.Contains(note, "does not verify") || c.originTip() != tip {
		t.Fatalf("note %q, origin moved %v", note, c.originTip() != tip)
	}
}

// TestRejectedPushLeavesNoCycle: a push origin refuses twice leaves origin as
// it was, the draw refused, and the window passed.
func TestRejectedPushLeavesNoCycle(t *testing.T) {
	c := newDrawCase(t)
	c.observe(w0)
	c.closeWindow()
	tip := c.originTip()
	hooks := t.TempDir()
	if err := os.WriteFile(filepath.Join(hooks, "pre-push"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	c.r.Git("config", "core.hooksPath", hooks)
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "refused twice") || c.originTip() != tip {
		t.Fatalf("note %q; origin moved %v", note, c.originTip() != tip)
	}
}

// TestDrawWithNoReaderIsRefused: a binary with no landings reader linked
// draws nothing.
func TestDrawWithNoReaderIsRefused(t *testing.T) {
	c := newDrawCase(t)
	c.observe(w0)
	RegisterLandingsReader(nil)
	c.closeWindow()
	tip := c.originTip()
	if note := c.observe(q1000.Add(time.Hour)); !strings.Contains(note, "no landings reader") || c.originTip() != tip {
		t.Fatalf("note %q", note)
	}
}

// TestPushCarriesOnlyRecords: a file of the local copy that does not read as
// a record, under the store's folders or outside them, never reaches origin.
func TestPushCarriesOnlyRecords(t *testing.T) {
	r, bare := originWithRemote(t)
	write(t, r.Root(), hb(handBack("iss-1", at0)))
	local := r.Git("rev-parse", LocalRef)
	stray := commitFilesAt(t, r, local, map[string]string{
		"notes/x.txt": "a note\n",
		HandBacksDir + "/20261113T070000Z-iss-9-0000000000000000.json": "{}\n",
		ItemsDir + "/forged.json":                                      "{}\n",
	})
	r.Git("update-ref", LocalRef, stray)
	res, err := push(t.Context(), r.Root())
	if err != nil || res.Pushed != 1 {
		t.Fatalf("push = %+v, %v", res, err)
	}
	files := remoteFiles(t, bare)
	if len(files) != 1 || !strings.HasPrefix(files[0], HandBacksDir+"/") || !strings.Contains(files[0], "-iss-1-") {
		t.Fatalf("origin holds %v; want the one record", files)
	}
}

// TestFreeTextIsNotSent: a record's free text stays on the local copy; origin
// gets its typed fields alone, under a name of their own.
func TestFreeTextIsNotSent(t *testing.T) {
	r, bare := originWithRemote(t)
	h := handBack("iss-1", at0)
	h.Rule = "the lane's rule as the drain wrote it"
	d := Decision{Kind: KindDrainJudgement, At: at0, Issue: "iss-2", Question: "may the drain take it", Answer: "yes", Model: "claude-opus-4-8", Rule: "adr-2609291342092738", Reason: "it looked fine"}
	g := gateDecision(4, sha('f'), true, at0)
	write(t, r.Root(), Batch{HandBacks: []HandBack{h}, Decisions: []Decision{d, g}})
	if _, err := push(t.Context(), r.Root()); err != nil {
		t.Fatal(err)
	}
	r.Git("fetch", "-q", "origin", "+"+BranchRef+":"+TrackingRef)
	rec, ok, err := read(t.Context(), r.Root(), TrackingRef)
	if err != nil || !ok || len(rec.HandBacks) != 1 || len(rec.Decisions) != 2 || len(rec.Unreadable) != 0 {
		t.Fatalf("origin's copy reads as %+v, %v, %v", rec, ok, err)
	}
	for _, f := range remoteFiles(t, bare) {
		body := gitIn(t, bare, "show", BranchRef+":"+f)
		for _, text := range []string{h.Reason, h.Rule, d.Question, d.Reason, "review receipt"} {
			if strings.Contains(body, text) {
				t.Errorf("origin's %s carries %q", f, text)
			}
		}
	}
	if rec.HandBacks[0].DrainRule != h.DrainRule || rec.HandBacks[0].Kind != h.Kind {
		t.Errorf("origin's hand-back lost a typed field: %+v", rec.HandBacks[0])
	}
	local := readLocal(t, r.Root())
	if local.HandBacks[0].Reason != h.Reason {
		t.Errorf("the local copy lost the reason: %+v", local.HandBacks[0])
	}
	// A second push carries nothing more.
	if res, err := push(t.Context(), r.Root()); err != nil || res.Pushed != 0 {
		t.Fatalf("a second push = %+v, %v", res, err)
	}
}
