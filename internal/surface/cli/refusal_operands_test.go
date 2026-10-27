package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/core/interview"
	"github.com/intentdriven/abcd/internal/core/question"
)

// refusalCase is one invocation whose refusal must name what was wrong — by
// flag, by position or by role — without repeating what was passed.
type refusalCase struct {
	name string
	// setup prepares a scratch directory (and the working directory, when the
	// verb reads one) for one run and returns it.
	setup func(t *testing.T) string
	args  func(dir, m string) []string
	stdin func(dir, m string) string
	// hide is the text that must not come back; the marker itself when nil.
	hide  func(m string) string
	code  int
	names []string // each must appear in the refusal
	// plainOnly marks a hook: it answers on stderr and exits 0, so it has no
	// --json error object to read.
	plainOnly bool
}

// digitsMarker is a run-time marker for a flag that takes a number.
func digitsMarker() string {
	return fmt.Sprintf("9%08d", time.Now().UnixNano()%100000000)
}

func scratchCwd(t *testing.T) string {
	t.Helper()
	dir := realPath(t, t.TempDir())
	t.Chdir(dir)
	return dir
}

func scratchRepo(t *testing.T) string {
	t.Helper()
	dir := realPath(t, t.TempDir())
	gitInitAt(t, dir)
	t.Chdir(dir)
	return dir
}

func scratchRepoWithCommit(t *testing.T) string {
	t.Helper()
	dir := scratchRepo(t)
	gitCommitAt(t, dir, "root")
	return dir
}

func writeScratch(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runRefusal runs the front door with stdin and a scratch home.
func runRefusal(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut strings.Builder
	code := run(args, strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

func checkRefusalCase(t *testing.T, c refusalCase) {
	t.Helper()
	forms := [][]string{nil}
	if !c.plainOnly {
		forms = append(forms, []string{"--json"})
	}
	for _, form := range forms {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("ABCD_PLUGIN_ROOT", "")
		t.Setenv("CLAUDE_PLUGIN_ROOT", "")
		t.Setenv("ABCD_BIN_TARGET", "")
		dir := c.setup(t)
		m := usageMarker()
		hide := m
		if c.hide != nil {
			hide = c.hide(m)
		}
		args := append(append([]string(nil), form...), c.args(dir, m)...)
		stdin := ""
		if c.stdin != nil {
			stdin = c.stdin(dir, m)
		}
		label := c.name
		if form != nil {
			label += " --json"
		}
		code, out, errOut := runRefusal(t, stdin, args...)
		if code != c.code {
			t.Errorf("%s: exit %d, want %d\nstdout: %s\nstderr: %s", label, code, c.code, out, errOut)
		}
		if repeats(out, hide) || repeats(errOut, hide) {
			t.Errorf("%s: a refusal repeated what was passed\nstdout: %s\nstderr: %s", label, out, errOut)
		}
		where := errOut
		if form != nil {
			var env struct {
				Abcd     string `json:"abcd"`
				Error    string `json:"error"`
				ExitCode int    `json:"exit_code"`
			}
			if err := json.Unmarshal([]byte(out), &env); err != nil {
				t.Errorf("%s: stdout is not the error object: %v\n%s", label, err, out)
				continue
			}
			if env.Abcd != "error" || env.ExitCode != c.code {
				t.Errorf("%s: object = %+v, want abcd=error exit_code=%d", label, env, c.code)
			}
			where = env.Error
		}
		for _, want := range c.names {
			if !strings.Contains(where, want) {
				t.Errorf("%s: the refusal does not name %q\n%s", label, want, where)
			}
		}
	}
}

// TestRefusalsNeverRepeatWhatWasPassed passes a run-time marker through every
// refusal that reads an operand, a flag value, an answers-file value or a path
// the caller typed, and holds that the marker reaches neither stream, in the
// plain and the --json form, while the refusal still names what to correct.
func TestRefusalsNeverRepeatWhatWasPassed(t *testing.T) {
	cases := []refusalCase{
		// Free-text create verbs read a lone word as a sub-verb.
		{name: "capture lone word", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"capture", m} }, code: 2,
			names: []string{"unknown capture subcommand", "first argument"}},
		{name: "capture far miss", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"capture", m, "iss-2608300000000004"} },
			code: 2, names: []string{"unknown capture subcommand", "the sub-verbs are"}},
		{name: "capture near miss", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"capture", "resolv" + m[2:4]} },
			hide: func(m string) string { return "resolv" + m[2:4] },
			code: 2, names: []string{"unknown capture subcommand", "did you mean"}},
		{name: "intent lone word", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"intent", m} }, code: 2,
			names: []string{"unknown intent subcommand", "first argument"}},
		{name: "intent far miss", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"intent", m, "itd-1"} },
			code: 2, names: []string{"unknown intent subcommand", "the sub-verbs are"}},
		// Closed-set flag values.
		{name: "capture --severity", setup: scratchRepo,
			args: func(_, m string) []string {
				return []string{"capture", "a finding with a guessed value", "--severity", m, "--remedy", "fix it"}
			}, code: 2, names: []string{"--severity", "critical"}},
		{name: "capture --category", setup: scratchRepo,
			args: func(_, m string) []string {
				return []string{"capture", "a finding with a guessed value", "--category", m, "--remedy", "fix it"}
			}, code: 2, names: []string{"--category"}},
		{name: "capture --source", setup: scratchRepo,
			args: func(_, m string) []string {
				return []string{"capture", "a finding with a guessed value", "--source", m, "--remedy", "fix it"}
			}, code: 2, names: []string{"--source"}},
		{name: "ahoy install --install-tool", setup: scratchCwd,
			args: func(_, m string) []string { return []string{"ahoy", "install", "--install-tool", m, "--refuse-adopt"} },
			code: 2, names: append([]string{"--install-tool"}, ahoy.DependencyTools...)},
		{name: "board --view", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"--view", m} }, code: 2,
			names: []string{"--view", viewProduct, viewFacilitator}},
		{name: "board --format", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"--format", m} }, code: 2,
			names: []string{"--format", formText, formMarkdown}},
		{name: "implement log --field shape", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"implement", "log", "agent_start", "--field", m} },
			code: 2, names: []string{"--field", "key=value"}},
		{name: "implement log --field twice", setup: scratchRepo,
			args: func(_, m string) []string {
				return []string{"implement", "log", "agent_start", "--field", m + "=1", "--field", m + "=2"}
			}, code: 2, names: []string{"--field", "twice"}},
		{name: "implement report --date", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"implement", "report", "--date", m} }, code: 2,
			names: []string{"--date", "YYYY-MM-DD"}},
		{name: "implement report --log", setup: scratchRepo,
			args: func(dir, m string) []string {
				return []string{"implement", "report", "--log", filepath.Join(dir, m)}
			}, code: 2, names: []string{"--log"}},
		{name: "implement load --site", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"implement", "load", "--site", m} }, code: 2,
			names: []string{"--site"}},
		{name: "implement outage probe --model", setup: scratchRepo,
			args: func(_, m string) []string {
				return []string{"implement", "outage", "probe", "--session", "s1", "--model", m}
			}, code: 2, names: []string{"--model", "ok", "fail"}},
		{name: "intent interview id", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"intent", "interview", m} }, code: 2,
			names: []string{"itd-N"}},
		{name: "history discard confirm", setup: scratchRepoWithCommit,
			args: func(_, m string) []string { return []string{"history", "discard", m} }, code: 1,
			names: []string{"--yes", "first argument"}},
		{name: "history --root-sha", setup: scratchCwd,
			args: func(_, m string) []string {
				return []string{"history", "discard", "x", "--root-sha", m}
			}, code: 1, names: []string{"--root-sha"}},
		{name: "history capture --all --kind", setup: scratchRepoWithCommit,
			args: func(_, m string) []string {
				return []string{"history", "capture", "--all", "--session", "s1", "--kind", m}
			}, code: 1, names: []string{"--kind"}},
		{name: "history capture --all --tool", setup: scratchRepoWithCommit,
			args: func(_, m string) []string {
				return []string{"history", "capture", "--all", "--session", "s1", "--tool", m}
			}, code: 1, names: []string{"--tool"}},
		{name: "history ingest --into", setup: scratchCwd,
			args: func(dir, m string) []string {
				into := filepath.Join(dir, m)
				_ = os.MkdirAll(into, 0o755)
				return []string{"history", "ingest", "--into", into, dir}
			}, code: 1, names: []string{"--into"}},
		// The setup interview's answers file.
		{name: "ahoy install --answers value", setup: scratchRepo,
			args: func(dir, m string) []string {
				p := filepath.Join(dir, "answers.json")
				b, _ := json.Marshal(map[string]any{
					"schema_version": interview.SchemaVersion,
					"interview":      ahoy.SetupInterview,
					"answers":        []map[string]string{{"id": "adopt", "value": m}},
				})
				_ = os.WriteFile(p, b, 0o644)
				return []string{"ahoy", "install", "--answers", p}
			}, code: 2, names: []string{"the answers file", "adopt", "yes"}},
		{name: "ahoy install --answers id twice", setup: scratchRepo,
			args: func(dir, m string) []string {
				return []string{"ahoy", "install", "--answers", setupAnswersFileFor(t, dir, ahoy.SetupInterview,
					[]map[string]string{{"id": m, "value": "yes"}, {"id": m, "value": "no"}})}
			}, code: 2, names: []string{"answer 2", "earlier answer"}},
		{name: "ahoy install --answers answered_in", setup: scratchRepo,
			args: func(dir, m string) []string {
				return []string{"ahoy", "install", "--answers", setupAnswersFileFor(t, dir, ahoy.SetupInterview,
					[]map[string]string{{"id": "adopt", "value": "yes", "answered_in": m}})}
			}, code: 2, names: []string{"answer 1", "answered_in"}},
		{name: "ahoy install --answers interview", setup: scratchRepo,
			args: func(dir, m string) []string {
				return []string{"ahoy", "install", "--answers", setupAnswersFileFor(t, dir, m, nil)}
			}, code: 2, names: []string{"another interview", ahoy.SetupInterview}},
		{name: "ahoy install --answers unknown key", setup: scratchRepo,
			args: func(dir, m string) []string {
				return []string{"ahoy", "install", "--answers", setupAnswersFileFor(t, dir, ahoy.SetupInterview,
					[]map[string]string{{"id": "adopt", "value": "yes", m: "x"}})}
			}, code: 2, names: []string{"does not define"}},
		// The intent pre-pass: its id operand and its findings file.
		{name: "intent prepass id", setup: scratchRepo,
			args: func(_, m string) []string { return []string{"intent", "prepass", m} }, code: 2,
			names: []string{"first argument", "itd-N"}},
		{name: "intent prepass absent id", setup: scratchRepo,
			args: func(_, _ string) []string { return []string{"intent", "prepass", "itd-" + portMarker} },
			hide: func(string) string { return portMarker }, code: 2, names: []string{"first argument"}},
		{name: "intent prepass --findings-json missing", setup: scratchRepo,
			args: func(dir, m string) []string {
				return []string{"intent", "prepass", "itd-1", "--findings-json", filepath.Join(dir, m)}
			}, code: 2, names: []string{"--findings-json"}},
		{name: "intent prepass --findings-json directory", setup: scratchRepo,
			args: func(dir, m string) []string {
				p := filepath.Join(dir, m)
				_ = os.MkdirAll(p, 0o755)
				return []string{"intent", "prepass", "itd-1", "--findings-json", p}
			}, code: 2, names: []string{"the findings file"}},
		// Paths the caller typed.
		{name: "lint docs --config", setup: scratchRepo,
			args: func(dir, m string) []string {
				return []string{"lint", "docs", "--config", filepath.Join(dir, m+".json")}
			}, code: 2, names: []string{"--config"}},
		{name: "docs cite refresh --config", setup: scratchRepo,
			args: func(dir, m string) []string {
				return []string{"docs", "cite", "refresh", "--config", filepath.Join(dir, m+".json")}
			}, code: 2, names: []string{"--config"}},
		{name: "disembark probe", setup: scratchCwd,
			args: func(dir, m string) []string { return []string{"disembark", "probe", filepath.Join(dir, m)} },
			code: 2, names: []string{"not a directory"}},
		{name: "disembark plan", setup: scratchCwd,
			args: func(dir, m string) []string { return []string{"disembark", "plan", filepath.Join(dir, m)} },
			code: 2, names: []string{"not a directory"}},
		{name: "disembark pack", setup: scratchCwd,
			args: func(dir, m string) []string {
				return []string{"disembark", "pack", filepath.Join(dir, m), filepath.Join(dir, "dest")}
			}, code: 2, names: []string{"not a directory"}},
		{name: "disembark coverage unreadable", setup: scratchCwd,
			args: func(dir, m string) []string {
				return []string{"disembark", "coverage", filepath.Join(dir, m+".json")}
			}, code: 2, names: []string{"report 1"}},
		{name: "disembark coverage not json", setup: scratchCwd,
			args: func(dir, m string) []string {
				p := filepath.Join(dir, m+".json")
				_ = os.WriteFile(p, []byte("[]"), 0o644)
				return []string{"disembark", "coverage", p}
			}, code: 2, names: []string{"report 1", "not a coverage report"}},
		{name: "disembark coverage no schema", setup: scratchCwd,
			args: func(dir, m string) []string {
				p := filepath.Join(dir, m+".json")
				_ = os.WriteFile(p, []byte("{}"), 0o644)
				return []string{"disembark", "coverage", p}
			}, code: 2, names: []string{"report 1", "schema_version"}},
		{name: "disembark coverage too new", setup: scratchCwd,
			args: func(dir, m string) []string {
				p := filepath.Join(dir, m+".json")
				_ = os.WriteFile(p, []byte(`{"schema_version": 999}`), 0o644)
				return []string{"disembark", "coverage", p}
			}, code: 2, names: []string{"report 1"}},
		{name: "lint --root", setup: scratchCwd,
			args: func(dir, m string) []string { return []string{"lint", "--root", filepath.Join(dir, m)} },
			code: 2, names: []string{"--root", "not a directory"}},
		{name: "lint outbound directory", setup: scratchCwd,
			args: func(dir, m string) []string {
				p := filepath.Join(dir, m)
				_ = os.MkdirAll(p, 0o755)
				return []string{"lint", "outbound", p}
			}, code: 2, names: []string{"directory"}},
		{name: "lint outbound empty", setup: scratchCwd,
			args: func(dir, m string) []string {
				p := filepath.Join(dir, m)
				_ = os.WriteFile(p, nil, 0o644)
				return []string{"lint", "outbound", p}
			}, code: 2, names: []string{"empty"}},
		{name: "lint outbound missing", setup: scratchCwd,
			args: func(dir, m string) []string { return []string{"lint", "outbound", filepath.Join(dir, m)} },
			code: 2, names: []string{"cannot read the artefact"}},
		{name: "lint outbound --root", setup: scratchCwd,
			args: func(dir, m string) []string {
				return []string{"lint", "outbound", "--root", filepath.Join(dir, m), "-"}
			}, stdin: func(_, _ string) string { return "a plain line\n" },
			code: 2, names: []string{"--root", "not a directory"}},
		{name: "library --corpus", setup: scratchRepoWithCommit,
			args: func(dir, m string) []string {
				return []string{"library", "ledger", "--corpus", filepath.Join(dir, m)}
			}, code: 3, names: []string{"--corpus", "library init"}},
		{name: "ahoy credential name", setup: scratchCwd,
			args: func(_, m string) []string { return []string{"ahoy", "credential", m} }, code: 2,
			names: []string{"abcd ahoy credential"}},
		{name: "dashboard start --port", setup: scratchCwd,
			args: func(_, _ string) []string { return []string{"dashboard", "start", "--port", portMarker} },
			hide: func(string) string { return portMarker }, code: 2, names: []string{"--port", "is not a port"}},
		// A hook reads the transcript path from its payload.
		{name: "hook session-end transcript path", setup: scratchRepoWithCommit, plainOnly: true,
			args: func(_, _ string) []string { return []string{"hook", "session-end"} },
			stdin: func(dir, m string) string {
				return endPayload(t, "s1", dir, filepath.Join(dir, m))
			}, code: 0, names: []string{"the transcript path"}},
	}
	for _, c := range cases {
		checkRefusalCase(t, c)
	}
}

// portMarker is the run's number for --port: a flag that takes a number
// cannot carry the letter marker. An intent id's digits reuse it.
var portMarker = digitsMarker()

// setupAnswersFileFor writes an answers file for the named interview into dir and
// returns its path.
func setupAnswersFileFor(t *testing.T, dir, name string, answers []map[string]string) string {
	t.Helper()
	if answers == nil {
		answers = []map[string]string{}
	}
	b, err := json.Marshal(map[string]any{
		"schema_version": interview.SchemaVersion,
		"interview":      name,
		"answers":        answers,
	})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "answers.json")
	writeScratch(t, p, string(b))
	return p
}

// TestRefusalHelpersNeverRepeatWhatWasPassed covers the refusals that only a
// running install, interview, hook or release cut reaches, by calling the
// helper each of those paths goes through with the same run-time marker.
func TestRefusalHelpersNeverRepeatWhatWasPassed(t *testing.T) {
	m := usageMarker()
	dir := realPath(t, t.TempDir())
	check := func(name string, err error, names ...string) {
		t.Helper()
		if err == nil {
			t.Errorf("%s: no refusal", name)
			return
		}
		if repeats(err.Error(), m) {
			t.Errorf("%s: %q repeats what was passed", name, err)
		}
		for _, want := range names {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: %q does not name %q", name, err, want)
			}
		}
	}

	missing := filepath.Join(dir, m)
	sub := filepath.Join(dir, m+"-dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(dir, m+"-empty")
	writeScratch(t, empty, "")
	big := filepath.Join(dir, m+"-big")
	writeScratch(t, big, "")
	if err := os.Truncate(big, maxTranscriptBytes+1); err != nil {
		t.Fatal(err)
	}

	_, err := readTranscript(missing)
	check("transcript missing", err, "the transcript path")
	_, err = readTranscript(sub)
	check("transcript not regular", err, "the transcript path", "not a readable regular file")
	_, err = readTranscript(empty)
	check("transcript empty", err, "the transcript path", "empty")
	_, err = readTranscript(big)
	check("transcript too big", err, "the transcript path", "over the", "cap")

	_, err = readGuardedOperand(missing, 8)
	check("operand missing", err, "the operand file")
	_, err = readGuardedOperand(sub, 8)
	check("operand not regular", err, "the operand file", "not a readable regular file")
	_, err = readGuardedOperand(big, 8)
	check("operand too big", err, "the operand file", "-byte cap")

	_, err = writeReconstruction(missing, history.Reconstruction{})
	check("reconstruct --out", err, "--out")

	repo := filepath.Join(dir, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	writeScratch(t, filepath.Join(repo, "CHANGELOG.md"), "# Changelog\n\n## [Unreleased]\n\n## [0.4.0] - 2026-07-21\n\n- a line\n")
	_, err = archiveRenderRequest(repo, "v"+m)
	check("archive --tag", err, "--tag", "v0.4.0")

	q := ahoy.SetupConfirmQuestion(0, "Adopt this repository?")
	q.ID = "adopt"
	file := interview.Answers{Answers: []interview.FileAnswer{{ID: q.ID, Value: m}}}
	_, err = fileAnswerer(io.Discard, "abcd intent interview", file, interview.Terminal, true)(question.Ask{Questions: []question.Question{q}})
	check("written interview answers value", err, "the answers file", "adopt")

	p := &answersPrompter{setupQuestions: setupQuestions{w: io.Discard}, file: file, ascii: true}
	func() {
		defer func() {
			r := recover()
			stop, ok := r.(*setupStop)
			if !ok {
				t.Errorf("setup answers value: recovered %v, want a setup stop", r)
				return
			}
			check("setup answers value", errors.New(stop.msg), "the answers file", "adopt")
		}()
		p.answer(q, "pass --yes")
	}()
}

// tokenMarker is a run-time argument shaped like an access token: mixed case,
// digits and an underscore, the shape a variable a script expanded carries.
func tokenMarker() string {
	return "Tk_" + strings.ToUpper(usageMarker()[2:8]) + usageMarker()[8:] + "9x"
}

// TestBoundaryWithholdsATokenGivenAsARecordID passes a token-shaped argument
// where three verbs take a record id or a release. Each verb's own refusal
// quotes what it was given; the boundary pass in Run withholds it from stderr
// and from the --json error object, and the refusal still says what shape the
// argument takes.
func TestBoundaryWithholdsATokenGivenAsARecordID(t *testing.T) {
	cases := []refusalCase{
		{
			name:  "spec close",
			setup: scratchRepoWithCommit,
			args:  func(_, m string) []string { return []string{"spec", "close", m} },
			code:  2,
			names: []string{"spc-", "not repeated here"},
		},
		{
			name:  "reserve",
			setup: scratchRepoWithCommit,
			args:  func(_, m string) []string { return []string{"reserve", m, "--footprint", "x"} },
			code:  2,
			names: []string{"iss-N or itd-N", "not repeated here"},
		},
	}
	for _, c := range cases {
		tok, args := tokenMarker(), c.args
		c.args = func(dir, _ string) []string { return args(dir, tok) }
		c.hide = func(string) string { return tok }
		checkRefusalCase(t, c)
	}

	// build answers --json with its own refusal document ahead of the error
	// object, which the boundary pass does not read: the key check words its
	// row without the key, so neither document carries it.
	t.Setenv("HOME", t.TempDir())
	dir := scratchRepoWithCommit(t)
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	tok := tokenMarker()
	for _, form := range [][]string{{"build", tok}, {"--json", "build", tok}} {
		code, out, errOut := runRefusal(t, "", form...)
		if code != 2 {
			t.Errorf("%v: exit %d, want 2\nstdout: %s\nstderr: %s", form[0], code, out, errOut)
		}
		if repeats(out, tok) || repeats(errOut, tok) {
			t.Errorf("%v: build repeated what was passed\nstdout: %s\nstderr: %s", form[0], out, errOut)
		}
		if !strings.Contains(out+errOut, "itd-N or iss-N") {
			t.Errorf("%v: build does not say the form a key takes\nstdout: %s\nstderr: %s", form[0], out, errOut)
		}
	}
}

// TestWithholdArgumentsKeepsWhatARefusalMustName holds the boundary pass to
// its three exemptions — a short argument, a command name, a well-formed record
// id — and to whole-word matching, so a refusal keeps naming its verb, the
// record it could not find and the words it is built from.
func TestWithholdArgumentsKeepsWhatARefusalMustName(t *testing.T) {
	root := NewRootCommand()
	tok := tokenMarker()
	cases := []struct {
		name string
		args []string
		msg  string
		want string
	}{
		{"positional", []string{"spec", "close", tok}, `spec id "` + tok + `" is malformed`, `spec id "(argument 3, not repeated here)" is malformed`},
		{"flag value with =", []string{"lint", "--config=" + tok}, "cannot read " + tok, "cannot read (argument 2, not repeated here)"},
		{"after the terminator", []string{"capture", "--", "-" + tok}, "refused -" + tok, "refused (argument 3, not repeated here)"},
		{"escaped by %q", []string{"build", "a\tb" + tok}, fmt.Sprintf("refused %q", "a\tb"+tok), `refused "(argument 2, not repeated here)"`},
		{"a record id stays", []string{"spec", "show", "spc-2610231038335615"}, "no spec spc-2610231038335615", "no spec spc-2610231038335615"},
		{"a command name stays", []string{"implement", "load"}, "abcd implement load: refused", "abcd implement load: refused"},
		{"a short argument stays", []string{"capture", "--limit=7"}, "limit 7 of 70", "limit 7 of 70"},
		{"inside a longer word stays", []string{"x", "drainage"}, "the drainagescheme", "the drainagescheme"},
		{"a flag name stays", []string{"--severity", tok}, "--severity is closed; " + tok, "--severity is closed; (argument 2, not repeated here)"},
	}
	for _, c := range cases {
		if got := withholdArguments(root, c.args, c.msg); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}
