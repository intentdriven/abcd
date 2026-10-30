package runner

// config.go reads the route per role and the runners this machine enabled,
// through the layered configuration resolver (the 2026-09-25 ruling that
// placed roles.<role>.runner in layered.Config):
//
//   - roles.<role>.runner names host (the default when unset) or a runner.
//     Only a personal layer (the machine, or the invocation's own flag) may
//     name a runner (rulings RN2 and OC2 of 2026-10-02); the repository may
//     name host alone, and a repository route to a runner is a diagnostic and
//     is skipped, so the next layer's route, or the host, applies. The higher
//     layer wins per role among the routes that stand. A role no agent
//     answers to is a diagnostic and is skipped, as the oracle's routes are.
//   - runner.<name> enables a shipped runner (claude, opencode) on this
//     machine, with an optional model route, <provider>/<model>, admitted
//     against that provider's allowlist when the configuration is read
//     (adr-2609221009491186; its Decision 2 superseded by adr-2609300107513982,
//     so abcd bundles no vendor denylist and the allowlist alone decides, and
//     an oracle.denylist entry the configuration writes still refuses a model
//     it matches), so a route off the list is refused before any runner can
//     be launched.
//   - runner.fallback_host names the enabled runner that runs a role when abcd
//     runs with no host session (the intent's Decision 2).
//
// The runner namespace is the machine's alone: which harness abcd starts, and
// on whose credential it runs, is the person's own machine's to say, as a
// provider block is (the product thinker's ruling AA(b) of 2026-09-29, which
// keeps a repository from spending the person's paid key). A repository that
// declares it is refused, naming the machine's file. Handing a role to a
// runner spends the same key, so that too is the person's alone: a
// repository's route to a runner is skipped (RN2: the claude runner stays
// bare on an API key, and a person whose account is a subscription runs the
// role in their own session). A role the person routes to a runner the
// machine did not enable runs nowhere new: the dispatch records it as an
// absent runner and falls back.

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The configuration keys, as every refusal names them.
const (
	rolesKey    = "roles"
	runnerKey   = "runner"
	fallbackKey = "fallback_host"
	modelKey    = "model"
)

// roleImplementer is the loop's implementer and roleDrainJudge the drain's
// judge (spc-2610161044461448 step 14), the two roles outside the agent roster
// (the loop's RoleImplementer and RoleDrainJudge; spelled here so the loop can
// import this package).
const (
	roleImplementer = "implementer"
	roleDrainJudge  = "drain-judge"
)

// Route is where a role runs and the layer that said so.
type Route struct {
	Runner string
	Layer  layered.Layer
	Origin string
}

// Override is the --route text verbatim when the person's flag routed the
// role for this invocation (Config.Override), "" otherwise.
func (r Route) Override() string {
	if r.Layer == layered.Flag {
		return r.Origin
	}
	return ""
}

// RunnerConfig is one runner this machine enabled.
type RunnerConfig struct {
	Name string
	// Model is the admitted <provider>/<model> route, "" for the harness's own
	// default.
	Model string
	// Pin is the machine's pin for that model on that provider
	// (oracle.Pin), nil when it sets none; the opencode runner holds a run
	// to it.
	Pin *oracle.Pin
	// Local says the model's provider is a block on this machine, and
	// LocalPin is the machine's local pin for the model, nil when it sets
	// none (oracle.LocalPin): opencode is held to it after a probe.
	Local    bool
	LocalPin *oracle.LocalPin
	Origin   string
}

// Config is the runner configuration one invocation read.
type Config struct {
	roles   map[string]Route
	runners map[string]RunnerConfig
	// flagModels is the model route each role's --route asks its runner for,
	// keyed by role (Override); a role absent here runs its runner's machine
	// model.
	flagModels map[string]string
	fallback   string
	api        *oracle.APIConfig
	// Diagnostics are the non-fatal reports the read produced, one line each,
	// for a front door to print on stderr.
	Diagnostics []string
	// diagRoles is the role each of Diagnostics reports on, index for index,
	// so Override can drop the lines about a route it replaces.
	diagRoles []string
}

// Load reads and validates the runner configuration. A fault is an error
// naming the file and the key; none falls through to a default.
func Load(r layered.Roots) (*Config, error) {
	s, err := layered.Load(layered.Config, r)
	if err != nil {
		return nil, fmt.Errorf("runner: %w", err)
	}
	if err := s.Claim(rolesKey+".*", runnerKey); err != nil {
		return nil, fmt.Errorf("runner: %w", err)
	}
	if err := s.Claim(runnerKey, append([]string{fallbackKey}, runnerNames...)...); err != nil {
		return nil, fmt.Errorf("runner: %w", err)
	}
	for _, n := range runnerNames {
		if err := s.Claim(runnerKey+"."+n, modelKey); err != nil {
			return nil, fmt.Errorf("runner: %w", err)
		}
	}
	found, err := s.Lookup(runnerKey)
	if err != nil {
		return nil, fmt.Errorf("runner: %w", err)
	}
	for _, f := range found {
		if f.Layer != layered.Machine {
			return nil, fmt.Errorf("runner: %s (%s layer): %s is set here, but which harness abcd starts, and on whose "+
				"credential, is this machine's alone to say; set it in %s and remove it from %s",
				f.Origin, f.Layer, runnerKey, layered.Config.MachineOrigin(), f.Origin)
		}
	}
	api, err := oracle.LoadAPI(r)
	if err != nil {
		return nil, fmt.Errorf("runner: the model routes are admitted against the provider configuration: %w", err)
	}
	c := &Config{roles: map[string]Route{}, runners: map[string]RunnerConfig{}, api: api}
	if err := c.readRunners(s); err != nil {
		return nil, err
	}
	if err := c.readFallback(s); err != nil {
		return nil, err
	}
	if err := c.readRoles(s); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) readRunners(s *layered.Stack) error {
	for _, n := range runnerNames {
		found, err := s.Lookup(runnerKey + "." + n)
		if err != nil {
			return fmt.Errorf("runner: %w", err)
		}
		if len(found) == 0 {
			continue
		}
		rc := RunnerConfig{Name: n, Origin: found[0].Origin}
		key := runnerKey + "." + n + "." + modelKey
		mf, err := s.Lookup(key)
		if err != nil {
			return fmt.Errorf("runner: %w", err)
		}
		if len(mf) > 0 {
			where := fmt.Sprintf("%s (%s layer): %s", mf[0].Origin, mf[0].Layer, key)
			text, err := layered.Decode[string](mf[0].Raw)
			if err != nil {
				return fmt.Errorf("runner: %s: %w", where, err)
			}
			if err := c.admitModel(text); err != nil {
				return fmt.Errorf("runner: %s is %q, %w", where, layered.BoundKey(text), err)
			}
			rc = c.modelled(rc, text)
		}
		c.runners[n] = rc
	}
	return nil
}

// modelled is rc asking for model, an admitted <provider>/<model> route, with
// the pins this machine sets for that model: its provider pin (PinFor),
// whether its provider is a block on this machine, and its local pin
// (LocalPinFor). The machine's runner.<name>.model and a --route's model are
// both read through it (iss-2610161801358637), so a runner is held to the
// pins of the model it asks for, whichever layer named it, and never carries
// another model's.
func (c *Config) modelled(rc RunnerConfig, model string) RunnerConfig {
	rc.Model, rc.Pin, rc.Local, rc.LocalPin = model, nil, false, nil
	provider, id, _ := strings.Cut(model, "/")
	if pin, ok := c.api.PinFor(provider, id); ok {
		rc.Pin = &pin
	}
	rc.Local = c.api.IsLocal(provider)
	if pin, ok := c.api.LocalPinFor(provider, id); ok {
		rc.LocalPin = &pin
	}
	return rc
}

// providerConfigured reports whether the machine configures provider.
func (c *Config) providerConfigured(provider string) bool {
	if c.api == nil {
		return false
	}
	_, ok := c.api.Provider(provider)
	return ok
}

// admitModel is the allowlist check a model route passes when it is read and
// again before a runner is launched.
func (c *Config) admitModel(route string) error {
	provider, model, ok := strings.Cut(route, "/")
	if !ok || provider == "" || model == "" {
		return fmt.Errorf("which is not <provider>/<model>")
	}
	if err := c.api.Admit(provider, model); err != nil {
		return fmt.Errorf("which is refused before any runner is launched: %w", err)
	}
	return nil
}

func (c *Config) readFallback(s *layered.Stack) error {
	key := runnerKey + "." + fallbackKey
	v, err := layered.Get(s, key, "", func(name string) error {
		if _, ok := c.runners[name]; !ok {
			return fmt.Errorf("the fallback host is a runner this machine enables under %s.<name> (%s)",
				runnerKey, strings.Join(runnerNames, ", "))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("runner: %w", err)
	}
	c.fallback = v.V
	return nil
}

func (c *Config) readRoles(s *layered.Stack) error {
	names := map[string]bool{}
	for _, l := range []layered.Layer{layered.Flag, layered.Repo, layered.Machine} {
		ns, err := s.Members(l, rolesKey)
		if err != nil {
			return fmt.Errorf("runner: %w", err)
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
	known := map[string]bool{roleImplementer: true, roleDrainJudge: true}
	for _, a := range oracle.Roster() {
		known[a] = true
	}
	for _, role := range sorted {
		if !roleRe.MatchString(role) {
			return fmt.Errorf("runner: %s.%s: the role is not a plain lower-case name", rolesKey, layered.BoundKey(role))
		}
		key := rolesKey + "." + role + "." + runnerKey
		found, err := s.Lookup(key)
		if err != nil {
			return fmt.Errorf("runner: %w", err)
		}
		if len(found) == 0 {
			continue
		}
		if !known[role] {
			where := fmt.Sprintf("%s (%s layer): %s", found[0].Origin, found[0].Layer, key)
			c.diagnose(role, fmt.Sprintf("runner: %s names %q, which is neither an agent in the roster "+
				"nor the implementer or the drain judge; the route is skipped and the remaining routes apply", where, role))
			continue
		}
		// found lists the layers that set the key, highest first (flag, repo,
		// machine): the first route that stands is the role's.
		for i, f := range found {
			where := fmt.Sprintf("%s (%s layer): %s", f.Origin, f.Layer, key)
			name, err := layered.Decode[string](f.Raw)
			if err != nil {
				return fmt.Errorf("runner: %s: %w", where, err)
			}
			if name != Host && !isRunnerName(name) {
				return fmt.Errorf("runner: %s is %q; a role runs on %s or one of the runners %s",
					where, layered.BoundKey(name), Host, strings.Join(runnerNames, ", "))
			}
			if name != Host && !personal(f.Layer) {
				c.diagnose(role, fmt.Sprintf("runner: %s is %q, which is skipped: only the person's own "+
					"route may hand a role to a runner, which spends their key, so the role runs as if the repository "+
					"had not routed it; to run it through %s, set %s in %s", where, name, name, key, layered.Config.MachineOrigin()))
				continue
			}
			if role == roleDrainJudge && name == OpenCode {
				// The drain's judge reads a record a stranger may have
				// written, and is held to the tools it is granted; opencode
				// runs under its own configuration's permissions instead.
				c.diagnose(role, fmt.Sprintf("runner: %s is %q, which is skipped: opencode runs under "+
					"the permissions its own configuration sets rather than the tools a role is granted, so it cannot be held "+
					"to the drain judge's Read and Write, and the judge runs as if it had not been routed; to route it to a "+
					"runner, name %s", where, name, Claude))
				continue
			}
			if name == Host && !personal(f.Layer) {
				if d, ok := displacedRunnerRoute(found[i+1:]); ok {
					c.diagnose(role, fmt.Sprintf("runner: %s is %q, which keeps the role on the host "+
						"over the person's own route %s (%s layer): %s, which names %q; the repository's route stands, "+
						"because it spends nothing of theirs, so the runner is not launched for this role",
						where, name, d.Origin, d.Layer, key, d.Runner))
				}
			}
			c.roles[role] = Route{Runner: name, Layer: f.Layer, Origin: f.Origin}
			break
		}
	}
	return nil
}

// displacedRunnerRoute returns the first personal route in lower, the layers
// beneath a repository route to the host, that names a runner: the person's
// own choice the repository's route displaces. A lower route that does not
// decode, or names the host, displaces nothing worth saying.
func displacedRunnerRoute(lower []layered.Found) (Route, bool) {
	for _, f := range lower {
		if !personal(f.Layer) {
			continue
		}
		name, err := layered.Decode[string](f.Raw)
		if err != nil || !isRunnerName(name) {
			return Route{}, false
		}
		return Route{Runner: name, Layer: f.Layer, Origin: f.Origin}, true
	}
	return Route{}, false
}

// personal reports whether a route set in layer l is the person's own, the
// only kind that may hand a role to a runner (rulings RN2 and OC2 of
// 2026-10-02): the machine's file under their home, and the flag layer, which
// is their own invocation for one run. The repository's file is committed
// content anyone with a pull request can write, so a route there may keep a
// role on the host and never hand it to a runner.
func personal(l layered.Layer) bool { return l == layered.Machine || l == layered.Flag }

func isRunnerName(n string) bool {
	for _, r := range runnerNames {
		if n == r {
			return true
		}
	}
	return false
}

// Override places the person's --route runner routes
// (oracle.ParseRunnerRoutes) in the flag layer, for this invocation alone
// (iss-2610161801358637): each hands its role to the runner it names, asking
// for the model it names in place of the runner's machine model, which stays
// every other role's. The flag is the person's own, so it may hand a role to a
// runner as the machine file may (personal), but only to a runner this machine
// enabled, and only for a model the runner's provider lists: the allowlist
// check a machine model route passes (adr-2609221009491186). A refusal routes
// nothing and names the fix.
func (c *Config) Override(routes []oracle.FlagRoute) error {
	known := map[string]bool{roleImplementer: true}
	for _, a := range oracle.Roster() {
		known[a] = true
	}
	// A refusal names the route by its position among the --route values, and
	// any part of it the machine does not know by its length, never by its text:
	// the build loop's step carries it into its --json refusal document, which
	// the error boundary's pass does not read. A part named in full is one the
	// machine knows: a roster agent, a runner this build ships, a provider it
	// configures or a model a provider lists.
	for i, fr := range routes {
		shown := oracle.RoutePosition(i)
		switch {
		case fr.Runner == "":
			return fmt.Errorf("runner: --route %s names no runner; want %s", shown, oracle.RunnerRouteSyntax)
		case !known[fr.Agent]:
			return fmt.Errorf("runner: --route %s names an agent (%s) that is neither an agent in the roster nor the implementer",
				shown, termsafe.DescribeRefused(fr.Agent))
		case !isRunnerName(fr.Runner):
			return fmt.Errorf("runner: --route %s names a runner (%s) this build does not ship; a role runs on one of the runners %s",
				shown, termsafe.DescribeRefused(fr.Runner), strings.Join(runnerNames, ", "))
		}
		if _, ok := c.runners[fr.Runner]; !ok {
			return fmt.Errorf("runner: --route %s names %s, which is not enabled on this machine; enable it under %s.%s in %s, "+
				"or name a runner it enables (%s)", shown, fr.Runner, runnerKey, fr.Runner, layered.Config.MachineOrigin(), c.enabled())
		}
		if fr.Model != "" {
			if provider, _, _ := strings.Cut(fr.Model, "/"); !c.providerConfigured(provider) {
				return fmt.Errorf("runner: --route %s asks %s for a model whose provider (%s) is not configured on this machine; "+
					"name a provider configured in %s", shown, fr.Runner, termsafe.DescribeRefused(provider), layered.Config.MachineOrigin())
			}
			if err := c.admitModel(fr.Model); err != nil {
				return fmt.Errorf("runner: --route %s asks %s for a model (%s), %w; name a model its provider lists, or add it to that "+
					"provider's models in %s", shown, fr.Runner, termsafe.DescribeRefused(fr.Model), err, layered.Config.MachineOrigin())
			}
		}
		// Only opencode holds a pin. A route the person names to another
		// runner for a model the machine pins could only end in that
		// runner's refusal and the role handed to the host, so it is
		// refused here, before anything is written, as every other route
		// the configuration shows cannot run is; a machine route keeps its
		// fallback, since nobody asked for it on this step.
		if fr.Runner != OpenCode {
			m := fr.Model
			if m == "" {
				m = c.runners[fr.Runner].Model
			}
			if r := c.modelled(c.runners[fr.Runner], m); m != "" && (r.Pin != nil || r.LocalPin != nil) {
				return fmt.Errorf("runner: --route %s asks %s for %s, which the machine pins, and %s can hold neither a "+
					"provider pin nor a local pin; route the role through opencode (runner.opencode.model), or remove the pin",
					shown, fr.Runner, layered.BoundKey(m), fr.Runner)
			}
		}
	}
	// A report about a route the flag replaces would contradict what then
	// runs, so it is dropped; every other role's stands.
	replaced := map[string]bool{}
	for _, fr := range routes {
		replaced[fr.Agent] = true
	}
	var diags, roles []string
	for i, d := range c.Diagnostics {
		if i < len(c.diagRoles) && replaced[c.diagRoles[i]] {
			continue
		}
		diags = append(diags, d)
		if i < len(c.diagRoles) {
			roles = append(roles, c.diagRoles[i])
		}
	}
	c.Diagnostics, c.diagRoles = diags, roles
	for _, fr := range routes {
		c.roles[fr.Agent] = Route{Runner: fr.Runner, Layer: layered.Flag, Origin: fr.Text}
		if fr.Model != "" {
			if c.flagModels == nil {
				c.flagModels = map[string]string{}
			}
			c.flagModels[fr.Agent] = fr.Model
		}
	}
	return nil
}

// diagnose records a non-fatal report about role's configured route.
func (c *Config) diagnose(role, line string) {
	c.Diagnostics = append(c.Diagnostics, line)
	c.diagRoles = append(c.diagRoles, role)
}

// FlagRoutes are the roles the person's --route routed for this invocation
// (Override), each with the --route text verbatim; nil when none did.
func (c *Config) FlagRoutes() map[string]string {
	var out map[string]string
	for role, r := range c.roles {
		if t := r.Override(); t != "" {
			if out == nil {
				out = map[string]string{}
			}
			out[role] = t
		}
	}
	return out
}

// enabled names the runners this machine enabled, for a refusal.
func (c *Config) enabled() string {
	var on []string
	for _, n := range runnerNames {
		if _, ok := c.runners[n]; ok {
			on = append(on, n)
		}
	}
	if len(on) == 0 {
		return "it enables none"
	}
	return strings.Join(on, ", ")
}

// runnerFor is the runner this machine enabled under name as it runs role:
// with the model role's --route asks for when that route named this runner,
// else the machine's, and in either case held to the pins this machine sets
// for the model it asks for (modelled), so the adapter probes a local pin and
// the claude runner refuses a pinned model exactly as for the machine's
// route. where names the setting the model came from, for a refusal.
func (c *Config) runnerFor(name, role string) (rc RunnerConfig, where string, ok bool) {
	rc, ok = c.runners[name]
	where = runnerKey + "." + name + "." + modelKey
	if r := c.roles[role]; ok && r.Override() != "" && r.Runner == name {
		if m := c.flagModels[role]; m != "" {
			rc, where = c.modelled(rc, m), "--route "+r.Origin
		}
	}
	return rc, where, ok
}

// RouteFor returns where role runs: its configured route, or the host from
// the bundled default.
func (c *Config) RouteFor(role string) Route {
	if r, ok := c.roles[role]; ok {
		return r
	}
	return Route{Runner: Host, Layer: layered.Bundled, Origin: "bundled"}
}

// Runner returns the runner this machine enabled under name.
func (c *Config) Runner(name string) (RunnerConfig, bool) {
	rc, ok := c.runners[name]
	return rc, ok
}

// FallbackHost is the runner that runs a role when there is no host session,
// "" when none is configured.
func (c *Config) FallbackHost() string { return c.fallback }

// Quota asks the runner the machine enabled under name for its remaining
// quota. The host, and a name the machine has not enabled, report none.
func (c *Config) Quota(ctx context.Context, name string) (Quota, bool, error) {
	rc, ok := c.runners[name]
	if !ok {
		return Quota{}, false, nil
	}
	return quotaOf(ctx, c.adapter(rc))
}

// adapter builds the enabled runner's adapter.
func (c *Config) adapter(rc RunnerConfig) Runner {
	if rc.Name == OpenCode {
		o := newOpenCode(rc.Model)
		o.pin, o.local, o.localPin = rc.Pin, rc.Local, rc.LocalPin
		if rc.LocalPin != nil {
			provider, model, _ := strings.Cut(rc.Model, "/")
			if p, ok := c.api.Provider(provider); ok {
				o.localBase = p.BaseURL
			}
			// Every role a runner runs is a tool-using one, so the probe holds
			// the server to tool calls and a context it can say it serves.
			o.probe = func(ctx context.Context) (*oracle.LocalServed, error) {
				_, served, err := c.api.ProbeLocal(ctx, nil, provider, model, true)
				return served, err
			}
		}
		return o
	}
	cl := newClaude(rc.Model)
	cl.local, cl.pinned = rc.Local, rc.Pin != nil || rc.LocalPin != nil
	return cl
}
