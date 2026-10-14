package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// completeMaskRunes are the runes the completeness test plants in every
// string of a state: a bidi override, the C1 control sequence introducer and a
// tag letter, one from each class termsafe.Sanitize masks.
const completeMaskRunes = "‮\u009b\U000E0041"

var timeType = reflect.TypeFor[time.Time]()

// plantEveryString sets every string the value reaches to a text carrying
// completeMaskRunes, giving every slice one element and every pointer a value,
// so a string field anywhere in the state is reached. A map, an interface or
// an unexported field is a shape the walk in state_mask.go does not cover, and
// fails the test until the walk and this planter are taught it.
func plantEveryString(t *testing.T, v reflect.Value, path string) {
	t.Helper()
	switch v.Kind() {
	case reflect.String:
		v.SetString("a" + completeMaskRunes + "b")
	case reflect.Bool, reflect.Int, reflect.Int64, reflect.Float64:
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		plantEveryString(t, v.Elem(), path)
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		plantEveryString(t, v.Index(0), path+"[0]")
	case reflect.Struct:
		if v.Type() == timeType {
			return
		}
		for i := range v.NumField() {
			f := v.Type().Field(i)
			if !f.IsExported() {
				t.Fatalf("%s.%s is unexported: teach masked and this test its shape", path, f.Name)
			}
			plantEveryString(t, v.Field(i), path+"."+f.Name)
		}
	default:
		t.Fatalf("%s is a %s: teach masked and this test its shape", path, v.Kind())
	}
}

// rawStrings names every string the value reaches that still carries one of
// completeMaskRunes.
func rawStrings(v reflect.Value, path string) []string {
	switch v.Kind() {
	case reflect.String:
		if strings.ContainsAny(v.String(), completeMaskRunes) {
			return []string{path}
		}
	case reflect.Pointer:
		if !v.IsNil() {
			return rawStrings(v.Elem(), path)
		}
	case reflect.Slice:
		var out []string
		for i := range v.Len() {
			out = append(out, rawStrings(v.Index(i), path+"[]")...)
		}
		return out
	case reflect.Struct:
		if v.Type() == timeType {
			return nil
		}
		var out []string
		for i := range v.NumField() {
			out = append(out, rawStrings(v.Field(i), path+"."+v.Type().Field(i).Name)...)
		}
		return out
	}
	return nil
}

// TestMaskedReachesEveryStringOfTheState: every string a run's state stores,
// at any depth, is masked by State.masked (the structural walk the ruling
// "mask when stored and when shown" asks for), and the caller's state is left
// as it was. A string field added to the state without a line in the walk
// fails this test by its path.
func TestMaskedReachesEveryStringOfTheState(t *testing.T) {
	var st, again State
	plantEveryString(t, reflect.ValueOf(&st).Elem(), "State")
	plantEveryString(t, reflect.ValueOf(&again).Elem(), "State")
	got := st.masked()
	for _, p := range rawStrings(reflect.ValueOf(got), "State") {
		t.Errorf("%s reaches the stored or shown state unmasked", p)
	}
	if !reflect.DeepEqual(st, again) {
		t.Error("masked changed the state it was called on")
	}
}

// TestAStateMasksItsIdsAndPathsWhenShown is the probe that found the gap: a
// tag letter planted in a run's key, intent, spec, and a lane's id, key,
// branch and worktree came out raw on the record. Each is read masked now,
// by the read, the record and the run list.
func TestAStateMasksItsIdsAndPathsWhenShown(t *testing.T) {
	repo, runID, _, _ := issueAwaiting(t)
	statePath := filepath.Join(repo.Root(), filepath.FromSlash(StateRelPath(runID)))
	var doc map[string]any
	if err := json.Unmarshal(stateBytes(t, repo.Root(), runID), &doc); err != nil {
		t.Fatal(err)
	}
	const tag = "\U000E0041"
	for _, k := range []string{"key", "intent", "spec"} {
		v, _ := doc[k].(string)
		doc[k] = v + "x" + tag
	}
	lane := doc["lanes"].([]any)[0].(map[string]any)
	for _, k := range []string{"key", "branch", "worktree", "brief", "receipt"} {
		v, _ := lane[k].(string)
		lane[k] = v + "x" + tag
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatalf("a state whose ids carry a tag letter is read, not refused: %v", err)
	}
	rec, err := ReadRecord(repo.Root(), runID)
	if err != nil {
		t.Fatalf("the record reads: %v", err)
	}
	runs, err := Runs(repo.Root())
	if err != nil {
		t.Fatalf("the run list reads: %v", err)
	}
	for what, v := range map[string]any{"the read": st, "the record": rec, "the run list": runs} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), tag) {
			t.Errorf("%s: a raw tag letter survives: %s", what, b)
		}
	}
	if !strings.HasSuffix(st.Lanes[0].Branch, "?") || !strings.HasSuffix(st.Key, "?") {
		t.Errorf("each forged id reads back masked: key %q, branch %q", st.Key, st.Lanes[0].Branch)
	}
}
