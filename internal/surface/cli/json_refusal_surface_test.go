package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// json_refusal_surface_test.go — iss-2609100519128005.
//
// A refused `--json` invocation used to write its envelope to standard ERROR and
// leave standard OUTPUT empty. A real operator merged the two streams, parsed the
// result as JSON, never read the exit status, and concluded two captures had been
// silently lost. Nothing was lost — the refusal is atomic and correct — but on a
// merged stream a refusal and a success were both well-formed JSON, and telling
// them apart needed either the exit status the merge discarded or foreknowledge
// that an error object carries an `error` key while a success object does not.
//
// The two claims these tests hold are therefore: the outcome of a machine-readable
// run is on the stream a machine-readable consumer reads, and a refusal announces
// itself as one.

// TestJSONRefusalIsOnStdoutAndSaysSoInTheDocument is the headline. `capture list
// --json` with no state flag is a stable refusal that writes nothing.
func TestJSONRefusalIsOnStdoutAndSaysSoInTheDocument(t *testing.T) {
	_ = captureLedgerRepo(t)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"capture", "list", "--json"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected a non-zero exit for `capture list` with no state flag")
	}
	if stdout.Len() == 0 {
		t.Fatalf("a --json refusal left standard output empty; the outcome of a machine-readable run must be on the stream a machine-readable consumer reads (stderr was %q)", stderr.String())
	}
	var env struct {
		Abcd     string `json:"abcd"`
		Error    string `json:"error"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("--json refusal on stdout is not one JSON document: %v\nstdout: %q", err, stdout.String())
	}
	if env.Abcd != "error" {
		t.Errorf(`a refusal must announce itself by shape alone: want a top-level "abcd":"error", got %q in %s`, env.Abcd, stdout.String())
	}
	if env.Error == "" {
		t.Errorf("--json refusal envelope has an empty message:\n%s", stdout.String())
	}
	if env.ExitCode != code {
		t.Errorf("the envelope must carry the exit status the merged stream discarded: want %d, got %d", code, env.ExitCode)
	}
	// Nothing on stderr: a prose line there would make the merged stream — the
	// thing the reporting operator actually built — unparseable again.
	if strings.TrimSpace(stderr.String()) != "" {
		t.Errorf("a --json refusal must not also write prose to stderr, or a merged stream stops being JSON:\n%s", stderr.String())
	}
}

// TestJSONSuccessCarriesNoRefusalDiscriminator is the other half: the
// discriminator only distinguishes if a success never carries it.
func TestJSONSuccessCarriesNoRefusalDiscriminator(t *testing.T) {
	_ = captureLedgerRepo(t)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"capture", "list", "--open", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("`capture list --open --json` should succeed on an empty ledger: exit %d\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &probe); err != nil {
		// A collection envelope is an array, which carries no keys at all and so
		// can never be confused with the refusal object.
		var arr []json.RawMessage
		if err2 := json.Unmarshal(stdout.Bytes(), &arr); err2 != nil {
			t.Fatalf("success envelope is neither object nor array: %v / %v\nstdout: %q", err, err2, stdout.String())
		}
		return
	}
	for _, k := range []string{"abcd", "error"} {
		if _, present := probe[k]; present {
			t.Errorf("a success envelope carries %q, so the refusal discriminator does not discriminate:\n%s", k, stdout.String())
		}
	}
}

// lastJSONDoc decodes the LAST JSON document on a --json stdout stream into v.
//
// Stdout under --json is a stream: a verb that renders a document and then
// refuses (`history drain`, `reading assemble`) puts its data first and the
// refusal envelope after it. The refusal is always last, because Run writes it
// once the command has returned, so a test that wants the outcome takes the tail.
func lastJSONDoc(t *testing.T, b []byte, v any) error {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(b))
	var last json.RawMessage
	for {
		var doc json.RawMessage
		if err := dec.Decode(&doc); err != nil {
			if last == nil {
				return err
			}
			break
		}
		last = doc
	}
	return json.Unmarshal(last, v)
}
