package runner

// opencode.go is the opencode runner (spc-2609221533057881 scope 2): run mode
// with raw JSON events, the repository as its directory, the brief attached,
// and no external plugins (--pure), the nearest the harness offers to the
// claude CLI's bare mode, so a plugin the target repository configures does
// not run. Permissions are left at the harness's own configuration: abcd does
// not pass the flag that approves every request unasked.
//
// The runner reaches opencode through run mode only; its server's session
// endpoints, where a server is already up, are a later adapter.

import (
	"bytes"
	"context"
	"encoding/json"
)

// OpenCodeCLI is the opencode runner.
type OpenCodeCLI struct {
	// model is the model route, <provider>/<model> as opencode spells it too,
	// or "" for the harness's own default.
	model  string
	launch launcher
}

// newOpenCode returns the opencode runner asking for model, a
// <provider>/<model> route the configuration admitted, or "".
func newOpenCode(model string) *OpenCodeCLI {
	return &OpenCodeCLI{model: model, launch: defaultLauncher()}
}

// Name is the route's name.
func (*OpenCodeCLI) Name() string { return OpenCode }

func (o *OpenCodeCLI) args(req Request) []string {
	a := []string{"run", "--format=json", "--pure", "--dir=" + req.Dir, "--file=" + req.Brief}
	if o.model != "" {
		a = append(a, "--model="+o.model)
	}
	return append(a, "--", prompt(req))
}

// Run runs the role through opencode.
func (o *OpenCodeCLI) Run(ctx context.Context, req Request) (Answer, []byte, error) {
	if err := req.check(); err != nil {
		return Answer{}, nil, err
	}
	bin, err := o.launch.admit(OpenCode, "opencode", req.Dir)
	if err != nil {
		return Answer{}, nil, err
	}
	res, err := o.launch.run(ctx, OpenCode, bin, o.args(req), req.Dir, req.timeout())
	transcript := res.transcript()
	if err != nil {
		return Answer{}, transcript, err
	}
	ans, err := parseOpenCode(res.stdout)
	if err != nil {
		return Answer{}, transcript, err
	}
	ans.Model = o.model
	return ans, transcript, nil
}

// openCodeEvent is the part of a run-mode JSON event the runner reads.
type openCodeEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionID"`
	Part      struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"part"`
}

// parseOpenCode reads the event stream: one JSON object per line. An error
// event is a refusal; a line that is not an event, or a stream with no text
// and no finished step, is unparsable. The last text part is the final
// message.
func parseOpenCode(out []byte) (Answer, error) {
	var ans Answer
	seen := false
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var ev openCodeEvent
		if json.Unmarshal(line, &ev) != nil || ev.Type == "" {
			return Answer{}, fail(OpenCode, ReasonUnparsable, "a line of its output is not a JSON event")
		}
		if ans.SessionID == "" && ev.SessionID != "" {
			ans.SessionID = ev.SessionID
		}
		switch ev.Type {
		case "error":
			return Answer{}, fail(OpenCode, ReasonRefused, "it reported an error event")
		case "text":
			ans.Text = ev.Part.Text
			seen = true
		case "step_finish":
			seen = true
		}
	}
	if !seen {
		return Answer{}, fail(OpenCode, ReasonUnparsable, "its output carries no text and no finished step")
	}
	return ans, nil
}
