package spotcheck

// afterlanding.go is the collection lane's one entry (spc-2610301925429250
// approaches 3 and 11): AfterLanding, called where abcd has just proved a
// landing. While the window has not passed it reads origin's copy as last
// fetched and the default branch's line, locally, and reaches the network
// only to carry records this checkout holds that origin's copy lacks. Once
// the window has closed and the round its close fixes is published, it
// fetches the store's branch and that round, reads the window's landings,
// draws, and pushes the cycle with its items in one commit over origin's
// copy. Whatever it does, its answer is a note and never an error, an
// outcome or a hold: a draw that refuses for any reason leaves the window
// passed and writes nothing, and the next landing observed tries again, to
// the same draw. Nothing it does decides anything a landing or a drain does.

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
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
	tip, err := defaultTip(ctx, o.Root, o.Default)
	if err != nil {
		return "spot-check: " + err.Error() + "; nothing was drawn"
	}
	st, err := readWindow(ctx, o.Root, o.Now, tip)
	if errors.Is(err, errBehind) {
		// A copy behind the mark reaches the network for the one fetch that
		// brings it up; a branch the fetch finds rewritten is refused there.
		if _, err = fetchTracking(ctx, o.Root); err == nil {
			st, err = readWindow(ctx, o.Root, o.Now, tip)
		} else {
			err = errors.New(redact(err))
		}
	}
	if err != nil {
		return "spot-check: " + err.Error() + "; nothing was drawn"
	}
	if st.open {
		cfg, why, err := windowConfig(ctx, o.Root, st.from)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		cl, err := windowClose(ctx, o.Root, cfg, st.from, st.fromTime, tip)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		if cl.by == "" {
			pending, _, err := outbound(ctx, o.Root, st.files)
			if err != nil {
				return "spot-check: " + err.Error()
			}
			if len(pending) == 0 {
				return withTail("", markNote(st.markErr)+defaultsNote(why))
			}
			return withTail(carry(ctx, o.Root), markNote(st.markErr)+defaultsNote(why))
		}
		if note := notYet(cl, o.Now); note != "" {
			return note + defaultsNote(why)
		}
	}
	return draw(ctx, o, tip)
}

// withTail is note with tail, each "" for nothing to say, as one note: a
// tail alone stands as the note's whole.
func withTail(note, tail string) string {
	if note == "" && tail != "" {
		return "spot-check: " + strings.TrimPrefix(tail, "; ")
	}
	return note + tail
}

// notYet is the note for a window that has closed before the round its
// close fixes is published, "" once it is: nothing is asked of the beacon
// and nothing is written, and a landing observed after the round draws it,
// the same draw whoever makes it.
func notYet(cl closing, at time.Time) string {
	r := roundAt(cl.at)
	if !at.Before(roundTime(r)) {
		return ""
	}
	return fmt.Sprintf("spot-check: the window that closed at %s is not yet drawable: its round %d of the beacon is published at %s",
		cl.at.Format(time.RFC3339), r, roundTime(r).Format(time.RFC3339))
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
// second time after a fresh fetch when origin rejected the first push. The
// window's close, its round and so its draw are fixed by the default
// branch's line and the chain's clock, so a second try, or another
// checkout's, draws the same items; nothing about the moment it runs
// chooses them.
func draw(ctx context.Context, o Observed, tip string) string {
	var beacon *Beacon
	var last error
	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := fetchTracking(ctx, o.Root); err != nil {
			return "spot-check: " + redact(err) + "; nothing was drawn"
		}
		st, err := readWindow(ctx, o.Root, o.Now, tip)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		pending, _, err := outbound(ctx, o.Root, st.files)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		// The records carried with a cycle leave room in the push for its
		// items and itself; any beyond go with a later push, into a later
		// window's population.
		pending = pending[:min(len(pending), maxPushFiles-len(groupOrder)*maxPerGroup-1)]
		if !st.open {
			// The opening is never earlier than its commit, so a tip
			// committed by a clock ahead of this one opens at the tip's own
			// time; readers past it hold that to their clock.
			at, err := committedAt(ctx, o.Root, tip)
			if err != nil {
				return "spot-check: " + err.Error() + "; nothing was drawn"
			}
			at = later(at, o.Now)
			w, err := encodeFile("", WindowOpenFile, WindowOpen{SchemaVersion: SchemaVersion, Commit: tip, At: at, Holder: Holder()})
			if err != nil {
				return "spot-check: " + err.Error()
			}
			commit, err := pushOver(ctx, o.Root, st.tracking, append(pending, w), "spotcheck: the first window opens")
			if err != nil {
				last = err
				continue
			}
			// The mark keeps the opening once its commit's child confirms it.
			note := markNote(writeMark(ctx, o.Root, mark{Commit: commit}))
			return "spot-check: the first window opened at " + tip[:12] + "; nothing is drawn until it passes" + setAside(st) + note
		}
		cfg, why, err := windowConfig(ctx, o.Root, st.from)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		cl, err := windowClose(ctx, o.Root, cfg, st.from, st.fromTime, tip)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		if cl.by == "" {
			// Another writer drew this window first.
			if len(pending) == 0 {
				return withTail("", defaultsNote(why))
			}
			if _, err := pushOver(ctx, o.Root, st.tracking, pending, fmt.Sprintf("spotcheck: %d record(s)", len(pending))); err != nil {
				last = err
				continue
			}
			return fmt.Sprintf("spot-check: carried %d record(s) to %s", len(pending), Branch) + defaultsNote(why)
		}
		if note := notYet(cl, o.Now); note != "" {
			return note + defaultsNote(why)
		}
		round := roundAt(cl.at)
		if beacon == nil || beacon.Round != round {
			b, err := fetchBeacon(ctx, round, o.Now)
			if err != nil {
				return "spot-check: " + err.Error() + "; nothing was drawn"
			}
			beacon = &b
		}
		c, items, unread, err := makeCycle(ctx, o, cfg, why != "", st, cl, *beacon, pending)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		files := append(pending, items...)
		cf, err := encodeFile(CyclesDir, c.ID+".json", c)
		if err != nil {
			return "spot-check: " + err.Error() + "; nothing was drawn"
		}
		files = append(files, cf)
		commit, err := pushOver(ctx, o.Root, st.tracking, files, "spotcheck: cycle "+c.ID)
		if err != nil {
			last = err
			continue
		}
		st.cycles = append(st.cycles, c)
		note := markNote(writeMark(ctx, o.Root, st.mark(commit)))
		drawn := 0
		for _, g := range c.Groups {
			drawn += len(g.Drawn)
		}
		return fmt.Sprintf("spot-check: drew cycle %s (round %d of the beacon): %d item(s) for review", c.ID, beacon.Round, drawn) + unreadNote(unread) + defaultsNote(why) + setAside(st) + note
	}
	return "spot-check: the push to " + Branch + " was refused twice (" + redact(last) + "); nothing was drawn, and the next landing tries again"
}

// markNote is the note's tail for a high-water mark that could not be kept,
// "" when it was.
func markNote(err error) string {
	if err == nil {
		return ""
	}
	return "; the high-water mark could not be kept (" + redact(err) + "), so this machine would not notice the branch rewritten past it"
}

// maxSetAside bounds the cycles a note names as set aside.
const maxSetAside = 3

// setAside is the note's tail naming the cycles of st that did not verify,
// "" for none.
func setAside(st windowState) string {
	if len(st.unverified) == 0 {
		return ""
	}
	named := st.unverified[:min(len(st.unverified), maxSetAside)]
	out := "; " + strings.Join(named, "; ")
	if more := len(st.unverified) - len(named); more > 0 {
		out += fmt.Sprintf("; and %d more", more)
	}
	return out
}

// makeCycle draws the cycle closing st's window at cl: the landings of the
// window (from st.from exclusive to cl.commit), the records origin's copy
// holds that no earlier verified cycle drew from (windowRecords, which reads
// only those), the size over the cycles
// before it, and an item file per drawn key; defaults says the bundled
// defaults counted the window (windowConfig). pending are the typed records
// this checkout carries in the same commit, read as origin's own.
func makeCycle(ctx context.Context, o Observed, cfg Config, defaults bool, st windowState, cl closing, b Beacon, pending []file) (Cycle, []file, int, error) {
	reader := registeredReader()
	if reader == nil {
		return Cycle{}, nil, 0, errNoReader
	}
	var rows []LandingRow
	if st.from != cl.commit {
		var err error
		if rows, err = reader(ctx, o.Root, st.from, cl.commit); err != nil {
			return Cycle{}, nil, 0, fmt.Errorf("the window's landings cannot be read: %s", redact(err))
		}
	}
	for _, r := range rows {
		if !slices.Contains(cl.commits, r.Commit) {
			return Cycle{}, nil, 0, fmt.Errorf("the landings reader named %s, which is not a commit of the window", termsafe.Sanitize(r.Commit))
		}
	}
	seen := map[string]map[string]bool{}
	for _, c := range st.cycles {
		for _, g := range c.Groups {
			if seen[g.Group] == nil {
				seen[g.Group] = map[string]bool{}
			}
			for _, k := range g.Population.Keys {
				seen[g.Group][k] = true
			}
		}
	}
	hbs, ds, unread, err := windowRecords(ctx, o.Root, st.files, pending, seen)
	if err != nil {
		return Cycle{}, nil, 0, err
	}
	groups, err := populations(rows, hbs, ds, seen)
	if err != nil {
		return Cycle{}, nil, 0, err
	}
	tallies, err := tallyCycles(ctx, o.Root, st)
	if err != nil {
		return Cycle{}, nil, 0, err
	}
	size, why, counted := sampleSize(cfg.PerGroup, tallies)
	randomness, err := hex.DecodeString(b.Randomness)
	if err != nil || len(randomness) != 32 {
		return Cycle{}, nil, 0, errors.New("the beacon's randomness does not read")
	}
	c := Cycle{SchemaVersion: SchemaVersion, ID: cycleID(cl.at, cl.commit), FromCommit: st.from, ToCommit: cl.commit,
		FromTime: st.fromTime, ToTime: cl.at, ClosedBy: cl.by, Window: CycleWindow{Days: cfg.WindowDays, Changes: cfg.WindowChanges, Commits: len(cl.commits), Defaults: defaults},
		Site: o.Site, Beacon: b, Draw: DrawRule,
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
		drawn := drawKeys(randomness, drawSeed(c), g, keys, size)
		c.Groups = append(c.Groups, GroupDraw{Group: g, Population: population(keys), Drawn: drawn})
		for _, k := range drawn {
			n++
			cd := byKey[k]
			it := Item{SchemaVersion: SchemaVersion, Cycle: c.ID, Number: n, Group: g, Key: k, Roles: cd.roles,
				Landing: cd.landing, HandBack: cd.handBack, Decision: cd.decision}
			f, err := encodeFile(ItemsDir+"/"+c.ID, itemStem(n, g, k)+".json", it)
			if err != nil {
				return Cycle{}, nil, 0, fmt.Errorf("item %s of the cycle cannot be kept: %w", strconv.Itoa(n), err)
			}
			items = append(items, f)
		}
	}
	return c, items, unread, nil
}

// maxFilesPerKey bounds the files of one record key a draw reads: records of
// one key are one item, and its greatest names are the ones it copies.
const maxFilesPerKey = 4

// windowRecords are the hand-backs and decisions a window draws from: of the
// records origin's copy holds (files, its listing) and those pending carries
// with the cycle, the ones whose key no earlier verified cycle's population of
// their group listed (seen). Only their files are read, so a draw reads what
// arrived since the cycles before it, never the store's whole life. Within a
// group the keys are read in sorted order, each by its maxFilesPerKey
// greatest names, until maxListedKeys of them read as records or half of
// maxReadFiles files are read; a key none of whose files reads takes no
// place, and a key beyond either bound waits for a later window, as a record
// that reaches origin late does. unread counts the files that did not read
// as records; a group whose read bound is spent before any of its records
// read, with some file unread, refuses the draw, naming the count. So every record population is listed
// whole, and the next draw knows each record already drawn from by its key.
// A file whose name is not a record's is never read, and one whose content
// does not read as its name gives is no record.
func windowRecords(ctx context.Context, root string, files map[string]entry, pending []file, seen map[string]map[string]bool) ([]namedHandBack, []namedDecision, int, error) {
	type src struct {
		dir, name string
		// data is the pending copy's content, nil for origin's file.
		data []byte
	}
	byKey := map[string]map[string]map[string]src{}
	add := func(dir, name string, data []byte) {
		g, re := GroupDrainHandBacks, handBackNameRe
		if dir == DecisionsDir {
			g, re = GroupFacilitatorDecisions, decisionNameRe
		}
		k := recordKey(name)
		if !re.MatchString(name) || seen[g][k] {
			return
		}
		if byKey[g] == nil {
			byKey[g] = map[string]map[string]src{}
		}
		if byKey[g][k] == nil {
			byKey[g][k] = map[string]src{}
		}
		byKey[g][k][name] = src{dir: dir, name: name, data: data}
	}
	for p, e := range files {
		dir, name, _ := strings.Cut(p, "/")
		if (dir != HandBacksDir && dir != DecisionsDir) || strings.Contains(name, "/") || e.typ != "blob" || e.mode != "100644" || e.size > maxRecordBytes {
			continue
		}
		add(dir, name, nil)
	}
	for _, f := range pending {
		if f.folder == HandBacksDir || f.folder == DecisionsDir {
			// A pending copy replaces origin's file of its name.
			add(f.folder, f.name, f.data)
		}
	}
	var hbs []namedHandBack
	var ds []namedDecision
	unread := 0
	for _, g := range []string{GroupDrainHandBacks, GroupFacilitatorDecisions} {
		keys := make([]string, 0, len(byKey[g]))
		for k := range byKey[g] {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		// The group's names in key order, each key by its greatest names,
		// read a batch at a time: a key none of whose files reads as a
		// record takes no place, and the next key is read in its stead,
		// until the window's places are filled or the group's files are
		// spent.
		budget, used, listed, next, missed := max(maxReadFiles/2, 1), 0, 0, 0, 0
		spent := false
		for listed < maxListedKeys && next < len(keys) && !spent {
			type keyFiles struct {
				key   string
				names []string
			}
			var batch []keyFiles
			var paths []string
			for next < len(keys) && len(batch) < maxListedKeys-listed {
				names := make([]string, 0, len(byKey[g][keys[next]]))
				for n := range byKey[g][keys[next]] {
					names = append(names, n)
				}
				slices.Sort(names)
				slices.Reverse(names)
				names = names[:min(len(names), maxFilesPerKey)]
				if used > 0 && used+len(names) > budget {
					spent = true
					break
				}
				used += len(names)
				batch = append(batch, keyFiles{key: keys[next], names: names})
				for _, n := range names {
					if src := byKey[g][keys[next]][n]; src.data == nil {
						paths = append(paths, src.dir+"/"+n)
					}
				}
				next++
			}
			blobs, err := readMany(ctx, root, files, paths)
			if err != nil {
				return nil, nil, 0, err
			}
			for _, kf := range batch {
				read := false
				for _, n := range kf.names {
					src := byKey[g][kf.key][n]
					data := src.data
					if data == nil {
						data = blobs[src.dir+"/"+n]
					}
					var one Records
					if data == nil || !readRecord(&one, src.dir, n, data) {
						missed++
						continue
					}
					read = true
					for _, h := range one.HandBacks {
						hbs = append(hbs, namedHandBack{name: n, h: h})
					}
					for _, d := range one.Decisions {
						ds = append(ds, namedDecision{name: n, d: d})
					}
				}
				if read {
					listed++
				}
			}
		}
		unread += missed
		if spent && listed == 0 && missed > 0 {
			// The group's whole read bound went on files that are not
			// records: the window is not drawn with that group empty while
			// records may read beyond them. A group that listed some is
			// drawn from those, and the note counts the rest.
			return nil, nil, unread, fmt.Errorf("the spot-check's %s listed none of its records within its read of %d files, and %d record file(s) on origin did not read as records", g, budget, missed)
		}
	}
	return hbs, ds, unread, nil
}

// unreadNote is the note's tail counting the record files on origin that
// did not read as records and so took no place in the window, "" for none.
func unreadNote(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("; %d record file(s) on origin did not read as records and took no place in the window", n)
}
