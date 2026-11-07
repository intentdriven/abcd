package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/landings"
)

// admittedEntry reports whether the run's record says the admission at its start.
func admittedEntry(st State, want string) bool {
	return slices.ContainsFunc(st.Record, func(e Entry) bool {
		return e.Stage == StageAdmitted && e.Note == "admitted under "+want
	})
}

// TestStartRecordsAdmission (itd-2610301850454860): a run's state names the rule
// its lanes were admitted under from its start, and its record says it: the spec
// for a run `abcd build <itd-N>` starts, the issue for `abcd build <iss-N>`, the
// spec of the intent `abcd build next` picked, and the drain's rule record, by id
// and path, for a lane the drain opens.
func TestStartRecordsAdmission(t *testing.T) {
	t.Run("build an intent", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		res, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
		if err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), res.RunID)
		if err != nil {
			t.Fatal(err)
		}
		want := Admitted{Kind: landings.KindBuild, Record: st.Spec}
		if st.Spec == "" || st.Admitted == nil || *st.Admitted != want || !admittedEntry(st, want.String()) {
			t.Fatalf("the run is admitted under its spec %q: %+v %+v", st.Spec, st.Admitted, st.Record)
		}
	})
	t.Run("build an issue", func(t *testing.T) {
		repo := issueRepo(t)
		res, err := startedOnce(t, repo, eligibleIssue)
		if err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), res.RunID)
		if err != nil {
			t.Fatal(err)
		}
		want := Admitted{Kind: landings.KindBuild, Record: eligibleIssue}
		if st.Admitted == nil || *st.Admitted != want || !admittedEntry(st, want.String()) {
			t.Fatalf("the run is admitted under its issue: %+v", st.Admitted)
		}
	})
	t.Run("build next", func(t *testing.T) {
		repo := pickRepo(t, map[string][2]string{
			"10": {pickIntent("10", "", settledQuestions, gwt), pickSpec("10", fpSmall)},
		})
		res, err := Next(repo.Root(), Options{}, NextOptions{})
		if err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), res.Start.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if st.Pick == nil || st.Spec == "" || st.Admitted == nil || *st.Admitted != (Admitted{Kind: landings.KindBuild, Record: st.Spec}) {
			t.Fatalf("a picked run is admitted under the picked intent's spec: %+v %+v", st.Admitted, st.Spec)
		}
	})
	t.Run("drain", func(t *testing.T) {
		repo := drainRepo(t)
		res := drainedToLaneOnce(t, repo, DrainOptions{})
		if res.Start == nil {
			t.Fatalf("the drain opened a lane: %+v", res)
		}
		st, err := ReadState(repo.Root(), res.Start.RunID)
		if err != nil {
			t.Fatal(err)
		}
		want := Admitted{Kind: landings.KindDrain, Record: "adr-2609300000000001",
			Path: drainrule.ADRsRelDir + "/2609300000000001-drain-rule.md"}
		if st.Admitted == nil || *st.Admitted != want || !admittedEntry(st, want.String()) {
			t.Fatalf("the drain's lane is admitted under the rule record, by id and path: %+v", st.Admitted)
		}
	})
}

// TestADrainsStartWithoutItsRuleIsRefused: a lane start the drain makes with no
// rule record given (as only the bundled baseline would be) is refused before a
// run exists, so no trailer can ever read `drain` with no id.
func TestADrainsStartWithoutItsRuleIsRefused(t *testing.T) {
	repo := issueRepo(t)
	for _, a := range []*Admitted{nil, {Kind: landings.KindDrain}, {Kind: landings.KindBuild, Record: eligibleIssue}} {
		_, err := Start(repo.Root(), eligibleIssue, Options{noReserve: true, drain: true, admitted: a})
		r := mustRefusal(t, err)
		if !strings.Contains(r.Remedy, "drainrule.Load") {
			t.Fatalf("%+v: the refusal names drainrule.Load: %+v", a, r)
		}
		// A rule given and refused is named as that, with why, never as none given.
		if given := a != nil; given == strings.Contains(r.Reason, "none was given") || given && !strings.Contains(r.Reason, "admission") {
			t.Fatalf("%+v: the refusal says whether a rule was given and why it was refused: %q", a, r.Reason)
		}
		if runs, _ := Runs(repo.Root()); len(runs) != 0 {
			t.Fatalf("%+v: no run is created: %d", a, len(runs))
		}
	}
}

// TestAnAdmissionIsVersion24s: a version-23 file carrying `admitted` is refused
// naming it, one without it reads as a run with no admission at the current
// version, and an admission the loop never writes is refused, never copied into
// a commit message.
func TestAnAdmissionIsVersion24s(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	res, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	root, id := repo.Root(), res.RunID
	data := stateBytes(t, root, id)
	state := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
	write := func(b []byte) {
		t.Helper()
		if err := os.WriteFile(state, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	at := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	older := fmt.Sprintf(`"schema_version": %d,`, schemaVersionUnadmitted)
	if !strings.Contains(string(data), at) || !strings.Contains(string(data), `"admitted"`) {
		t.Fatalf("premise: the file is the current version and carries the field: %s", data)
	}
	write([]byte(strings.Replace(string(data), at, older, 1)))
	if _, err := ReadState(root, id); err == nil || !strings.Contains(err.Error(), "admission") {
		t.Fatalf("an older file carrying the field is refused naming it: %v", err)
	}
	write(downgraded(t, data, schemaVersionUnadmitted))
	st, err := ReadState(root, id)
	if err != nil || st.SchemaVersion != SchemaVersion || st.Admitted != nil {
		t.Fatalf("an older file without the field reads as the current version with no admission: %+v %v", st.Admitted, err)
	}
	for _, bad := range []map[string]any{
		{"kind": "build", "record": "spc-1\nResolves: iss-1"},
		{"kind": "drain", "record": "adr-1"},
		{"kind": "build", "record": "itd-10"},
	} {
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}
		m["admitted"] = bad
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		write(b)
		_, err = ReadState(root, id)
		if r := mustRefusal(t, err); !strings.Contains(r.Reason, "admission the loop never writes") || strings.Contains(r.Reason, "Resolves: iss-1") {
			t.Fatalf("%v: refused, without echoing it: %+v", bad, r)
		}
	}
	write(data)
}
