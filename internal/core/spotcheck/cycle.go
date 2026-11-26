package spotcheck

// cycle.go is what the draw writes beside the records (spc-2610301925429250
// approach 1): window-open.json, written once by the first check, a cycle
// record per draw under cycles/, and an item file per drawn item under
// items/<cycle-id>/. They are written straight over origin's copy as last
// fetched and pushed at once, never onto the local copy, and hold typed
// fields only: object names, times, record ids, words from fixed sets and the
// beacon's values. No free text a model or a person wrote is among them.

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The draw's folders and files.
const (
	CyclesDir      = "cycles"
	ItemsDir       = "items"
	JudgementsDir  = "judgements"
	WindowOpenFile = "window-open.json"
)

// The words a cycle's window closes by.
const (
	ClosedByDays    = "days"
	ClosedByChanges = "changes"
)

// SiteDrain is the third observing site, a drain move, beside SiteLand and
// SiteLane.
const SiteDrain = "drain"

// The sizes a sample takes.
const (
	SizeFull   = "full"
	SizeHalved = "halved"
)

// DrawRule says how a cycle's draw is recomputed from its record, so anyone
// can repeat it.
const DrawRule = "each key of a group, sorted, is given HMAC-SHA256 keyed by the beacon's randomness over from-commit NUL to-time NUL group NUL key, to-time as YYYYMMDDTHHMMSSZ; the size smallest are drawn"

// drawSeed is the part of a draw's message its window fixes: the commit the
// window starts at and the moment it closed, both settled before its round
// exists and both recomputed by every reader, never the cycle's id.
func drawSeed(c Cycle) string {
	return c.FromCommit + "\x00" + c.ToTime.UTC().Format(stamp)
}

// maxListedKeys is the most keys a population lists whole: above it, the
// cycle keeps the first and last key and the SHA-256 of the sorted keys, one
// per line. Four groups listed whole stay within maxCycleBytes.
const maxListedKeys = 1000

// maxCycleBytes bounds one cycle file, on write and on read, as the pre-push
// hook bounds it: a cycle lists its populations, so it is allowed more than a
// record's maxRecordBytes.
const maxCycleBytes = 256 << 10

// WindowOpen is window-open.json: where and when the first window opened.
type WindowOpen struct {
	SchemaVersion int       `json:"schema_version"`
	Commit        string    `json:"commit"`
	At            time.Time `json:"at"`
	Holder        string    `json:"holder,omitempty"`
}

// Cycle is one draw, cycles/<id>.json.
type Cycle struct {
	SchemaVersion int         `json:"schema_version"`
	ID            string      `json:"id"`
	FromCommit    string      `json:"from_commit"`
	ToCommit      string      `json:"to_commit"`
	FromTime      time.Time   `json:"from_time"`
	ToTime        time.Time   `json:"to_time"`
	ClosedBy      string      `json:"closed_by"`
	Window        CycleWindow `json:"window"`
	Site          string      `json:"site"`
	Beacon        Beacon      `json:"beacon"`
	Draw          string      `json:"draw"`
	Sample        Sample      `json:"sample"`
	Groups        []GroupDraw `json:"groups"`
	Holder        string      `json:"holder,omitempty"`
}

// CycleWindow is the window's bounds as the default branch configured them
// at the window's start (windowConfig): WindowDays, and WindowChanges, 0
// when unset; Commits, the first-parent commits the window spans; and
// Defaults, true when the block did not read there and the bundled defaults
// counted the window.
type CycleWindow struct {
	Days     int  `json:"days"`
	Changes  int  `json:"changes"`
	Commits  int  `json:"commits"`
	Defaults bool `json:"defaults,omitempty"`
}

// Sample is the size a cycle drew at and why.
type Sample struct {
	PerGroup int    `json:"per_group"`
	Size     int    `json:"size"`
	Why      string `json:"why"`
	// Counted are the closed cycles the size was decided over, newest first.
	Counted []string `json:"counted"`
}

// GroupDraw is one group of a cycle: its population and the keys drawn.
type GroupDraw struct {
	Group      string     `json:"group"`
	Population Population `json:"population"`
	Drawn      []string   `json:"drawn"`
}

// Population is a group's keys, whole up to maxListedKeys and otherwise by
// their range and hash.
type Population struct {
	Count  int      `json:"count"`
	Keys   []string `json:"keys,omitempty"`
	First  string   `json:"first,omitempty"`
	Last   string   `json:"last,omitempty"`
	SHA256 string   `json:"sha256,omitempty"`
}

// Item is one drawn item, items/<cycle-id>/<nnn>-<group>-<key>.json: the
// state it had at the draw, copied in, so what a person later sees as then
// never rests on a later read.
type Item struct {
	SchemaVersion int         `json:"schema_version"`
	Cycle         string      `json:"cycle"`
	Number        int         `json:"number"`
	Group         string      `json:"group"`
	Key           string      `json:"key"`
	Roles         []string    `json:"roles"`
	Landing       *LandingRow `json:"landing,omitempty"`
	HandBack      *HandBack   `json:"hand_back,omitempty"`
	Decision      *Decision   `json:"decision,omitempty"`
}

var (
	cycleIDRe   = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z-[0-9a-f]{12}$`)
	cycleNameRe = regexp.MustCompile(`^` + CyclesDir + `/([0-9]{8}T[0-9]{6}Z-[0-9a-f]{12})\.json$`)
)

// cycleID is the id of a cycle whose window closed at to and ends at commit:
// its close first, so ids sort in window order. Both are read from the
// default branch's line (windowClose), so every drawer of a window names its
// cycle alike, and the draw it keys is the same whenever it is made.
func cycleID(to time.Time, commit string) string {
	return to.UTC().Format(stamp) + "-" + commit[:12]
}

// itemStem is an item's file name without .json, which its judgements are
// named after.
func itemStem(number int, group, key string) string {
	return fmt.Sprintf("%03d-%s-%s", number, group, key)
}

// windowState is the store's window as origin's copy holds it, every cycle
// verified as it is read.
type windowState struct {
	// tracking is origin's copy as last fetched, "" when this checkout holds
	// none.
	tracking string
	files    map[string]entry
	// open says window-open.json is there and its opening verifies
	// (verifyOpening); opened and openedAt are that opening, and from and
	// fromTime are where the window now open began: the last verified
	// cycle's end, else the opening. confirmed says the opening's commit has
	// its child on the default branch, so its bounds are all checked and the
	// mark may keep it.
	open      bool
	confirmed bool
	opened    string
	openedAt  time.Time
	from      string
	fromTime  time.Time
	// cycles are the verified cycles, in window order: each starts where the
	// one before it ended, the first at the opening.
	cycles []Cycle
	// unverified names each cycle file that is not among them, and why.
	unverified []string
	// markErr is why the high-water mark could not be kept, nil when it was
	// or needed no change.
	markErr error
}

// clockSkew is how far past the reader's clock a cycle's close, or the
// window's opening, may lie and still be read: one beyond it was never
// written then, whatever its file says.
const clockSkew = 5 * time.Minute

// errAhead is a cycle whose window this checkout's copy of the default
// branch does not close yet: drawn, it may be, from a tip not fetched here.
var errAhead = errors.New("its window does not close on this checkout's copy of the default branch")

// verifyOpening checks the window's opening against origin's default branch
// up to tip, as stored. Its moment is its writer's word, held to the line: no
// later than now allows, no earlier than its commit was committed, and, once
// that commit has its first-parent child on the line, no later than the
// child was committed; an honest writer opens at the tip it read, between
// the two. why is the bound an opening fails, "" when it holds: one that
// fails is no window's start, and the next observer opens the window afresh
// over it. An opening at a commit the line does not hold, or not on its
// first parents, passes here and is refused loudly by windowClose, as a
// commit this checkout has not fetched yet may be the line's next. confirmed
// says the child is there, so every bound was checked and none can fail
// later. err is a line that could not be read, which sets nothing aside.
func verifyOpening(ctx context.Context, root, tip string, w WindowOpen, now time.Time) (confirmed bool, why string, err error) {
	if w.At.After(now.Add(clockSkew)) {
		return false, "it opens after now", nil
	}
	if on, err := isAncestor(ctx, root, w.Commit, tip); err != nil || !on {
		return false, "", nil
	}
	ct, err := committedAt(ctx, root, w.Commit)
	if err != nil {
		return false, "", err
	}
	if w.At.Before(ct) {
		return false, "it opens before its commit was committed", nil
	}
	if w.Commit == tip {
		return false, "", nil
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", maxListingBytes, "log", "--first-parent", "--reverse", "--format=%H %ct %P", w.Commit+".."+tip, "--")
	if err != nil {
		return false, "", fmt.Errorf("reading the spot-check's opening: %s", redact(err))
	}
	first, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	f := strings.Fields(first)
	if len(f) < 3 || f[2] != w.Commit {
		return false, "", nil
	}
	secs, err := strconv.ParseInt(f[1], 10, 64)
	if err != nil {
		return false, "", errors.New("reading the spot-check's opening: git answered a line of an unexpected shape")
	}
	if w.At.After(time.Unix(secs, 0)) {
		return false, "it opens after its commit's child on the default branch was committed", nil
	}
	return true, "", nil
}

// committedAt is when commit was committed, as stored.
func committedAt(ctx context.Context, root, commit string) (time.Time, error) {
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", 64, "log", "-1", "--format=%ct", commit, "--")
	secs, perr := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil || perr != nil {
		return time.Time{}, fmt.Errorf("git answered no commit time for %s, so the window's opening cannot be checked", commit[:12])
	}
	return time.Unix(secs, 0).UTC(), nil
}

// readWindow is the window as origin's copy, last fetched, holds it, read
// locally and checked against this machine's mark. The opening is verified
// (verifyOpening), and one that fails is set aside, named in unverified, and
// is no window's start, so the window is opened afresh. Every cycle is
// verified (verifyCycle) and chained from the opening in window order; a
// cycle that fails either is set aside the same way and is no window
// boundary, so its window is drawn afresh. A copy that does not hold the
// mark's commit, its opening or its newest verified cycle is refused as
// rewritten. A copy that holds files the store does not write under the
// draw's names, or an opening that does not read, is an error: a window is
// never guessed. A copy that reads is kept as the new mark, its opening once
// confirmed.
func readWindow(ctx context.Context, root string, now time.Time, tip string) (windowState, error) {
	tracking, err := storeRef(ctx, root, TrackingRef)
	if err != nil {
		return windowState{}, err
	}
	if err := holdsMark(ctx, root, tracking); err != nil || tracking == "" {
		return windowState{}, err
	}
	m, marked, err := readMark(ctx, root)
	if err != nil {
		return windowState{}, err
	}
	files, err := listFiles(ctx, root, tracking)
	if err != nil {
		return windowState{}, err
	}
	st := windowState{tracking: tracking, files: files}
	var ids []string
	for p := range files {
		if m := cycleNameRe.FindStringSubmatch(p); m != nil {
			ids = append(ids, m[1])
		}
	}
	slices.Sort(ids)
	if _, ok := files[WindowOpenFile]; ok {
		var w WindowOpen
		if err := readJSON(ctx, root, files, WindowOpenFile, &w); err != nil {
			return windowState{}, err
		}
		if !gitutil.IsFullSHA(w.Commit) || w.At.IsZero() {
			return windowState{}, errors.New("the spot-check's window-open.json does not read as one this package writes")
		}
		confirmed, why, err := verifyOpening(ctx, root, tip, w, now)
		if err != nil {
			return windowState{}, err
		}
		if why != "" {
			st.unverified = append(st.unverified, "the window's opening on origin does not verify ("+why+"), so it was set aside")
		} else {
			st.open, st.confirmed, st.opened, st.openedAt, st.from, st.fromTime = true, confirmed, w.Commit, w.At, w.Commit, w.At
		}
	}
	if marked && m.Opened != "" && (m.Opened != st.opened || !m.OpenedAt.Equal(st.openedAt)) {
		return windowState{}, rewritten("the window's opening (" + m.Opened[:12] + ")")
	}
	for _, id := range ids {
		var c Cycle
		err := readJSON(ctx, root, files, CyclesDir+"/"+id+".json", &c)
		switch {
		case err != nil:
			err = errors.New("it does not read")
		case !st.open:
			err = errors.New("no window opened before it")
		default:
			err = verifyCycle(ctx, root, tip, c, id, st.from, st.fromTime, now)
		}
		if errors.Is(err, errAhead) {
			st.unverified = append(st.unverified, "cycle "+id+" on origin cannot be checked against this checkout's copy of the default branch yet, so it was set aside")
			continue
		}
		if err != nil {
			st.unverified = append(st.unverified, "cycle "+id+" on origin does not verify ("+err.Error()+"), so it was set aside")
			continue
		}
		st.cycles = append(st.cycles, c)
		st.from, st.fromTime = c.ToCommit, c.ToTime
	}
	if marked && m.Cycle != "" && !slices.ContainsFunc(st.cycles, func(c Cycle) bool { return c.ID == m.Cycle }) {
		return windowState{}, rewritten("the cycle " + m.Cycle)
	}
	if st.open && (!marked || m.Commit != tracking || st.confirmed && m.Opened == "") {
		// A mark this machine cannot write leaves the last one standing,
		// which still refuses a branch rewritten past it; the note says so.
		st.markErr = writeMark(ctx, root, st.mark(tracking))
	}
	return st, nil
}

// mark is st as the high-water mark keeps it, at the store's commit: its
// opening only once confirmed, as one not yet confirmed may still fail its
// bounds, and is then opened afresh rather than refused as rewritten.
func (st windowState) mark(commit string) mark {
	m := mark{Commit: commit}
	if st.confirmed {
		m.Opened, m.OpenedAt = st.opened, st.openedAt
	}
	if n := len(st.cycles); n > 0 {
		c := st.cycles[n-1]
		m.Cycle, m.CycleFrom, m.CycleTo = c.ID, c.FromCommit, c.ToCommit
	}
	return m
}

// verifyCycle checks a cycle read back from origin's copy as the cycle of
// the window that began at from (fromTime), against origin's default branch
// up to tip, as stored. Its window is recomputed, never read from the cycle:
// the spotcheck block as the branch stores it at from (windowConfig) must be
// the cycle's own window and per-group values, and windowClose over the
// branch's first-parent line must give the cycle's bound, end commit and
// close, so its end is on that line and its close and round are the line's.
// Its id must be its close's, its close no later than now allows, its
// signature that of the round its close fixes under the pinned key, its
// sample size one the size rule gives, every landing it lists a commit of
// the window, and each drawn list the one that round's randomness gives over
// its population, keyed by the window alone (drawSeed). A population kept by
// its hash alone is checked by its count. What it cannot check, which records
// a holder of push rights listed and why a size was halved, is the residual
// adr-2611251806496719 names.
func verifyCycle(ctx context.Context, root, tip string, c Cycle, id, from string, fromTime, now time.Time) error {
	switch {
	case c.ID != id || !gitutil.IsFullSHA(c.FromCommit) || !gitutil.IsFullSHA(c.ToCommit) || cycleID(c.ToTime, c.ToCommit) != id:
		return errors.New("its id is not the one its close gives")
	case c.FromCommit != from || !c.FromTime.Equal(fromTime):
		return errors.New("it does not start where its window began")
	case c.ToTime.After(now.Add(clockSkew)):
		return errors.New("it closes after now")
	}
	cfg, why, err := windowConfig(ctx, root, from)
	if err != nil {
		return errors.New("its window's start is not held here")
	}
	if c.Window.Days != cfg.WindowDays || c.Window.Changes != cfg.WindowChanges || c.Sample.PerGroup != cfg.PerGroup || c.Window.Defaults != (why != "") {
		return errors.New("its window is not the one the default branch configures")
	}
	cl, err := windowClose(ctx, root, cfg, from, fromTime, tip)
	if err != nil {
		return errors.New("the default branch gives its window no close")
	}
	if cl.by == "" {
		return errAhead
	}
	if c.ClosedBy != cl.by || c.ToCommit != cl.commit || !c.ToTime.Equal(cl.at) || c.Window.Commits != len(cl.commits) {
		return errors.New("its close is not the one the default branch gives")
	}
	inWindow := map[string]bool{}
	for _, k := range cl.commits {
		inWindow[k] = true
	}
	if err := verifyBeacon(c.Beacon); err != nil {
		return err
	}
	if want := roundAt(c.ToTime); c.Beacon.Round != want {
		return fmt.Errorf("its round %d is not round %d, which its close fixes", c.Beacon.Round, want)
	}
	s := c.Sample
	if !(s.Size == s.PerGroup && s.Why == SizeFull || s.Size == max((s.PerGroup+1)/2, 1) && s.Why == SizeHalved) {
		return errors.New("its sample size is not one the size rule gives")
	}
	randomness, _ := hex.DecodeString(c.Beacon.Randomness)
	if len(c.Groups) != len(groupOrder) {
		return errors.New("it does not hold the four groups")
	}
	for i, g := range c.Groups {
		p := g.Population
		switch {
		case g.Group != groupOrder[i]:
			return errors.New("its groups are not the four in their order")
		case (g.Group == GroupDrainLanded || g.Group == GroupAutomaticMerges) &&
			slices.ContainsFunc(p.Keys, func(k string) bool { return !inWindow[k] }):
			return fmt.Errorf("its %s population names a commit outside its window", g.Group)
		case p.Count <= maxListedKeys && (len(p.Keys) != p.Count || !slices.IsSorted(p.Keys) || len(slices.Compact(slices.Clone(p.Keys))) != p.Count):
			return fmt.Errorf("its %s population does not list its keys", g.Group)
		case p.Count <= maxListedKeys && !slices.Equal(g.Drawn, drawKeys(randomness, drawSeed(c), g.Group, p.Keys, s.Size)):
			return fmt.Errorf("its %s draw is not the one its round gives", g.Group)
		case p.Count > maxListedKeys && (len(p.Keys) != 0 || len(g.Drawn) != min(s.Size, p.Count)):
			return fmt.Errorf("its %s draw is not of its size", g.Group)
		}
	}
	return nil
}

// readJSON decodes the store's file at p, strictly and within
// maxRecordBytes (maxCycleBytes for a cycle), into v; a file of another
// schema version is refused.
func readJSON(ctx context.Context, root string, files map[string]entry, p string, v any) error {
	e, ok := files[p]
	if !ok || e.typ != "blob" || e.mode != "100644" || e.size > fileBound(p) {
		return fmt.Errorf("the spot-check's %s cannot be read", p)
	}
	blobs, err := readBlobs(ctx, root, []entry{e})
	if err != nil {
		return err
	}
	return decodeStoreFile(p, blobs[0], v)
}

// decodeStoreFile is data, the store's file at p, decoded strictly into v;
// a file of another schema version is refused.
func decodeStoreFile(p string, data []byte, v any) error {
	if err := jsonstrict.Decode(data, v); err != nil {
		return fmt.Errorf("the spot-check's %s does not read: %s", p, redact(err))
	}
	var head struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &head); err != nil || head.SchemaVersion != SchemaVersion {
		return fmt.Errorf("the spot-check's %s is of another schema version", p)
	}
	return nil
}

// fileBound is the most bytes the store's file at p may hold: maxCycleBytes
// for a cycle, which lists its populations, and maxRecordBytes for any other.
func fileBound(p string) int {
	if strings.HasPrefix(p, CyclesDir+"/") {
		return maxCycleBytes
	}
	return maxRecordBytes
}

// readMany is the content of each of paths that names a regular blob of the
// store within its bound, by path, read in one batch; a path that does not is
// left out.
func readMany(ctx context.Context, root string, files map[string]entry, paths []string) (map[string][]byte, error) {
	var es []entry
	var kept []string
	for _, p := range paths {
		if e, ok := files[p]; ok && e.typ == "blob" && e.mode == "100644" && e.size <= fileBound(p) {
			es, kept = append(es, e), append(kept, p)
		}
	}
	out := map[string][]byte{}
	if len(es) == 0 {
		return out, nil
	}
	blobs, err := readBlobs(ctx, root, es)
	if err != nil {
		return nil, err
	}
	for i, p := range kept {
		out[p] = blobs[i]
	}
	return out, nil
}

// cycleTally is one cycle as the sample size reads it.
type cycleTally struct {
	id       string
	closed   bool
	findings int
}

// maxTallied bounds the cycles the size reads back, newest first, looking
// for the three closed ones it decides on.
const maxTallied = 24

// The verdicts a judgement gives.
const (
	VerdictSound = "sound"
	VerdictWrong = "wrong"
)

// Judgement is the part of a judgement file, judgements/<cycle-id>/<item
// stem>.<role>.json, the sample size reads: the cycle, item and role it
// judges, which its path must repeat, and its verdict. Its other fields are
// the review lane's.
type Judgement struct {
	SchemaVersion int    `json:"schema_version"`
	Cycle         string `json:"cycle"`
	Item          string `json:"item"`
	Role          string `json:"role"`
	Verdict       string `json:"verdict"`
}

// tallyCycles reads the newest verified cycles of st, newest first, until
// three closed ones are found or maxTallied are read. A cycle's items are
// the ones its drawn lists name, at the paths the draw gave them; a cycle is
// closed when each of them reads as the item drawn there, with its group's
// role among its roles, and every role of each has a judgement that reads
// (readJudgement); its findings are the judgements whose verdict is wrong. A
// file that does not read leaves its cycle open, never closed and clean. A
// cycle that drew nothing is closed and clean.
func tallyCycles(ctx context.Context, root string, st windowState) ([]cycleTally, error) {
	var out []cycleTally
	closed := 0
	for i := len(st.cycles) - 1; i >= 0 && closed < 3 && len(out) < maxTallied; i-- {
		c := st.cycles[i]
		t := cycleTally{id: c.ID, closed: true}
		type slot struct{ stem, group, key string }
		var slots []slot
		var paths []string
		for _, g := range c.Groups {
			for _, k := range g.Drawn {
				s := slot{stem: itemStem(len(slots)+1, g.Group, k), group: g.Group, key: k}
				slots = append(slots, s)
				paths = append(paths, ItemsDir+"/"+c.ID+"/"+s.stem+".json")
			}
		}
		items, err := readMany(ctx, root, st.files, paths)
		if err != nil {
			return nil, err
		}
		var judged []string
		roles := map[string][]string{}
		for n, s := range slots {
			var it Item
			data, ok := items[paths[n]]
			if !ok || decodeStoreFile(paths[n], data, &it) != nil || it.Cycle != c.ID || it.Number != n+1 || it.Group != s.group ||
				it.Key != s.key || !slices.Contains(it.Roles, groupRole(s.group)) {
				t.closed = false
				continue
			}
			roles[s.stem] = it.Roles
			for _, role := range it.Roles {
				judged = append(judged, JudgementsDir+"/"+c.ID+"/"+s.stem+"."+role+".json")
			}
		}
		verdicts, err := readMany(ctx, root, st.files, judged)
		if err != nil {
			return nil, err
		}
		for _, s := range slots {
			for _, role := range roles[s.stem] {
				v, ok := readJudgement(verdicts[JudgementsDir+"/"+c.ID+"/"+s.stem+"."+role+".json"], c.ID, s.stem, role)
				if !ok {
					t.closed = false
					continue
				}
				if v == VerdictWrong {
					t.findings++
				}
			}
		}
		if t.closed {
			closed++
		}
		out = append(out, t)
	}
	return out, nil
}

// groupRole is the role every item of group is for.
func groupRole(group string) string {
	if group == GroupFacilitatorDecisions {
		return RoleHumanFacilitator
	}
	return RoleFacilitator
}

// readJudgement is a judgement's verdict when data, its file, reads as one:
// one JSON object of no repeated key, of this schema version, naming the
// cycle, item and role its path names, with a verdict from the fixed set.
// Anything else, a missing file included, is no judgement.
func readJudgement(data []byte, cycle, item, role string) (string, bool) {
	if data == nil || jsonstrict.NoDuplicateKeys(data) != nil {
		return "", false
	}
	var j Judgement
	if err := json.Unmarshal(data, &j); err != nil || j.SchemaVersion != SchemaVersion || j.Cycle != cycle || j.Item != item || j.Role != role ||
		(j.Verdict != VerdictSound && j.Verdict != VerdictWrong) {
		return "", false
	}
	return j.Verdict, true
}

// encodeFile is v as the store's file at folder/name.
func encodeFile(folder, name string, v any) (file, error) {
	data, err := encodeWithin(v, fileBound(strings.TrimPrefix(folder+"/"+name, "/")))
	if err != nil {
		return file{}, err
	}
	return file{folder: folder, name: name, data: data}, nil
}
