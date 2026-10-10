package loop

// drive_test.go proves the process driver (spc-2609202134338445 piece 3) and
// the runner's loop half (itd-2609201916056194 phase 2) without a model or a
// real harness. The test binary re-executes itself under the name of the
// harness a test puts on PATH (claude, opencode), and TestMain, seeing
// ABCD_LOOP_FAKE_HARNESS, plays that harness. PATH holds the fake's directory
// alone, so no real harness on the machine can be reached.
//
// The events each fake prints follow what the harnesses document, read on
// 2026-09-30. The claude CLI (code.claude.com/docs/en/headless): print mode's
// stream-json output is one JSON object per line, the system/init event
// carries the session metadata including the model, and the last line is a
// result message with the final text and the session; --bare skips hooks,
// plugins, MCP servers and CLAUDE.md, and never reads OAuth credentials or the
// keychain; dontAsk denies every call that would otherwise prompt while the
// --allowedTools entries run. opencode (opencode.ai/docs/cli): `run --format
// json` prints "raw JSON events", --pure (a global flag) runs "without
// external plugins", --dir, --file and --model provider/model; the page does
// NOT document the events' shape, so the step_start/text/step_finish lines
// with a sessionID and a part are the adapter's assumption, and the live
// shape is owed to a person's check.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
)

const (
	loopFakeEnv    = "ABCD_LOOP_FAKE_HARNESS"
	loopFakeLogEnv = "ABCD_LOOP_FAKE_LOG"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(loopFakeEnv); mode != "" {
		os.Exit(loopFakeHarness(mode))
	}
	os.Exit(m.Run())
}

// loopFakeHarness plays the harness its launch names: opencode's argv starts
// with "run". It writes what it was launched with under the log directory,
// then, by mode, writes the receipt its prompt names (a reviewer's return for
// a reviewer, an implementer's receipt otherwise) or does not.
func loopFakeHarness(mode string) int {
	name := "claude"
	if len(os.Args) > 1 && os.Args[1] == "run" {
		name = "opencode"
	}
	if dir := os.Getenv(loopFakeLogEnv); dir != "" {
		argv, _ := json.Marshal(os.Args[1:])
		_ = os.WriteFile(filepath.Join(dir, name+".argv.json"), argv, 0o600)
	}
	prompt := os.Args[len(os.Args)-1]
	field := func(prefix string) string {
		sc := bufio.NewScanner(strings.NewReader(prompt))
		for sc.Scan() {
			if v, ok := strings.CutPrefix(sc.Text(), prefix); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	if mode == "ratelimit" {
		// The claude CLI cut off by a rate limit mid-work: an edit left
		// uncommitted in the worktree it runs in, a partial receipt, and the
		// rejected rate_limit_event and error result its stream carries.
		_ = os.WriteFile("halfway.txt", []byte("half done\n"), 0o600)
		_ = os.WriteFile(field("Receipt: "), []byte(`{"schema_version": 1, "run_id": "`), 0o600)
		fmt.Println(`{"type":"system","subtype":"init","session_id":"fake-session-3","model":"fake-model"}`)
		fmt.Println(`{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1791000000,"rateLimitType":"five_hour"},"session_id":"fake-session-3"}`)
		fmt.Println(`{"type":"result","subtype":"success","is_error":true,"result":"limit reached","session_id":"fake-session-3"}`)
		return 1
	}
	if mode == "ok" || strings.HasPrefix(mode, "model-") {
		body := "{}\n"
		if role := field("You are the "); strings.HasPrefix(role, RoleRuthless) {
			body = reviewReturn
		}
		_ = os.WriteFile(field("Receipt: "), []byte(body), 0o600)
	}
	if name == "opencode" {
		fmt.Println(`{"type":"step_start","sessionID":"ses_fake2","part":{"type":"step-start"}}`)
		fmt.Println(`{"type":"text","sessionID":"ses_fake2","part":{"type":"text","text":"done"}}`)
		fmt.Println(`{"type":"step_finish","sessionID":"ses_fake2","part":{"type":"step-finish"}}`)
		return 0
	}
	model := "fake-model"
	switch mode {
	case "model-huge":
		model = strings.Repeat("m", 5<<20)
	case "model-ctrl":
		model = "fake\x1b[2Jmodel\r\n"
	}
	init, _ := json.Marshal(map[string]string{"type": "system", "subtype": "init", "session_id": "fake-session-2", "model": model})
	fmt.Println(string(init))
	fmt.Println(`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"fake-session-2"}`)
	return 0
}

// reviewReturn is a ruthless reviewer's return, the same bytes whichever route
// wrote it.
const reviewReturn = "# Review\n\nNothing to fix.\n\n### Verdict\n\nSHIP\n"

// driveEnv is one test's harness set-up: the fakes on a PATH of their own and
// the log they write.
type driveEnv struct{ bin, log string }

func newDriveEnv(t *testing.T, mode string, harnesses ...string) driveEnv {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	e := driveEnv{bin: filepath.Join(root, "bin"), log: filepath.Join(root, "log")}
	for _, d := range []string{e.bin, e.log} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, h := range harnesses {
		if err := os.Symlink(self, filepath.Join(e.bin, h)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", e.bin)
	t.Setenv(loopFakeEnv, mode)
	t.Setenv(loopFakeLogEnv, e.log)
	return e
}

func (e driveEnv) argv(t *testing.T, harness string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(e.log, harness+".argv.json"))
	if err != nil {
		t.Fatalf("the %s fake was not launched: %v", harness, err)
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// runnerConfig reads a runner configuration from a machine and a repository
// config file written for the test.
func runnerConfig(t *testing.T, machine, repo string) *runner.Config {
	t.Helper()
	r := layered.Roots{Repo: t.TempDir(), Home: t.TempDir()}
	for path, body := range map[string]string{
		abcdhome.Path(r.Home, "config.json"):          machine,
		filepath.Join(r.Repo, ".abcd", "config.json"): repo,
	} {
		if body == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c, err := runner.Load(r)
	if err != nil {
		t.Fatalf("runner config: %v", err)
	}
	return c
}

// memTranscripts is a transcript store that keeps what it is handed.
type memTranscripts struct{ stored []string }

func (m *memTranscripts) Store(name string, req runner.Request, _ runner.Answer, raw []byte) error {
	m.stored = append(m.stored, name+" "+req.Role+" "+fmt.Sprint(len(raw) > 0))
	return nil
}

// driveSteps is fakeSteps with a worktree the runner can run in and an
// implement stage whose verifier reads the receipt the agent wrote.
func driveSteps(worktree string) Stages {
	return Stages{
		{Name: StageWorktree, Piece: 6, Run: func(c Context, l *Lane) (Outcome, error) {
			l.Branch, l.Worktree = "build/"+l.ID, worktree
			return Outcome{Note: "worktree done"}, nil
		}},
		{Name: StageBrief, Piece: 5, Run: func(c Context, l *Lane) (Outcome, error) {
			l.Brief = RunRelDir + "/brief.md"
			return Outcome{Note: "brief done"}, nil
		}},
		{Name: StageImplement, Piece: 7,
			Run: func(c Context, l *Lane) (Outcome, error) {
				return Outcome{Await: &Await{Role: RoleImplementer, Brief: filepath.Join(c.RepoRoot, l.Brief),
					Receipt: filepath.Join(c.RepoRoot, "receipt-"+l.ID+".json")}}, nil
			},
			Verify: func(c Context, l *Lane, receipt string) error {
				raw, err := os.ReadFile(receipt)
				if err != nil || strings.TrimSpace(string(raw)) != "{}" {
					return refuse("receipt", "", l.ID, "no receipt the contract accepts at "+receipt, "write it, then hand it back")
				}
				l.Receipts = append(l.Receipts, ReceiptRecord{Role: RoleImplementer, Receipt: receipt})
				return nil
			}},
		{Name: StageValidate, Piece: 8, Run: func(Context, *Lane) (Outcome, error) { return Outcome{Note: "validate done"}, nil }},
		{Name: StageLand, Piece: 9, Run: func(Context, *Lane) (Outcome, error) { return Outcome{Note: "land done"}, nil }},
	}
}

// startToImplement starts a run and takes its lane to the implement stage.
func startToImplement(t *testing.T) (root, id string, steps Stages, o Options) {
	t.Helper()
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	wt := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), "brief.md"), []byte("# brief\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	steps = driveSteps(wt)
	clock := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	o = Options{Now: func() time.Time { return clock }}
	for range 2 {
		if _, err := advance(repo.Root(), start.RunID, steps, o); err != nil {
			t.Fatal(err)
		}
	}
	return repo.Root(), start.RunID, steps, o
}

// TestAnUnsetRouteLeavesTheAwaitAndTheStateByteIdentical is criterion 2: with
// no runner configured, the driven step hands the host exactly what a plain
// step hands it, and writes exactly the same state.
func TestAnUnsetRouteLeavesTheAwaitAndTheStateByteIdentical(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	newDriveEnv(t, "ok", "claude", "opencode")
	path := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
	before := stateBytes(t, root, id)

	plain, err := advance(root, id, steps, o)
	if err != nil {
		t.Fatal(err)
	}
	plainState := stateBytes(t, root, id)
	if err := os.WriteFile(path, before, 0o600); err != nil {
		t.Fatal(err)
	}
	store := &memTranscripts{}
	driven, err := Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, "", ""), Transcripts: store})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plain, driven) {
		t.Fatalf("an unset route changed the step's result:\nplain  %+v\ndriven %+v", plain, driven)
	}
	if !bytes.Equal(plainState, stateBytes(t, root, id)) {
		t.Fatalf("an unset route changed the state:\nplain  %s\ndriven %s", plainState, stateBytes(t, root, id))
	}
	if len(store.stored) != 0 {
		t.Fatalf("an unset route launched a runner: %v", store.stored)
	}
}

// TestARoutedRoleRunsThroughItsRunner is criterion 1 and the loop's criterion
// 9: a role routed to a runner is started by the loop itself with the brief
// and the receipt path the host would get, its receipt is verified by the
// stage's own verifier, its transcript is stored, and the record names the
// runner that ran it.
func TestARoutedRoleRunsThroughItsRunner(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	env := newDriveEnv(t, "ok", "claude")
	store := &memTranscripts{}
	cfg := runnerConfig(t, `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`, "")
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: store})
	if err != nil {
		t.Fatal(err)
	}
	if res.PerformedStage != StageImplement || res.Stage != StageValidate || res.Awaiting != nil {
		t.Fatalf("the runner's verified receipt completes the stage: %+v", res)
	}
	if res.Route == nil || res.Route.Asked != runner.Claude || res.Route.Ran != runner.Claude || res.Route.Model != "fake-model" {
		t.Fatalf("the result names the route that ran: %+v", res.Route)
	}
	argv := strings.Join(env.argv(t, "claude"), "\n")
	for _, want := range []string{"--bare", "--allowedTools=" + strings.Join(toolsFor(RoleImplementer), ","),
		"Brief: " + filepath.Join(root, filepath.FromSlash(RunRelDir), "brief.md")} {
		if !strings.Contains(argv, want) {
			t.Fatalf("the launch carries %q:\n%s", want, argv)
		}
	}
	if len(store.stored) != 1 || store.stored[0] != "claude implementer true" {
		t.Fatalf("the transcript lands in abcd's store: %v", store.stored)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	rs := st.Lanes[0].Receipts
	if len(rs) != 1 || rs[0].Route == nil || rs[0].Route.Ran != runner.Claude {
		t.Fatalf("the verified receipt names the route that ran it: %+v", rs)
	}
	named := false
	for _, e := range st.Record {
		if e.Stage == StageRunner && strings.Contains(e.Note, "claude") {
			named = true
		}
	}
	if !named || len(st.Fallbacks) != 0 {
		t.Fatalf("the run record names the runner and records no fallback: %+v %+v", st.Record, st.Fallbacks)
	}
}

// TestARunnerThatCannotRunTheRoleFallsBackAndIsRecorded is criterion 3: an
// absent runner, and a runner whose answer the stage's verifier refuses, each
// hand the role to the host, which is told what to start as before, and each
// leaves one fallback receipt naming the role, the runner, the reason and the
// route that ran.
func TestARunnerThatCannotRunTheRoleFallsBackAndIsRecorded(t *testing.T) {
	for _, tc := range []struct {
		name, mode, route string
		harnesses         []string
		reason            runner.Reason
	}{
		{"absent", "ok", "opencode", nil, runner.ReasonAbsent},
		{"invalid", "noreceipt", "claude", []string{"claude"}, runner.ReasonInvalid},
		// A model past its bound or carrying control bytes is a refusal of
		// the route, never written into the state: a 5 MiB one there would
		// push state.json past its read bound and brick the run.
		{"huge-model", "model-huge", "claude", []string{"claude"}, runner.ReasonUnparsable},
		{"control-model", "model-ctrl", "claude", []string{"claude"}, runner.ReasonUnparsable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, id, steps, o := startToImplement(t)
			newDriveEnv(t, tc.mode, tc.harnesses...)
			cfg := runnerConfig(t, `{"roles":{"implementer":{"runner":"`+tc.route+`"}},"runner":{"claude":{},"opencode":{}}}`, "")
			res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
			if err != nil {
				t.Fatal(err)
			}
			if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer || res.PerformedStage != "" {
				t.Fatalf("the host is handed the role: %+v", res)
			}
			fb := res.Fallback
			if fb == nil || fb.Role != RoleImplementer || fb.Asked != tc.route || fb.Reason != tc.reason || fb.Ran != runner.Host {
				t.Fatalf("the result names the fallback: %+v", fb)
			}
			st, err := ReadState(root, id)
			if err != nil {
				t.Fatal(err)
			}
			if len(st.Fallbacks) != 1 || st.Fallbacks[0] != *fb {
				t.Fatalf("the state carries the one fallback receipt: %+v", st.Fallbacks)
			}
			if len(st.Lanes[0].Awaits) == 0 {
				t.Fatal("the lane still awaits the host's receipt")
			}
			if len(st.Lanes[0].Receipts) != 0 {
				t.Fatalf("a refused route verified no receipt: %+v", st.Lanes[0].Receipts)
			}
			rec, err := ReadRecord(root, id)
			if err != nil {
				t.Fatal(err)
			}
			c := rec.FallbackCounts
			if c.Total != 1 || c.ByRunner[tc.route] != 1 || c.ByRole[RoleImplementer] != 1 {
				t.Fatalf("the run record counts the fallback per runner and per role (criterion 4): %+v", c)
			}
		})
	}
}

// TestAReviewThroughARunnerDiffersFromAHostReviewOnlyInItsRoute is criterion
// 7's structural half: one review, returned with the same bytes once by the
// host and once by the opencode runner, is recorded by the stage's own
// verifier with records that differ in the route alone.
func TestAReviewThroughARunnerDiffersFromAHostReviewOnlyInItsRoute(t *testing.T) {
	review := func(t *testing.T, routed bool) ValidatorRun {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		start, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		wt := t.TempDir()
		steps := driveSteps(wt)
		const briefRel, returnRel = "review/brief.md", "review/return.md"
		if err := os.MkdirAll(filepath.Join(repo.Root(), "review"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo.Root(), "review", "brief.md"), []byte("# review\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		steps[2] = StageDef{Name: StageImplement, Piece: 7,
			Run: func(c Context, l *Lane) (Outcome, error) {
				if len(l.Validation) == 0 {
					l.Validation = []ValidationRound{{Round: 1, HeadSHA: strings.Repeat("a", 40),
						Validators: []ValidatorRun{{Role: RoleRuthless, Brief: briefRel, Return: returnRel}}}}
				}
				return Outcome{Await: &Await{Role: RoleRuthless, Brief: briefRel, Receipt: returnRel}}, nil
			},
			Verify: func(c Context, l *Lane, receipt string) error {
				c.Await.Role = RoleRuthless
				return verifyValidation(c, l, receipt)
			}}
		o := Options{Now: func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) }}
		for range 2 {
			if _, err := advance(repo.Root(), start.RunID, steps, o); err != nil {
				t.Fatal(err)
			}
		}
		if routed {
			newDriveEnv(t, "ok", "opencode")
			cfg := runnerConfig(t, `{"roles":{"ruthless-reviewer":{"runner":"opencode"}},"runner":{"opencode":{}}}`, "")
			res, err := Drive(context.Background(), repo.Root(), start.RunID, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
			if err != nil || res.PerformedStage != StageImplement {
				t.Fatalf("the opencode review is verified: %+v %v", res, err)
			}
		} else {
			res, err := advance(repo.Root(), start.RunID, steps, o)
			if err != nil || res.Awaiting == nil {
				t.Fatalf("the host is handed the review: %+v %v", res, err)
			}
			if err := os.WriteFile(filepath.Join(repo.Root(), returnRel), []byte(reviewReturn), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Receipt(repo.Root(), start.RunID, returnRel, steps, o); err != nil {
				t.Fatal(err)
			}
		}
		st, err := ReadState(repo.Root(), start.RunID)
		if err != nil {
			t.Fatal(err)
		}
		return st.Lanes[0].Validation[0].Validators[0]
	}
	host, routed := review(t, false), review(t, true)
	if host.Route != nil {
		t.Fatalf("a host-run review names no runner route: %+v", host.Route)
	}
	if routed.Route == nil || routed.Route.Asked != runner.OpenCode || routed.Route.Ran != runner.OpenCode {
		t.Fatalf("the opencode review names its route: %+v", routed.Route)
	}
	if host.Verdict != "SHIP" {
		t.Fatalf("the host review is recorded: %+v", host)
	}
	routed.Route = nil
	if !reflect.DeepEqual(host, routed) {
		t.Fatalf("the two reviews differ beyond the route:\nhost   %+v\nrouted %+v", host, routed)
	}
}

// TestAVersion7StateCarryingARunnersRecordIsRefused: version 7 never wrote a
// fallback or a route, so a file of that version carrying one is refused, and
// one without either is read and written back at the current version.
func TestAVersion7StateCarryingARunnersRecordIsRefused(t *testing.T) {
	root, id, _, _ := startToImplement(t)
	path := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
	current := stateBytes(t, root, id)
	cur := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	old := strings.Replace(string(current), cur, `"schema_version": 7,`, 1)
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	if st, err := ReadState(root, id); err != nil || st.SchemaVersion != SchemaVersion {
		t.Fatalf("a version-7 file is read as the current version: %v", err)
	}
	carrying := strings.Replace(old, `"record": [`, `"fallbacks": [{"at":"2026-09-30T12:00:00Z","role":"implementer","asked":"claude","reason":"absent","detail":"x","ran":"host"}],
  "record": [`, 1)
	if err := os.WriteFile(path, []byte(carrying), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(root, id); err == nil {
		t.Fatal("a version-7 file carrying a fallback must be refused")
	}
}

// TestRoleToolsFollowTheAgentDefinitions: a reviewer run through a runner is
// granted the tools its agent definition names, plus Write for the return its
// brief tells it to write, and nothing its definition does not name.
func TestRoleToolsFollowTheAgentDefinitions(t *testing.T) {
	for _, role := range []string{RoleRuthless, RoleSecurity} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "agents", role+".md"))
		if err != nil {
			t.Fatal(err)
		}
		var def []string
		for _, ln := range strings.Split(string(raw), "\n") {
			if v, ok := strings.CutPrefix(ln, "tools:"); ok {
				for _, tool := range strings.Split(v, ",") {
					def = append(def, strings.TrimSpace(tool))
				}
				break
			}
		}
		want := append(def, "Write")
		if got := toolsFor(role); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: tools = %v, want its definition's %v plus Write", role, got, def)
		}
	}
	if got := toolsFor("scribe"); got != nil {
		t.Fatalf("a role the loop does not start is granted nothing: %v", got)
	}
}

// TestAStepThatReTellsAnAwaitStartsNoRunner: once a runner's fallback has
// handed the host the role, stepping again re-tells the await; it neither
// starts the runner again nor records a second fallback.
func TestAStepThatReTellsAnAwaitStartsNoRunner(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	newDriveEnv(t, "ok")
	cfg := runnerConfig(t, `{"roles":{"implementer":{"runner":"opencode"}},"runner":{"opencode":{}}}`, "")
	for range 2 {
		if _, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}}); err != nil {
			t.Fatal(err)
		}
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Fallbacks) != 1 {
		t.Fatalf("one fallback for one hand-out, got %d", len(st.Fallbacks))
	}
}
