package matrix

// rerun.go is a re-run (spc-2610161949559050 step 8; decision 13): a model run
// is re-runnable, not reproducible, so a re-run starts from a recorded run (its
// declaration, at the commit that run froze) under the person's consent to
// its own plan, keeps a copy of the first run's records beside its own, and
// its report shows, field by field, where it drifted from them.
//
// The first run is named by its id in the lab, or by the path of a run's
// folder carried from elsewhere (another machine's lab, say); a folder is
// read as untrusted input, each record validated and passed through the secret
// screen before it is copied. Either way the lab's pin must be the commit the
// first run froze: a re-run from other code would drift by construction.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// firstDir is the folder of a re-run's records that holds the first run's.
const firstDir = "first"

// maxDriftFields bounds the drifted fields a report lists; past it the report
// says how many more drifted.
const maxDriftFields = 2000

// Drift is where a re-run drifted from the run it repeats: every field of the
// two runs' records and of their copies' trials, compared one by one.
type Drift struct {
	// Of is the run the re-run repeats.
	Of string `json:"of"`
	// Fields are the fields that drifted, in order, each with both values as
	// JSON ("absent" where one side has no such field).
	Fields []DriftField `json:"fields"`
	// More counts the drifted fields past those listed.
	More int `json:"more"`
	// Same counts the fields that stayed the same.
	Same int `json:"same"`
}

// DriftField is one field that drifted.
type DriftField struct {
	Field string `json:"field"`
	First string `json:"first"`
	Now   string `json:"now"`
}

// rerun is a recorded run read to be repeated: its record, its declaration's
// bytes and the records the re-run keeps a copy of.
type rerun struct {
	First       RunRecord
	Declaration []byte
	files       []lab.StateFile
}

// loadRerun reads the run from names (a run id in the lab labID, or a run's
// folder) to be repeated in that lab, refusing one whose frozen commit is not
// the lab's pin, and one whose records the secret screen refuses.
func loadRerun(repoRoot, labID, from string) (rerun, error) {
	proc, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return rerun{}, err
	}
	defer proc.Close()
	var src stateReader = proc
	run := from
	if !runIDRe.MatchString(from) {
		dir, err := openRunDir(from)
		if err != nil {
			return rerun{}, err
		}
		defer dir.Close()
		raw, err := dir.ReadState("run.json")
		if err != nil {
			return rerun{}, fmt.Errorf("the run's folder holds no run record: %w", err)
		}
		var rec RunRecord
		if err := strictRun(raw, &rec); err != nil {
			return rerun{}, err
		}
		run, src = rec.RunID, prefixed{dir, rec.RunID}
	}
	rec, _, err := readRun(src, run)
	if err != nil {
		return rerun{}, err
	}
	if pin := proc.Entry().Pin; pin != rec.FrozenCommit {
		return rerun{}, fmt.Errorf("the run %s froze %s, and this lab is pinned at %s: a re-run starts from the frozen commit of the run it repeats; mint a lab at that commit", rec.RunID, rec.FrozenCommit, pin)
	}
	trials, err := readTrials(src, run, run+"/trials")
	if err != nil {
		return rerun{}, err
	}
	raw, err := src.ReadState(run + "/run.json")
	if err != nil {
		return rerun{}, err
	}
	decl, err := src.ReadState(run + "/declaration.json")
	if err != nil {
		return rerun{}, err
	}
	files := []lab.StateFile{{Rel: "run.json", Data: raw}, {Rel: "declaration.json", Data: decl}}
	for _, t := range trials {
		data, err := encodeCheck(t)
		if err != nil {
			return rerun{}, err
		}
		files = append(files, lab.StateFile{Rel: t.Key.rel(), Data: data})
	}
	// The records the re-run keeps a copy of pass the secret screen as they
	// are read, so a run the screen refuses is refused at its plan, before
	// the re-run's record is written; the starter screens them again.
	sc := make([]screened, 0, len(files))
	for _, f := range files {
		sc = append(sc, screened{f, f.Data})
	}
	if err := screen(repoRoot, sc); err != nil {
		return rerun{}, fmt.Errorf("the first run's records are not copied, and the re-run is not started: %w", err)
	}
	return rerun{First: rec, Declaration: decl, files: files}, nil
}

// strictRun decodes a run record and checks its id's shape.
func strictRun(raw []byte, rec *RunRecord) error {
	if err := jsonstrict.Decode(raw, rec); err != nil || !runIDRe.MatchString(rec.RunID) {
		return fmt.Errorf("the run to repeat has a record that does not parse as one")
	}
	return nil
}

// prefixed reads a carried run's folder as if it were the run's folder in a
// lab: every path under the run's id.
type prefixed struct {
	src stateReader
	run string
}

func (p prefixed) strip(rel string) (string, error) {
	r, ok := strings.CutPrefix(rel, p.run+"/")
	if !ok {
		return "", fmt.Errorf("%s is not in the run %s's folder: %w", rel, p.run, fs.ErrNotExist)
	}
	return r, nil
}

func (p prefixed) ReadState(rel string) ([]byte, error) {
	r, err := p.strip(rel)
	if err != nil {
		return nil, err
	}
	return p.src.ReadState(r)
}

func (p prefixed) ListState(rel string) ([]string, error) {
	r, err := p.strip(rel)
	if err != nil {
		return nil, err
	}
	return p.src.ListState(r)
}

// starter is next, run once the first run's records are copied under the
// re-run's own, each passed through the secret screen first: a copy that
// cannot be made starts nothing.
func (r rerun) starter(repoRoot, labID string, next Starter) Starter {
	return rerunStarter{r: r, repoRoot: repoRoot, labID: labID, next: next}
}

type rerunStarter struct {
	r               rerun
	repoRoot, labID string
	next            Starter
}

func (s rerunStarter) Start(p Plan, rec RunRecord) error {
	files := make([]lab.StateFile, 0, len(s.r.files))
	var sc []screened
	for _, f := range s.r.files {
		f.Rel = rec.RunID + "/" + firstDir + "/" + f.Rel
		files = append(files, f)
		sc = append(sc, screened{f, f.Data})
	}
	if err := screen(s.repoRoot, sc); err != nil {
		return fmt.Errorf("the first run's records are not copied, and the re-run is not started: %w", err)
	}
	proc, err := lab.OpenProcedure(s.repoRoot, s.labID)
	if err != nil {
		return err
	}
	err = proc.WriteStates(files...)
	proc.Close()
	if err != nil {
		return fmt.Errorf("the first run's records are not copied, and the re-run is not started: %w", err)
	}
	return s.next.Start(p, rec)
}

// PlanRerun builds the plan for a re-run of the run from names in the lab
// labID, as PlanForLab builds a run's, from that run's recorded declaration.
// It writes nothing.
func PlanRerun(repoRoot, labID, from string, cores int) (Plan, []string, error) {
	r, err := loadRerun(repoRoot, labID, from)
	if err != nil {
		return Plan{}, nil, err
	}
	return PlanForLab(repoRoot, labID, r.Declaration, cores)
}

// StartRerun starts a re-run of the run from names, with approve as the
// person's consent to the plan PlanRerun showed them, as StartRun starts a run:
// the re-run's record is written, the first run's records are copied under
// it, and only then does s start it.
func StartRerun(repoRoot, labID, from, approve string, cores int, executable, version string, s Starter) (RunRecord, Plan, error) {
	r, err := loadRerun(repoRoot, labID, from)
	if err != nil {
		return RunRecord{}, Plan{}, err
	}
	return StartRun(repoRoot, labID, r.Declaration, approve, cores, executable, version, r.starter(repoRoot, labID, s))
}

// rerunDrift is where run rec drifted from the run whose copy it keeps, nil
// where it keeps none.
func rerunDrift(proc *lab.Procedure, rec RunRecord, trials []TrialRecord) (*Drift, error) {
	dir := rec.RunID + "/" + firstDir
	raw, err := proc.ReadState(dir + "/run.json")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var first RunRecord
	if err := strictRun(raw, &first); err != nil {
		return nil, err
	}
	firstTrials, err := readTrials(proc, first.RunID, dir+"/trials")
	if err != nil {
		return nil, err
	}
	a, b := map[string]string{}, map[string]string{}
	if err := flattenInto(a, "", first, runClocks); err != nil {
		return nil, err
	}
	if err := flattenInto(b, "", rec, runClocks); err != nil {
		return nil, err
	}
	for _, t := range firstTrials {
		if err := flattenInto(a, "trials."+t.Key.String(), t, trialClocks); err != nil {
			return nil, err
		}
	}
	for _, t := range trials {
		if err := flattenInto(b, "trials."+t.Key.String(), t, trialClocks); err != nil {
			return nil, err
		}
	}
	d := &Drift{Of: first.RunID, Fields: []DriftField{}}
	keys := make([]string, 0, len(a)+len(b))
	for k := range a {
		keys = append(keys, k)
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		va, oka := a[k]
		vb, okb := b[k]
		if oka && okb && va == vb {
			d.Same++
			continue
		}
		if !oka {
			va = "absent"
		}
		if !okb {
			vb = "absent"
		}
		if len(d.Fields) == maxDriftFields {
			d.More++
			continue
		}
		d.Fields = append(d.Fields, DriftField{Field: k, First: va, Now: vb})
	}
	return d, nil
}

// runClocks and trialClocks are the fields that differ between a run and its
// re-run by construction (an id, a clock), so they are never drift.
var (
	runClocks   = map[string]bool{"run_id": true}
	trialClocks = map[string]bool{"run": true, "started_at": true, "finished_at": true}
)

// flattenInto adds v's fields to out, one per leaf, each path its keys joined
// by dots under prefix, and each value its JSON. A list whose items each carry
// an identifying field (a cell, a field, a rule, a name, a key) is keyed by
// it, so one cell added does not shift every other; any other list is keyed by
// position. A top-level field skip names is left out.
func flattenInto(out map[string]string, prefix string, v any, skip map[string]bool) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var tree any
	if err := json.Unmarshal(raw, &tree); err != nil {
		return err
	}
	if m, ok := tree.(map[string]any); ok {
		for k := range skip {
			delete(m, k)
		}
	}
	flatten(out, prefix, tree)
	return nil
}

func flatten(out map[string]string, path string, v any) {
	join := func(k string) string {
		if path == "" {
			return k
		}
		return path + "." + k
	}
	switch x := v.(type) {
	case map[string]any:
		if len(x) == 0 {
			out[path] = "{}"
		}
		for k, e := range x {
			flatten(out, join(k), e)
		}
	case []any:
		if len(x) == 0 {
			out[path] = "[]"
		}
		ids := listIDs(x)
		for i, e := range x {
			k := strconv.Itoa(i)
			if ids != nil {
				k = ids[i]
			}
			flatten(out, join(k), e)
		}
	default:
		js, _ := json.Marshal(x)
		out[path] = string(js)
	}
}

// listIDs is each item's identifying field, nil unless every item carries one
// and no two share it.
func listIDs(items []any) []string {
	for _, field := range []string{"cell", "field", "rule", "name", "key", "role", "agent"} {
		ids := make([]string, 0, len(items))
		seen := map[string]bool{}
		for _, it := range items {
			m, ok := it.(map[string]any)
			if !ok {
				break
			}
			var id string
			switch v := m[field].(type) {
			case string:
				id = v
			case map[string]any:
				js, _ := json.Marshal(v)
				id = string(js)
			}
			if id == "" || seen[id] {
				break
			}
			seen[id] = true
			ids = append(ids, id)
		}
		if len(ids) == len(items) {
			return ids
		}
	}
	return nil
}

// runDirReader reads a run's folder carried from elsewhere, contained by an
// os.Root so no link in it reads outside it.
type runDirReader struct {
	root *os.Root
}

// openRunDir opens path as a run's folder.
func openRunDir(path string) (*runDirReader, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, fmt.Errorf("the run to repeat is neither a run id nor a run's folder: %v", fsutil.RedactHome(err.Error()))
	}
	return &runDirReader{root: root}, nil
}

func (d *runDirReader) Close() error { return d.root.Close() }

func (d *runDirReader) ReadState(rel string) ([]byte, error) {
	data, err := fsutil.ReadGuardedInRoot(d.root, filepath.FromSlash(rel), lab.MaxStateBytes)
	if err != nil {
		return nil, fmt.Errorf("%s in the run's folder: %w", rel, err)
	}
	return data, nil
}

// ListState lists the folder at rel, refusing, without opening it, anything
// there that is not a directory, and any entry in it that is neither a
// regular file nor a directory: a FIFO or a device in a carried folder would
// otherwise block the open that reads it.
func (d *runDirReader) ListState(rel string) ([]string, error) {
	name := filepath.FromSlash(rel)
	fi, err := d.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s in the run's folder is not a folder", rel)
	}
	f, err := d.root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	names, err := f.Readdirnames(MaxRepeats + maxTasks + maxCells + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	for _, n := range names {
		fi, err := d.root.Lstat(filepath.Join(name, n))
		if err != nil {
			return nil, err
		}
		if !fi.Mode().IsRegular() && !fi.IsDir() {
			return nil, fmt.Errorf("%s/%s in the run's folder is neither a file nor a folder", rel, n)
		}
	}
	sort.Strings(names)
	return names, nil
}
