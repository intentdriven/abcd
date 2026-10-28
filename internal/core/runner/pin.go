package runner

// pin.go is the opencode runner's half of iss-2610161734413178: a role routed
// through opencode to a model this machine pins to one OpenRouter provider
// (oracle.Pin) runs against a sealed opencode configuration carrying the pin,
// written for that run alone.
//
// opencode 1.18.31 loads OPENCODE_CONFIG, the path of one more configuration
// file, after the person's global configuration (the opencode.json of its
// config directory) and merges it over it (read in the installed binary on
// 2026-10-17, and checked with `opencode debug config` against a scratch
// home: a pinned order replaced the global one, and the global file was left
// as it was). abcd writes the pin there, under the model's options as
// opencode's OpenRouter provider passes them on
// (provider.<name>.models.<model>.options.provider, opencode's documented
// shape for OpenRouter's routing object), and never touches the person's own
// configuration or a file of the repository: the file sits in a directory
// made for the run (mode 0700, the file 0600) under the system's temporary
// directory, refused when that resolves inside the repository or its
// checkout, and removed when the run ends.
//
// What opencode merges after OPENCODE_CONFIG, in the same binary and checked
// the same way, in order: the project's configuration files and the
// opencode.json of each .opencode directory up from the working directory
// (both turned off by the seal, OPENCODE_DISABLE_PROJECT_CONFIG), then
// ~/.opencode/opencode.json, then the opencode.json of OPENCODE_CONFIG_DIR,
// then OPENCODE_CONFIG_CONTENT, then an organisation's remote account
// configuration and its managed configuration. The config directory is in
// that same directory walk, but only its agents, commands and plugins load
// there: its opencode.json is read once, before OPENCODE_CONFIG. The
// organisation's configuration is the person's own and still applies.
//
// A person who points opencode at a configuration of their own, a file
// through OPENCODE_CONFIG, a directory through OPENCODE_CONFIG_DIR or inline
// through OPENCODE_CONFIG_CONTENT, would lose it to the sealed file, or have
// it override the pin, and with it the key, permission denies and servers it
// carries. abcd never changes how the person's tools work, so a pinned run is
// refused before anything is written or launched while any of the three is
// set and not empty (personsOpenCodeConfig); an unpinned run passes them
// through. ~/.opencode/opencode.json is deliberately not refused: it is a
// file the person keeps, not a variable they set for a run, and refusing it
// would block every pinned run of everyone who keeps one. A routing entry for the pinned model in that file
// overrides the pin, which is one reason the pin is recorded as not verified
// on this path.
//
// The route records the configuration's SHA-256 and opencode's version, asked
// with --version under the same seal before the run. The key stays in
// opencode's own configuration or environment and abcd never reads it, so the
// generation readback is not made on this path and the served provider, the
// attempts and the provider's own counts stay not reported. Nothing on this
// path shows the pin reached OpenRouter either, so the record carries the pin
// and lists it as not verified: configured, not verified on the wire.
//
// A role routed to a model a block on this machine pins down to its weights
// (oracle.LocalPin, iss-2610161734428291) runs the same way: refused while the
// person names a configuration of their own, then the server is probed
// (oracle's ProbeLocal, held to tool calls and a context it can say it
// serves, since every role a runner runs uses tools), and only then is the
// sealed configuration written, naming the provider's base URL so opencode
// reaches the server the probe read (localConfig).

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// openCodeConfigSchema is the schema opencode's own configuration names.
const openCodeConfigSchema = "https://opencode.ai/config.json"

// versionTimeout bounds opencode's --version.
const versionTimeout = 15 * time.Second

// harnessVersionRe is a version as a harness prints one (1.18.31, v1.2.3-rc1).
var harnessVersionRe = regexp.MustCompile(`^v?[0-9][0-9A-Za-z.+-]{0,31}$`)

// sealedPin is one run's sealed configuration: the directory made for it,
// the file in it and the file's SHA-256.
type sealedPin struct {
	dir, path, sha256 string
}

// pinConfig is the sealed configuration's text for model, a
// <provider>/<model> route, pinned by pin: the routing object under the
// model's options, and nothing else. Every value in it was checked when the
// configuration was read, and none can carry a brace, so opencode reads no
// {env:} or {file:} substitution from it.
func pinConfig(model string, pin oracle.Pin) ([]byte, error) {
	provider, name, _ := strings.Cut(model, "/")
	type options struct {
		Provider any `json:"provider"`
	}
	type entry struct {
		Options options `json:"options"`
	}
	type block struct {
		Models map[string]entry `json:"models"`
	}
	cfg := struct {
		Schema   string           `json:"$schema"`
		Provider map[string]block `json:"provider"`
	}{Schema: openCodeConfigSchema, Provider: map[string]block{
		provider: {Models: map[string]entry{name: {Options: options{Provider: pin.Routing()}}}},
	}}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(cfg); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// localConfig is the sealed configuration's text for model, a
// <provider>/<model> route to a server on this machine at base: the
// provider's base URL, so opencode reaches the server the probe read, and
// the model's entry, and nothing else. opencode pins no served context on any
// of the three runtimes (the context is the server's setting, and its own
// limit.context is opencode's reckoning, which requires limit.output beside
// it and would replace the person's), so the probe confirms the context and
// the configuration names none.
func localConfig(model, base string) ([]byte, error) {
	provider, name, _ := strings.Cut(model, "/")
	if strings.ContainsAny(base, "{}") {
		return nil, errors.New("the base URL carries a brace, which opencode would read as a substitution")
	}
	type options struct {
		BaseURL string `json:"baseURL"`
	}
	type block struct {
		Options options                   `json:"options"`
		Models  map[string]map[string]any `json:"models"`
	}
	cfg := struct {
		Schema   string           `json:"$schema"`
		Provider map[string]block `json:"provider"`
	}{Schema: openCodeConfigSchema, Provider: map[string]block{
		provider: {Options: options{BaseURL: base}, Models: map[string]map[string]any{name: {}}},
	}}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(cfg); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// sealedConfig is the run's sealed configuration: the OpenRouter pin's, or
// the local pin's.
func (o *OpenCodeCLI) sealedConfig() ([]byte, error) {
	var raw []byte
	var err error
	if o.localPin != nil {
		raw, err = localConfig(o.model, o.localBase)
	} else {
		raw, err = pinConfig(o.model, *o.pin)
	}
	if err != nil {
		return nil, fail(OpenCode, ReasonFailed, "the pinned configuration could not be built")
	}
	return raw, nil
}

// probeLocal probes the local pin's server before the run, refusing it when
// the server could not hold the pin for a tool-using role.
func (o *OpenCodeCLI) probeLocal(ctx context.Context) (*oracle.LocalServed, error) {
	if o.probe == nil || o.localBase == "" {
		return nil, fail(OpenCode, ReasonFailed, "the local pin's server is not configured, so it cannot be probed")
	}
	served, err := o.probe(ctx)
	if err != nil {
		return nil, failAt(err, OpenCode, ReasonFailed, "%s", err.Error())
	}
	return served, nil
}

// sealPin writes raw, the run's sealed configuration, for one run of req,
// outside the repository and the checkout, and returns where.
func (o *OpenCodeCLI) sealPin(req Request, raw []byte) (sealedPin, error) {
	dir, err := os.MkdirTemp("", "abcd-opencode-pin-")
	if err != nil {
		return sealedPin{}, fail(OpenCode, ReasonFailed, "no directory could be made for the run's pinned configuration")
	}
	if inside(dir, req.Dir, req.Checkout) {
		_ = os.Remove(dir)
		return sealedPin{}, fail(OpenCode, ReasonFailed, "the temporary directory resolves inside the repository the role runs in, "+
			"and the run's pinned configuration is never written there; point TMPDIR outside it")
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.RemoveAll(dir)
		return sealedPin{}, fail(OpenCode, ReasonFailed, "the run's configuration directory could not be made private")
	}
	path := filepath.Join(dir, "opencode.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return sealedPin{}, fail(OpenCode, ReasonFailed, "the run's pinned configuration could not be written")
	}
	sum := sha256.Sum256(raw)
	return sealedPin{dir: dir, path: path, sha256: hex.EncodeToString(sum[:])}, nil
}

// inside reports whether dir is, lexically or through a link, within any of
// roots.
func inside(dir string, roots ...string) bool {
	cands := []string{filepath.Clean(dir)}
	if r, err := filepath.EvalSymlinks(dir); err == nil {
		cands = append(cands, r)
	}
	fold := fsutil.CaseFoldingFS()
	for _, root := range roots {
		if root == "" {
			continue
		}
		guards := []string{filepath.Clean(root)}
		if g, err := filepath.EvalSymlinks(root); err == nil {
			guards = append(guards, g)
		}
		for _, g := range guards {
			for _, c := range cands {
				if fsutil.PathWithin(c, g, fold) {
					return true
				}
			}
		}
	}
	return false
}

// personsConfigVars are the variables through which a person points opencode
// at a configuration of their own, which a pinned run's sealed file would
// set aside (OPENCODE_CONFIG) or be overridden by (OPENCODE_CONFIG_DIR's
// opencode.json and OPENCODE_CONFIG_CONTENT, both merged after it).
var personsConfigVars = []string{"OPENCODE_CONFIG", "OPENCODE_CONFIG_DIR", "OPENCODE_CONFIG_CONTENT"}

// personsOpenCodeConfig refuses a pinned run while the environment the
// harness would inherit names a configuration of the person's own for
// opencode: the variable is named, its value never quoted.
func personsOpenCodeConfig() error {
	for _, name := range personsConfigVars {
		if os.Getenv(name) != "" {
			return fail(OpenCode, ReasonFailed, "%s is set, and a pinned run points opencode at a configuration abcd writes "+
				"for it, which the configuration %s names would be set aside by or override; abcd never changes how your "+
				"tools work, so the run is not started: unset %s for pinned runs, or route the role to a model without a pin",
				name, name, name)
		}
	}
	return nil
}

// pinnedSeal is the seal of a pinned run: opencode's own and the run's
// configuration. personsOpenCodeConfig has refused a parent that names one
// of its own, so nothing the person set is replaced.
func pinnedSeal(p sealedPin) []string {
	return append(slices.Clone(openCodeSeal), "OPENCODE_CONFIG="+p.path)
}

// version asks bin for its version under seal, "" when it gives none that
// reads as one.
func (o *OpenCodeCLI) version(ctx context.Context, bin string, seal []string, dir string) string {
	res, err := o.launch.run(ctx, OpenCode, bin, []string{"--version"}, seal, dir, versionTimeout)
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(res.stdout)), "\n")
	line = strings.TrimSpace(line)
	if !harnessVersionRe.MatchString(line) {
		return ""
	}
	return line
}
