package oracle

// localpin.go names a local model down to its weights (iss-2610161734428291,
// itd-2610161733316518's local cells):
//
//   - oracle.api.<provider>.pins.<model>, on a block whose base URL is on
//     this machine, is a local pin: the runtime serving it (ollama, llama.cpp
//     or mlx), the runtime's version, the weight file's SHA-256 (or Ollama's
//     model digest), the quantisation, the context length it is served at and
//     the KV cache's type. It sits where an OpenRouter pin does, in the
//     machine's provider block, so a repository never declares one.
//   - abcd never hashes a weight file of its own accord: the person declares
//     the digest, and the probe compares it with the runtime's own where the
//     runtime reports one (Ollama does; llama-server and mlx_lm.server do not).
//   - Before a pinned local route is first used, the server is probed once in
//     the process (openaiapi.ProbeLocal): what it reports is compared with the
//     pin, and a field it reports otherwise refuses the route, naming the
//     field. A route that runs a tool-using role (every role a runner runs,
//     the implementer among them) is also refused when the server reports no
//     tool calls for the model, when Ollama cannot say what context it serves
//     (it defaults to 4096 and truncates a longer prompt silently), or when a
//     llama-server pin declares a q4 KV cache, which degrades tool calling.
//     The probe is bounded, made only to this machine, and its failure sends
//     nothing anywhere else.
//   - The record carries the pin on the asked side and the probe's report on
//     the served side: a pinned local field the server confirmed is pinned,
//     one it did not report is not verified, and each served local field is
//     self-reported (the server's own word) or not reported.

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// MaxContextLength bounds a pinned context length.
const MaxContextLength = 1 << 24

var (
	// localVersionRe is a runtime's version as it prints one (0.14.2, b9860).
	localVersionRe = regexp.MustCompile(`^v?[0-9A-Za-z][0-9A-Za-z.+-]{0,31}$`)
	// localDigestRe is a SHA-256 in hex, optionally sha256:-prefixed.
	localDigestRe = regexp.MustCompile(`^(sha256:)?[0-9A-Fa-f]{64}$`)
	// localTokenRe is a quantisation or a cache type (Q4_K_M, 4bit, q8_0).
	localTokenRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,31}$`)
)

// LocalPin is one local model's pin, as the configuration declares it and a
// record carries it.
type LocalPin struct {
	Runtime        string `json:"runtime"`
	RuntimeVersion string `json:"runtime_version"`
	// WeightsSHA256 is the weight file's SHA-256 or Ollama's model digest,
	// in lower-case hex.
	WeightsSHA256 string `json:"weights_sha256"`
	Quantisation  string `json:"quantisation"`
	ContextLength int    `json:"context_length"`
	KVCacheType   string `json:"kv_cache_type"`
}

// LocalServed is what the probe read from the server before the route was
// used: each value the server's own word, empty (0, nil) where it reported
// none.
type LocalServed struct {
	Model          string `json:"model,omitempty"`
	RuntimeVersion string `json:"runtime_version,omitempty"`
	WeightsSHA256  string `json:"weights_sha256,omitempty"`
	Quantisation   string `json:"quantisation,omitempty"`
	ContextLength  int    `json:"context_length,omitempty"`
	KVCacheType    string `json:"kv_cache_type,omitempty"`
	ToolCalls      *bool  `json:"tool_calls,omitempty"`
}

// localPinFile is a local pin as the file spells it, decoded strictly.
type localPinFile struct {
	Runtime        string `json:"runtime"`
	RuntimeVersion string `json:"runtime_version"`
	WeightsSHA256  string `json:"weights_sha256"`
	Quantisation   string `json:"quantisation"`
	ContextLength  int    `json:"context_length"`
	KVCacheType    string `json:"kv_cache_type"`
}

// readLocalPins checks a local block's pins: each pinned model on its list,
// and every field of the pin declared, since a pin that leaves one out names
// a cell no re-run can find again.
func readLocalPins(models []string, raw map[string]localPinFile) (map[string]LocalPin, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(raw))
	for m := range raw {
		names = append(names, m)
	}
	sort.Strings(names)
	pins := map[string]LocalPin{}
	for _, m := range names {
		where := fmt.Sprintf("%s.%s", pinsKey, layered.BoundKey(m))
		pf := raw[m]
		switch {
		case !slices.Contains(models, m):
			return nil, fmt.Errorf("%s: the model is not on its models, and a block pins only a model it serves", where)
		case !slices.Contains(openaiapi.Runtimes, pf.Runtime):
			return nil, fmt.Errorf("%s: runtime %q is not one abcd probes (%s)", where, layered.BoundKey(pf.Runtime),
				strings.Join(openaiapi.Runtimes, ", "))
		case !localVersionRe.MatchString(pf.RuntimeVersion):
			return nil, fmt.Errorf("%s: runtime_version names the runtime's version as it prints it (0.14.2, b9860)", where)
		case !localDigestRe.MatchString(pf.WeightsSHA256):
			return nil, fmt.Errorf("%s: weights_sha256 is the weight file's SHA-256, or Ollama's model digest, in hex "+
				"(shasum -a 256 <file>; ollama list shows the digest's start)", where)
		case !localTokenRe.MatchString(pf.Quantisation):
			return nil, fmt.Errorf("%s: quantisation names the weights' quantisation (Q4_K_M, 4bit, bf16)", where)
		case pf.ContextLength <= 0 || pf.ContextLength > MaxContextLength:
			return nil, fmt.Errorf("%s: context_length is the context the server serves the model at, from 1 to %d", where, MaxContextLength)
		case !localTokenRe.MatchString(pf.KVCacheType):
			return nil, fmt.Errorf("%s: kv_cache_type names the KV cache's type (f16, q8_0)", where)
		}
		pins[m] = LocalPin{Runtime: pf.Runtime, RuntimeVersion: pf.RuntimeVersion,
			WeightsSHA256: strings.ToLower(strings.TrimPrefix(pf.WeightsSHA256, "sha256:")),
			Quantisation:  pf.Quantisation, ContextLength: pf.ContextLength, KVCacheType: pf.KVCacheType}
	}
	return pins, nil
}

// LocalPinFor returns the local pin this machine sets for model on provider.
func (c *APIConfig) LocalPinFor(provider, model string) (LocalPin, bool) {
	p, ok := c.providers[provider]
	if !ok {
		return LocalPin{}, false
	}
	pin, ok := p.LocalPins[model]
	return pin, ok
}

// IsLocal reports whether provider is a configured block whose base URL is on
// this machine.
func (c *APIConfig) IsLocal(provider string) bool {
	p, ok := c.providers[provider]
	return ok && openaiapi.IsLocal(p.BaseURL)
}

// probes is this process's reports, one per provider, base URL and model:
// a server is probed once before its first use, and a probe that read
// nothing is made again at the next use.
var probes = struct {
	sync.Mutex
	m map[string]*probeOnce
}{m: map[string]*probeOnce{}}

type probeOnce struct {
	sync.Mutex
	done bool
	rep  openaiapi.LocalReport
}

// probe is openaiapi's probe through client, a variable so a test counts the
// probes made.
var probe = func(ctx context.Context, client *openaiapi.Client, runtime, model string) (openaiapi.LocalReport, error) {
	return client.ProbeLocal(ctx, runtime, model)
}

// probeLocal is the report of p's server for model, read once per process.
func probeLocal(ctx context.Context, client *openaiapi.Client, p Provider, pin LocalPin, model string) (openaiapi.LocalReport, error) {
	key := p.Name + "\x00" + p.BaseURL + "\x00" + pin.Runtime + "\x00" + model
	probes.Lock()
	o, ok := probes.m[key]
	if !ok {
		o = &probeOnce{}
		probes.m[key] = o
	}
	probes.Unlock()
	o.Lock()
	defer o.Unlock()
	if o.done {
		return o.rep, nil
	}
	rep, err := probe(ctx, client, pin.Runtime, model)
	if err != nil {
		return openaiapi.LocalReport{}, err
	}
	o.rep, o.done = rep, true
	return rep, nil
}

// ProbeLocal probes the server behind a pinned local route before it is used
// and checks what it reports against the pin; build says the route runs a
// tool-using role. It returns the pin and what the server reported, or
// neither when the route is not a pinned local one; a route the probe refuses
// is an error naming the field and the setting that would fix it. The block's
// key, when it names one, is resolved through creds (the person's store when
// nil) and sent to this machine alone.
func (c *APIConfig) ProbeLocal(ctx context.Context, creds credential.Source, provider, model string, build bool) (*LocalPin, *LocalServed, error) {
	p, ok := c.providers[provider]
	if !ok {
		return nil, nil, nil
	}
	pin, ok := p.LocalPins[model]
	if !ok {
		return nil, nil, nil
	}
	key, err := resolveKey(creds, p)
	if err != nil {
		return nil, nil, err
	}
	client, err := openaiapi.New(p.BaseURL, key)
	if err != nil {
		return nil, nil, fmt.Errorf("oracle adapter: provider %s: %w", provider, err)
	}
	served, err := checkLocal(ctx, client, p, pin, model, build)
	if err != nil {
		return nil, nil, err
	}
	return &pin, served, nil
}

// checkLocal probes p's server through client and holds its report to pin.
func checkLocal(ctx context.Context, client *openaiapi.Client, p Provider, pin LocalPin, model string, build bool) (*LocalServed, error) {
	route := fmt.Sprintf("oracle adapter: %s/%s is pinned to %s on this machine, and", p.Name, model, runtimeLabel(pin.Runtime))
	rep, err := probeLocal(ctx, client, p, pin, model)
	if err != nil {
		return nil, fmt.Errorf("%s the probe before its first use failed, so nothing is sent: %w", route, err)
	}
	s := &LocalServed{Model: rep.Model, RuntimeVersion: rep.Version, WeightsSHA256: rep.Digest,
		Quantisation: rep.Quantisation, ContextLength: rep.Context, KVCacheType: rep.KVCacheType, ToolCalls: rep.ToolCalls}
	differs := func(field, pinned, served, fix string) error {
		return fmt.Errorf("%s the server reports %s %s where the pin names %s, so the route is refused: %s",
			route, field, served, pinned, fix)
	}
	if s.RuntimeVersion != "" && normVersion(s.RuntimeVersion) != normVersion(pin.RuntimeVersion) {
		return nil, differs("runtime_version", pin.RuntimeVersion, s.RuntimeVersion,
			"serve the model with the version pinned, or pin the version that serves it")
	}
	if s.WeightsSHA256 != "" && s.WeightsSHA256 != pin.WeightsSHA256 {
		return nil, differs("weights_sha256", pin.WeightsSHA256, s.WeightsSHA256,
			"the model under that name is other weights; pull the pinned ones, or pin these")
	}
	if s.Quantisation != "" && !strings.EqualFold(s.Quantisation, pin.Quantisation) {
		return nil, differs("quantisation", pin.Quantisation, s.Quantisation, "serve the quantisation pinned, or pin this one")
	}
	if s.ContextLength != 0 && s.ContextLength != pin.ContextLength {
		return nil, differs("context_length", fmt.Sprint(pin.ContextLength), fmt.Sprint(s.ContextLength), contextFix(pin))
	}
	if s.KVCacheType != "" && !strings.EqualFold(s.KVCacheType, pin.KVCacheType) {
		return nil, differs("kv_cache_type", pin.KVCacheType, s.KVCacheType, "serve the cache type pinned, or pin this one")
	}
	if !build {
		return s, nil
	}
	tool := fmt.Sprintf("%s it runs a tool-using role, and", route)
	switch {
	case s.ToolCalls != nil && !*s.ToolCalls:
		return nil, fmt.Errorf("%s the server reports that the model's template takes no tool calls, so the route is refused: %s",
			tool, toolFix(pin, model))
	case pin.Runtime == openaiapi.RuntimeOllama && s.ContextLength == 0:
		return nil, fmt.Errorf("%s Ollama does not say what context it serves the model at: it defaults to 4096 and "+
			"truncates a longer prompt silently, so the route is refused: %s", tool, contextFix(pin))
	case pin.Runtime == openaiapi.RuntimeLlamaCpp && quantisedCache(pin.KVCacheType):
		return nil, fmt.Errorf("%s the pin declares a %s KV cache, which degrades tool calling on llama-server, so the route is "+
			"refused: start llama-server with --cache-type-k and --cache-type-v f16 or q8_0, and pin that", tool, pin.KVCacheType)
	}
	return s, nil
}

// quantisedCache reports whether a KV-cache type is a 4-bit one.
func quantisedCache(t string) bool {
	t = strings.ToLower(t)
	return strings.HasPrefix(t, "q4") || strings.HasPrefix(t, "iq4")
}

// normVersion is a version as two spellings of it compare: without a leading
// v, or llama.cpp's b.
func normVersion(v string) string {
	v = strings.ToLower(v)
	if len(v) > 1 && (v[0] == 'v' || v[0] == 'b') && v[1] >= '0' && v[1] <= '9' {
		v = v[1:]
	}
	return v
}

// runtimeLabel is a runtime as a refusal names it.
func runtimeLabel(runtime string) string {
	switch runtime {
	case openaiapi.RuntimeOllama:
		return "Ollama"
	case openaiapi.RuntimeLlamaCpp:
		return "llama-server"
	case openaiapi.RuntimeMLX:
		return "mlx_lm.server"
	}
	return runtime
}

// contextFix is the setting that serves the pinned context on pin's runtime.
func contextFix(pin LocalPin) string {
	n := pin.ContextLength
	switch pin.Runtime {
	case openaiapi.RuntimeOllama:
		return fmt.Sprintf("set PARAMETER num_ctx %d in the model's Modelfile (ollama create), or start the server with "+
			"OLLAMA_CONTEXT_LENGTH=%d and load the model so ollama ps reports it", n, n)
	case openaiapi.RuntimeLlamaCpp:
		return fmt.Sprintf("start llama-server with --ctx-size %d per slot (the size is shared across --parallel slots)", n)
	}
	return fmt.Sprintf("serve the model at a context of %d, or pin the one it is served at", n)
}

// toolFix is the setting that gives the model tool calls on pin's runtime.
func toolFix(pin LocalPin, model string) string {
	switch pin.Runtime {
	case openaiapi.RuntimeOllama:
		return fmt.Sprintf("use a model whose template takes tools (ollama show %s lists tools under its capabilities)", layered.BoundKey(model))
	case openaiapi.RuntimeLlamaCpp:
		return "start llama-server with --jinja and a chat template that takes tool calls (--chat-template-file)"
	}
	return "serve the model with a chat template that takes tool calls"
}

// LocalAsked lists the asked side's local fields of a route to this machine:
// each not configured without a pin, and with one, each the server did not
// confirm not verified (a field it reported otherwise refused the route).
func LocalAsked(un *Unpin, pin *LocalPin, s *LocalServed) {
	if pin == nil {
		for _, f := range LocalFields {
			un.Add(f, NotConfigured)
		}
		return
	}
	if s == nil {
		s = &LocalServed{}
	}
	un.Unless(s.WeightsSHA256 != "", AskedWeightsSHA256, NotVerified)
	un.Unless(s.Quantisation != "", AskedQuantisation, NotVerified)
	un.Unless(s.RuntimeVersion != "", AskedRuntimeVersion, NotVerified)
}

// LocalContext lists the asked context of a route to this machine: not
// configured without a pin, not verified when the server did not report the
// context it serves, and pinned when it confirmed the pin's.
func LocalContext(un *Unpin, pin *LocalPin, s *LocalServed) {
	switch {
	case pin == nil:
		un.Add(AskedContext, NotConfigured)
	case s == nil || s.ContextLength == 0:
		un.Add(AskedContext, NotVerified)
	}
}

// LocalServedList lists the served side's local fields of a route to this
// machine: each value the probe read is the server's own word, and each it
// did not read is not reported.
func LocalServedList(un *Unpin, s *LocalServed) {
	if s == nil {
		s = &LocalServed{}
	}
	for _, f := range []struct {
		f    Field
		have bool
	}{
		{ServedWeightsSHA256, s.WeightsSHA256 != ""}, {ServedQuantisation, s.Quantisation != ""},
		{ServedRuntimeVersion, s.RuntimeVersion != ""}, {ServedContextLength, s.ContextLength != 0},
		{ServedKVCacheType, s.KVCacheType != ""}, {ServedToolCalls, s.ToolCalls != nil},
	} {
		if f.have {
			un.Add(f.f, SelfReported)
		} else {
			un.Add(f.f, NotReported)
		}
	}
}
