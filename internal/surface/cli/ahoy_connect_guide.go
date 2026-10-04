package cli

// ahoy_connect_guide.go is the front door of the guided connect
// (spc-2610031241482088, "The guided path in the session"):
// `abcd ahoy connect [<provider>] --guide [--base-url <url>]` starts it, and
// `--guide --resume - --answer <value>` takes the next turn, the previous
// turn's resume object on stdin. Each run prints one turn: a question to put
// to the person through the host's question tool, the end (the one command
// and every path it writes, for the person to paste into a terminal), or a
// stop. It writes nothing, asks for no key, and never runs the command: no
// flag runs it, so --guide refuses every flag that would set the connection
// up (decision 3).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/surface/cli/ask"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// maxResumeBytes bounds the resume object read back: the listed ids it
// carries (oracle.MaxCarriedBytes), with 64 KiB more for the answers and the
// JSON around them, which keeps it below the 128 KiB one argument may hold on
// linux (MAX_ARG_STRLEN), with room for the rest of the command.
const maxResumeBytes = oracle.MaxCarriedBytes + 64<<10

// guideSetupFlags are the flags that set a connection up: the guide prints
// the command that carries them and takes none of them itself.
var guideSetupFlags = []string{"home", "model", "key", "env", "file", "field"}

// guideOut is one turn as the front door prints it: the core's turn, and the
// question as the host's question tool takes it, so the page passes it on
// unchanged.
type guideOut struct {
	oracle.GuideTurn
	Tool *hostQuestionCall `json:"tool,omitempty"`
}

// hostQuestionCall is a question tool's input, in the host's own key names
// (those hostQuestionInput decodes for the guard).
type hostQuestionCall struct {
	Questions []hostQuestionOut `json:"questions"`
}

type hostQuestionOut struct {
	Header      string          `json:"header"`
	Question    string          `json:"question"`
	Options     []hostOptionOut `json:"options"`
	MultiSelect bool            `json:"multiSelect"`
}

type hostOptionOut struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// hostCall maps a onto the host's question tool through the field view: the
// chip, the question text, the options and decide later. A typed part is the
// host's free-text row, which the host draws under every question; where the
// question lists fewer options than the host's tool takes (a typed part and
// decide later alone), one option labelled question.TypedRowLabel points at
// that row, and the guide asks again if it is chosen.
func hostCall(a question.Ask) *hostQuestionCall {
	f := a.Fields()
	out := &hostQuestionCall{}
	for _, t := range f.Tabs {
		q := hostQuestionOut{Header: t.Header, Question: t.Text}
		opts := t.Options
		if t.Typed != "" && len(opts) < question.Default.OptionsPerQ[0] {
			row := question.Choice{Label: question.TypedRowLabel,
				Description: "Choose the row for typing below instead, and type it there."}
			opts = append([]question.Choice{row}, opts...)
		}
		for _, o := range opts {
			q.Options = append(q.Options, hostOptionOut{Label: o.Label, Description: o.Description})
		}
		out.Questions = append(out.Questions, q)
	}
	return out
}

// envNames are the names of the variables in this process's environment;
// each value is cut away here and never passed on.
func envNames() []string {
	var out []string
	for _, kv := range os.Environ() {
		if name, _, ok := strings.Cut(kv, "="); ok && name != "" {
			out = append(out, name)
		}
	}
	return out
}

// runConnectGuide is one turn of `ahoy connect --guide`.
func runConnectGuide(cmd *cobra.Command, args []string, baseURL, resume string, answered bool, answer string, asJSON bool) error {
	fail := func(msg string) error {
		return &exitError{Code: 2, Msg: "abcd ahoy connect --guide: " + termsafe.Sanitize(fsutil.RedactHome(msg))}
	}
	for _, f := range guideSetupFlags {
		if cmd.Flags().Changed(f) {
			return fail(fmt.Sprintf("--%s sets the connection up, and the guide never does: it prints the command "+
				"for you to run in a terminal, so run the guide without it", f))
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	roots, notes := layered.RootsFor(cwd)
	for _, n := range notes {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd %s\n", termsafe.Sanitize(fsutil.RedactHome(n)))
	}
	req := oracle.GuideRequest{Roots: roots, BaseURL: baseURL, EnvNames: envNames()}
	if len(args) == 1 {
		req.Provider = args[0]
	}
	switch {
	case resume != "":
		st, err := readResume(cmd.InOrStdin(), resume)
		if err != nil {
			return fail(err.Error())
		}
		req.Resume = &st
	case answered:
		return fail("--answer needs --resume, the resume object of the turn it answers")
	}
	if answered {
		req.Answer = &answer
	}
	turn, err := oracle.Guide(context.Background(), req)
	if err != nil {
		return fail(err.Error())
	}
	out := guideOut{GuideTurn: turn}
	if turn.Ask != nil {
		out.Tool = hostCall(*turn.Ask)
	}
	return render(cmd.OutOrStdout(), asJSON, out, func(w io.Writer) { writeGuideText(w, out) })
}

// readResume reads the resume object: from stdin for "-", else the flag's
// value itself, bounded and strictly decoded.
func readResume(stdin io.Reader, flag string) (oracle.GuideState, error) {
	raw := []byte(flag)
	if flag == "-" {
		var err error
		if raw, err = io.ReadAll(io.LimitReader(stdin, maxResumeBytes+1)); err != nil {
			return oracle.GuideState{}, errors.New("the resume object could not be read from stdin")
		}
	}
	if len(raw) > maxResumeBytes {
		return oracle.GuideState{}, fmt.Errorf("the resume object is larger than %d bytes", maxResumeBytes)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var st oracle.GuideState
	if err := dec.Decode(&st); err != nil {
		return oracle.GuideState{}, errors.New("the resume object is not the guide's: pass the resume member of the last turn unchanged")
	}
	return st, nil
}

// writeGuideText is a turn as text: the question drawn as the Terminal draws
// it, then the resume object to pass back; or the command on a line of its
// own and every path it writes; or the stop.
func writeGuideText(w io.Writer, out guideOut) {
	switch {
	case out.Done != nil:
		d := out.Done
		fmt.Fprintln(w, termsafe.Sanitize(d.Command))
		fmt.Fprintln(w, "When it runs, this command writes:")
		for _, p := range d.Writes {
			fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(p))
		}
		if d.Home == oracle.KeyHomeABCD || d.Home == oracle.KeyHomeKeychain {
			fmt.Fprintln(w, "It asks for the key there, hidden as you paste it.")
		}
		if d.PicksInTerminal {
			fmt.Fprintln(w, "It lists the service's models there once it has the key, and you pick one.")
		}
		fmt.Fprintln(w, "Paste it into a terminal on this machine; nothing is set up until it runs there.")
	case out.Stopped != "":
		fmt.Fprintln(w, out.Stopped)
		writeResume(w, out.Resume)
	case out.Ask != nil:
		for _, l := range ask.Layout(*out.Ask, 80, term.Mono) {
			fmt.Fprintln(w, l)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Answer with --guide --resume - --answer <value>, this resume object on stdin:")
		writeResume(w, out.Resume)
	}
}

func writeResume(w io.Writer, st oracle.GuideState) {
	b, err := json.Marshal(st)
	if err != nil {
		return
	}
	fmt.Fprintln(w, termsafe.Sanitize(string(b)))
}
