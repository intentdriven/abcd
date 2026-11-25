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
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
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
const DrawRule = "each key of a group, sorted, is given HMAC-SHA256 keyed by the beacon's randomness over cycle-id NUL group NUL key; the size smallest are drawn"

// maxListedKeys is the most keys a population lists whole: above it, the
// cycle keeps the first and last key and the SHA-256 of the sorted keys, one
// per line.
const maxListedKeys = 5000

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
	Site          string      `json:"site"`
	Beacon        Beacon      `json:"beacon"`
	Draw          string      `json:"draw"`
	Sample        Sample      `json:"sample"`
	Groups        []GroupDraw `json:"groups"`
	Holder        string      `json:"holder,omitempty"`
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

// cycleID is the id of a cycle closed at to over commit: its time first, so
// ids sort in window order.
func cycleID(to time.Time, commit string) string {
	return to.UTC().Format(stamp) + "-" + commit[:12]
}

// itemStem is an item's file name without .json, which its judgements are
// named after.
func itemStem(number int, group, key string) string {
	return fmt.Sprintf("%03d-%s-%s", number, group, key)
}

// windowState is the store's window as origin's copy holds it.
type windowState struct {
	// tracking is origin's copy as last fetched, "" when this checkout holds
	// none.
	tracking string
	files    map[string]entry
	// open says window-open.json or a cycle is there; from and fromTime are
	// where the window now open began: the last cycle's end, else the
	// window's opening.
	open     bool
	from     string
	fromTime time.Time
	// cycles are the ids of every cycle, in window order.
	cycles []string
}

// readWindow is the window as origin's copy, last fetched, holds it: a
// listing and at most one file read, all local. A copy that holds files the
// store does not write under the draw's names, or a cycle or opening that
// does not read, is an error: a window is never guessed.
func readWindow(ctx context.Context, root string) (windowState, error) {
	tracking, err := storeRef(ctx, root, TrackingRef)
	if err != nil || tracking == "" {
		return windowState{}, err
	}
	files, err := listFiles(ctx, root, tracking)
	if err != nil {
		return windowState{}, err
	}
	st := windowState{tracking: tracking, files: files}
	for p := range files {
		if m := cycleNameRe.FindStringSubmatch(p); m != nil {
			st.cycles = append(st.cycles, m[1])
		}
	}
	slices.Sort(st.cycles)
	if n := len(st.cycles); n > 0 {
		var c Cycle
		if err := readJSON(ctx, root, files, CyclesDir+"/"+st.cycles[n-1]+".json", &c); err != nil {
			return windowState{}, err
		}
		if c.ID != st.cycles[n-1] || !gitutil.IsFullSHA(c.ToCommit) || c.ToTime.IsZero() {
			return windowState{}, fmt.Errorf("the spot-check's cycle %s does not read as one this package writes", st.cycles[n-1])
		}
		st.open, st.from, st.fromTime = true, c.ToCommit, c.ToTime
		return st, nil
	}
	if _, ok := files[WindowOpenFile]; ok {
		var w WindowOpen
		if err := readJSON(ctx, root, files, WindowOpenFile, &w); err != nil {
			return windowState{}, err
		}
		if !gitutil.IsFullSHA(w.Commit) || w.At.IsZero() {
			return windowState{}, errors.New("the spot-check's window-open.json does not read as one this package writes")
		}
		st.open, st.from, st.fromTime = true, w.Commit, w.At
	}
	return st, nil
}

// readJSON decodes the store's file at p, strictly and within
// maxRecordBytes, into v; a file of another schema version is refused.
func readJSON(ctx context.Context, root string, files map[string]entry, p string, v any) error {
	e, ok := files[p]
	if !ok || e.typ != "blob" || e.mode != "100644" || e.size > maxRecordBytes {
		return fmt.Errorf("the spot-check's %s cannot be read", p)
	}
	blobs, err := readBlobs(ctx, root, []entry{e})
	if err != nil {
		return err
	}
	if err := jsonstrict.Decode(blobs[0], v); err != nil {
		return fmt.Errorf("the spot-check's %s does not read: %s", p, redact(err))
	}
	var head struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(blobs[0], &head); err != nil || head.SchemaVersion != SchemaVersion {
		return fmt.Errorf("the spot-check's %s is of another schema version", p)
	}
	return nil
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

// tallyCycles reads the newest cycles of st, newest first, until three closed
// ones are found or maxTallied are read: a cycle is closed when every role of
// every item it drew has a judgement file, and its findings are the
// judgements whose verdict is wrong. A cycle that drew nothing is closed and
// clean.
func tallyCycles(ctx context.Context, root string, st windowState) ([]cycleTally, error) {
	var out []cycleTally
	closed := 0
	for i := len(st.cycles) - 1; i >= 0 && closed < 3 && len(out) < maxTallied; i-- {
		id := st.cycles[i]
		t := cycleTally{id: id, closed: true}
		var items []string
		judged := map[string]string{}
		for p := range st.files {
			if rest, ok := strings.CutPrefix(p, ItemsDir+"/"+id+"/"); ok && strings.HasSuffix(rest, ".json") {
				items = append(items, p)
			}
			if rest, ok := strings.CutPrefix(p, JudgementsDir+"/"+id+"/"); ok && strings.HasSuffix(rest, ".json") {
				judged[strings.TrimSuffix(rest, ".json")] = p
			}
		}
		slices.Sort(items)
		for _, p := range items {
			var it Item
			if err := readJSON(ctx, root, st.files, p, &it); err != nil {
				return nil, err
			}
			stem := strings.TrimSuffix(p[len(ItemsDir+"/"+id+"/"):], ".json")
			for _, role := range it.Roles {
				jp, ok := judged[stem+"."+role]
				if !ok {
					t.closed = false
					continue
				}
				wrong, err := judgedWrong(ctx, root, st.files, jp)
				if err != nil {
					return nil, err
				}
				if wrong {
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

// judgedWrong reads a judgement's verdict: true when it is wrong. Only the
// verdict is read, so a judgement's other fields are the review lane's.
func judgedWrong(ctx context.Context, root string, files map[string]entry, p string) (bool, error) {
	e := files[p]
	if e.typ != "blob" || e.size > maxRecordBytes {
		return false, fmt.Errorf("the spot-check's %s cannot be read", p)
	}
	blobs, err := readBlobs(ctx, root, []entry{e})
	if err != nil {
		return false, err
	}
	var j struct {
		Verdict string `json:"verdict"`
	}
	if err := json.Unmarshal(blobs[0], &j); err != nil {
		return false, fmt.Errorf("the spot-check's %s does not read", p)
	}
	return j.Verdict == "wrong", nil
}

// encodeFile is v as the store's file at folder/name.
func encodeFile(folder, name string, v any) (file, error) {
	data, err := encode(v)
	if err != nil {
		return file{}, err
	}
	return file{folder: folder, name: name, data: data}, nil
}
