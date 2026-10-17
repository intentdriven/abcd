package runner

// pin.go is the opencode runner's half of iss-2610161734413178: a role routed
// through opencode to a model this machine pins to one OpenRouter provider
// (oracle.Pin) runs against a sealed opencode configuration carrying the pin,
// written for that run alone.
//
// opencode 1.18.31 loads OPENCODE_CONFIG, the path of one more configuration
// file, after the person's global configuration and merges it over it (read
// in the installed binary on 2026-10-17, and checked with `opencode debug
// config` against a scratch home: a pinned order replaced the global one, and
// the global file was left as it was). abcd writes the pin there, under the
// model's options as opencode's OpenRouter provider passes them on
// (provider.<name>.models.<model>.options.provider, opencode's documented
// shape for OpenRouter's routing object), and never touches the person's own
// configuration or a file of the repository: the file sits in a directory
// made for the run (mode 0700, the file 0600) under the system's temporary
// directory, refused when that resolves inside the repository or its
// checkout, and removed when the run ends. OPENCODE_CONFIG_CONTENT, which
// opencode merges later still, is cleared for the run, so an inline
// configuration the parent holds cannot override the pin. A configuration
// opencode reads after both (an organisation's managed configuration or its
// remote account configuration) is the person's own and still applies.
//
// The route records the configuration's SHA-256 and opencode's version, asked
// with --version under the same seal before the run. The key stays in
// opencode's own configuration or environment and abcd never reads it, so the
// generation readback is not made on this path and the served provider, the
// attempts and the provider's own counts stay not reported.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// sealPin writes the pin's configuration for one run of req, outside the
// repository and the checkout, and returns where.
func (o *OpenCodeCLI) sealPin(req Request) (sealedPin, error) {
	raw, err := pinConfig(o.model, *o.pin)
	if err != nil {
		return sealedPin{}, fail(OpenCode, ReasonFailed, "the pinned configuration could not be built")
	}
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

// pinnedSeal is the seal of a pinned run: opencode's own, the run's
// configuration, and the parent's inline configuration cleared.
func pinnedSeal(p sealedPin) []string {
	return append(slices.Clone(openCodeSeal), "OPENCODE_CONFIG="+p.path, "OPENCODE_CONFIG_CONTENT=")
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
