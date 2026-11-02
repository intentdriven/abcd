package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/rules"
)

// fillEnvelope is the prompt router's --json envelope with the fill
// (spc-2610111627213138 step 5).
type fillEnvelope struct {
	Text  string      `json:"text"`
	Fill  *rules.Fill `json:"fill"`
	Error string      `json:"error"`
}

// promptPayload is a UserPromptSubmit payload naming the session's
// transcript, as the host hands it to the prompt router.
func promptPayload(t *testing.T, session, cwd, prompt, transcript string) string {
	t.Helper()
	b, err := json.Marshal(map[string]string{
		"session_id": session, "cwd": cwd, "prompt": prompt,
		"transcript_path": transcript, "hook_event_name": "UserPromptSubmit",
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func fillRouterJSON(t *testing.T, payload string) fillEnvelope {
	t.Helper()
	out, _ := runHook(t, payload, "hook", "prompt-router", "--json")
	var env fillEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("prompt-router --json is not one JSON document: %v\n%s", err, out)
	}
	return env
}

// fillTranscript writes a transcript whose last response, from
// claude-opus-4-1 (a 200,000-token window), held input tokens in all.
func fillTranscript(t *testing.T, input int) string {
	t.Helper()
	p := filepath.Join(realPath(t, t.TempDir()), "session.jsonl")
	body := `{"type":"user","message":{"role":"user","content":"hello"}}` + "\n" +
		`{"type":"assistant","message":{"id":"msg_1","role":"assistant","model":"claude-opus-4-1-20250805","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":` + itoa(input) + `,"output_tokens":3}}}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// TestHookEnvelopeNamesTheFillAndItsSource: the router's --json envelope
// carries how full the session is and where that came from, on every
// evaluated prompt: the status line's fill file first, the transcript's tail
// without one, and unknown, with the reason, when neither tells. A fresh fill
// and an unknown one both keep the rules in full (step 6's bracket table),
// so with no note or page in reach the text is the same for both.
func TestHookEnvelopeNamesTheFillAndItsSource(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	writeRepoRules(t, dir, widgetsRules)
	if err := rules.ResetFill(home, ""); err != nil {
		t.Fatal(err)
	}
	transcript := fillTranscript(t, 50_000)

	// Unknown: no fill file, and no transcript named.
	unknown := fillRouterJSON(t, promptPayload(t, "fill-a", dir, "count the widget", ""))
	if unknown.Fill == nil || unknown.Fill.Source != rules.FillSourceUnknown || unknown.Fill.UsedPercentage != nil || unknown.Fill.Reason == "" {
		t.Fatalf("fill = %+v, want unknown with its reason", unknown.Fill)
	}
	if !strings.Contains(unknown.Text, "## WIDGETS") {
		t.Fatalf("the rules did not inject: %q", unknown.Text)
	}

	// The transcript's tail, without a fill file for the session.
	fromTranscript := fillRouterJSON(t, promptPayload(t, "fill-b", dir, "count the widget", transcript))
	if f := fromTranscript.Fill; f == nil || f.Source != rules.FillSourceTranscript || f.UsedPercentage == nil || *f.UsedPercentage != 25 || f.Model != "claude-opus-4-1" {
		t.Fatalf("fill = %+v, want 25%% from the transcript of claude-opus-4-1", f)
	}
	if fromTranscript.Text != unknown.Text {
		t.Errorf("the fill changed what was injected:\n%q\n%q", fromTranscript.Text, unknown.Text)
	}

	// The status line's figure for the session wins over the transcript.
	if err := rules.WriteFill(home, "fill-c", 71, time.Now()); err != nil {
		t.Fatal(err)
	}
	fromStatus := fillRouterJSON(t, promptPayload(t, "fill-c", dir, "count the widget", transcript))
	if f := fromStatus.Fill; f == nil || f.Source != rules.FillSourceStatusLine || f.UsedPercentage == nil || *f.UsedPercentage != 71 {
		t.Fatalf("fill = %+v, want the status line's 71%%", f)
	}

	// A prompt the router could not evaluate carries no fill.
	writeRepoRules(t, dir, `{"schema_version": 1, "domains": "not an object"}`)
	refused := fillRouterJSON(t, promptPayload(t, "fill-c", dir, "count the widget", transcript))
	if refused.Error == "" || refused.Fill != nil {
		t.Errorf("an unevaluated prompt: error %q, fill %+v; want an error and no fill", refused.Error, refused.Fill)
	}
}

// TestPlainRouterOutputFollowsTheTranscriptsFill: the plain form, which the
// host injects, is sent in the bracket the fill puts the prompt in, the
// transcript's fill included: a transcript whose last response used 95% of
// its model's window shrinks a domain to its heading and first rule, and a
// session with no fill keeps it whole (spc-2610111627213138 step 6).
func TestPlainRouterOutputFollowsTheTranscriptsFill(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	writeRepoRules(t, dir, threeRules)
	with, _ := runHook(t, promptPayload(t, "plain-a", dir, "count the widget", fillTranscript(t, 190_000)), "hook", "prompt-router")
	without, _ := runHook(t, promptPayload(t, "plain-b", dir, "count the widget", ""), "hook", "prompt-router")
	if !strings.Contains(with, "Count widgets twice.") || strings.Contains(with, "Ship widgets in pairs.") || !strings.Contains(with, "`abcd rules WIDGETS`") {
		t.Errorf("a 95%% transcript fill did not shrink the domain:\n%s", with)
	}
	if !strings.Contains(without, "Ship widgets in pairs.") {
		t.Errorf("an unknown fill shrank the domain:\n%s", without)
	}
}

// TestRouterAfterACompactionKeepsTheRulesWhole: on the first prompt after a
// compaction, the reset has deleted the fill file and the transcript's last
// response is the one before the compaction boundary, which measured the
// context the compaction replaced. The fill is unknown, so the rules come back
// in full rather than shrunk by the old figure and then pinned in that form by
// the ledger.
func TestRouterAfterACompactionKeepsTheRulesWhole(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	writeRepoRules(t, dir, threeRules)
	p := filepath.Join(realPath(t, t.TempDir()), "session.jsonl")
	body := `{"type":"user","message":{"role":"user","content":"hello"}}` + "\n" +
		`{"type":"assistant","message":{"id":"msg_1","role":"assistant","model":"claude-opus-4-1-20250805","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":160000,"output_tokens":3}}}` + "\n" +
		`{"parentUuid":null,"isSidechain":false,"type":"system","subtype":"compact_boundary","content":"Conversation compacted","isMeta":false,"level":"info","compactMetadata":{"trigger":"manual","preTokens":160000}}` + "\n" +
		`{"type":"user","isSidechain":false,"isCompactSummary":true,"message":{"role":"user","content":"This session is being continued from a previous conversation."}}` + "\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	env := fillRouterJSON(t, promptPayload(t, "compacted-a", dir, "count the widget", p))
	if f := env.Fill; f == nil || f.Source != rules.FillSourceUnknown || !strings.Contains(f.Reason, "compaction") {
		t.Fatalf("fill = %+v, want unknown past the compaction", f)
	}
	plain, _ := runHook(t, promptPayload(t, "compacted-b", dir, "count the widget", p), "hook", "prompt-router")
	if !strings.Contains(plain, "Ship widgets in pairs.") {
		t.Errorf("the pre-compaction figure shrank the domain:\n%s", plain)
	}
}

// TestPromptRouterResetMakesTheFillDirectoryAndDeletesTheSessionsFile: the
// session-start and pre-compact reset makes ~/.abcd.noindex/fill/ for the
// status line to write into, and deletes the session's fill file with its
// ledgers; behind a symlinked ~/.abcd.noindex it makes nothing and says
// nothing about it.
func TestPromptRouterResetMakesTheFillDirectoryAndDeletesTheSessionsFile(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	reset := func(session string) string {
		t.Helper()
		in, err := json.Marshal(map[string]string{"session_id": session, "hook_event_name": "SessionStart"})
		if err != nil {
			t.Fatal(err)
		}
		_, stderr := runHook(t, string(in), "hook", "prompt-router-reset")
		return stderr
	}
	reset("reset-a")
	fi, err := os.Lstat(abcdhome.Path(home, "fill"))
	if err != nil || !fi.IsDir() || fi.Mode().Perm() != 0o700 {
		t.Fatalf("the reset did not make the fill directory 0700: %v %v", fi, err)
	}
	if err := rules.WriteFill(home, "reset-a", 88, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := rules.WriteFill(home, "reset-b", 44, time.Now()); err != nil {
		t.Fatal(err)
	}
	reset("reset-a")
	if f := rules.ReadFill(home, "reset-a", ""); f.Known() {
		t.Errorf("the session's fill survived its reset: %+v", f)
	}
	if f := rules.ReadFill(home, "reset-b", ""); !f.Known() {
		t.Errorf("another session's fill was deleted: %+v", f)
	}

	linked := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, abcdhome.Path(linked)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", linked)
	if stderr := reset("reset-c"); strings.Contains(stderr, "fill") {
		t.Errorf("the symlinked home's refusal was said: %q", stderr)
	}
	if _, err := os.Lstat(filepath.Join(target, "fill")); !os.IsNotExist(err) {
		t.Errorf("the reset made a fill directory under the link's target: %v", err)
	}
}
