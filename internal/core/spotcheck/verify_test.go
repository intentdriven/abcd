package spotcheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

// The rounds these tests draw at: quicknet publishes round r at genesis plus
// (r-1) periods, so round 1000 is the first round at or after q1000, and a
// window of seven days that closes there is followed by one closing at
// round 202600's time.
var (
	q1000   = time.Unix(1692806364, 0).UTC()
	q202600 = q1000.Add(7 * 24 * time.Hour)
	w0      = q1000.Add(-7 * 24 * time.Hour)
)

// Round 202600 of the quicknet chain, as its hosts publish it.
const (
	vector202600Signature  = "89abe9bbda7ccd0b01da48622e63dcc01738928427e959754bb5903f65dc27bacb210429158723dbcc1ea87381805ce6"
	vector202600Randomness = "2272535ec3cbb11d4d66f3d8d00710d971fedda3f095018e21a1c8f627fc0bb7"
)

// serveRounds answers a round asked by number with its published reply, and
// the latest round with the one *latest names.
func (c *drawCase) serveRounds(latest *string) {
	c.t.Helper()
	old := beaconGet
	beaconGet = func(ctx context.Context, url string) ([]byte, error) {
		c.beacons++
		round := url[strings.LastIndex(url, "/")+1:]
		if round == "latest" {
			round = *latest
		}
		switch round {
		case "1000":
			return beaconReply("1000", vectorSignature, vectorRandomness), nil
		case "202600":
			return beaconReply("202600", vector202600Signature, vector202600Randomness), nil
		}
		return nil, errors.New("answered 404 Not Found")
	}
	c.t.Cleanup(func() { beaconGet = old })
}

// landAt puts n first-parent commits on origin's main, committed a second
// apart from at, and fetches it.
func (c *drawCase) landAt(n int, at time.Time) string {
	c.t.Helper()
	for i := range n {
		commitAt(c.t, c.r, at.Add(time.Duration(i)*time.Second))
	}
	c.r.Git("push", "-q", "origin", "HEAD:refs/heads/main")
	c.r.Git("fetch", "-q", "origin")
	return c.r.Git("rev-parse", "origin/main")
}

// plant commits files over origin's store branch, as a holder of push rights
// could with a plain push, and fetches nothing.
func (c *drawCase) plant(files map[string]string) string {
	c.t.Helper()
	commit := commitFilesAt(c.t, c.r, c.originTip(), files)
	c.r.Git("push", "-q", "origin", commit+":"+BranchRef)
	return commit
}

// cycleJSON is a cycle record with the window bounds it names.
func cycleJSON(t *testing.T, c Cycle, window map[string]int) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(encodeOrFail(t, c)), &m); err != nil {
		t.Fatal(err)
	}
	m["window"] = window
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(b) + "\n"
}

// newestCycle is origin's newest cycle record.
func (c *drawCase) newestCycle() Cycle {
	c.t.Helper()
	var names []string
	for _, f := range remoteFiles(c.t, c.bare) {
		if strings.HasPrefix(f, CyclesDir+"/") {
			names = append(names, f)
		}
	}
	slices.Sort(names)
	if len(names) == 0 {
		c.t.Fatal("origin holds no cycle")
	}
	var cy Cycle
	var m map[string]json.RawMessage
	body := c.originFile(names[len(names)-1])
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		c.t.Fatal(err)
	}
	delete(m, "window")
	b, _ := json.Marshal(m)
	if err := json.Unmarshal(b, &cy); err != nil {
		c.t.Fatal(err)
	}
	return cy
}

// openWindow opens the first window at at, over origin/main as it stands.
func (c *drawCase) openWindow(at time.Time) string {
	c.t.Helper()
	from := c.r.Git("rev-parse", "origin/main")
	if note := c.observe(at); !strings.Contains(note, "first window opened") {
		c.t.Fatalf("the first check said %q", note)
	}
	return from
}

func hexKey(i int) string { return fmt.Sprintf("%040x", i) }

// sevenDays is the window a cycle names under the bundled configuration,
// spanning n commits.
func sevenDays(n int) map[string]int { return map[string]int{"days": 7, "changes": 0, "commits": n} }

// TestAForgedCycleIsSetAsideAndRedrawn: a cycle on origin for the window now
// closing whose signature does not verify, whose round is not the one the
// window's close fixes, or whose drawn list is not the one its randomness
// gives, is not read as the window's draw: the next observation names it as
// unverified and draws the window afresh, over its file.
func TestAForgedCycleIsSetAsideAndRedrawn(t *testing.T) {
	sig202600, _ := hex.DecodeString(vector202600Signature)
	badRand := sha256.Sum256(sig202600)
	for name, edit := range map[string]func(*Cycle){
		"bad signature": func(cy *Cycle) {
			cy.Beacon.Signature, cy.Beacon.Randomness = vector202600Signature, hex.EncodeToString(badRand[:])
		},
		"wrong round": func(cy *Cycle) {
			cy.Beacon.Round, cy.Beacon.Signature, cy.Beacon.Randomness = 202600, vector202600Signature, vector202600Randomness
		},
		"wrong list": func(cy *Cycle) { cy.Groups[0].Drawn = cy.Groups[0].Drawn[:2] },
	} {
		t.Run(name, func(t *testing.T) {
			latest := "1000"
			c := newDrawCase(t)
			c.serveRounds(&latest)
			from := c.openWindow(w0)
			to := c.landAt(3, w0.Add(time.Hour))
			keys := strings.Fields(c.r.Git("rev-list", from+".."+to))
			slices.Sort(keys)
			for _, k := range keys {
				c.rows = append(c.rows, row(k, ClassDrain))
			}
			c.closeWindow()
			id := cycleID(q1000, to)
			randomness, _ := hex.DecodeString(vectorRandomness)
			drawn := drawKeys(randomness, drawSeed(Cycle{FromCommit: from, ToTime: q1000}), GroupDrainLanded, keys, 10)
			forged := Cycle{SchemaVersion: SchemaVersion, ID: id, FromCommit: from, ToCommit: to, FromTime: w0, ToTime: q1000,
				ClosedBy: ClosedByDays, Site: SiteLane, Draw: DrawRule,
				Beacon: Beacon{URL: "https://api.drand.sh/" + ChainHash + "/public/1000", ChainHash: ChainHash, Round: 1000,
					Signature: vectorSignature, Randomness: vectorRandomness, FetchedAt: q1000},
				Sample: Sample{PerGroup: 10, Size: 10, Why: SizeFull, Counted: []string{}},
				Groups: []GroupDraw{{Group: GroupDrainLanded, Population: Population{Count: 3, Keys: keys}, Drawn: drawn},
					{Group: GroupAutomaticMerges, Population: Population{}, Drawn: []string{}},
					{Group: GroupDrainHandBacks, Population: Population{}, Drawn: []string{}},
					{Group: GroupFacilitatorDecisions, Population: Population{}, Drawn: []string{}}}}
			edit(&forged)
			c.plant(map[string]string{CyclesDir + "/" + id + ".json": cycleJSON(t, forged, sevenDays(3))})
			note := c.observe(q1000.Add(time.Hour))
			if !strings.Contains(note, "drew cycle "+id) || !strings.Contains(note, id+" on origin does not verify") {
				t.Fatalf("the observation over a forged cycle said %q", note)
			}
			cy := c.newestCycle()
			if cy.ID != id || cy.Beacon.Round != 1000 || cy.Beacon.Signature != vectorSignature || len(cy.Groups[0].Drawn) != 3 {
				t.Fatalf("origin's cycle after the redraw is %+v", cy)
			}
		})
	}
}

// TestAFutureDatedCycleDoesNotStopTheNextDraw: a cycle whose window closes
// past now and the clock's allowance is no window boundary: the window it
// claims to end still closes on time and is drawn.
func TestAFutureDatedCycleDoesNotStopTheNextDraw(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	from := c.openWindow(w0)
	far := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	id := "99991231T235959Z-000000000000"
	forged := Cycle{SchemaVersion: SchemaVersion, ID: id, FromCommit: from, ToCommit: strings.Repeat("0", 40), FromTime: w0, ToTime: far,
		ClosedBy: ClosedByDays, Site: SiteLane, Draw: DrawRule,
		Beacon: Beacon{ChainHash: ChainHash, Round: 1000, Signature: vectorSignature, Randomness: vectorRandomness},
		Sample: Sample{PerGroup: 10, Size: 10, Why: SizeFull, Counted: []string{}}}
	c.plant(map[string]string{CyclesDir + "/" + id + ".json": cycleJSON(t, forged, sevenDays(0))})
	c.closeWindow()
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "drew cycle") || !strings.Contains(note, id+" on origin does not verify") {
		t.Fatalf("the observation over a future-dated cycle said %q", note)
	}
}

// TestAStaleRoundIsRefused: a beacon that answers another round than the one
// the window's close fixes is refused, and nothing is drawn.
func TestAStaleRoundIsRefused(t *testing.T) {
	c := newDrawCase(t)
	old := beaconGet
	beaconGet = func(ctx context.Context, url string) ([]byte, error) {
		c.beacons++
		return beaconReply("1000", vectorSignature, vectorRandomness), nil
	}
	t.Cleanup(func() { beaconGet = old })
	c.openWindow(q1000)
	c.landAt(1, q202600.Add(time.Second))
	tip := c.originTip()
	note := c.observe(q202600.Add(time.Hour))
	if !strings.Contains(note, "nothing was drawn") || !strings.Contains(note, "202600") || !strings.Contains(note, "round 1000") {
		t.Fatalf("a stale round's note is %q", note)
	}
	if c.originTip() != tip {
		t.Fatal("a stale round moved origin's branch")
	}
}

// TestADrawBeforeItsRoundSaysNotYetDrawable: a window that has closed before
// the round its close fixes is published asks no beacon and writes nothing,
// and says it is not yet drawable; once the round is out it draws.
func TestADrawBeforeItsRoundSaysNotYetDrawable(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	c.openWindow(w0.Add(-2 * time.Second))
	c.landAt(1, q1000.Add(-time.Second))
	tip := c.originTip()
	note := c.observe(q1000.Add(-time.Second))
	if !strings.Contains(note, "not yet drawable") || c.beacons != 0 || c.originTip() != tip {
		t.Fatalf("a draw before its round: note %q, beacon asked %d times, origin moved %v", note, c.beacons, c.originTip() != tip)
	}
	if note := c.observe(q1000.Add(time.Second)); !strings.Contains(note, "drew cycle") || !strings.Contains(note, "round 1000") {
		t.Fatalf("once the round is out the observation said %q", note)
	}
}

// TestARewrittenBranchRefuses: once this machine has verified a draw, a store
// branch on origin that no longer holds it, deleted or rewritten, refuses
// with one line, carries nothing, and draws or opens no window over it.
func TestARewrittenBranchRefuses(t *testing.T) {
	for _, how := range []string{"deleted", "rewritten"} {
		t.Run(how, func(t *testing.T) {
			latest := "1000"
			c := newDrawCase(t)
			c.serveRounds(&latest)
			c.openWindow(w0)
			c.closeWindow()
			if note := c.observe(q1000.Add(time.Hour)); !strings.Contains(note, "drew cycle") {
				t.Fatalf("the draw said %q", note)
			}
			switch how {
			case "deleted":
				gitIn(t, c.bare, "update-ref", "-d", BranchRef)
			case "rewritten":
				fresh := commitFilesAt(t, c.r, "", map[string]string{WindowOpenFile: encodeOrFail(t, WindowOpen{SchemaVersion: SchemaVersion,
					Commit: c.r.Git("rev-parse", "origin/main"), At: q1000})})
				c.r.Git("push", "-q", "--no-verify", "origin", "+"+fresh+":"+BranchRef)
			}
			tip := c.originTip()
			write(t, c.r.Root(), hb(handBack("iss-8", q1000.Add(time.Hour))))
			note := c.observe(q1000.Add(2 * time.Hour))
			if !strings.Contains(note, "rewritten") || strings.Contains(note, "first window") || c.originTip() != tip {
				t.Fatalf("a carry over a %s branch said %q (origin moved %v)", how, note, c.originTip() != tip)
			}
			c.landAt(1, q202600.Add(time.Second))
			latest = "202600"
			note = c.observe(q202600.Add(time.Hour))
			if !strings.Contains(note, "rewritten") || strings.Contains(note, "first window") || strings.Contains(note, "drew") || c.originTip() != tip {
				t.Fatalf("a draw over a %s branch said %q (origin moved %v)", how, note, c.originTip() != tip)
			}
		})
	}
}

// TestALateRecordIsDrawnInTheNextWindow: a hand-back made inside a window
// that reaches origin only after that window was drawn is in the next
// window's population.
func TestALateRecordIsDrawnInTheNextWindow(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	c.openWindow(w0)
	c.closeWindow()
	if note := c.observe(q1000.Add(time.Hour)); !strings.Contains(note, "drew cycle") {
		t.Fatalf("the first draw said %q", note)
	}
	write(t, c.r.Root(), hb(handBack("iss-77", w0.Add(time.Hour))))
	if note := c.observe(q1000.Add(2 * time.Hour)); !strings.Contains(note, "carried 1 record") {
		t.Fatalf("the late record was not carried: %q", note)
	}
	latest = "202600"
	c.landAt(1, q202600.Add(time.Second))
	if note := c.observe(q202600.Add(time.Hour)); !strings.Contains(note, "drew cycle") {
		t.Fatalf("the second draw said %q", note)
	}
	cy := c.newestCycle()
	for _, g := range cy.Groups {
		if g.Group == GroupDrainHandBacks && slices.ContainsFunc(g.Population.Keys, func(k string) bool { return strings.Contains(k, "-iss-77") }) {
			return
		}
	}
	t.Fatalf("the late hand-back is in no population of the next cycle: %+v", cy.Groups)
}

// emptyGroups are the four groups of a cycle that drew from no one.
func emptyGroups() []GroupDraw {
	var gs []GroupDraw
	for _, g := range groupOrder {
		gs = append(gs, GroupDraw{Group: g, Population: Population{Keys: []string{}}, Drawn: []string{}})
	}
	return gs
}

// handMade is a cycle of round 1000 for the window that opened at from, at
// w0, closing at q1000 by and ending at to, drawing from no one.
func handMade(from, to, by string) Cycle {
	return Cycle{SchemaVersion: SchemaVersion, ID: cycleID(q1000, to), FromCommit: from, ToCommit: to, FromTime: w0, ToTime: q1000,
		ClosedBy: by, Site: SiteLane, Draw: DrawRule,
		Beacon: Beacon{URL: "https://api.drand.sh/" + ChainHash + "/public/1000", ChainHash: ChainHash, Round: 1000,
			Signature: vectorSignature, Randomness: vectorRandomness, FetchedAt: q1000},
		Sample: Sample{PerGroup: 10, Size: 10, Why: SizeFull, Counted: []string{}}, Groups: emptyGroups()}
}

// TestACycleEndingOffTheDefaultBranchIsSetAside: a cycle that chains from
// the window and carries a real round, but ends at a commit the default
// branch does not hold, is no window boundary: observations go on, and the
// window is drawn.
func TestACycleEndingOffTheDefaultBranchIsSetAside(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	from := c.openWindow(w0)
	forged := handMade(from, hexKey(77), ClosedByChanges)
	c.plant(map[string]string{CyclesDir + "/" + forged.ID + ".json": cycleJSON(t, forged, map[string]int{"days": 7, "changes": 10, "commits": 0})})
	c.closeWindow()
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "drew cycle "+cycleID(q1000, from)) || !strings.Contains(note, forged.ID+" on origin does not verify") {
		t.Fatalf("the observation over a cycle ending off the default branch said %q", note)
	}
	if note := c.observe(q1000.Add(2 * time.Hour)); note != "" {
		t.Fatalf("the next observation said %q", note)
	}
}

// TestACycleWhoseCloseIsNotTheBranchsIsRedrawn: a cycle whose window values
// disagree with the close the default branch gives (here it ends two commits
// early) is set aside, and the window is drawn at its true close.
func TestACycleWhoseCloseIsNotTheBranchsIsRedrawn(t *testing.T) {
	latest := "1000"
	c := newDrawCase(t)
	c.serveRounds(&latest)
	from := c.openWindow(w0)
	line := c.landAt(3, w0.Add(time.Hour))
	first := strings.Fields(c.r.Git("rev-list", "--reverse", from+".."+line))[0]
	c.closeWindow()
	forged := handMade(from, first, ClosedByDays)
	c.plant(map[string]string{CyclesDir + "/" + forged.ID + ".json": cycleJSON(t, forged, sevenDays(1))})
	note := c.observe(q1000.Add(time.Hour))
	if !strings.Contains(note, "drew cycle "+cycleID(q1000, line)) || !strings.Contains(note, forged.ID+" on origin does not verify") {
		t.Fatalf("the observation over a cycle with another close said %q", note)
	}
}

// TestTheDrawIsKeyedByTheWindowNotTheID: two cycles of one window whose ids
// differ only in their digits draw the same list, so the id's digits are no
// free choice.
func TestTheDrawIsKeyedByTheWindowNotTheID(t *testing.T) {
	randomness, _ := hex.DecodeString(vectorRandomness)
	keys := keysN(50)
	a := Cycle{ID: cycleID(q1000, sha('b')), FromCommit: sha('a'), ToTime: q1000}
	b := a
	b.ID = cycleID(q1000, sha('c'))
	for _, g := range groupOrder {
		if x, y := drawKeys(randomness, drawSeed(a), g, keys, 10), drawKeys(randomness, drawSeed(b), g, keys, 10); !slices.Equal(x, y) {
			t.Fatalf("ids %s and %s draw %v and %v from one window", a.ID, b.ID, x, y)
		}
	}
}
