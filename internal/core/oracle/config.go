package oracle

// config.go is the provider configuration of the OpenAI-compatible API adapter
// (itd-2609081951381895, spc-2609221011153746 scope 1) and the resolver that
// validates it when it is read (adr-2609221009491186):
//
//   - oracle.api.<provider> is a provider block: base_url (pinned; https, or
//     http to this machine), key (a credential NAME, resolved through
//     internal/core/credential; omitted for a server that needs none) and
//     models (the allowlist: the only models the provider may serve). A block
//     is read from the machine's ~/.abcd/config.json alone. A repository
//     declaring one is refused, because a block names the address a key is sent
//     to, and a checkout must never be able to aim the person's key at a
//     server of its choosing.
//   - oracle.denylist extends the bundled vendor denylist (anthropic/* at
//     minimum). The repository and the machine add entries; nothing removes a
//     bundled one, so a listing mistake can never reach a frontier model.
//   - oracle.roles.<agent> and oracle.judgements.<type> point a role (an agent
//     in the roster) or a judgement type at <provider>/<model>. The repository
//     and the machine may both point; the higher layer wins per name.
//
// Every route is checked here, before any call: a model its provider does not
// list is refused naming the list, and a listed model the denylist matches is
// refused naming the entry, whatever the allowlist says. A route naming a
// provider this machine has not configured is a diagnostic, not a refusal: the
// step stays on the host, exactly as it would with nothing configured (adr-25).
//
// Like the rest of the package, the resolver never writes, never reaches a
// network and never prints.

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// The configuration keys, as every refusal names them.
const (
	apiKey        = "oracle.api"
	denylistKey   = "oracle.denylist"
	rolesKey      = "oracle.roles"
	judgementsKey = "oracle.judgements"
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

// bundledDenylist is the vendor denylist abcd ships: the frontier vendor whose
// models the person already pays for through the host (adr-2609221009491186
// Decision 2). A layer may extend it and never shorten it.
var bundledDenylist = []string{"anthropic/*"}

// BundledDenylist returns the denylist abcd ships. The slice is a copy.
func BundledDenylist() []string { return append([]string(nil), bundledDenylist...) }

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
}

// Target is where a role or a judgement type is pointed: a configured
// provider and a model on its list.
type Target struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	// Origin names the file the route came from.
	Origin string `json:"origin"`
}

// String is the route as the configuration spells it.
func (t Target) String() string { return t.Provider + "/" + t.Model }

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
	// Diagnostics are the non-fatal reports the read produced, one line each,
	// for a front door to print on stderr: a route naming a provider this
	// machine has not configured, and a role outside the roster.
	Diagnostics []string
}

// providerFile is a provider block as the file spells it, decoded strictly.
type providerFile struct {
	BaseURL string   `json:"base_url"`
	Key     *string  `json:"key"`
	Models  []string `json:"models"`
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
	if err := c.readRoutes(s, rolesKey, c.roles); err != nil {
		return nil, err
	}
	if err := c.readRoutes(s, judgementsKey, c.judgements); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *APIConfig) readDenylist(s *layered.Stack) error {
	for _, p := range bundledDenylist {
		c.denylist = append(c.denylist, DenyEntry{Pattern: p, Origin: "bundled"})
	}
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
	if err := s.Claim(apiKey+".*", "base_url", "key", "models"); err != nil {
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
	p := Provider{Name: name, BaseURL: pf.BaseURL, Origin: fd.Origin}
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
	return p, nil
}

func deniedError(model string, e DenyEntry) error {
	return fmt.Errorf("lists %s, %s", model, denial(e))
}

// denial is the denylist refusal's clause, shared by the read and Admit.
func denial(e DenyEntry) string {
	return fmt.Sprintf("which the vendor denylist refuses (%s, from %s); no allowlist entry overrides the denylist, "+
		"so a frontier model the host serves is never billed or routed through a provider", e.Pattern, e.Origin)
}

// readRoutes reads one route family (roles or judgement types) from the repo
// and machine layers, the higher layer winning per name.
func (c *APIConfig) readRoutes(s *layered.Stack, key string, into map[string]Target) error {
	names := map[string]bool{}
	for _, l := range []layered.Layer{layered.Repo, layered.Machine} {
		ns, err := s.Members(l, key)
		if err != nil {
			return fmt.Errorf("oracle adapter: %w", err)
		}
		for _, n := range ns {
			names[n] = true
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
			return fmt.Errorf("oracle adapter: %s.%s: the name is not a plain lower-case name", key, layered.BoundKey(name))
		}
		found, err := s.Lookup(key + "." + name)
		if err != nil {
			return fmt.Errorf("oracle adapter: %w", err)
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
		provider, model, ok := strings.Cut(text, "/")
		if !ok || provider == "" || model == "" {
			return fmt.Errorf("oracle adapter: %s is %q; a route is <provider>/<model>", where, layered.BoundKey(text))
		}
		if _, configured := c.providers[provider]; !configured {
			c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("oracle adapter: %s points at provider %q, which is not configured on this machine; "+
				"it runs on the host, as it would with no provider configured", where, layered.BoundKey(provider)))
			continue
		}
		if err := c.Admit(provider, model); err != nil {
			return fmt.Errorf("oracle adapter: %s points at %s, %w", where, layered.BoundKey(text), err)
		}
		into[name] = Target{Provider: provider, Model: model, Origin: win.Origin}
	}
	return nil
}

// Admit is the refusal adr-2609221009491186 names: it returns nil only when
// provider is configured, model is on its list, and no denylist entry matches
// the model. It is consulted when the configuration is read and again by any
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

// Denylist returns the denylist in force: the bundled entries, then the
// machine's, then the repository's.
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
// tier-only steps stay on the harness. Named returns the provider's connection
// carrying its allowlist and the settings the adapter accepts.
func (c *APIConfig) Connections() Connections { return apiConnections{c: c} }

type apiConnections struct{ c *APIConfig }

func (apiConnections) Serves(Tier) (Connection, bool) { return Connection{}, false }

func (a apiConnections) Named(name string) (Connection, bool) {
	p, ok := a.c.providers[name]
	if !ok {
		return Connection{}, false
	}
	return Connection{
		Name:    p.Name,
		Models:  append([]string(nil), p.Models...),
		Accepts: openaiapi.AcceptedSettings(),
	}, true
}
