package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The drain state's version 2: a routed judge's stored route (a
// DrainJudgement's runner.RouteRecord) can carry a route's pins and say that
// its model is local, which version 1 never wrote. Version 1 is read as
// version 2's strict subset and written back at 2; a version-1 file carrying
// any of those fields is refused naming it; a version above 2 is a newer
// abcd's. Version 3 added a judge fallback receipt's `spent`
// (iss-2611021751406429), on the same terms.

// judgedDrain is writtenDrain with its first judgement answered by a routed
// judge whose route carries sides.
func judgedDrain(sides runner.Sides) DrainState {
	st := writtenDrain()
	st.Judgements[0].Route = &runner.RouteRecord{Asked: "opencode", Ran: "opencode", Model: "qwen3-coder", Sides: sides}
	return st
}

// writeDrainAt writes st with its schema_version set to version, as an abcd
// that wrote that version would have left it, and returns the file's path.
func writeDrainAt(t *testing.T, root string, st DrainState, version int) string {
	t.Helper()
	if err := fsutil.EnsureRealDirAll(root, RunRelDir, dirPerm); err != nil {
		t.Fatal(err)
	}
	st.SchemaVersion = version
	data, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(DrainStateRel))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func storedDrainVersion(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var peek struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &peek); err != nil {
		t.Fatal(err)
	}
	return peek.SchemaVersion
}

func TestDrainSchemaVersionIsFive(t *testing.T) {
	if DrainSchemaVersion != 5 || drainSchemaVersionUnworked != 4 || drainSchemaVersionUnbundled != 3 || drainSchemaVersionUnspent != 2 || drainSchemaVersionUnpinned != 1 {
		t.Fatalf("DrainSchemaVersion = %d (%d, %d, %d, %d), want 5 (4, 3, 2, 1)", DrainSchemaVersion, drainSchemaVersionUnworked,
			drainSchemaVersionUnbundled, drainSchemaVersionUnspent, drainSchemaVersionUnpinned)
	}
}

// TestAVersionFourDrainStateReadsAsOneLaneInWork: a version-4 file reads as a
// drain that started with the bundled limit of one lane in work, its layer
// recorded as bundled, and is written back at 5; version 5's own limit reads
// back as written.
func TestAVersionFourDrainStateReadsAsOneLaneInWork(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	path := writeDrainAt(t, root, bundledDrain(), 4)
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatalf("a version-4 drain state reads: %v", err)
	}
	if st.SchemaVersion != 5 {
		t.Errorf("a version-4 drain state is read at version %d, want 5", st.SchemaVersion)
	}
	if l := st.InWorkMax; l == nil || *l != bundledInWorkMax() {
		t.Errorf("a version-4 drain started with the bundled limit of one lane in work: %+v", l)
	}
	if err := writeDrain(root, st); err != nil {
		t.Fatal(err)
	}
	if v := storedDrainVersion(t, path); v != 5 {
		t.Errorf("the drain writes a version-4 state back at version %d, want 5", v)
	}
	worked := writtenDrain()
	worked.InWorkMax = &PaceValue{Value: 3, Layer: "repo", Origin: ".abcd/config.json at origin/main"}
	writeDrainAt(t, root, worked, 5)
	if st, _, err = readDrain(root); err != nil || st.InWorkMax == nil || *st.InWorkMax != *worked.InWorkMax {
		t.Errorf("version 5's limit reads back as written: %+v %v", st.InWorkMax, err)
	}
}

// TestAnInWorkLimitTheDrainNeverWritesIsRefused: a version-4 file carrying
// in_work_max, and a version-5 limit out of range or from the machine's
// layer, are refused when the state is read.
func TestAnInWorkLimitTheDrainNeverWritesIsRefused(t *testing.T) {
	for name, c := range map[string]struct {
		version int
		limit   PaceValue
		says    string
	}{
		"version 4 carrying it": {4, PaceValue{Value: 1, Layer: "bundled", Origin: "bundled"}, "schema version 4 but carries `in_work_max`"},
		"a limit of 11":         {5, PaceValue{Value: 11, Layer: "flag", Origin: "--in-work 11"}, "a limit on lanes in work the drain never writes"},
		"a limit of 0":          {5, PaceValue{Value: 0, Layer: "bundled", Origin: "bundled"}, "a limit on lanes in work the drain never writes"},
		"the machine's layer":   {5, PaceValue{Value: 2, Layer: "machine", Origin: "config.json"}, "a limit on lanes in work the drain never writes"},
	} {
		t.Run(name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			root := repo.Root()
			st := writtenDrain()
			limit := c.limit
			st.InWorkMax = &limit
			writeDrainAt(t, root, st, c.version)
			_, _, err := readDrain(root)
			if rf := mustRefusal(t, err); !strings.Contains(rf.Reason, c.says) {
				t.Errorf("refused naming %q: %q", c.says, rf.Reason)
			}
		})
	}
}

// TestAJudgeFallbacksSpendIsVersionThrees: a judge fallback receipt's
// `spent` is version 3's (iss-2611021751406429): a version-2 file carrying one
// is refused naming it, one carrying none reads and is written back at 3, and
// version 3's own reads with the spend kept.
func TestAJudgeFallbacksSpendIsVersionThrees(t *testing.T) {
	spent := func() DrainState {
		st := judgedDrain(runner.Sides{})
		st.JudgeFallbacks = []runner.FallbackReceipt{{Role: "drain-judge", Asked: "claude", Reason: runner.ReasonFailed,
			Detail: "killed", Ran: "none", Spent: &runner.Spend{ChargedRuns: 1}}}
		return st
	}
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	writeDrainAt(t, root, spent(), 2)
	_, _, err := readDrain(root)
	rf := mustRefusal(t, err)
	if !strings.Contains(rf.Reason, "schema version 2") || !strings.Contains(rf.Reason, "`spent`") {
		t.Errorf("a version-2 drain state carrying a spend is refused naming it: %q", rf.Reason)
	}
	path := writeDrainAt(t, root, judgedDrain(runner.Sides{}), 2)
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatalf("a version-2 drain state with no spend reads: %v", err)
	}
	if err := writeDrain(root, st); err != nil {
		t.Fatal(err)
	}
	if v := storedDrainVersion(t, path); v != DrainSchemaVersion {
		t.Errorf("the drain writes a version-2 state back at version %d, want %d", v, DrainSchemaVersion)
	}
	writeDrainAt(t, root, spent(), 3)
	st, _, err = readDrain(root)
	if err != nil {
		t.Fatalf("a version-3 drain state carrying a spend reads: %v", err)
	}
	if s := st.JudgeFallbacks[0].Spent; s == nil || s.ChargedRuns != 1 {
		t.Errorf("the judge fallback's spend is kept: %+v", st.JudgeFallbacks)
	}
}

// TestAVersionOneDrainStateReadsAndIsWrittenBackAtTwo: a version-1 drain
// state carrying none of version 2's fields, the sides version 1 already
// recorded kept, reads as the current version, and the drain's next write
// stores it at 2.
func TestAVersionOneDrainStateReadsAndIsWrittenBackAtTwo(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	path := writeDrainAt(t, root, judgedDrain(runner.Sides{ModelAsked: "qwen3-coder", HarnessVersion: "1.2.3"}), 1)
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatalf("a version-1 drain state reads: %v", err)
	}
	if st.SchemaVersion != DrainSchemaVersion {
		t.Errorf("a version-1 drain state is read at version %d, want %d", st.SchemaVersion, DrainSchemaVersion)
	}
	if r := st.Judgements[0].Route; r == nil || r.HarnessVersion != "1.2.3" {
		t.Errorf("the version-1 judge route's sides are kept: %+v", r)
	}
	if err := writeDrain(root, st); err != nil {
		t.Fatal(err)
	}
	if v := storedDrainVersion(t, path); v != DrainSchemaVersion {
		t.Errorf("the drain writes a version-1 state back at version %d, want %d", v, DrainSchemaVersion)
	}
}

// TestAVersionOneDrainStateCarryingAPinIsRefusedNamingIt: version 1 never
// wrote a pin or `local` on the judge's route, so a version-1 file carrying
// one is not a file version 1 wrote; it is refused naming the key and where
// it sits.
func TestAVersionOneDrainStateCarryingAPinIsRefusedNamingIt(t *testing.T) {
	cases := []struct {
		key   string
		sides runner.Sides
	}{
		{"local", runner.Sides{Local: true}},
		{"provider_pin", runner.Sides{ProviderPin: &oracle.Pin{Order: []string{"deepinfra"}, DataCollection: "deny"}}},
		{"pin_config_sha256", runner.Sides{PinConfigSHA256: strings.Repeat("ab", 32)}},
		{"local_pin", runner.Sides{LocalPin: &oracle.LocalPin{Runtime: "ollama", RuntimeVersion: "0.12.6"}}},
		{"local_served", runner.Sides{LocalServed: &oracle.LocalServed{Model: "qwen3-coder"}}},
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			root := repo.Root()
			st := judgedDrain(c.sides)
			writeDrainAt(t, root, st, 1)
			_, _, err := readDrain(root)
			rf := mustRefusal(t, err)
			if !strings.Contains(rf.Reason, "schema version 1") || !strings.Contains(rf.Reason, "`"+c.key+"`") ||
				!strings.Contains(rf.Reason, st.Judgements[0].Issue) {
				t.Errorf("a version-1 drain state carrying %s is refused naming it, the judgement and the version: %q", c.key, rf.Reason)
			}
			if !strings.Contains(rf.Remedy, "restore it or remove it") {
				t.Errorf("the remedy is the drain's own: %q", rf.Remedy)
			}
		})
	}
}

// TestAVersionTwoDrainStateCarryingLocalReads: version 2 writes the judge
// route's pins and `local`, so its own file carrying one reads.
func TestAVersionTwoDrainStateCarryingLocalReads(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	writeDrainAt(t, root, judgedDrain(runner.Sides{Local: true}), 2)
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatalf("a version-2 drain state carrying local reads: %v", err)
	}
	if r := st.Judgements[0].Route; r == nil || !r.Local {
		t.Errorf("the judge route's local is kept: %+v", r)
	}
}

// TestAVersionSixDrainStateIsRefusedAsNewer: a version above 5 is a newer
// abcd's, refused as that before any strict decode (iss-2610230223231843), as
// releases that write 4 refuse a version-5 file.
func TestAVersionSixDrainStateIsRefusedAsNewer(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	writeDrainAt(t, root, bundledDrain(), 6)
	_, _, err := readDrain(root)
	rf := mustRefusal(t, err)
	if !strings.Contains(rf.Reason, "schema version 6, which a newer abcd wrote") || !strings.Contains(rf.Reason, "reads version 5") {
		t.Errorf("a version-6 drain state is refused as a newer abcd's: %q", rf.Reason)
	}
}

// bundledDrain is writtenDrain carrying every field version 4 adds.
func bundledDrain() DrainState {
	st := writtenDrain()
	st.BundleMax = &PaceValue{Value: 5, Layer: "repo", Origin: ".abcd/config.json"}
	st.Forming = &DrainForming{Lead: "iss-2610010000000006", Hold: &DrainHold{Commit: strings.Repeat("a", 40), ForkAllowed: "owner/name"},
		Members: []DrainFormingMember{{Issue: "iss-2610010000000007", Hold: &DrainHold{Commit: strings.Repeat("b", 40),
			ForkUnchecked: "the forge did not answer"}}}}
	minutes := 12.5
	st.Lanes[0].Bundle = []string{"iss-2610010000000009"}
	st.Lanes[0].Split = true
	st.Lanes[1].CIMinutes = &minutes
	return st
}

// TestVersionFoursBundleFieldsReadBack: the drain's cap and forming bundle,
// and a lane's members, split and CI minutes, are version 4's and read back as
// written.
func TestVersionFoursBundleFieldsReadBack(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	writeDrainAt(t, root, bundledDrain(), 4)
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatalf("a version-4 drain state carrying its fields reads: %v", err)
	}
	if b := st.BundleMax; b == nil || b.Value != 5 || b.Layer != "repo" {
		t.Errorf("bundle_max is kept: %+v", b)
	}
	if f := st.Forming; f == nil || f.Lead != "iss-2610010000000006" || len(f.Members) != 1 || f.Members[0].Hold == nil {
		t.Errorf("forming is kept: %+v", f)
	}
	if l := st.Lanes; len(l[0].Bundle) != 1 || !l[0].Split || l[1].CIMinutes == nil || *l[1].CIMinutes != 12.5 {
		t.Errorf("a lane's bundle, split and ci_minutes are kept: %+v", l)
	}
	path := filepath.Join(root, filepath.FromSlash(DrainStateRel))
	plain := writtenDrain()
	if err := writeDrain(root, plain); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"bundle_max", "forming", "\"bundle\"", "split", "ci_minutes"} {
		if strings.Contains(string(data), key) {
			t.Errorf("a drain that set none of version 4's fields writes no %s:\n%s", key, data)
		}
	}
}

// TestAVersionThreeDrainStateReadsWithTheBundledCap: a version-3 file reads as
// a drain that started with the bundled cap, its layer recorded as bundled,
// and is written back at 5.
func TestAVersionThreeDrainStateReadsWithTheBundledCap(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	path := writeDrainAt(t, root, writtenDrain(), 3)
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatalf("a version-3 drain state reads: %v", err)
	}
	if st.SchemaVersion != 5 {
		t.Errorf("a version-3 drain state is read at version %d, want 5", st.SchemaVersion)
	}
	if b := st.BundleMax; b == nil || b.Value != 1 || b.Layer != "bundled" || b.Origin != "bundled" {
		t.Errorf("a version-3 drain started with the bundled cap: %+v", b)
	}
	if err := writeDrain(root, st); err != nil {
		t.Fatal(err)
	}
	if v := storedDrainVersion(t, path); v != 5 {
		t.Errorf("the drain writes a version-3 state back at version %d, want 5", v)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var back struct {
		BundleMax *PaceValue `json:"bundle_max"`
	}
	if err := json.Unmarshal(data, &back); err != nil || back.BundleMax == nil || *back.BundleMax != bundledBundleMax() {
		t.Errorf("the written-back state records the bundled cap (%v):\n%s", err, data)
	}
}

// TestAVersionThreeDrainStateCarryingABundleFieldIsRefusedNamingIt: version 3
// never wrote any of version 4's fields, so a version-3 file carrying one is
// refused naming it.
func TestAVersionThreeDrainStateCarryingABundleFieldIsRefusedNamingIt(t *testing.T) {
	minutes := 3.0
	for key, forge := range map[string]func(*DrainState){
		"bundle_max": func(s *DrainState) { s.BundleMax = &PaceValue{Value: 1, Layer: "bundled", Origin: "bundled"} },
		"forming":    func(s *DrainState) { s.Forming = &DrainForming{Lead: "iss-2610010000000006"} },
		"bundle":     func(s *DrainState) { s.Lanes[1].Bundle = []string{"iss-2610010000000009"} },
		"split":      func(s *DrainState) { s.Lanes[1].Split = true },
		"ci_minutes": func(s *DrainState) { s.Lanes[1].CIMinutes = &minutes },
	} {
		t.Run(key, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			root := repo.Root()
			st := writtenDrain()
			forge(&st)
			writeDrainAt(t, root, st, 3)
			_, _, err := readDrain(root)
			rf := mustRefusal(t, err)
			if !strings.Contains(rf.Reason, "schema version 3") || !strings.Contains(rf.Reason, "`"+key+"`") {
				t.Errorf("a version-3 drain state carrying %s is refused naming it: %q", key, rf.Reason)
			}
		})
	}
}

// TestAVersionFourBundleFieldTheDrainNeverWritesIsRefused: a cap out of range
// or from a layer the resolver never writes, and a member that is no issue id,
// are refused when the state is read.
func TestAVersionFourBundleFieldTheDrainNeverWritesIsRefused(t *testing.T) {
	minutes := -1.0
	for name, forge := range map[string]func(*DrainState){
		"cap of 11":              func(s *DrainState) { s.BundleMax.Value = 11 },
		"cap from a flag":        func(s *DrainState) { s.BundleMax.Layer = "flag" },
		"cap origin bidi":        func(s *DrainState) { s.BundleMax.Origin += "\u202e" },
		"member":                 func(s *DrainState) { s.Lanes[0].Bundle = []string{"not-an-issue"} },
		"forming member":         func(s *DrainState) { s.Forming.Members[0].Issue = "itd-2610010000000007" },
		"forming lead hold":      func(s *DrainState) { s.Forming.Hold.Commit = "abc123" },
		"forming member hold":    func(s *DrainState) { s.Forming.Members[0].Hold.Commit = strings.Repeat("g", 40) },
		"forming fork allowed":   func(s *DrainState) { s.Forming.Hold.ForkAllowed = "owner/name/extra" },
		"forming fork unchecked": func(s *DrainState) { s.Forming.Members[0].Hold.ForkUnchecked = "the forge\u202e did not answer" },
		"ci minutes":             func(s *DrainState) { s.Lanes[1].CIMinutes = &minutes },
	} {
		t.Run(name, func(t *testing.T) {
			repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
			root := repo.Root()
			st := bundledDrain()
			forge(&st)
			writeDrainAt(t, root, st, 4)
			_, _, err := readDrain(root)
			rf := mustRefusal(t, err)
			noRawRune(t, name, rf)
			if !strings.Contains(rf.Reason, "never writes") {
				t.Errorf("%s is refused as one the drain never writes: %q", name, rf.Reason)
			}
		})
	}
}

// filledJudgeRoute is a judge route carrying every string both sides and the
// pins can hold, each clean.
func filledJudgeRoute() runner.Sides {
	return runner.Sides{ModelAsked: "qwen3-coder", ServedModels: []string{"qwen/qwen3-coder"}, ServedProvider: "deepinfra",
		FinishReason: "stop", HarnessVersion: "1.0.0", PromptSHA256: strings.Repeat("a", 64),
		Cost: &oracle.Cost{Amount: 0.01, Unit: "usd"}, Unpinned: []oracle.Unpinned{{Field: "served.model", Reason: "not_verified"}},
		ProviderPin: &oracle.Pin{Order: []string{"deepinfra"}, DataCollection: "deny", Quantizations: []string{"fp8"},
			CanonicalSlug: "qwen/qwen3-coder"},
		PinConfigSHA256: strings.Repeat("b", 64),
		LocalPin: &oracle.LocalPin{Runtime: "ollama", RuntimeVersion: "0.12.6", WeightsSHA256: strings.Repeat("c", 64),
			WeightsFile: "qwen3-coder.gguf", Quantisation: "Q4_K_M", KVCacheType: "f16"},
		LocalServed: &oracle.LocalServed{Model: "qwen3-coder", RuntimeVersion: "0.12.6", WeightsSHA256: strings.Repeat("c", 64),
			Quantisation: "Q4_K_M"},
		Local: true}
}

// TestAJudgeRouteCarryingTextTheDrainNeverWritesIsRefused: every string a
// version-2 judge route stores, both sides and its pins included, reaches
// `drain --json` as stored, so a bidi rune in any of them is refused when the
// drain state is read, as it is in the judge route's asked, ran and model.
func TestAJudgeRouteCarryingTextTheDrainNeverWritesIsRefused(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	writeDrainAt(t, root, judgedDrain(filledJudgeRoute()), 2)
	if _, _, err := readDrain(root); err != nil {
		t.Fatalf("a version-2 judge route carrying clean text reads: %v", err)
	}
	const bad = "\u202e"
	for name, forge := range map[string]func(*runner.Sides){
		"model_asked":                 func(s *runner.Sides) { s.ModelAsked += bad },
		"served_models":               func(s *runner.Sides) { s.ServedModels[0] += bad },
		"served_provider":             func(s *runner.Sides) { s.ServedProvider += bad },
		"finish_reason":               func(s *runner.Sides) { s.FinishReason += bad },
		"harness_version":             func(s *runner.Sides) { s.HarnessVersion += bad },
		"prompt_sha256":               func(s *runner.Sides) { s.PromptSHA256 += bad },
		"cost.unit":                   func(s *runner.Sides) { s.Cost.Unit += bad },
		"unpinned.reason":             func(s *runner.Sides) { s.Unpinned[0].Reason += bad },
		"provider_pin.order":          func(s *runner.Sides) { s.ProviderPin.Order[0] += bad },
		"provider_pin.canonical_slug": func(s *runner.Sides) { s.ProviderPin.CanonicalSlug += bad },
		"pin_config_sha256":           func(s *runner.Sides) { s.PinConfigSHA256 += bad },
		"local_pin.runtime":           func(s *runner.Sides) { s.LocalPin.Runtime += bad },
		"local_pin.weights_file":      func(s *runner.Sides) { s.LocalPin.WeightsFile += bad },
		"local_served.model":          func(s *runner.Sides) { s.LocalServed.Model += bad },
		"local_served.quantisation":   func(s *runner.Sides) { s.LocalServed.Quantisation += bad },
	} {
		t.Run(name, func(t *testing.T) {
			sides := filledJudgeRoute()
			forge(&sides)
			writeDrainAt(t, root, judgedDrain(sides), 2)
			_, _, err := readDrain(root)
			r := mustRefusal(t, err)
			noRawRune(t, name, r)
			if r.Stage != StageDrain || !strings.Contains(r.Reason, "text the drain never writes") {
				t.Fatalf("a judge route's %s carrying U+202E: refused at the drain as text it never writes: %+v", name, r)
			}
		})
	}
}

// TestADrainJudgeRouteOverrideIsVersionTwos: a judge route's `override` (the
// --route text that routed it) is version 2's as the pins are, so a version-1
// drain state carrying it is refused naming it, and a version-2 one reads.
func TestADrainJudgeRouteOverrideIsVersionTwos(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	root := repo.Root()
	st := judgedDrain(runner.Sides{})
	st.Judgements[0].Route.Override = "opencode:qwen3-coder"
	writeDrainAt(t, root, st, 1)
	_, _, err := readDrain(root)
	rf := mustRefusal(t, err)
	if !strings.Contains(rf.Reason, "schema version 1") || !strings.Contains(rf.Reason, "`override`") ||
		!strings.Contains(rf.Reason, st.Judgements[0].Issue) {
		t.Errorf("a version-1 drain state carrying override is refused naming it: %q", rf.Reason)
	}
	writeDrainAt(t, root, st, 2)
	got, _, err := readDrain(root)
	if err != nil {
		t.Fatalf("a version-2 drain state carrying override reads: %v", err)
	}
	if r := got.Judgements[0].Route; r == nil || r.Override != "opencode:qwen3-coder" {
		t.Errorf("the judge route's override is kept: %+v", r)
	}
}
