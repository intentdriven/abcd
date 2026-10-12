package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The stored pace's provenance (iss-2610121206163414): each pace setting's
// layer and origin are read back from the series' state, a run's state and the
// drain's state, and echoed on --json streams and in a resume refusal. A file
// whose layer is not one abcd writes, or whose origin carries a control, bidi
// or zero-width rune, is refused when it is read, never echoing the value.

// hostilePaces are the forged settings every reader refuses: U+202E (a bidi
// override) and U+009B (the C1 CSI) in a layer and in an origin, a layer abcd
// never writes, and a fix-rounds setting whose layer was emptied while its
// value and origin stayed.
var hostilePaces = []struct{ setting, field, bad string }{
	{"work_minutes", "layer", "‮flag"},
	{"sub_agents", "layer", "bundled\u009b"},
	{"pause_minutes", "layer", "file"},
	{"fix_rounds", "layer", ""},
	{"work_minutes", "origin", "--pace 90/0‮"},
	{"fix_rounds", "origin", "\u009b--fix-rounds 1"},
}

// forgePace writes good to path with one pace setting's field replaced by bad.
func forgePace(t *testing.T, path string, good []byte, setting, field, bad string) {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(good, &doc); err != nil {
		t.Fatal(err)
	}
	pace, ok := doc["pace"].(map[string]any)
	if !ok {
		t.Fatalf("the file carries a pace: %s", good)
	}
	v, ok := pace[setting].(map[string]any)
	if !ok {
		t.Fatalf("the pace carries %s: %s", setting, good)
	}
	v[field] = bad
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// noRawRune fails when a refusal carries either rune the forgeries plant.
func noRawRune(t *testing.T, what string, r *Refusal) {
	t.Helper()
	if strings.ContainsAny(r.Reason+r.Remedy, "‮\u009b") {
		t.Fatalf("%s: the refusal carries a raw control rune: %q", what, r.Reason+" | "+r.Remedy)
	}
}

// TestASeriesRefusesAPaceItNeverWrites: a series begun with pace flags reads
// its pace back, each setting with its layer and origin; one whose pace names
// a layer the series never writes or an origin with a control or bidi rune is
// refused at the pick, on a plain move and on one typed with another pace
// (resumeWithFlags' route, whose refusal rendered the stored pace raw).
func TestASeriesRefusesAPaceItNeverWrites(t *testing.T) {
	root := seriesRepo(t)
	if _, err := Next(root, Options{Pace: strp("90/0"), FixRounds: strp("1")}, NextOptions{Max: 2}); err != nil {
		t.Fatal(err)
	}
	res, err := Next(root, Options{}, NextOptions{})
	if err != nil || res.Series == nil || res.Series.Pace == nil {
		t.Fatalf("a series written with real flags reads back: %+v %v", res.Series, err)
	}
	want := Pace{
		WorkMinutes:  PaceValue{Value: 90, Layer: "flag", Origin: "--pace 90/0"},
		PauseMinutes: PaceValue{Value: 0, Layer: "flag", Origin: "--pace 90/0"},
		SubAgents:    PaceValue{Value: BundledSubAgents, Layer: "bundled", Origin: "bundled"},
		FixRounds:    PaceValue{Value: 1, Layer: "flag", Origin: "--fix-rounds 1"},
	}
	if *res.Series.Pace != want {
		t.Fatalf("the series' pace round-trips: %+v", *res.Series.Pace)
	}
	path := filepath.Join(root, filepath.FromSlash(NextSeriesRel))
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range hostilePaces {
		forgePace(t, path, good, c.setting, c.field, c.bad)
		for _, o := range []Options{{}, {Pace: strp("60/0")}} {
			what := fmt.Sprintf("%s.%s %q (pace flag %v)", c.setting, c.field, c.bad, o.Pace != nil)
			_, err := Next(root, o, NextOptions{})
			r := mustRefusal(t, err)
			noRawRune(t, what, r)
			if r.Stage != StagePick || !strings.Contains(r.Reason, "a pace the series never writes") ||
				!strings.Contains(r.Remedy, "restore it or remove it") {
				t.Fatalf("%s: refused at the pick: %+v", what, r)
			}
		}
		if runs, _ := Runs(root); len(runs) != 1 {
			t.Fatalf("a refused series starts no run: %d runs", len(runs))
		}
	}
}

// TestARunStateRefusesAPaceItNeverWrites: a run whose pace comes from all four
// layers reads back unchanged, at the current version and as version 9; one
// whose stored pace names a layer the loop never writes or an origin with a
// control or bidi rune is refused as the run's state, with the remedy every
// state refusal names, by the direct read, the record and a resume typed with
// another pace.
func TestARunStateRefusesAPaceItNeverWrites(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	machineConfig(t, `{"pace": {"pause_minutes": 30}}`)
	repoConfig(t, repo, `{"pace": {"work_minutes": 100}}`)
	start, err := Start(repo.Root(), "itd-10", Options{FixRounds: strp("1")})
	if err != nil {
		t.Fatal(err)
	}
	want := Pace{
		WorkMinutes:  PaceValue{Value: 100, Layer: "repo", Origin: ".abcd/config.json"},
		PauseMinutes: PaceValue{Value: 30, Layer: "machine", Origin: abcdhome.Display("config.json")},
		SubAgents:    PaceValue{Value: BundledSubAgents, Layer: "bundled", Origin: "bundled"},
		FixRounds:    PaceValue{Value: 1, Layer: "flag", Origin: "--fix-rounds 1"},
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(start.RunID)))
	good := stateBytes(t, repo.Root(), start.RunID)
	cur := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	v9 := []byte(strings.Replace(string(good), cur, fmt.Sprintf(`"schema_version": %d,`, schemaVersionUnkept), 1))
	if string(v9) == string(good) {
		t.Fatal("the state names its version")
	}
	for _, version := range []struct {
		name string
		good []byte
	}{{"current", good}, {"version 9", v9}} {
		if err := os.WriteFile(path, version.good, 0o600); err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), start.RunID)
		if err != nil || st.Pace == nil || *st.Pace != want {
			t.Fatalf("%s: a run's pace from every layer round-trips: %+v %v", version.name, st.Pace, err)
		}
		rec, err := ReadRecord(repo.Root(), start.RunID)
		if err != nil || rec.Pace == nil || !reflect.DeepEqual(*rec.Pace, want) {
			t.Fatalf("%s: the record carries the run's pace: %+v %v", version.name, rec.Pace, err)
		}
		for _, c := range hostilePaces {
			forgePace(t, path, version.good, c.setting, c.field, c.bad)
			what := fmt.Sprintf("%s: %s.%s %q", version.name, c.setting, c.field, c.bad)
			reads := map[string]func() error{
				"the read":   func() error { _, err := ReadState(repo.Root(), start.RunID); return err },
				"the record": func() error { _, err := ReadRecord(repo.Root(), start.RunID); return err },
				"a resume":   func() error { _, err := Start(repo.Root(), "itd-10", Options{Pace: strp("60/0")}); return err },
			}
			for name, read := range reads {
				r := mustRefusal(t, read())
				noRawRune(t, what+", "+name, r)
				if r.Stage != "state" || !strings.Contains(r.Reason, StateRelPath(start.RunID)+" carries a pace the loop never writes") ||
					r.Remedy != "the loop is the file's only writer; restore it or remove the run directory "+runRel(start.RunID) {
					t.Fatalf("%s, %s: refused as the run's state: %+v", what, name, r)
				}
			}
		}
	}
}

// TestADrainStateRefusesAPaceItNeverWrites: the drain's state is the third
// file a pace is read back from; a pace the resolver wrote round-trips, and
// one naming a layer the drain never writes or an origin with a control or
// bidi rune is refused at the drain stage.
func TestADrainStateRefusesAPaceItNeverWrites(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	flags, err := parsePaceFlags(strp("90/0"), nil, strp("1"))
	if err != nil {
		t.Fatal(err)
	}
	pace, err := resolvePace(Options{}.roots(root), flags)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsutil.EnsureRealDirAll(root, RunRelDir, dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := writeDrain(root, DrainState{SchemaVersion: DrainSchemaVersion, Rule: "adr-1", Pace: &pace,
		Lanes: []DrainLane{}, HandBacks: []DrainRoute{}}); err != nil {
		t.Fatal(err)
	}
	st, live, err := readDrain(root)
	if err != nil || !live || st.Pace == nil || *st.Pace != pace {
		t.Fatalf("a drain's pace round-trips: %+v %v %v", st.Pace, live, err)
	}
	path := filepath.Join(root, filepath.FromSlash(DrainStateRel))
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range hostilePaces {
		forgePace(t, path, good, c.setting, c.field, c.bad)
		what := fmt.Sprintf("%s.%s %q", c.setting, c.field, c.bad)
		_, _, err := readDrain(root)
		r := mustRefusal(t, err)
		noRawRune(t, what, r)
		if r.Stage != StageDrain || !strings.Contains(r.Reason, "a pace the drain never writes") ||
			!strings.Contains(r.Remedy, "restore it or remove it") {
			t.Fatalf("%s: refused at the drain: %+v", what, r)
		}
	}
}
