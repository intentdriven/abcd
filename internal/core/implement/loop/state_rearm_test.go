package loop

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Version 32 kept a landing's re-arms (itd-2611141610201875,
// spc-2611251808377435 step 1): `rearms`, `rearmed_removal` and `armed_at`.
// Version 31 is its strict subset, and a file of it or earlier carrying any
// of the three is refused naming it.

func TestTheReArmFieldsAreTheVersionAfterTheDigest(t *testing.T) {
	if SchemaVersion != schemaVersionUnrearmed+1 || schemaVersionUnrearmed != schemaVersionUndigested+1 {
		t.Fatalf("a landing's re-arms are the version after %d: %d, %d", schemaVersionUndigested, schemaVersionUnrearmed, SchemaVersion)
	}
}

// A version-31 file reads as a run that re-armed nothing, its landings keeping
// no arming time, and the run's next write takes it to the current version.
func TestAVersion31FileReadsAsARunThatReArmedNothing(t *testing.T) {
	f, _ := rearmFixture(t)
	data := stateAtVersion(t, f.repo.Root(), f.runID, schemaVersionUnrearmed)
	for _, field := range []string{`"armed_at"`, `"rearms"`, `"rearmed_removal"`} {
		if bytes.Contains(data, []byte(field)) {
			t.Fatalf("premise: the version-31 file carries no %s:\n%s", field, data)
		}
	}
	ld := f.landingNow(t)
	if ld.Rearms != 0 || ld.RearmedRemoval != nil || ld.ArmedAt != nil || !ld.Armed {
		t.Fatalf("it reads as an armed landing that re-armed nothing and keeps no arming time: %+v", ld)
	}
	// The run's next write is its driver's, claimed as a step claims it.
	if err := claimToSave(f.repo.Root(), f.runID, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.UpdatedAt = st.UpdatedAt.Add(time.Second)
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if got := stateBytes(t, f.repo.Root(), f.runID); !bytes.Contains(got, []byte(fmt.Sprintf(`"schema_version": %d,`, SchemaVersion))) {
		t.Fatalf("the next write states the current version:\n%s", got)
	}
}

// A file of version 31 carrying any of the three fields is not one its version
// wrote, and is refused naming it (an earlier version is refused by the same
// row, after the rows of what it lacks besides: versionChecks).
func TestAnEarlierFileCarryingAReArmFieldIsRefused(t *testing.T) {
	at := time.Date(2026, 10, 15, 7, 30, 0, 0, time.UTC)
	for _, c := range []struct {
		field string
		set   func(ld *Landing)
	}{
		{"`rearms`", func(ld *Landing) { ld.Rearms, ld.ArmedAt = 2, nil }},
		{"`rearmed_removal`", func(ld *Landing) { ld.RearmedRemoval, ld.ArmedAt = &at, nil }},
		{"`armed_at`", func(ld *Landing) { ld.ArmedAt = &at }},
	} {
		for _, version := range []int{schemaVersionUnrearmed} {
			t.Run(fmt.Sprintf("%s at version %d", c.field, version), func(t *testing.T) {
				f, _ := rearmFixture(t)
				f.setLanding(t, c.set)
				good := stateBytes(t, f.repo.Root(), f.runID)
				older := bytes.Replace(good, []byte(fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)), []byte(fmt.Sprintf(`"schema_version": %d,`, version)), 1)
				if bytes.Equal(older, good) {
					t.Fatal("premise: the version changed")
				}
				if err := os.WriteFile(runStatePath(f.repo.Root(), f.runID), older, 0o600); err != nil {
					t.Fatal(err)
				}
				_, err := ReadState(f.repo.Root(), f.runID)
				r, ok := AsRefusal(err)
				if !ok || !strings.Contains(r.Reason, c.field+" on lane-1") || !strings.Contains(r.Reason, fmt.Sprintf("which version %d never wrote", version)) {
					t.Fatalf("the read refuses naming %s: %v", c.field, err)
				}
			})
		}
	}
}

// The current version keeps all three as written.
func TestTheReArmFieldsRoundTrip(t *testing.T) {
	f, _ := rearmFixture(t)
	removed, armed := time.Date(2026, 10, 15, 7, 48, 4, 0, time.UTC), time.Date(2026, 10, 15, 7, 50, 0, 0, time.UTC)
	f.setLanding(t, func(ld *Landing) { ld.Rearms, ld.RearmedRemoval, ld.ArmedAt = 2, &removed, &armed })
	ld := f.landingNow(t)
	if ld.Rearms != 2 || ld.RearmedRemoval == nil || !ld.RearmedRemoval.Equal(removed) || ld.ArmedAt == nil || !ld.ArmedAt.Equal(armed) {
		t.Fatalf("the landing's re-arms read back as written: %+v", ld)
	}
	data := stateBytes(t, f.repo.Root(), f.runID)
	for _, want := range []string{`"rearms": 2`, `"rearmed_removal": "2026-10-15T07:48:04Z"`, `"armed_at": "2026-10-15T07:50:00Z"`} {
		if !bytes.Contains(data, []byte(want)) {
			t.Fatalf("the file carries %s:\n%s", want, data)
		}
	}
}
