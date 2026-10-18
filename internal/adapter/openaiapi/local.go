package openaiapi

// local.go is the adapter's half of iss-2610161734428291: what a server on
// this machine says it serves, read through each runtime's own endpoints
// before a pinned local route is used (internal/core/oracle decides what the
// report must say, and when).
//
// Each runtime reports a different part of what a pin names (the endpoints
// read in their documentation and checked against the versions on the
// machine the issue names, 2026-10-16):
//
//   - Ollama (0.14): GET /api/version is its version; POST /api/show is the
//     model's quantisation (details.quantization_level), its Modelfile's
//     parameters (num_ctx) and whether its template takes tools
//     (capabilities); GET /api/tags is the digest of each pulled model; and
//     GET /api/ps is the context each loaded model is served at
//     (context_length). None of them loads a model. The KV-cache type is the
//     server's environment (OLLAMA_KV_CACHE_TYPE), which no endpoint reports.
//   - llama-server (b9860): GET /props is its build (build_info), the context
//     each slot serves (default_generation_settings.n_ctx), the model file's
//     path, and whether its chat template calls tools (chat_template_caps).
//     It reports no digest, quantisation or KV-cache type.
//   - mlx_lm.server (0.31): GET /v1/models lists the models it serves, and
//     nothing else is reported.
//
// The probe is a few small GETs and one POST, made only to a base URL on this
// machine, through the client's own transport (no redirect followed, the
// block's key as the bearer token when it names one), each body bounded and
// the whole probe bounded by LocalProbeTimeout. It never asks a model for
// anything, so no inference runs.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The runtimes a local pin names.
const (
	RuntimeOllama   = "ollama"
	RuntimeLlamaCpp = "llama.cpp"
	RuntimeMLX      = "mlx"
)

// Runtimes is every runtime a local pin may name, in the order a refusal
// lists them.
var Runtimes = []string{RuntimeOllama, RuntimeLlamaCpp, RuntimeMLX}

// LocalProbeTimeout bounds one probe end to end, every request in it.
const LocalProbeTimeout = 5 * time.Second

// errProbeTimeout is the cause LocalProbeTimeout ends a probe with.
var errProbeTimeout = errors.New("local probe timeout")

// LocalReport is what a local server says it serves. Every string is the
// server's own text, scrubbed of the key, cleaned and bounded; a value the
// runtime did not report readably is empty, 0, or nil.
type LocalReport struct {
	// Version is the runtime's version (llama-server's build, b9860).
	Version string
	// Digest is the model's sha256 in hex, as the runtime names it.
	Digest string
	// Quantisation is the model's quantisation as the runtime names it.
	Quantisation string
	// Context is the context length the model is served at.
	Context int
	// KVCacheType is the KV cache's type.
	KVCacheType string
	// ToolCalls is whether the model's template takes tool calls.
	ToolCalls *bool
	// Model is the model as the server names it: Ollama's name, the
	// llama-server model file's name, mlx's listed id.
	Model string
}

// numCtxRe is a Modelfile parameter line setting num_ctx.
var numCtxRe = regexp.MustCompile(`(?m)^\s*num_ctx\s+([0-9]{1,9})\s*$`)

// ProbeLocal reads what the server at the client's base URL says it serves
// for model, through runtime's own endpoints. A base URL off this machine is
// never probed. A server that cannot be reached, does not answer as runtime,
// or does not serve model fails the probe; a detail it does not report is
// left empty in the report, never a failure.
func (c *Client) ProbeLocal(ctx context.Context, runtime, model string) (LocalReport, error) {
	if c.base == nil || !loopback(c.base.Hostname()) {
		return LocalReport{}, c.fail("a local server is probed only on this machine (localhost or a loopback address), and the base URL is not")
	}
	pctx, cancel := context.WithTimeoutCause(ctx, c.probeWait, errProbeTimeout)
	defer cancel()
	p := prober{c: c, parent: ctx, ctx: pctx, runtime: runtime}
	switch runtime {
	case RuntimeOllama:
		return p.ollama(model)
	case RuntimeLlamaCpp:
		return p.llamaServer()
	case RuntimeMLX:
		return p.mlx(model)
	}
	return LocalReport{}, c.fail(fmt.Sprintf("%q is not a runtime abcd probes (%s)", bound(runtime), strings.Join(Runtimes, ", ")))
}

// prober is one probe's requests.
type prober struct {
	c           *Client
	parent, ctx context.Context
	runtime     string
}

// root is the server's own root: the base URL without the /v1 its
// OpenAI-compatible endpoints sit under.
func (p prober) root(rel string) string {
	base := strings.TrimSuffix(basePath(p.c.base), "/v1")
	return under(p.c.base, base+rel)
}

// ask makes one request and returns the body of a 200, or the status alone.
func (p prober) ask(method, addr string, body []byte) ([]byte, int, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(p.ctx, method, addr, rd)
	if err != nil {
		return nil, 0, p.c.fail("the probe's request could not be built")
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if p.c.key != "" {
		req.Header.Set("Authorization", "Bearer "+p.c.key)
	}
	resp, err := p.c.hc.Do(req)
	if err != nil {
		switch {
		case p.ctx.Err() != nil:
			return nil, 0, p.ended()
		case errors.Is(err, errRedirect):
			return nil, 0, p.c.fail(p.c.host + " answered the probe with a redirect, which abcd never follows")
		}
		return nil, 0, p.c.fail(p.c.host + " could not be reached to probe what it serves")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, nil
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes+1))
	switch {
	case err != nil && p.ctx.Err() != nil:
		return nil, 0, p.ended()
	case err != nil:
		return nil, 0, p.c.fail("the probe's answer from " + p.c.host + " was cut off")
	case len(raw) > MaxResponseBytes:
		return nil, 0, p.c.fail(fmt.Sprintf("%s answered the probe with more than %d bytes, so it is refused unread", p.c.host, MaxResponseBytes))
	}
	return raw, http.StatusOK, nil
}

// ended is a probe its bound or its caller ended.
func (p prober) ended() error {
	if p.parent.Err() != nil {
		return p.c.fail("the probe was not waited for: the call it belongs to ended")
	}
	return p.c.fail(fmt.Sprintf("%s did not answer the probe within %s", p.c.host, p.c.probeWait))
}

// notRuntime is the failure of a server that does not answer as the pin's
// runtime.
func (p prober) notRuntime(what string, code int) error {
	return p.c.fail(fmt.Sprintf("%s does not answer as %s: %s answered HTTP %d", p.c.host, runtimeName(p.runtime), what, code))
}

// runtimeName is the runtime as its own documentation names it.
func runtimeName(runtime string) string {
	switch runtime {
	case RuntimeOllama:
		return "Ollama"
	case RuntimeLlamaCpp:
		return "llama-server"
	case RuntimeMLX:
		return "mlx_lm.server"
	}
	return runtime
}

// clean is a string the server supplied, as a report carries it.
func (p prober) clean(s string) string { return cleanModel(p.c.scrub(s)) }

func (p prober) ollama(model string) (LocalReport, error) {
	raw, code, err := p.ask(http.MethodGet, p.root("/api/version"), nil)
	if err != nil {
		return LocalReport{}, err
	}
	var ver struct {
		Version json.RawMessage `json:"version"`
	}
	if code != http.StatusOK || json.Unmarshal(raw, &ver) != nil || ver.Version == nil {
		return LocalReport{}, p.notRuntime("GET /api/version", code)
	}
	rep := LocalReport{Version: p.clean(lenientString(ver.Version))}
	body, _ := json.Marshal(map[string]string{"model": model})
	raw, code, err = p.ask(http.MethodPost, p.root("/api/show"), body)
	switch {
	case err != nil:
		return LocalReport{}, err
	case code == http.StatusNotFound:
		return LocalReport{}, p.c.fail(fmt.Sprintf("Ollama at %s has no model %s; `ollama pull %s` fetches it", p.c.host, bound(model), bound(model)))
	case code != http.StatusOK:
		return LocalReport{}, p.notRuntime("POST /api/show", code)
	}
	var show struct {
		Parameters   json.RawMessage `json:"parameters"`
		Capabilities *[]string       `json:"capabilities"`
		Details      struct {
			Quantisation json.RawMessage `json:"quantization_level"`
		} `json:"details"`
	}
	if json.Unmarshal(raw, &show) != nil {
		return LocalReport{}, p.c.fail(p.c.host + " answered POST /api/show with something that is not Ollama's model details")
	}
	rep.Model = p.clean(model)
	rep.Quantisation = p.clean(lenientString(show.Details.Quantisation))
	if show.Capabilities != nil {
		tools := false
		for _, cap := range *show.Capabilities {
			tools = tools || cap == "tools"
		}
		rep.ToolCalls = &tools
	}
	if m := numCtxRe.FindStringSubmatch(lenientString(show.Parameters)); m != nil {
		rep.Context, _ = strconv.Atoi(m[1])
	}
	// The digest of the pulled model, and the context of the loaded one, which
	// is what serves a request now and so stands over the Modelfile's.
	type listed struct {
		Name    string          `json:"name"`
		Model   string          `json:"model"`
		Digest  json.RawMessage `json:"digest"`
		Context json.RawMessage `json:"context_length"`
	}
	read := func(rel string) ([]listed, error) {
		raw, code, err := p.ask(http.MethodGet, p.root(rel), nil)
		if err != nil || code != http.StatusOK {
			return nil, err
		}
		var l struct {
			Models []listed `json:"models"`
		}
		_ = json.Unmarshal(raw, &l)
		return l.Models, nil
	}
	is := func(l listed) bool { return ollamaSame(l.Name, model) || ollamaSame(l.Model, model) }
	tags, err := read("/api/tags")
	if err != nil {
		return LocalReport{}, err
	}
	for _, l := range tags {
		if is(l) {
			if d := strings.ToLower(lenientString(l.Digest)); sha256Re.MatchString(d) {
				rep.Digest = d
			}
			break
		}
	}
	loaded, err := read("/api/ps")
	if err != nil {
		return LocalReport{}, err
	}
	for _, l := range loaded {
		if n := readCount(l.Context); is(l) && n != nil && *n > 0 {
			rep.Context = *n
			break
		}
	}
	return rep, nil
}

// sha256Re is a SHA-256 in lower-case hex.
var sha256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ollamaSame reports whether Ollama's name a is the model b names: a name
// without a tag is the :latest one.
func ollamaSame(a, b string) bool {
	tagged := func(s string) string {
		if s == "" || strings.Contains(path.Base(s), ":") {
			return s
		}
		return s + ":latest"
	}
	return a != "" && tagged(a) == tagged(b)
}

func (p prober) llamaServer() (LocalReport, error) {
	raw, code, err := p.ask(http.MethodGet, p.root("/props"), nil)
	switch {
	case err != nil:
		return LocalReport{}, err
	case code == http.StatusUnauthorized:
		return LocalReport{}, p.c.fail(p.c.host + " refused the probe's GET /props without a key it accepts; name the key llama-server " +
			"was started with (--api-key) in the provider block")
	case code != http.StatusOK:
		return LocalReport{}, p.notRuntime("GET /props", code)
	}
	var props struct {
		BuildInfo json.RawMessage `json:"build_info"`
		ModelPath json.RawMessage `json:"model_path"`
		Defaults  struct {
			NCtx json.RawMessage `json:"n_ctx"`
		} `json:"default_generation_settings"`
		Caps map[string]json.RawMessage `json:"chat_template_caps"`
	}
	if json.Unmarshal(raw, &props) != nil || props.BuildInfo == nil {
		return LocalReport{}, p.c.fail(p.c.host + " answered GET /props with something that is not llama-server's properties")
	}
	rep := LocalReport{}
	build, _, _ := strings.Cut(lenientString(props.BuildInfo), "-")
	rep.Version = p.clean(build)
	if mp := lenientString(props.ModelPath); mp != "" {
		rep.Model = p.clean(path.Base(strings.ReplaceAll(mp, `\`, "/")))
	}
	if n := readCount(props.Defaults.NCtx); n != nil && *n > 0 {
		rep.Context = *n
	}
	for _, key := range []string{"supports_tool_calls", "supports_tools"} {
		var b bool
		if r, ok := props.Caps[key]; ok && json.Unmarshal(r, &b) == nil && !isNull(r) {
			rep.ToolCalls = &b
			break
		}
	}
	return rep, nil
}

func (p prober) mlx(model string) (LocalReport, error) {
	raw, code, err := p.ask(http.MethodGet, p.c.models, nil)
	if err != nil {
		return LocalReport{}, err
	}
	var list struct {
		Data *[]struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if code != http.StatusOK || json.Unmarshal(raw, &list) != nil || list.Data == nil {
		return LocalReport{}, p.notRuntime("GET /v1/models", code)
	}
	for _, m := range *list.Data {
		if m.ID == model {
			return LocalReport{Model: p.clean(m.ID)}, nil
		}
	}
	return LocalReport{}, p.c.fail(fmt.Sprintf("mlx_lm.server at %s does not list %s among its models", p.c.host, bound(model)))
}

// parsedBase is the base URL New admitted, parsed; kept so the probe derives
// the runtime's own endpoints from it.
func parsedBase(u *url.URL) *url.URL {
	d := *u
	return &d
}
