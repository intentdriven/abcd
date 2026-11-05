package runner

// limit_probe.go is what a drive learns about a limit from the harness: the one
// short launch that asks whether a limit has lifted, and the shape of what a
// failed launch showed, for the run's evidence file.
//
// Nothing here is read to decide anything but the probe's answer. The probe asks
// the least abcd knows to ask: one word, asking for no tools, on the model of the
// launch that met the limit, on the same profile. That this tells a lifted limit
// from a held one is what a run's evidence file is there to show. Only the word it
// asked for, from a launch that held no tool, counts as an answer.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// limitProbeTimeout bounds one probe, past which it has said nothing.
const limitProbeTimeout = 3 * time.Minute

// limitProbePrompt is all a probe asks.
const limitProbePrompt = "Reply with the single word: ok"

// ProbeLimit asks whether a limit has lifted for role, with one short launch on the
// profile, on model, a model route, or the runner's own for "". It reports true when
// the launch answers what it was asked; otherwise the failure says what came back: a
// rate-limited one with the reset it named, if any, or any other, which says nothing
// about the limit.
func (c *Config) ProbeLimit(ctx context.Context, role, model string) (bool, *Failure) {
	if c.profile == nil {
		return false, fail(Claude, ReasonFailed, "the probe has no profile to run on")
	}
	rc, ok := c.Runner(c.RouteFor(role).Runner)
	if !ok || rc.Name != Claude {
		return false, fail(Claude, ReasonFailed, "the probe has no claude route for the %s", role)
	}
	cl, ok := c.adapter(rc).(*ClaudeCLI)
	if !ok {
		return false, fail(Claude, ReasonFailed, "the probe has no claude route for the %s", role)
	}
	return cl.probeLimit(ctx, c.ledger, model)
}

// probeLimit is ProbeLimit's launch, in a directory of its own, asking for no tools.
func (c *ClaudeCLI) probeLimit(ctx context.Context, ledger *Ledger, model string) (bool, *Failure) {
	dir, err := os.MkdirTemp("", "abcd-limit-probe-")
	if err != nil {
		return false, fail(Claude, ReasonFailed, "the probe's working directory could not be made")
	}
	defer os.RemoveAll(dir)
	if dir, err = filepath.EvalSymlinks(dir); err != nil {
		return false, fail(Claude, ReasonFailed, "the probe's working directory could not be resolved")
	}
	p := *c.profile
	c.launch.gateWaiting = nil
	c.launch.carry = stageEnv
	bin, err := c.launch.admit(Claude, "claude", p.Dir, p.Repo)
	if err != nil {
		return false, fail(Claude, ReasonFailed, "the harness could not be run for the probe")
	}
	req := Request{Role: "limit-probe", Brief: filepath.Join(dir, "brief.md"), Receipt: filepath.Join(dir, "receipt.json"),
		Dir: dir, SessionID: "limit-probe", Timeout: limitProbeTimeout, Model: model}
	args := append(c.walledFlags(req), "--", limitProbePrompt)
	// The probe spends the same subscription as a stage, so its launch is written
	// to the same ledger, and one the ledger cannot hold is not started.
	var res procResult
	end, lerr := RecordLaunch(ctx, ledger, LaunchStart{Role: req.Role, Runner: Claude, Repo: p.Repo, Fallback: req.SessionID}, func(ctx context.Context) LaunchEnd {
		var runErr error
		res, runErr = c.launch.run(ctx, Claude, bin, args, p.Env, dir, limitProbeTimeout)
		var ans Answer
		if runErr == nil {
			ans, runErr = parseClaude(res.stdout)
		}
		return LaunchEnd{Answer: ans, Transcript: res.transcript(), Err: runErr}
	})
	if lerr != nil {
		return false, fail(Claude, ReasonFailed, "the probe could not be written to the launch ledger")
	}
	if what := claudeRateLimit(res.stdout); what != "" {
		fl := limitFailure(what, res.stdout)
		fl.Seen = claudeSeen(res)
		return false, fl
	}
	var fl *Failure
	switch {
	case errors.As(end.Err, &fl):
	case end.Err != nil:
		fl = fail(Claude, ReasonFailed, "the probe did not complete")
	case !claudeInitHeldNoTools(res.stdout):
		fl = fail(Claude, ReasonFailed, "the probe's launch did not show it held no tool, so its answer says nothing")
	case !probeAnswered(end.Answer.Text):
		fl = fail(Claude, ReasonFailed, "the probe's launch did not answer the word it was asked for")
	default:
		return true, nil
	}
	fl.Seen = claudeSeen(res)
	return false, fl
}

// probeAnswered reports whether text is the word the probe asked for.
func probeAnswered(text string) bool {
	return strings.EqualFold(strings.Trim(strings.TrimSpace(text), ".!\"'`"), "ok")
}

// claudeInitHeldNoTools reports whether the stream's init event names a tool list and
// it is empty, which is how version 2.1.318 was seen to answer a launch asking for
// none. Anything else fails closed: such a launch cannot show what it held.
func claudeInitHeldNoTools(out []byte) bool {
	for _, line := range bytes.Split(out, []byte("\n")) {
		var ev struct {
			Type    string    `json:"type"`
			Subtype string    `json:"subtype"`
			Tools   *[]string `json:"tools"`
		}
		if json.Unmarshal(bytes.TrimSpace(line), &ev) == nil && ev.Type == "system" && ev.Subtype == "init" {
			return ev.Tools != nil && len(*ev.Tools) == 0
		}
	}
	return false
}

// claudeResultFailed reports whether the stream's result reports an error.
func claudeResultFailed(out []byte) bool {
	for _, line := range bytes.Split(out, []byte("\n")) {
		var ev struct {
			Type    string `json:"type"`
			IsError bool   `json:"is_error"`
		}
		if json.Unmarshal(bytes.TrimSpace(line), &ev) == nil && ev.Type == "result" && ev.IsError {
			return true
		}
	}
	return false
}

// claudeSeen is what a failed launch showed, spend read on whichever exit it took.
func claudeSeen(res procResult) *HarnessSeen {
	first, _, _ := strings.Cut(string(bytes.TrimSpace(res.stderr)), "\n")
	s := &HarnessSeen{ExitCode: res.exit, Shape: claudeShape(res.stdout), Stderr: first, SilentFor: res.silent}
	s.Spent = claudeMeasuredSpend(res.stdout)
	for _, line := range bytes.Split(res.stdout, []byte("\n")) {
		var ev struct {
			Type    string          `json:"type"`
			Subtype string          `json:"subtype"`
			Status  any             `json:"api_error_status"`
			IsError bool            `json:"is_error"`
			Error   json.RawMessage `json:"error"`
		}
		if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil || ev.Type == "" {
			continue
		}
		switch {
		case ev.Type == "system" && ev.Subtype == "thinking_tokens":
			continue
		case ev.Type == "system" && ev.Subtype == "api_retry":
			s.Retries++
			s.Retrying = true
			continue
		case ev.Type == "assistant" && len(ev.Error) > 0:
			// How running out of retries closes, as 2.1.318 was seen to: an
			// assistant event carrying the error, then a failed result. It has
			// not moved on from retrying.
			continue
		case ev.Type == "result":
			if ev.Status != nil {
				s.APIStatus = fmt.Sprint(ev.Status)
			}
			if ev.IsError {
				continue
			}
		}
		s.Retrying = false
	}
	return s
}

// claudeMeasuredSpend reports whether any event of the stream shows tokens or a cost
// above zero. A closing result of nothing, which a launch that lost the service writes,
// takes nothing back from what its assistant messages showed.
func claudeMeasuredSpend(out []byte) bool {
	type usage struct {
		Input      int `json:"input_tokens"`
		Output     int `json:"output_tokens"`
		CacheRead  int `json:"cache_read_input_tokens"`
		CacheWrite int `json:"cache_creation_input_tokens"`
	}
	some := func(u usage) bool { return u.Input > 0 || u.Output > 0 || u.CacheRead > 0 || u.CacheWrite > 0 }
	for _, line := range bytes.Split(out, []byte("\n")) {
		var ev struct {
			Type    string  `json:"type"`
			Cost    float64 `json:"total_cost_usd"`
			Usage   usage   `json:"usage"`
			Message struct {
				Usage usage `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "assistant":
			if some(ev.Message.Usage) {
				return true
			}
		case "result":
			if some(ev.Usage) || ev.Cost > 0 {
				return true
			}
		}
	}
	return false
}

// keepSeen puts what a failed launch showed on its failure, saying in abcd's words
// when that was the model service being retried: the outage is then what is named.
func keepSeen(fl *Failure, res procResult, model string) {
	fl.Seen, fl.Model = claudeSeen(res), model
	if fl.Outage() {
		fl.Detail += fmt.Sprintf("; it was retrying the model service (%d attempts), which reads as an outage", fl.Seen.Retries)
	}
	// So an expired login says so, rather than an exit status under "outage".
	if status := fl.Seen.APIStatus; status != "" && safeWord(status) == status {
		fl.Detail += "; its result names API status " + status
	}
}

// Limits on the shape a failed launch keeps. Nothing is cut here: whoever keeps it
// redacts first, so the redactor sees a whole token, and cuts afterwards.
const (
	maxShapeLines = 12
	maxShapeDepth = 4
	// maxShapeText is the longest failed result kept to be read at all; a longer
	// one reduces to its length.
	maxShapeText = 256 << 10
)

// shapeWords are the keys whose string values are kept as they are: each names a
// state or a kind, never a person.
var shapeWords = []string{"type", "subtype", "status", "rateLimitType", "error", "api_error", "stop_reason", "overageStatus", "model"}

// shapeDropped are parts of the keys left out whole, whatever their values: each
// names a session, a person or a place.
var shapeDropped = []string{"session", "uuid", "email", "account", "org", "user", "name", "cwd", "path", "token", "key", "apikey"}

// claudeShape is the end of the event stream, the events a limit shows in and the
// stream's last, one line each, the harness's version kept: keys that name a session, a person or a place
// dropped, a string reduced to its length unless its key names a state, numbers
// and booleans kept. A failed result's text is kept, since it is where a harness
// says what it met; the caller passes it through the scanner.
func claudeShape(out []byte) []string {
	var keep []string
	lines := bytes.Split(out, []byte("\n"))
	for i, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		last := i >= len(lines)-2
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.UseNumber()
		var ev map[string]any
		if dec.Decode(&ev) != nil {
			keep = append(keep, fmt.Sprintf("[not json, %d bytes]", len(line)))
			continue
		}
		if !last && !shapeWanted(ev) {
			continue
		}
		raw, err := json.Marshal(shapeValue("", ev, 0, ev["is_error"] == true))
		if err != nil {
			continue
		}
		s := string(raw)
		// A status repeated turn after turn says it once.
		if n := len(keep); n == 0 || keep[n-1] != s {
			keep = append(keep, s)
		}
	}
	if len(keep) > maxShapeLines {
		keep = keep[len(keep)-maxShapeLines:]
	}
	return keep
}

// shapeWanted reports whether an event is one a limit shows in: anything but the
// assistant's and the user's own turns, or one of those naming an error.
func shapeWanted(ev map[string]any) bool {
	switch ev["type"] {
	case "assistant", "user", "stream_event":
		_, failed := ev["error"]
		return failed
	case "system":
		return ev["subtype"] != "thinking_tokens"
	}
	return true
}

// shapeValue is one value of an event, as shape keeps it.
func shapeValue(key string, v any, depth int, failed bool) any {
	switch x := v.(type) {
	case map[string]any:
		if depth >= maxShapeDepth {
			return fmt.Sprintf("[object, %d keys]", len(x))
		}
		out := map[string]any{}
		for k, val := range x {
			if !shapeDrops(k) {
				out[k] = shapeValue(k, val, depth+1, failed)
			}
		}
		return out
	case []any:
		return fmt.Sprintf("[list, %d items]", len(x))
	case string:
		switch {
		case slices.Contains(shapeWords, key) && safeWord(x) == x:
			return x
		case key == "claude_code_version":
			return cleanHarness(x)
		case key == "result" && failed && len(x) <= maxShapeText:
			return x
		}
		return fmt.Sprintf("[text, %d bytes]", len(x))
	}
	return v
}

// shapeDrops reports whether a key names a session, a person or a place.
func shapeDrops(key string) bool {
	k := strings.ToLower(key)
	if k == "id" || strings.HasSuffix(k, "_id") || strings.HasSuffix(key, "Id") {
		return true
	}
	return slices.ContainsFunc(shapeDropped, func(part string) bool { return strings.Contains(k, part) })
}
