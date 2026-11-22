package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/refusal"
)

// refusal_envelope_test.go — the refusal's code, hint and next command on
// the --json envelope and the text form (spc-2610301431108175 step 1).

// envelopeKeys is a JSON object's top-level keys in the order they stand.
func envelopeKeys(t *testing.T, doc string) []string {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(doc))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		t.Fatalf("not a JSON object: %v\n%s", err, doc)
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, tok.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatal(err)
		}
	}
	return keys
}

// TestRefusedCaptureCarriesItsCodeHintAndNext is the step's acceptance: a
// refused `capture` with no --remedy carries capture/remedy-missing, its
// hint, and a next command naming `abcd capture`, after the three keys every
// reader of the envelope already reads, which keep their order and values.
func TestRefusedCaptureCarriesItsCodeHintAndNext(t *testing.T) {
	_ = captureLedgerRepo(t)
	var stdout, stderr strings.Builder
	code := Run([]string{"capture", "the parser drops a line", "--json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit %d, want 2:\n%s%s", code, stdout.String(), stderr.String())
	}
	if got := envelopeKeys(t, stdout.String()); strings.Join(got, ",") != "abcd,error,exit_code,code,hint,next" {
		t.Fatalf("envelope keys = %v, want abcd, error, exit_code, then code, hint, next", got)
	}
	var env errorEnvelope
	if err := json.Unmarshal([]byte(stdout.String()), &env); err != nil {
		t.Fatal(err)
	}
	entry, _ := refusal.Lookup(refusal.CaptureRemedyMissing)
	if env.Abcd != "error" || env.ExitCode != 2 || !strings.Contains(env.Error, "--remedy is required") {
		t.Errorf("the envelope's first three keys changed: %+v", env)
	}
	if env.Code != string(refusal.CaptureRemedyMissing) || env.Hint != entry.Summary {
		t.Errorf("code %q hint %q, want %s with its summary", env.Code, env.Hint, refusal.CaptureRemedyMissing)
	}
	if !strings.HasPrefix(env.Next, "abcd capture ") {
		t.Errorf("next = %q, want a command naming abcd capture", env.Next)
	}
	if stderr.Len() != 0 && strings.Contains(stderr.String(), "abcd explain") {
		t.Errorf("a --json refusal wrote the text form's line to stderr:\n%s", stderr.String())
	}
}

// TestRefusalTextFormAddsOneLine: without --json, the refusal's line stands
// as it was, and one line after it gives the hint, the command to try and the
// call that explains the code.
func TestRefusalTextFormAddsOneLine(t *testing.T) {
	_ = captureLedgerRepo(t)
	var stdout, stderr strings.Builder
	if code := Run([]string{"capture", "the parser drops a line"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d, want 2:\n%s", code, stderr.String())
	}
	entry, _ := refusal.Lookup(refusal.CaptureRemedyMissing)
	lines := strings.Split(strings.TrimRight(stderr.String(), "\n"), "\n")
	last := lines[len(lines)-1]
	want := "abcd: " + entry.Summary + " — try: " + entry.Next + " (abcd explain capture/remedy-missing)"
	if last != want {
		t.Fatalf("the refusal's last line is\n%q\nwant\n%q", last, want)
	}
	if prev := lines[len(lines)-2]; !strings.HasPrefix(prev, "abcd: abcd capture: --remedy is required") {
		t.Errorf("the refusal's own line changed: %q", prev)
	}
}

// TestRefusalNextNeverCarriesTheCallersArguments: the next command is filled
// from the refused call's command path alone, so nothing the caller typed
// reaches it, whatever the refusal.
func TestRefusalNextNeverCarriesTheCallersArguments(t *testing.T) {
	_ = captureLedgerRepo(t)
	const secret = "zebra-operand-1234"
	for _, args := range [][]string{
		{"capture", "the " + secret + " drops a line"},
		{"capture", "list", "--" + secret},
		{"capture", "resolve", secret},
	} {
		code, env, out := codedRefusal(t, args...)
		if code != 2 || env.Code == "" {
			t.Fatalf("abcd %s: exit %d code %q, want a coded refusal:\n%s", strings.Join(args, " "), code, env.Code, out)
		}
		if strings.Contains(env.Next, "zebra") || strings.Contains(env.Hint, "zebra") {
			t.Errorf("abcd %s: next %q or hint %q carries the caller's text", strings.Join(args, " "), env.Next, env.Hint)
		}
	}
}

// TestUncodedRefusalEnvelopeIsAsBefore: a refusal with no code, one the
// migration has not reached, encodes byte for byte as the envelope always
// has; one carrying internal/uncoded carries the code alone, with no hint, no
// next command and no text line.
func TestUncodedRefusalEnvelopeIsAsBefore(t *testing.T) {
	env := newErrorEnvelope("abcd x: refused", 2)
	env.withRefusal(refusal.None, nil)
	got, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"abcd":"error","error":"abcd x: refused","exit_code":2}`; string(got) != want {
		t.Errorf("an uncoded envelope is %s, want %s", got, want)
	}
	if line := env.refusalLine(); line != "" {
		t.Errorf("an uncoded refusal has a text line: %q", line)
	}
	env.withRefusal(refusal.Uncoded, nil)
	if env.Code != string(refusal.Uncoded) || env.Hint != "" || env.Next != "" || env.refusalLine() != "" {
		t.Errorf("internal/uncoded rendered as %+v", env)
	}
}

// TestRefusalWithNoNextCommandSaysSo: a code whose next is none carries the
// word in --json, and the text line leaves out the command to try.
func TestRefusalWithNoNextCommandSaysSo(t *testing.T) {
	env := newErrorEnvelope("abcd explain: refused", 2)
	env.withRefusal(refusal.ExplainUnknownCode, nil)
	if env.Next != refusal.NoNext {
		t.Fatalf("next = %q, want %q", env.Next, refusal.NoNext)
	}
	if line := env.refusalLine(); strings.Contains(line, "try:") || !strings.HasSuffix(line, "(abcd explain explain/unknown-code)") {
		t.Errorf("the text line is %q", line)
	}
}
