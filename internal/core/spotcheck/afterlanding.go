package spotcheck

// afterlanding.go is the collection lane's one entry (spc-2610301925429250
// approaches 3 and 11): AfterLanding, called where abcd has just proved a
// landing. While the window has not passed it reads two files and runs one
// local git command, and reaches the network only to carry records this
// checkout holds that origin's copy, as last fetched, lacks. Once the window
// has passed it fetches the store's branch, the beacon's latest round, reads
// the window's landings, draws, and pushes the cycle with its items in one
// commit over origin's copy. Whatever it does, its answer is a note and
// never an error, an outcome or a hold: a draw that refuses for any reason
// leaves the window passed and writes nothing, and the next landing observed
// tries again. Nothing it does decides anything a landing or a drain does.

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// DrawTimeout bounds one observation that draws: the fetch, the beacon, the
// landings read and the push together.
const DrawTimeout = 3 * time.Minute

// Observed is one landing abcd proved: the checkout, origin's default
// branch as just fetched, and the site that proved it.
type Observed struct {
	Root    string
	Default string
	// Site is SiteLane, SiteLand or SiteDrain.
	Site string
	// Now is the observing instant; zero is time.Now.
	Now time.Time
}

// now is the clock AfterLanding reads when an observation names none.
var now = time.Now

// AfterLanding observes a landing for the spot-check and answers a note for
// the caller's own text, "" when there is nothing to say. A checkout with no
// remote named origin keeps no spot-check, and the note is "".
func AfterLanding(parent context.Context, o Observed) string {
	if o.Now.IsZero() {
		o.Now = now()
	}
	o.Now = o.Now.UTC().Truncate(time.Second)
	if o.Site != SiteLand && o.Site != SiteLane && o.Site != SiteDrain {
		return "spot-check: " + fmt.Sprintf("%q is not an observing site, so nothing was drawn", o.Site)
	}
	ctx, cancel := context.WithTimeout(parent, DrawTimeout)
	defer cancel()
	if err := originPresent(ctx, o.Root); err != nil {
		if errors.Is(err, ErrNoOrigin) {
			return ""
		}
		return "spot-check: " + err.Error() + "; nothing was drawn"
	}
	roots, _ := layered.RootsFor(o.Root)
	cfg, err := readConfig(roots)
	if err != nil {
		return "spot-check: the spotcheck configuration cannot be read (" + redact(err) + "); nothing was drawn"
	}
	tip, err := defaultTip(ctx, o.Root, o.Default)
	if err != nil {
		return "spot-check: " + err.Error() + "; nothing was drawn"
	}
	st, err := readWindow(ctx, o.Root)
	if err != nil {
		return "spot-check: " + err.Error() + "; nothing was drawn"
	}
	if st.open {
		closedBy, err := closedNow(ctx, o.Root, cfg, st, tip, o.Now)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		if closedBy == "" {
			pending, _, err := outbound(ctx, o.Root, st.files)
			if err != nil {
				return "spot-check: " + err.Error()
			}
			if len(pending) == 0 {
				return ""
			}
			return carry(ctx, o.Root)
		}
	}
	return draw(ctx, o, cfg, tip)
}

// defaultTip is origin's default branch as this checkout last fetched it.
func defaultTip(ctx context.Context, root, def string) (string, error) {
	if def == "" || strings.ContainsAny(def, " \t\n:^~?*[\\") || strings.HasPrefix(def, "-") {
		return "", errors.New("origin's default branch is not named, so the window cannot be counted")
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", 256, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+def+"^{commit}", "--")
	tip := strings.TrimSpace(string(out))
	if err != nil || !gitutil.IsFullSHA(tip) {
		return "", fmt.Errorf("origin/%s cannot be read, so the window cannot be counted", def)
	}
	return tip, nil
}

// closedNow is the bound the window st holds has passed by now, at tip.
func closedNow(ctx context.Context, root string, cfg Config, st windowState, tip string, at time.Time) (string, error) {
	changes := 0
	if cfg.WindowChanges > 0 {
		n, err := changesSince(ctx, root, st.from, tip)
		if err != nil {
			return "", err
		}
		changes = n
	}
	return windowClosedBy(cfg, st.fromTime, changes, at), nil
}

// carry pushes the records origin's copy lacks, with nothing drawn.
func carry(ctx context.Context, root string) string {
	res, err := push(ctx, root)
	switch {
	case err != nil:
		return "spot-check: " + redact(err)
	case res.Pushed > 0:
		return fmt.Sprintf("spot-check: carried %d record(s) to %s", res.Pushed, Branch)
	}
	return ""
}

// draw is AfterLanding once the window may have passed: tried twice, the
// second time after a fresh fetch when origin rejected the first push.
func draw(ctx context.Context, o Observed, cfg Config, tip string) string {
	var beacon *Beacon
	var last error
	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := fetchTracking(ctx, o.Root); err != nil {
			return "spot-check: " + redact(err) + "; nothing was drawn"
		}
		st, err := readWindow(ctx, o.Root)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		pending, _, err := outbound(ctx, o.Root, st.files)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		if !st.open {
			w, err := encodeFile("", WindowOpenFile, WindowOpen{SchemaVersion: SchemaVersion, Commit: tip, At: o.Now, Holder: Holder()})
			if err != nil {
				return "spot-check: " + err.Error()
			}
			if _, err := pushOver(ctx, o.Root, st.tracking, append(pending, w), "spotcheck: the first window opens"); err != nil {
				last = err
				continue
			}
			return "spot-check: the first window opened at " + tip[:12] + "; nothing is drawn until it passes"
		}
		closedBy, err := closedNow(ctx, o.Root, cfg, st, tip, o.Now)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		if closedBy == "" {
			// Another writer drew this window first.
			if len(pending) == 0 {
				return ""
			}
			if _, err := pushOver(ctx, o.Root, st.tracking, pending, fmt.Sprintf("spotcheck: %d record(s)", len(pending))); err != nil {
				last = err
				continue
			}
			return fmt.Sprintf("spot-check: carried %d record(s) to %s", len(pending), Branch)
		}
		if beacon == nil {
			b, err := fetchBeacon(ctx, o.Now)
			if err != nil {
				return "spot-check: " + err.Error() + "; nothing was drawn"
			}
			beacon = &b
		}
		c, items, err := makeCycle(ctx, o, cfg, st, tip, closedBy, *beacon, pending)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		files := append(pending, items...)
		cf, err := encodeFile(CyclesDir, c.ID+".json", c)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		files = append(files, cf)
		if _, err := pushOver(ctx, o.Root, st.tracking, files, "spotcheck: cycle "+c.ID); err != nil {
			last = err
			continue
		}
		drawn := 0
		for _, g := range c.Groups {
			drawn += len(g.Drawn)
		}
		return fmt.Sprintf("spot-check: drew cycle %s (round %d of the beacon): %d item(s) for review", c.ID, beacon.Round, drawn)
	}
	return "spot-check: the push to " + Branch + " was refused twice (" + redact(last) + "); nothing was drawn, and the next landing tries again"
}

// makeCycle draws the cycle closing st's window at tip: the populations of
// the window (from st.from exclusive to tip, from st.fromTime exclusive to
// o.Now), the size over the cycles before it, and an item file per drawn key.
// pending are the typed records this checkout carries in the same commit,
// read as origin's own.
func makeCycle(ctx context.Context, o Observed, cfg Config, st windowState, tip, closedBy string, b Beacon, pending []file) (Cycle, []file, error) {
	reader := registeredReader()
	if reader == nil {
		return Cycle{}, nil, errNoReader
	}
	var rows []LandingRow
	if st.from != tip {
		var err error
		if rows, err = reader(ctx, o.Root, st.from, tip); err != nil {
			return Cycle{}, nil, fmt.Errorf("the window's landings cannot be read: %s", redact(err))
		}
	}
	rec, _, err := read(ctx, o.Root, TrackingRef)
	if err != nil {
		return Cycle{}, nil, err
	}
	var hbs []namedHandBack
	var ds []namedDecision
	for i, h := range rec.HandBacks {
		hbs = append(hbs, namedHandBack{name: rec.handBackNames[i], h: h})
	}
	for i, d := range rec.Decisions {
		ds = append(ds, namedDecision{name: rec.decisionNames[i], d: d})
	}
	for _, f := range pending {
		var one Records
		if !readRecord(&one, f.folder, f.name, f.data) {
			continue
		}
		for _, h := range one.HandBacks {
			hbs = append(hbs, namedHandBack{name: f.name, h: h})
		}
		for _, d := range one.Decisions {
			ds = append(ds, namedDecision{name: f.name, d: d})
		}
	}
	groups, err := populations(rows, hbs, ds, st.fromTime, o.Now)
	if err != nil {
		return Cycle{}, nil, err
	}
	tallies, err := tallyCycles(ctx, o.Root, st)
	if err != nil {
		return Cycle{}, nil, err
	}
	size, why, counted := sampleSize(cfg.PerGroup, tallies)
	randomness, err := hex.DecodeString(b.Randomness)
	if err != nil || len(randomness) != 32 {
		return Cycle{}, nil, errors.New("the beacon's randomness does not read")
	}
	c := Cycle{SchemaVersion: SchemaVersion, ID: cycleID(o.Now, tip), FromCommit: st.from, ToCommit: tip,
		FromTime: st.fromTime, ToTime: o.Now, ClosedBy: closedBy, Site: o.Site, Beacon: b, Draw: DrawRule,
		Sample: Sample{PerGroup: cfg.PerGroup, Size: size, Why: why, Counted: counted}, Holder: Holder()}
	var items []file
	n := 0
	for _, g := range groupOrder {
		cands := groups[g]
		keys := make([]string, len(cands))
		byKey := map[string]candidate{}
		for i, cd := range cands {
			keys[i], byKey[cd.key] = cd.key, cd
		}
		drawn := drawKeys(randomness, c.ID, g, keys, size)
		c.Groups = append(c.Groups, GroupDraw{Group: g, Population: population(keys), Drawn: drawn})
		for _, k := range drawn {
			n++
			cd := byKey[k]
			it := Item{SchemaVersion: SchemaVersion, Cycle: c.ID, Number: n, Group: g, Key: k, Roles: cd.roles,
				Landing: cd.landing, HandBack: cd.handBack, Decision: cd.decision}
			f, err := encodeFile(ItemsDir+"/"+c.ID, itemStem(n, g, k)+".json", it)
			if err != nil {
				return Cycle{}, nil, fmt.Errorf("item %s of the cycle cannot be kept: %w", strconv.Itoa(n), err)
			}
			items = append(items, f)
		}
	}
	return c, items, nil
}
