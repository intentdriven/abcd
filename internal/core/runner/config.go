package runner

// config.go reads the route per role and the runners this machine enabled,
// through the layered configuration resolver (the 2026-09-25 ruling that
// placed roles.<role>.runner in layered.Config):
//
//   - roles.<role>.runner names host (the default when unset) or a runner.
//     The repository and the machine may set it; the higher layer wins per
//     role. A role no agent answers to is a diagnostic and is skipped, as the
//     oracle's routes are.
//   - runner.<name> enables a shipped runner (claude, opencode) on this
//     machine, with an optional model route, <provider>/<model>, admitted
//     against that provider's allowlist and the vendor denylist
//     (adr-2609221009491186) when the configuration is read, so a route off
//     the list is refused before any runner can be launched.
//   - runner.fallback_host names the enabled runner that runs a role when abcd
//     runs with no host session (the intent's Decision 2).
//
// The runner namespace is the machine's alone: which harness abcd starts, and
// on whose credential it runs, is the person's own machine's to say, as a
// provider block is (the product thinker's ruling AA(b) of 2026-09-29, which
// keeps a repository from spending the person's paid key). A repository that
// declares it is refused, naming the machine's file. A role a repository
// routes to a runner the machine did not enable runs nowhere new: the dispatch
// records it as an absent runner and falls back.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// The configuration keys, as every refusal names them.
const (
	rolesKey    = "roles"
	runnerKey   = "runner"
	fallbackKey = "fallback_host"
	modelKey    = "model"
)

// roleImplementer is the loop's implementer, the one role outside the agent
// roster (the loop's RoleImplementer; spelled here so the loop can import this
// package).
const roleImplementer = "implementer"

// Route is where a role runs and the layer that said so.
type Route struct {
	Runner string
	Layer  layered.Layer
	Origin string
}

// RunnerConfig is one runner this machine enabled.
type RunnerConfig struct {
	Name string
	// Model is the admitted <provider>/<model> route, "" for the harness's own
	// default.
	Model  string
	Origin string
}

// Config is the runner configuration one invocation read.
type Config struct {
	roles    map[string]Route
	runners  map[string]RunnerConfig
	fallback string
	api      *oracle.APIConfig
	// Diagnostics are the non-fatal reports the read produced, one line each,
	// for a front door to print on stderr.
	Diagnostics []string
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
			rc.Model = text
		}
		c.runners[n] = rc
	}
	return nil
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
	known := map[string]bool{roleImplementer: true}
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
		win := found[0]
		where := fmt.Sprintf("%s (%s layer): %s", win.Origin, win.Layer, key)
		if !known[role] {
			c.Diagnostics = append(c.Diagnostics, fmt.Sprintf("runner: %s names %q, which is neither an agent in the roster "+
				"nor the implementer; the route is skipped and the remaining routes apply", where, role))
			continue
		}
		name, err := layered.Decode[string](win.Raw)
		if err != nil {
			return fmt.Errorf("runner: %s: %w", where, err)
		}
		if name != Host && !isRunnerName(name) {
			return fmt.Errorf("runner: %s is %q; a role runs on %s or one of the runners %s",
				where, layered.BoundKey(name), Host, strings.Join(runnerNames, ", "))
		}
		c.roles[role] = Route{Runner: name, Layer: win.Layer, Origin: win.Origin}
	}
	return nil
}

func isRunnerName(n string) bool {
	for _, r := range runnerNames {
		if n == r {
			return true
		}
	}
	return false
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

// adapter builds the enabled runner's adapter.
func (c *Config) adapter(rc RunnerConfig) Runner {
	if rc.Name == OpenCode {
		return newOpenCode(rc.Model)
	}
	return newClaude(rc.Model)
}
