package cli

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/cobra"
)

// next_steps.go — what usually comes next, carried on a successful --json
// result (itd-2610301428378830, spc-2610301431108175 approach 7, step 6).
// The steps are the command facts' (internal/core/surface/next.go), and they
// reach the result here, where the document is written, so run, the verbs'
// structs and the byte order of every key a result already has stay exactly
// as they were:
//
//   - The root's pre-run hook wraps a visible command's stdout, under --json,
//     in a nextStepsWriter that carries the command's path and its entry. A
//     hidden command, the hook plane's, keeps its stdout as it is.
//   - render encodes the result to bytes as it always has, then, given that
//     writer, splices `"next_steps": [...]` in before the document's closing
//     brace. Nothing is decoded and encoded again, so no key moves and no
//     number changes. A document that is not an object takes no key, and one
//     that already carries a top-level next_steps keeps its own.
//   - A streaming command's documents before its last carry no key; its
//     closing summary is written through renderFinal, which carries it.
//
// The key is next_steps, never next, which several verbs' results use for
// something of their own.

// nextStepsWriter is a visible command's stdout under --json, carrying what
// render needs to add the command's next steps to its result.
type nextStepsWriter struct {
	io.Writer
	path string
	next surface.Next
	// check is the test binary's check (nextStepsCheck) as it stood when the
	// writer was made; nil in the binary.
	check func(path string, doc []byte)
}

// nextStepsCheck, when set, is told every document render writes to a
// visible command's stdout under --json, with the command's path, before any
// key is added. It is nil in the binary; the package's tests set it so the
// test binary fails on a result already carrying a top-level next_steps, or
// one that is not an object from a command not exempted (next_steps_test.go).
var nextStepsCheck func(path string, doc []byte)

// installNextSteps is the root's pre-run hook: it wraps the stdout of the
// command about to run, when the call asked for --json and the command is
// visible and has an entry, so render can find the command's next steps.
func installNextSteps(cmd *cobra.Command, asJSON bool, nextFor func(string) (surface.Next, bool)) {
	if !asJSON {
		return
	}
	for c := cmd; c != nil; c = c.Parent() {
		if c.Hidden {
			return
		}
	}
	n, ok := nextFor(cmd.CommandPath())
	if !ok {
		return
	}
	cmd.SetOut(&nextStepsWriter{Writer: cmd.OutOrStdout(), path: cmd.CommandPath(), next: n, check: nextStepsCheck})
}

// plainStdout is w with the next-steps wrapper taken off, for a caller that
// needs the stream itself: a terminal test, or a child process's stdout,
// which exec hands over directly only when it is a file.
func plainStdout(w io.Writer) io.Writer {
	if nw, ok := w.(*nextStepsWriter); ok {
		return nw.Writer
	}
	return w
}

// docStyle is how a document was encoded, so the spliced key is encoded the
// same way: indented two spaces or compact, and with or without HTML
// escaping.
type docStyle struct {
	indent     bool
	escapeHTML bool
}

// renderStyle is render's own encoding: indented, with encoding/json's HTML
// escaping.
var renderStyle = docStyle{indent: true, escapeHTML: true}

// The three ways a document meets the next steps.
const (
	// docOne is a command's one document, or a streaming command's
	// document before its last.
	docOne = iota
	// docFinal is a streaming command's closing summary.
	docFinal
	// docRefused is a document a refusal follows or stands in for, which is
	// no success and carries no next steps.
	docRefused
)

// renderFinal is render for a streaming command's closing summary: the one
// document of its run that carries next_steps.
func renderFinal(w io.Writer, asJSON bool, v any, text func(io.Writer)) error {
	return renderAs(w, asJSON, v, text, docFinal)
}

// renderRefused is render for a document a refusal follows or stands in for,
// such as what a refused ingest landed before it refused, or a refusal
// receipt that is the error object itself: it is no success, so it carries
// no next steps.
func renderRefused(w io.Writer, asJSON bool, v any, text func(io.Writer)) error {
	return renderAs(w, asJSON, v, text, docRefused)
}

// renderAs is render with the document's kind.
func renderAs(w io.Writer, asJSON bool, v any, text func(io.Writer), kind int) error {
	if !asJSON {
		text(w)
		return nil
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	return writeDocument(w, b.Bytes(), kind, renderStyle)
}

// writeDocument writes one encoded --json document to w. Written to a
// visible command's stdout, a document of a kind that takes them gets the
// command's next steps spliced in, encoded in the document's own style.
func writeDocument(w io.Writer, doc []byte, kind int, style docStyle) error {
	if nw, ok := w.(*nextStepsWriter); ok && kind != docRefused {
		if nw.check != nil {
			nw.check(nw.path, doc)
		}
		if kind == docFinal || !nw.next.Streaming {
			spliced, err := spliceNextSteps(doc, nw.next.Steps, style)
			if err != nil {
				return err
			}
			doc = spliced
		}
	}
	_, err := w.Write(doc)
	return err
}

// nextStepsKey is the key the steps join a result under.
const nextStepsKey = "next_steps"

// spliceNextSteps is doc with `"next_steps": steps` added as its last
// top-level key, every byte before it unchanged. A document that is not a
// JSON object, or one that already carries the key at its top level, is
// returned as it is.
func spliceNextSteps(doc []byte, steps []surface.NextStep, style docStyle) ([]byte, error) {
	end := len(bytes.TrimRight(doc, " \t\r\n"))
	body, tail := doc[:end], doc[end:]
	if len(body) < 2 || body[0] != '{' || body[end-1] != '}' {
		return doc, nil
	}
	if topLevelKey(body, nextStepsKey) {
		return doc, nil
	}
	if steps == nil {
		steps = []surface.NextStep{}
	}
	var val bytes.Buffer
	enc := json.NewEncoder(&val)
	enc.SetEscapeHTML(style.escapeHTML)
	if style.indent {
		enc.SetIndent("  ", "  ")
	}
	if err := enc.Encode(steps); err != nil {
		return nil, err
	}
	value := bytes.TrimRight(val.Bytes(), "\n")
	inner := bytes.TrimRight(body[:end-1], " \t\r\n")
	empty := len(bytes.TrimSpace(inner[1:])) == 0

	out := make([]byte, 0, len(doc)+len(value)+32)
	switch {
	case style.indent && empty:
		out = append(out, "{\n  \""+nextStepsKey+"\": "...)
	case style.indent:
		out = append(append(out, inner...), ",\n  \""+nextStepsKey+"\": "...)
	case empty:
		out = append(out, "{\""+nextStepsKey+"\":"...)
	default:
		out = append(append(out, inner...), ",\""+nextStepsKey+"\":"...)
	}
	out = append(out, value...)
	if style.indent {
		out = append(out, '\n')
	}
	out = append(out, '}')
	return append(out, tail...), nil
}

// topLevelKey reports whether the object body has key at its top level. The
// cheap test comes first: a document that never spells the key holds none.
func topLevelKey(body []byte, key string) bool {
	if !bytes.Contains(body, []byte(`"`+key+`"`)) {
		return false
	}
	var top map[string]json.RawMessage
	if json.Unmarshal(body, &top) != nil {
		return false
	}
	_, ok := top[key]
	return ok
}
