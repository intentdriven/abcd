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
//     in a nextStepsWriter that carries the command's path and its entry, and
//     wraps the command's run so the writer hears how it ended. A hidden
//     command, the hook plane's, keeps its stdout as it is.
//   - render encodes the result to bytes as it always has and hands them to
//     that writer, which holds the document back until the command's run
//     returns. Only a run that returns nil has `"next_steps": [...]` spliced
//     in before the document's closing brace; a run that fails releases the
//     document as it was, so no document a non-zero exit or an error object
//     follows carries the key, whichever helper wrote it. Nothing is decoded
//     and encoded again, so no key moves and no number changes. A document
//     that is not an object takes no key, and one that already carries a
//     top-level next_steps keeps its own.
//   - Anything else written to the stream, a second document included, and a
//     caller that takes the stream itself (plainStdout), first release a held
//     document as it was: only a command's last document carries the key.
//   - A streaming command's documents before its last are written at once and
//     carry no key; its closing summary is written through renderFinal, which
//     is held like any other command's document.
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
	check *nextStepsChecks

	// held is the document waiting for the run's end, in its style.
	held      []byte
	heldStyle docStyle
	// spliced says the key was written, so run can tie it to exit 0.
	spliced bool
	// raw is what was written to the stream outside writeDocument, kept only
	// under the test binary's check, so a document that went around the
	// splice is heard.
	raw bytes.Buffer
}

// nextStepsChecks are the test binary's checks on the next steps. Each is
// told, with the command's path:
type nextStepsChecks struct {
	// document: every document written through writeDocument to a visible
	// command's stdout under --json, before any key is added.
	document func(path string, doc []byte)
	// around: what a run that returned nil wrote to that stdout outside
	// writeDocument, so a --json document that went around the splice is
	// heard.
	around func(path string, raw []byte)
	// exit: run's exit code, and whether the key was written, so a key that
	// a non-zero exit or an error object follows is heard.
	exit func(path string, spliced bool, code int)
}

// nextStepsCheck, when set, holds the checks above. It is nil in the binary;
// the package's tests set it so the test binary fails on a result already
// carrying a top-level next_steps, one that is not an object from a command
// not exempted, a --json document written around the splice, and a key a
// failed run wrote (next_steps_test.go).
var nextStepsCheck *nextStepsChecks

// installNextSteps is the root's pre-run hook: it wraps the stdout of the
// command about to run, when the call asked for --json and the command is
// visible and has an entry, so render can find the command's next steps, and
// wraps its run so the document it wrote is released, with the key or
// without it, by how the run ended.
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
	// A root executed again keeps the leaf's writer from the run before:
	// it is taken off rather than wrapped a second time.
	w := &nextStepsWriter{Writer: plainStdout(cmd.OutOrStdout()), path: cmd.CommandPath(), next: n, check: nextStepsCheck}
	cmd.SetOut(w)
	// A Run is taken in as a RunE for this run alone, which cobra prefers,
	// so a document that cannot be written fails the run either way.
	run, restore := cmd.RunE, cmd.RunE
	if run == nil && cmd.Run != nil {
		plain := cmd.Run
		run = func(c *cobra.Command, args []string) error { plain(c, args); return nil }
	}
	if run == nil {
		return
	}
	cmd.RunE = func(c *cobra.Command, args []string) (err error) {
		cmd.RunE = restore
		done := false
		// The document is written here, after the run, so a write that
		// fails fails the run, as it did inside it; a run that already
		// failed keeps its own error.
		defer func() {
			if ferr := w.finish(done && err == nil); err == nil {
				err = ferr
			}
		}()
		err = run(c, args)
		done = true
		return err
	}
}

// Write is the stream itself: a held document is released as it was first,
// since what follows it makes it no longer the command's last.
func (w *nextStepsWriter) Write(p []byte) (int, error) {
	if err := w.release(); err != nil {
		return 0, err
	}
	if w.check != nil {
		w.raw.Write(p)
	}
	return w.Writer.Write(p)
}

// hold keeps doc back for the run's end, releasing any document held before
// it as it was.
func (w *nextStepsWriter) hold(doc []byte, style docStyle) error {
	if err := w.release(); err != nil {
		return err
	}
	w.held, w.heldStyle = append([]byte(nil), doc...), style
	return nil
}

// release writes a held document as it was, with no key.
func (w *nextStepsWriter) release() error {
	if w.held == nil {
		return nil
	}
	doc := w.held
	w.held = nil
	_, err := w.Writer.Write(doc)
	return err
}

// finish is the run's end: the held document is written with the command's
// next steps when the run succeeded and as it was when it did not. The
// write's error is returned, for the run to fail with; a key that did not
// reach the stream is not counted as written.
func (w *nextStepsWriter) finish(ok bool) error {
	if w.held != nil && ok {
		if spliced, err := spliceNextSteps(w.held, w.next.Steps, w.heldStyle); err == nil {
			w.spliced = !bytes.Equal(spliced, w.held)
			w.held = spliced
		}
	}
	err := w.release()
	if err != nil {
		w.spliced = false
	}
	if ok && w.check != nil && w.check.around != nil && w.raw.Len() > 0 {
		w.check.around(w.path, w.raw.Bytes())
	}
	return err
}

// checkNextStepsExit is run's half of the test binary's check: the code a
// run exits with, and whether its command's writer wrote the key.
func checkNextStepsExit(called *cobra.Command, code int) {
	if called == nil {
		return
	}
	if w, ok := called.OutOrStdout().(*nextStepsWriter); ok && w.check != nil && w.check.exit != nil {
		w.check.exit(w.path, w.spliced, code)
	}
}

// plainStdout is w with the next-steps wrapper taken off, for a caller that
// needs the stream itself: a terminal test, or a child process's stdout,
// which exec hands over directly only when it is a file. A held document is
// released first, so nothing written beneath the wrapper overtakes it.
func plainStdout(w io.Writer) io.Writer {
	if nw, ok := w.(*nextStepsWriter); ok {
		_ = nw.release()
		return nw.Writer
	}
	return w
}

// streamOf is w with the next-steps wrapper taken off and nothing released:
// for a test of what the stream is, such as whether it is a terminal, which
// writes nothing and so leaves a held document the command's last.
func streamOf(w io.Writer) io.Writer {
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
// no next steps even when the run goes on to return nil.
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
// visible command's stdout, a document of a kind that takes them is held for
// the run's end, when it gets the command's next steps spliced in, encoded in
// the document's own style, if the run succeeded.
func writeDocument(w io.Writer, doc []byte, kind int, style docStyle) error {
	nw, ok := w.(*nextStepsWriter)
	if !ok {
		_, err := w.Write(doc)
		return err
	}
	if kind != docRefused && nw.check != nil && nw.check.document != nil {
		nw.check.document(nw.path, doc)
	}
	if kind == docFinal || kind == docOne && !nw.next.Streaming {
		return nw.hold(doc, style)
	}
	if err := nw.release(); err != nil {
		return err
	}
	_, err := nw.Writer.Write(doc)
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
