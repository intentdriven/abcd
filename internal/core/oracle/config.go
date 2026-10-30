package oracle

// config.go is the provider configuration of the OpenAI-compatible API adapter
// (itd-2609081951381895, spc-2609221011153746 scope 1) and the resolver that
// validates it when it is read (adr-2609221009491186, whose Decision 2
// adr-2609300107513982 supersedes):
//
//   - oracle.api.<provider> is a provider block: base_url (pinned; https, or
//     http to this machine), key (a credential NAME, resolved through
//     internal/core/credential; omitted for a server that needs none) and
//     models (the allowlist: the only models the provider may serve, and
//     alone what decides which models it serves). A block
//     is read from the machine's ~/.abcd.noindex/config.json alone. A repository
//     declaring one is refused, because a block names the address a key is sent
//     to, and a checkout must never be able to aim the person's key at a
//     server of its choosing.
//   - oracle.denylist is optional and the configuration's own: abcd bundles
//     no vendor denylist. An entry the repository or the machine writes
//     refuses a model it matches even when a provider lists it; the layers
//     are a union, so neither removes the other's entries.
//   - oracle.roles.<agent> and oracle.judgements.<type> point a role (an agent
//     in the roster) or a judgement type at <provider>/<model>. The machine
//     may point at any provider it configures. The repository may point only
//     at a provider that holds no key, and there its route wins per name; a
//     repository route to a provider that holds a key is skipped, because only
//     a route the person sets on their own machine may spend their paid key
//     (the product thinker's ruling AA(b) of 2026-09-29, which reverses the
//     route half of itd-2609081951381895 Decision 8).
//   - A judgement type's route carries a mode (itd-2609221009495079): `host`
//     keeps the type on the host and points it nowhere, `shadow:<provider>/
//     <model>` runs the decision adapter beside the host, which still decides,
//     and a bare `<provider>/<model>` turns the type over to the adapter. A
//     shadow route is judged exactly as one turned on: the same skips, the same
//     allowlist and the same denylist, before any call. A role takes no mode.
//
// Every route is checked here, before any call: a repository route to a
// provider that holds a key is skipped with a diagnostic naming the machine's
// file as where to set it, as is a repository route that is not
// <provider>/<model> or whose name is not a plain name (ruling CD2 of
// 2026-09-29), and a repository route to a model a keyless provider does not
// list (ruling CD3 of 2026-10-02); the machine's own route to a model its
// provider does not list is refused naming the list, and a model an
// oracle.denylist entry matches is refused naming the entry, from any layer.
// A route naming a provider this machine has not configured is a diagnostic,
// not a refusal: a repository's such route is skipped where the machine routes
// the same name, so the owner's setting applies (ruling CD4 of 2026-10-02),
// and otherwise the step stays on the host, exactly as it would with nothing
// configured (adr-25, amended 2026-10-02).
//
// Like the rest of the package, the resolver never writes, never reaches a
// network and never prints.

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The configuration keys, as every refusal names them.
const (
	apiKey        = "oracle.api"
	denylistKey   = "oracle.denylist"
	rolesKey      = "oracle.roles"
	judgementsKey = "oracle.judgements"
	// streamUsageKey is a provider block's opt-out of usage in the stream.
	streamUsageKey = "stream_usage"
	// readGenerationKey is a provider block's opt-in to the generation
	// readback (pin.go).
	readGenerationKey = "read_generation"
	// logprobModelsKey is a provider block's list of the listed models a
	// judgement routed to is scored by log-probability (ScoresByLogprob).
	logprobModelsKey = "logprob_models"
)

// Bounds on what the configuration may carry.
const (
	// MaxProviders bounds the provider blocks one machine declares.
	MaxProviders = 16
	// MaxModels bounds one provider's allowlist.
	MaxModels = 64
	// MaxDenylist bounds the entries one layer adds to the denylist.
	MaxDenylist = 64
)

var (
	providerNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	// modelRe is a model identifier as providers spell them: slash-separated
	// segments (a vendor path), an optional variant after a colon, OpenRouter's
	// ~ alias prefix; no empty segment. Its length is bounded by validModel.
	modelRe = regexp.MustCompile(`^[A-Za-z0-9~][A-Za-z0-9._:@+~-]*(/[A-Za-z0-9._:@+~-]+)*$`)
	// vendorRe is a denylist vendor prefix: vendor/*.
	vendorRe     = regexp.MustCompile(`^~?[A-Za-z0-9][A-Za-z0-9._-]{0,63}/\*$`)
	routeNameRe  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	judgementsRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
)

// validModel reports whether m is a model identifier of at most 128 bytes.
func validModel(m string) bool { return len(m) <= 128 && modelRe.MatchString(m) }

// Provider is one configured provider block.
type Provider struct {
	Name    string   `json:"name"`
	BaseURL string   `json:"base_url"`
	Key     string   `json:"key,omitempty"`
	Models  []string `json:"models"`
	Origin  string   `json:"origin"`
	// StreamUsage is the block's stream_usage: false stops a call asking for
	// usage in the stream (openaiapi.StreamUsageField), for a server that
	// refuses the field. nil, the default, asks (itd-166 decision 4).
	StreamUsage *bool `json:"stream_usage,omitempty"`
	// Pins is the block's pins, by the listed model each holds to one
	// OpenRouter provider (pin.go); nil when it pins none.
	Pins map[string]Pin `json:"pins,omitempty"`
	// LocalPins is a block on this machine's pins, by the listed model each
	// names down to its weights (localpin.go); nil when it pins none.
	LocalPins map[string]LocalPin `json:"local_pins,omitempty"`
	// ReadGeneration is the block's read_generation: true makes every call
	// through it read OpenRouter's generation record back once it is
	// answered (pin.go), a request of its own. false, the default, makes
	// none; true is read only on a block whose base URL is OpenRouter's.
	ReadGeneration bool `json:"read_generation,omitempty"`
	// LogprobModels is the block's logprob_models: the listed models a
	// judgement routed to is asked as a small text model whose closed
	// options are scored by the log-probability it gives each option's
	// answer token, rather than for a typed answer (itd-2609221009495079's
	// second candidate). Empty, the default, scores none that way.
	LogprobModels []string `json:"logprob_models,omitempty"`
}

// AsksStreamUsage reports whether a call through p asks for usage in the
// stream: unless its block sets stream_usage to false.
func (p Provider) AsksStreamUsage() bool { return p.StreamUsage == nil || *p.StreamUsage }

// Target is where a role or a judgement type is pointed: a configured
// provider and a model on its list.
type Target struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	// Origin names the file the route came from.
	Origin string `json:"origin"`
	// Mode is a judgement type's mode, JudgementShadow or JudgementOn; ""
	// on a role's target.
	Mode string `json:"mode,omitempty"`
}

// The modes a judgement type's route carries (itd-2609221009495079). A role's
// target carries none.
const (
	// JudgementHost keeps the type on the host: the route points nowhere.
	JudgementHost = "host"
	// JudgementShadow runs the adapter beside the host, which decides.
	JudgementShadow = "shadow"
	// JudgementOn turns the type over to the adapter.
	JudgementOn = "on"
)

// shadowPrefix spells a shadow route in the configuration.
const shadowPrefix = JudgementShadow + ":"

// String is the target as <provider>/<model>, whatever its mode.
func (t Target) String() string { return t.Provider + "/" + t.Model }

// Route is the route as the configuration spells it: a shadow judgement's with
// its shadow: prefix.
func (t Target) Route() string {
	if t.Mode == JudgementShadow {
		return shadowPrefix + t.String()
	}
	return t.String()
}

// routeValue splits a route's mode from its <provider>/<model> text. Only a
// judgement type's route has a mode: host (target ""), shadow (the prefix
// cut) or on; a role's text is returned whole, with no mode.
func routeValue(key, text string) (target, mode string) {
	if key != judgementsKey {
		return text, ""
	}
	if text == JudgementHost {
		return "", JudgementHost
	}
	if t, ok := strings.CutPrefix(text, shadowPrefix); ok {
		return t, JudgementShadow
	}
	return text, JudgementOn
}

// DenyEntry is one denylist entry and the layer that added it.
type DenyEntry struct {
	Pattern string `json:"pattern"`
	Origin  string `json:"origin"`
}

// APIConfig is the adapter configuration one invocation read.
type APIConfig struct {
	providers  map[string]Provider
	denylist   []DenyEntry
	roles      map[string]Target
	judgements map[string]Target
	// bundled is the providers oracle.bundled_context_providers names
	// (selfcontained.go), read from the machine layer alone.
	bundled map[string]bool
	// Diagnostics are the non-fatal reports the read produced, one line each,
	// for a front door to print on stderr: a route naming a provider this
	// machine has not configured, a role outside the roster, and a skipped
	// repository route (one to a provider that holds a key, one whose name or
	// value is malformed, one to a model a keyless provider does not list, and
	// one to an unconfigured provider where the machine routes the name).
	Diagnostics []string
}

// providerFile is a provider block as the file spells it, decoded strictly.
type providerFile struct {
	BaseURL     string   `json:"base_url"`
	Key         *string  `json:"key"`
	Models      []string `json:"models"`
	StreamUsage *bool    `json:"stream_usage"`
	// Pins is pins, each decoded by the block it sits in: an OpenRouter
	// block's as a routing pin, a block on this machine's as a local pin.
	Pins map[string]json.RawMessage `json:"pins"`
	// ReadGeneration is read_generation, nil when the block omits it.
	ReadGeneration *bool `json:"read_generation"`
	// LogprobModels is logprob_models, nil when the block omits it.
	LogprobModels []string `json:"logprob_models"`
}

// LoadAPI reads the provider configuration through the layered resolver and
// validates all of it. A fault is an error naming the file and the key; none
// falls through to a default.
func LoadAPI(r layered.Roots) (*APIConfig, error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return nil, fmt.Errorf("oracle adapter: %w", err)
	}
	c := &APIConfig{providers: map[string]Provider{}, roles: map[string]Target{}, judgements: map[string]Target{}}
	if err := c.readDenylist(s); err != nil {
		return nil, err
	}
	if err := c.readProviders(s); err != nil {
		return nil, err
	}
	if err := c.readBundled(s); err != nil {
		return nil, err
	}
	if err := c.readRoutes(s, rolesKey, c.roles); err != nil {
		return nil, err
	}
	if err := c.readRoutes(s, judgementsKey, c.judgements); err != nil {
		return nil, err
	}
	return c, nil
}

// readDenylist reads oracle.denylist. abcd bundles no entry
// (adr-2609300107513982): with none written the list is empty and a provider's
// allowlist alone decides.
func (c *APIConfig) readDenylist(s *layered.Stack) error {
	found, err := s.Lookup(denylistKey)
	if err != nil {
		return fmt.Errorf("oracle adapter: %w", err)
	}
	// Every layer's entries apply, lowest first: the denylist is a union, so a
	// higher layer adds to it and never replaces a lower one.
	for i := len(found) - 1; i >= 0; i-- {
		fd := found[i]
		entries, err := layered.Decode[[]string](fd.Raw)
		if err != nil {
			return fmt.Errorf("oracle adapter: %s (%s layer): %s: %w", fd.Origin, fd.Layer, denylistKey, err)
		}
		if len(entries) > MaxDenylist {
			return fmt.Errorf("oracle adapter: %s (%s layer): %s has %d entries; a layer adds at most %d",
				fd.Origin, fd.Layer, denylistKey, len(entries), MaxDenylist)
		}
		for _, e := range entries {
			if !vendorRe.MatchString(e) && !validModel(e) {
				return fmt.Errorf("oracle adapter: %s (%s layer): %s entry %q is neither a vendor prefix (vendor/*) nor a model identifier",
					fd.Origin, fd.Layer, denylistKey, layered.BoundKey(e))
			}
			c.denylist = append(c.denylist, DenyEntry{Pattern: e, Origin: fd.Origin})
		}
	}
	return nil
}

func (c *APIConfig) readProviders(s *layered.Stack) error {
	if s.Present(layered.Repo) {
		found, err := s.Lookup(apiKey)
		if err != nil {
			return fmt.Errorf("oracle adapter: %w", err)
		}
		for _, fd := range found {
			if fd.Layer == layered.Repo {
				return fmt.Errorf("oracle adapter: %s (repo layer): %s is refused in a repository's configuration: "+
					"a provider block names the address a key is sent to, so only this machine's %s declares one",
					fd.Origin, apiKey, layered.Config.MachineOrigin())
			}
		}
	}
	names, err := s.Members(layered.Machine, apiKey)
	if err != nil {
		return fmt.Errorf("oracle adapter: %w", err)
	}
	origin := layered.Config.MachineOrigin()
	if len(names) > MaxProviders {
		return fmt.Errorf("oracle adapter: %s (machine layer): %s declares %d providers; a machine declares at most %d",
			origin, apiKey, len(names), MaxProviders)
	}
	for _, name := range names {
		if !providerNameRe.MatchString(name) {
			return fmt.Errorf("oracle adapter: %s (machine layer): %s.%s: a provider's name is lower case letters, digits, - and _",
				origin, apiKey, layered.BoundKey(name))
		}
		if name == Harness {
			return fmt.Errorf("oracle adapter: %s (machine layer): %s.%s: %q names the host's own leg and is reserved",
				origin, apiKey, name, Harness)
		}
	}
	if err := s.Claim(apiKey+".*", "base_url", "key", "models", streamUsageKey, pinsKey, readGenerationKey, logprobModelsKey); err != nil {
		return fmt.Errorf("oracle adapter: %w", err)
	}
	for _, name := range names {
		found, err := s.Lookup(apiKey + "." + name)
		if err != nil {
			return fmt.Errorf("oracle adapter: %w", err)
		}
		for _, fd := range found {
			if fd.Layer != layered.Machine {
				continue
			}
			p, err := c.provider(name, fd)
			if err != nil {
				return fmt.Errorf("oracle adapter: %s (machine layer): %s.%s: %w", origin, apiKey, name, err)
			}
			c.providers[name] = p
		}
	}
	return nil
}

// provider decodes and checks one block.
func (c *APIConfig) provider(name string, fd layered.Found) (Provider, error) {
	pf, err := layered.Decode[providerFile](fd.Raw)
	if err != nil {
		return Provider{}, err
	}
	if err := openaiapi.ValidateBaseURL(pf.BaseURL); err != nil {
		return Provider{}, err
	}
	p := Provider{Name: name, BaseURL: pf.BaseURL, Origin: fd.Origin, StreamUsage: pf.StreamUsage}
	if pf.Key != nil {
		if !credential.ValidName(*pf.Key) {
			return Provider{}, fmt.Errorf("key %q is not a plain credential name (lower case letters, digits, '.', '_' and '-'); "+
				"omit key for a server that needs none", layered.BoundKey(*pf.Key))
		}
		p.Key = *pf.Key
	}
	if len(pf.Models) == 0 {
		return Provider{}, fmt.Errorf("models is empty or absent; a provider serves only the models it lists, so a block lists at least one")
	}
	if len(pf.Models) > MaxModels {
		return Provider{}, fmt.Errorf("models has %d entries; a provider lists at most %d", len(pf.Models), MaxModels)
	}
	seen := map[string]bool{}
	for _, m := range pf.Models {
		if !validModel(m) {
			return Provider{}, fmt.Errorf("model %q is not a model identifier", layered.BoundKey(m))
		}
		if seen[m] {
			return Provider{}, fmt.Errorf("model %s is listed twice", m)
		}
		seen[m] = true
		if e, denied := Denied(c.denylist, m); denied {
			return Provider{}, deniedError(m, e)
		}
	}
	p.Models = append([]string(nil), pf.Models...)
	if err := p.readAnyPins(pf.Pins); err != nil {
		return Provider{}, err
	}
	if p.ReadGeneration, err = readGenerationSetting(p.BaseURL, pf.ReadGeneration); err != nil {
		return Provider{}, err
	}
	if p.LogprobModels, err = readLogprobModels(p.Models, pf.LogprobModels); err != nil {
		return Provider{}, err
	}
	return p, nil
}

// readLogprobModels checks a block's logprob_models: each a model the block
// lists, once. It marks how a listed model is asked, never what may be
// asked, so it admits nothing the models list does not.
func readLogprobModels(models, marked []string) ([]string, error) {
	if len(marked) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	for _, m := range marked {
		if !slices.Contains(models, m) {
			return nil, fmt.Errorf("%s names %q, which is not on the block's models list; it marks only a model the block lists",
				logprobModelsKey, layered.BoundKey(m))
		}
		if seen[m] {
			return nil, fmt.Errorf("%s names %s twice", logprobModelsKey, m)
		}
		seen[m] = true
	}
	return append([]string(nil), marked...), nil
}

// ScoresByLogprob reports whether a judgement routed to t is asked as a small
// text model scored by log-probability: its provider's block names t's model
// in logprob_models. A target this configuration does not admit is not.
func (c *APIConfig) ScoresByLogprob(t Target) bool {
	if c.Admit(t.Provider, t.Model) != nil {
		return false
	}
	return slices.Contains(c.providers[t.Provider].LogprobModels, t.Model)
}

// readAnyPins reads a block's pins as the block's base URL decides: an
// OpenRouter block's pin a model to one provider, a block on this machine's
// name a model down to its weights, and any other block's are refused.
func (p *Provider) readAnyPins(raw map[string]json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	if !isOpenRouter(p.BaseURL) && openaiapi.IsLocal(p.BaseURL) {
		local := make(map[string]localPinFile, len(raw))
		for _, m := range slices.Sorted(maps.Keys(raw)) {
			pf, err := layered.Decode[localPinFile](raw[m])
			if err != nil {
				return fmt.Errorf("%s.%s, a local pin: %w", pinsKey, layered.BoundKey(m), err)
			}
			local[m] = pf
		}
		var err error
		p.LocalPins, err = readLocalPins(p.Models, local)
		return err
	}
	if !isOpenRouter(p.BaseURL) {
		return errNotPinnable()
	}
	routing := make(map[string]pinFile, len(raw))
	for _, m := range slices.Sorted(maps.Keys(raw)) {
		pf, err := layered.Decode[pinFile](raw[m])
		if err != nil {
			return fmt.Errorf("%s.%s, an OpenRouter pin: %w", pinsKey, layered.BoundKey(m), err)
		}
		routing[m] = pf
	}
	var err error
	p.Pins, err = readPins(p.BaseURL, p.Models, routing)
	return err
}

func deniedError(model string, e DenyEntry) error {
	return fmt.Errorf("lists %s, %s", model, denial(e))
}

// denial is the denylist refusal's clause, shared by the read and Admit.
func denial(e DenyEntry) string {
	return fmt.Sprintf("which %s refuses (%s, from %s); an entry there refuses a model even when a provider lists it, "+
		"so remove the entry to serve it", denylistKey, e.Pattern, e.Origin)
}

// readRoutes reads one route family (roles or judgement types) from the repo
// and machine layers, the higher layer winning per name. A route from any layer
// but the machine's that names a provider holding a key is skipped with a
// diagnostic (skipKeyedRoute), and the machine's own route to the name, if it
// has one, wins in its place: only the person's own machine may point a route
// at their key. A repository route that is not <provider>/<model>, and a route
// name only the repository spells that is not a plain name, are skipped with a
// diagnostic too; the machine's own are refused. So is a repository route to a
// model a keyless provider does not list (skipUnlistedRoute), and one to a
// provider this machine has not configured where the machine routes the same
// name (skipUnconfiguredRoute).
func (c *APIConfig) readRoutes(s *layered.Stack, key string, into map[string]Target) error {
	names := map[string]bool{}
	onMachine := map[string]bool{}
	for _, l := range []layered.Layer{layered.Repo, layered.Machine} {
		ns, err := s.Members(l, key)
		if err != nil {
			return fmt.Errorf("oracle adapter: %w", err)
		}
		for _, n := range ns {
			names[n] = true
			if l == layered.Machine {
				onMachine[n] = true
			}
		}
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	for _, name := range sorted {
		re := routeNameRe
		if key == judgementsKey {
			re = judgementsRe
		}
		if !re.MatchString(name) {
			// A malformed name in the machine's file is the person's own
			// mistake in their own file, and refuses like every other machine
			// fault, so it is fixed rather than silently dropping a route they
			// set. One only a repository spells is skipped with a diagnostic
			// (ruling CD2 of 2026-09-29: a checkout's configuration never takes
			// the commands that read it down), and the name is never looked up,
			// since it may not even split into a key.
			if onMachine[name] {
				return fmt.Errorf("oracle adapter: %s (machine layer): %s name %s is not a plain lower-case name",
					layered.Config.MachineOrigin(), key, quoteUntrusted(name))
			}
			c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("oracle adapter: %s (repo layer): %s name %s is not a plain lower-case name "+
				"(lower case letters, digits and -); the route is skipped and the rest of the configuration applies",
				layered.Config.RepoOrigin(), key, quoteUntrusted(name)))
			continue
		}
		found, err := s.Lookup(key + "." + name)
		if err != nil {
			return fmt.Errorf("oracle adapter: %w", err)
		}
		// A repository's route that is not <provider>/<model>, or that points
		// at a provider holding a key (ruling CD2 of 2026-09-29), at a provider
		// this machine has not configured while the machine routes the name
		// itself (ruling CD4 of 2026-10-02), or at a model a keyless provider
		// does not list (ruling CD3 of 2026-10-02), is skipped, and the next
		// layer's route to the name, the machine's own, applies in its place.
		// The order matters: skipKeyedRoute is judged before skipUnlistedRoute,
		// so a keyed provider's list is never consulted on a repository's
		// behalf, and its route is skipped as one that would spend a key
		// whatever model it names. A judgement type's route that turns the
		// type on is the person's own to set (skipTurnedOnRoute), so one
		// from any other layer is skipped last, after the reasons above.
		for len(found) > 0 && (c.skipMalformedRoute(key, name, found[0]) || c.skipKeyedRoute(key, name, found[0]) ||
			c.skipUnconfiguredRoute(key, name, found) || c.skipUnlistedRoute(key, name, found[0]) ||
			c.skipTurnedOnRoute(key, name, found[0])) {
			found = found[1:]
		}
		if len(found) == 0 {
			continue
		}
		win := found[0]
		where := fmt.Sprintf("%s (%s layer): %s.%s", win.Origin, win.Layer, key, name)
		if key == rolesKey && !inRoster(name) {
			c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("oracle adapter: %s names %q, which is not an agent in the roster; "+
				"the route is skipped and the remaining routes apply", where, name))
			continue
		}
		text, err := layered.Decode[string](win.Raw)
		if err != nil {
			return fmt.Errorf("oracle adapter: %s: %w", where, err)
		}
		spelled := text
		text, mode := routeValue(key, text)
		if mode == JudgementHost {
			// The winning layer keeps the type on the host: no route, and
			// the lower layers' routes to it do not apply.
			continue
		}
		provider, model, ok := strings.Cut(text, "/")
		if !ok || provider == "" || model == "" {
			forms := "a route is <provider>/<model>"
			if key == judgementsKey {
				forms = "a judgement type's route is host, shadow:<provider>/<model> or <provider>/<model>"
			}
			return fmt.Errorf("oracle adapter: %s is %q; %s", where, layered.BoundKey(spelled), forms)
		}
		if _, configured := c.providers[provider]; !configured {
			c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("oracle adapter: %s points at provider %q, which is not configured on this machine; "+
				"it runs on the host, as it would with no provider configured", where, layered.BoundKey(provider)))
			continue
		}
		if err := c.Admit(provider, model); err != nil {
			return fmt.Errorf("oracle adapter: %s points at %s, %w", where, layered.BoundKey(text), err)
		}
		into[name] = Target{Provider: provider, Model: model, Origin: win.Origin, Mode: mode}
	}
	return nil
}

// quoteUntrusted renders a route name that failed its pattern, or a route
// value that failed its shape, for a diagnostic or a refusal. It is
// file-authored bytes, a repository's among them, so termsafe masks every
// terminal-attack rune first, and the quoting then spells any other non-ASCII
// rune as an escape: a lookalike letter (the Cyrillic U+0456 in "scribe") is
// why such a name fails, and printed as itself it would read as the plain name
// it imitates.
func quoteUntrusted(v string) string {
	return strconv.QuoteToASCII(termsafe.Sanitize(layered.BoundKey(v)))
}

// skipMalformedRoute reports whether one layer's route to name is a route from
// any layer but the machine's that is not <provider>/<model>, and says so in a
// diagnostic when it is. Such a route names no provider, so no key and no
// denylist entry is in question, and skipping it keeps every command that
// reads the configuration working (ruling CD2 of 2026-09-29). The machine's
// own malformed route is left to the refusal below: that file is the person's,
// and a route they set is never dropped silently.
func (c *APIConfig) skipMalformedRoute(key, name string, fd layered.Found) bool {
	if fd.Layer == layered.Machine {
		return false
	}
	what := "is not a string"
	if text, err := layered.Decode[string](fd.Raw); err == nil {
		target, mode := routeValue(key, text)
		if mode == JudgementHost {
			return false
		}
		provider, model, ok := strings.Cut(target, "/")
		if ok && provider != "" && model != "" {
			return false
		}
		what = "is " + quoteUntrusted(text)
	}
	c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("oracle adapter: %s (%s layer): %s.%s %s; a route is <provider>/<model>, "+
		"so this route is skipped and the rest of the configuration applies", fd.Origin, fd.Layer, key, name, what))
	return true
}

// skipKeyedRoute reports whether one layer's route to name is a route from any
// layer but the machine's to a configured provider that holds a key, and says
// so in a diagnostic when it is. Only a route the person set up on their own
// machine may spend their paid key (ruling AA(b) of 2026-09-29), and such a
// route is skipped rather than refusing the whole configuration (ruling CD2 of
// the same day), so every other route and every command that reads the
// configuration keeps working. It is judged before the model is checked, so a
// keyed provider's list is never consulted on a repository's behalf; a route
// the denylist matches is left to the refusal below, which no layer softens,
// and a route this reader cannot parse is too.
func (c *APIConfig) skipKeyedRoute(key, name string, fd layered.Found) bool {
	if fd.Layer == layered.Machine {
		return false
	}
	text, err := layered.Decode[string](fd.Raw)
	if err != nil {
		return false
	}
	text, _ = routeValue(key, text)
	provider, model, ok := strings.Cut(text, "/")
	if !ok || provider == "" || model == "" {
		return false
	}
	p, configured := c.providers[provider]
	if !configured || !keyed(p) {
		return false
	}
	if _, denied := Denied(c.denylist, model); denied {
		return false
	}
	c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("oracle adapter: %s (%s layer): %s.%s points at %s, a provider that holds a key "+
		"(its block in %s names the credential %s); only a route set on this machine may spend that key, so this route is skipped "+
		"and the rest of the configuration applies: set %s.%s in %s and remove it from %s, or point it at a provider whose block names no key",
		fd.Origin, fd.Layer, key, name, layered.BoundKey(text), p.Origin, p.Key, key, name, layered.Config.MachineOrigin(), fd.Origin))
	return true
}

// skipUnconfiguredRoute reports whether the first of found, the layers' routes
// to name highest first, is a route from any layer but the machine's to a
// provider this machine has not configured while the machine's own file routes
// the same name, and says so in a diagnostic when it is. Such a route can
// reach no provider, and letting it win would displace the person's own
// setting with a step on the host, so the owner's setting applies (ruling CD4
// of 2026-10-02). With no machine route to the name it is not skipped, and the
// read below reports that the step runs on the host, as it would with nothing
// configured (adr-25).
func (c *APIConfig) skipUnconfiguredRoute(key, name string, found []layered.Found) bool {
	fd := found[0]
	if fd.Layer == layered.Machine {
		return false
	}
	machineRoute := false
	for _, below := range found[1:] {
		if below.Layer == layered.Machine {
			machineRoute = true
		}
	}
	if !machineRoute {
		return false
	}
	text, err := layered.Decode[string](fd.Raw)
	if err != nil {
		return false
	}
	text, _ = routeValue(key, text)
	provider, _, ok := strings.Cut(text, "/")
	if !ok || provider == "" {
		return false
	}
	if _, configured := c.providers[provider]; configured {
		return false
	}
	c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("oracle adapter: %s (%s layer): %s.%s points at provider %s, which is not configured "+
		"on this machine, so this route is skipped and the route %s sets for %s.%s applies in its place: "+
		"configure the provider in %s, or remove the route from %s",
		fd.Origin, fd.Layer, key, name, quoteUntrusted(provider), layered.Config.MachineOrigin(), key, name,
		layered.Config.MachineOrigin(), fd.Origin))
	return true
}

// skipTurnedOnRoute reports whether one layer's route to name is a judgement
// type's route that turns the type on, from a layer that is not the person's
// own, and says so in a diagnostic when it is. A type turned on hands its
// decision to the adapter, and only the person's own configuration may do
// that: the machine's file under their home, or their own flag for one run,
// the layers runner.personal names for a role handed to a runner (rulings RN2
// and OC2 of 2026-10-02). The rule is restated here rather than shared,
// because runner imports this package. Skipping the route lets the next
// layer's route to the type apply, the machine's in its own mode, or the
// host when the machine sets none, so a repository never turns a machine's
// shadow into on. A repository's host and shadow are not skipped here: host
// keeps work on the host, and in shadow the host still decides. A route the
// denylist matches is left to the refusal below, which no layer softens.
func (c *APIConfig) skipTurnedOnRoute(key, name string, fd layered.Found) bool {
	if key != judgementsKey || fd.Layer == layered.Machine || fd.Layer == layered.Flag {
		return false
	}
	text, err := layered.Decode[string](fd.Raw)
	if err != nil {
		return false
	}
	target, mode := routeValue(key, text)
	if mode != JudgementOn {
		return false
	}
	_, model, _ := strings.Cut(target, "/")
	if _, denied := Denied(c.denylist, model); denied {
		return false
	}
	c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("oracle adapter: %s (%s layer): %s.%s is %s, which turns the type on; "+
		"only the person's own configuration (%s) turns a judgement type on, so this route is skipped and the route %s sets "+
		"for the type, or the host, applies in its place: set it in %s, or spell it %s to run the adapter beside the host",
		fd.Origin, fd.Layer, key, name, quoteUntrusted(text), layered.Config.MachineOrigin(), layered.Config.MachineOrigin(),
		layered.Config.MachineOrigin(), quoteUntrusted(shadowPrefix+target)))
	return true
}

// skipUnlistedRoute reports whether one layer's route to name is a route from
// any layer but the machine's to a configured provider that holds no key,
// naming a model that provider does not list, and says so in a diagnostic
// when it is. Such a route can never be sent, and skipping it keeps every
// other route and every command that reads the configuration working (ruling
// CD3 of 2026-10-02); the machine's own route to the name, if it sets one,
// applies in its place. A keyed provider's list is never consulted here: its
// route is skipped by skipKeyedRoute first, and this predicate declines one
// on its own account too, so reordering the two cannot make it consult that
// list. The machine's own route to an unlisted model is left to the refusal
// below, since that file is the person's, and so is a route the denylist
// matches from any layer, which no layer softens.
func (c *APIConfig) skipUnlistedRoute(key, name string, fd layered.Found) bool {
	if fd.Layer == layered.Machine {
		return false
	}
	text, err := layered.Decode[string](fd.Raw)
	if err != nil {
		return false
	}
	text, _ = routeValue(key, text)
	provider, model, ok := strings.Cut(text, "/")
	if !ok || provider == "" || model == "" {
		return false
	}
	p, configured := c.providers[provider]
	if !configured || keyed(p) {
		return false
	}
	if _, denied := Denied(c.denylist, model); denied {
		return false
	}
	for _, m := range p.Models {
		if m == model {
			return false
		}
	}
	c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("oracle adapter: %s (%s layer): %s.%s points at %s, which is not on %s's list (%s); "+
		"a provider serves only the models it lists, so this route is skipped and the rest of the configuration applies, "+
		"the route %s sets for the name, if any, in its place: point it at a model on the list, or add the model to the list in %s",
		fd.Origin, fd.Layer, key, name, quoteUntrusted(text), provider, listNames(p.Models), layered.Config.MachineOrigin(),
		layered.Config.MachineOrigin()))
	return true
}

// keyed reports whether p holds a key: whether its block names a credential.
// It is judged from the block alone and never from the credential store, so no
// secret is read to answer it, and a key stored or removed later cannot change
// the answer: a block that names a key is keyed before its key is stored.
func keyed(p Provider) bool { return p.Key != "" }

// Admit is the refusal adr-2609221009491186 names: it returns nil only when
// provider is configured, model is on its list, and no oracle.denylist entry
// the configuration wrote matches the model. It is consulted when the configuration is read and again by any
// dispatch, so a route never reaches a provider on a stale answer.
func (c *APIConfig) Admit(provider, model string) error {
	p, ok := c.providers[provider]
	if !ok {
		return fmt.Errorf("provider %q is not configured on this machine", layered.BoundKey(provider))
	}
	if e, denied := Denied(c.denylist, model); denied {
		return errors.New(denial(e))
	}
	for _, m := range p.Models {
		if m == model {
			return nil
		}
	}
	return fmt.Errorf("which is not on %s's list (%s); a provider serves only the models it lists, so it is refused before any call",
		provider, listNames(p.Models))
}

// listNames renders a list for a refusal, bounded.
func listNames(ms []string) string {
	const shown = 10
	if len(ms) <= shown {
		return strings.Join(ms, ", ")
	}
	return strings.Join(ms[:shown], ", ") + fmt.Sprintf(" and %d more", len(ms)-shown)
}

// Denied reports the first denylist entry that matches model. Matching ignores
// case, OpenRouter's ~ alias prefix, and for an exact entry a :variant suffix,
// so a spelling cannot slip a denied model past its entry.
func Denied(denylist []DenyEntry, model string) (DenyEntry, bool) {
	m := normalizeModel(model)
	for _, e := range denylist {
		p := strings.TrimPrefix(strings.ToLower(e.Pattern), "~")
		if vendor, ok := strings.CutSuffix(p, "/*"); ok {
			if strings.HasPrefix(m, vendor+"/") {
				return e, true
			}
			continue
		}
		if m == normalizeModel(p) {
			return e, true
		}
	}
	return DenyEntry{}, false
}

// normalizeModel lower-cases a model identifier, drops a leading ~ and a
// :variant suffix on its last segment.
func normalizeModel(m string) string {
	m = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(m)), "~")
	slash := strings.LastIndexByte(m, '/')
	if i := strings.IndexByte(m[slash+1:], ':'); i >= 0 {
		m = m[:slash+1+i]
	}
	return m
}

// Providers returns the configured providers, sorted by name.
func (c *APIConfig) Providers() []Provider {
	out := make([]Provider, 0, len(c.providers))
	for _, p := range c.providers {
		p.Models = append([]string(nil), p.Models...)
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Provider returns one configured provider.
func (c *APIConfig) Provider(name string) (Provider, bool) {
	p, ok := c.providers[name]
	p.Models = append([]string(nil), p.Models...)
	return p, ok
}

// Denylist returns the denylist in force, the entries oracle.denylist holds,
// lowest layer first; empty when none is written, since abcd bundles none.
func (c *APIConfig) Denylist() []DenyEntry { return append([]DenyEntry(nil), c.denylist...) }

// Role returns where an agent is pointed, if it is.
func (c *APIConfig) Role(agent string) (Target, bool) {
	t, ok := c.roles[agent]
	return t, ok
}

// Judgement returns where a judgement type is pointed, if it is.
func (c *APIConfig) Judgement(kind string) (Target, bool) {
	t, ok := c.judgements[kind]
	return t, ok
}

// PointedRoute is one role or judgement type and its target, for a board.
type PointedRoute struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Target Target `json:"target"`
}

// Routes returns every role and judgement type pointed at a provider, roles
// first, each family sorted by name.
func (c *APIConfig) Routes() []PointedRoute {
	var out []PointedRoute
	for _, fam := range []struct {
		kind string
		m    map[string]Target
	}{{"role", c.roles}, {"judgement", c.judgements}} {
		names := make([]string, 0, len(fam.m))
		for n := range fam.m {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			out = append(out, PointedRoute{Kind: fam.kind, Name: n, Target: fam.m[n]})
		}
	}
	return out
}

// Connections returns the machine's provider connections, the implementation
// of Connections this configuration backs. A provider claims no tier: it is
// reached by a role or a judgement type pointed at it, or by a --route naming
// it, never by a tier alone, so Serves answers false for every tier and the
// tier-only steps stay on the harness. Pointed returns the connection an
// agent's role points at. Named returns the provider's connection
// carrying its allowlist, the settings the adapter accepts, and the model each
// role pointed at it asks for.
func (c *APIConfig) Connections() Connections { return apiConnections{c: c} }

type apiConnections struct{ c *APIConfig }

func (apiConnections) Serves(Tier) (Connection, bool) { return Connection{}, false }

// Pointed returns the connection agent's oracle.roles.<agent> points at: a
// provider is reached by the role pointed at it (Decision 9), so this is how
// a step with no --route finds its provider.
func (a apiConnections) Pointed(agent string) (Connection, bool) {
	t, ok := a.c.roles[agent]
	if !ok {
		return Connection{}, false
	}
	return a.Named(t.Provider)
}

func (a apiConnections) Named(name string) (Connection, bool) {
	p, ok := a.c.providers[name]
	if !ok {
		return Connection{}, false
	}
	var roles map[string]string
	for agent, t := range a.c.roles {
		if t.Provider == name {
			if roles == nil {
				roles = map[string]string{}
			}
			roles[agent] = t.Model
		}
	}
	return Connection{
		Name:    p.Name,
		Models:  append([]string(nil), p.Models...),
		Accepts: openaiapi.AcceptedSettings(),
		Roles:   roles,
		Keyed:   keyed(p),
	}, true
}
