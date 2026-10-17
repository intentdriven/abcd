package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// The tests in this file are iss-2610161734413178's opencode half: a role
// routed through opencode to a pinned OpenRouter model runs against a sealed
// per-run opencode configuration carrying the pin, written into a directory
// private to the run, never the person's own opencode configuration and
// never the repository; the route records the configuration's SHA-256 and
// opencode's version. The readback is not made on this path: opencode holds
// the key, so the served provider stays not reported.

const orModel = "meta-llama/llama-3.1-8b-instruct"

var testPin = oracle.Pin{Order: []string{"deepinfra"}, RequireParameters: true, DataCollection: "deny",
	Quantizations: []string{"fp8"}, CanonicalSlug: "meta-llama/llama-3.1-8b-instruct-20240723"}

// wantPinConfig is the sealed configuration, byte for byte.
const wantPinConfig = `{"$schema":"https://opencode.ai/config.json","provider":{"openrouter":{"models":{"` + orModel +
	`":{"options":{"provider":{"order":["deepinfra"],"allow_fallbacks":false,"require_parameters":true,` +
	`"data_collection":"deny","quantizations":["fp8"]}}}}}}}` + "\n"

// envOf reads the environment the named fake launch saw.
func envOf(t *testing.T, f fakeEnv, file string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.log, file))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(string(raw), "\n")
}

// envValue is key's value in env and whether it is set, the last entry
// winning as exec's does.
func envValue(env []string, key string) (string, bool) {
	v, set := "", false
	for _, kv := range env {
		if k, val, ok := strings.Cut(kv, "="); ok && k == key {
			v, set = val, true
		}
	}
	return v, set
}

// personsOpenCode plants the person's own opencode configuration under a
// home of the test's and returns its path and bytes.
func personsOpenCode(t *testing.T) (string, []byte) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	path := filepath.Join(home, ".config", "opencode", "opencode.json")
	body := []byte(`{"provider":{"openrouter":{"options":{"apiKey":"{env:OPENROUTER_API_KEY}"}}}}` + "\n")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, body
}

// TestAPinnedOpenCodeRunIsPointedAtASealedConfiguration: the run's opencode
// reads the pin from a configuration abcd wrote for it alone (OPENCODE_CONFIG,
// which opencode 1.18.31 merges over the person's own); the file is private,
// outside the repository, and gone when the run ends; the person's own
// configuration is untouched, and no inline configuration is set for the run.
func TestAPinnedOpenCodeRunIsPointedAtASealedConfiguration(t *testing.T) {
	own, ownBody := personsOpenCode(t)
	f := newFake(t, "ok", OpenCode)
	// Empty is as good as unset: the parent names no configuration of its
	// own for opencode, so the pinned run is not refused.
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	pin := testPin
	o := newOpenCode("openrouter/" + orModel)
	o.pin = &pin
	ans, _, err := o.Run(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	env := envOf(t, f, OpenCode+".env")
	cfg, set := envValue(env, "OPENCODE_CONFIG")
	if !set || !filepath.IsAbs(cfg) {
		t.Fatalf("OPENCODE_CONFIG = %q, want the run's own", cfg)
	}
	if v, _ := envValue(env, "OPENCODE_CONFIG_CONTENT"); v != "" {
		t.Fatalf("OPENCODE_CONFIG_CONTENT = %q, want no inline configuration", v)
	}
	for _, under := range []string{f.repo, filepath.Dir(own)} {
		if strings.HasPrefix(cfg, under+string(filepath.Separator)) {
			t.Fatalf("the sealed configuration %s is under %s", cfg, under)
		}
	}
	raw, err := os.ReadFile(filepath.Join(f.log, "opencode.config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != wantPinConfig {
		t.Fatalf("sealed configuration =\n%s\nwant\n%s", raw, wantPinConfig)
	}
	if mode, _ := os.ReadFile(filepath.Join(f.log, "opencode.config.mode")); string(mode) != "600 700" {
		t.Fatalf("file and directory modes = %s, want 600 700", mode)
	}
	if _, err := os.Stat(filepath.Dir(cfg)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the run's configuration directory outlived it: %v", err)
	}
	if got, _ := os.ReadFile(own); string(got) != string(ownBody) {
		t.Fatalf("the person's own opencode configuration changed: %s", got)
	}
	sum := sha256.Sum256([]byte(wantPinConfig))
	if ans.PinConfigSHA256 != hex.EncodeToString(sum[:]) || ans.HarnessVersion != "1.18.31" {
		t.Fatalf("pin config sha256 %q, harness version %q", ans.PinConfigSHA256, ans.HarnessVersion)
	}
	if ans.ProviderPin == nil || ans.ProviderPin.CanonicalSlug != testPin.CanonicalSlug {
		t.Fatalf("provider pin = %+v", ans.ProviderPin)
	}
	got := reasons(ans.Unpinned)
	if r, listed := got[oracle.AskedHarnessVersion]; listed {
		t.Errorf("%s is listed as %s on a pinned run", oracle.AskedHarnessVersion, r)
	}
	// The pin was written where opencode reads it, but nothing on this path
	// shows it reached OpenRouter: it is recorded and listed as not verified.
	if got[oracle.AskedProviderPin] != oracle.NotVerified {
		t.Errorf("provider pin listed as %q, want %q", got[oracle.AskedProviderPin], oracle.NotVerified)
	}
	// The readback is not made here: opencode holds the key.
	if got[oracle.ServedProvider] != oracle.NotReported {
		t.Errorf("served provider = %q, want not reported", got[oracle.ServedProvider])
	}
	assertFixedLists(t, ans.Unpinned)
	// The version was asked under the same seal.
	if v, _ := envValue(envOf(t, f, "version.env"), "OPENCODE_CONFIG"); v != cfg {
		t.Errorf("the version was asked with OPENCODE_CONFIG = %q", v)
	}
}

// TestAPinnedRunRefusesThePersonsOwnOpenCodeConfiguration: a parent that
// names a configuration of its own for opencode (a file through
// OPENCODE_CONFIG, inline through OPENCODE_CONFIG_CONTENT, or a directory
// through OPENCODE_CONFIG_DIR, whose opencode.json opencode merges over the
// sealed file) would lose it to the sealed file or override the pin, and with it the key, permission denies or servers it
// carries; abcd never changes how the person's tools work, so the pinned run
// is refused before anything is written or launched, naming the variable and
// what to do.
func TestAPinnedRunRefusesThePersonsOwnOpenCodeConfiguration(t *testing.T) {
	for _, name := range []string{"OPENCODE_CONFIG", "OPENCODE_CONFIG_CONTENT", "OPENCODE_CONFIG_DIR"} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, "ok", OpenCode)
			tmp := t.TempDir()
			t.Setenv("TMPDIR", tmp)
			t.Setenv("OPENCODE_CONFIG", "")
			t.Setenv("OPENCODE_CONFIG_CONTENT", "")
			t.Setenv("OPENCODE_CONFIG_DIR", "")
			t.Setenv(name, "/home/someone/opencode-work.json")
			pin := testPin
			o := newOpenCode("openrouter/" + orModel)
			o.pin = &pin
			_, _, err := o.Run(context.Background(), f.request("ruthless-reviewer"))
			var fl *Failure
			if !errors.As(err, &fl) || fl.Reason != ReasonFailed {
				t.Fatalf("err = %v, want a failure", err)
			}
			for _, want := range []string{name, "unset", "without a pin"} {
				if !strings.Contains(fl.Detail, want) {
					t.Errorf("the refusal does not say %q: %s", want, fl.Detail)
				}
			}
			if strings.Contains(fl.Detail, "opencode-work.json") {
				t.Errorf("the refusal quotes the variable's value: %s", fl.Detail)
			}
			if f.launched(OpenCode) {
				t.Fatal("opencode was launched")
			}
			if _, err := os.Stat(filepath.Join(f.log, "version.env")); err == nil {
				t.Error("the version was asked")
			}
			if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
				t.Fatalf("a configuration was written: %v", entries)
			}
		})
	}
}

// TestAnUnpinnedRunKeepsThePersonsOpenCodeConfiguration: without a pin the
// person's own OPENCODE_CONFIG, OPENCODE_CONFIG_CONTENT and
// OPENCODE_CONFIG_DIR reach opencode as they are.
func TestAnUnpinnedRunKeepsThePersonsOpenCodeConfiguration(t *testing.T) {
	f := newFake(t, "ok", OpenCode)
	t.Setenv("OPENCODE_CONFIG", "/home/someone/opencode-work.json")
	t.Setenv("OPENCODE_CONFIG_CONTENT", `{"permission":{"bash":"deny"}}`)
	t.Setenv("OPENCODE_CONFIG_DIR", "/home/someone/opencode-dir")
	if _, _, err := newOpenCode("openrouter/"+orModel).Run(context.Background(), f.request("ruthless-reviewer")); err != nil {
		t.Fatal(err)
	}
	env := envOf(t, f, OpenCode+".env")
	if v, _ := envValue(env, "OPENCODE_CONFIG"); v != "/home/someone/opencode-work.json" {
		t.Errorf("OPENCODE_CONFIG = %q", v)
	}
	if v, _ := envValue(env, "OPENCODE_CONFIG_CONTENT"); v != `{"permission":{"bash":"deny"}}` {
		t.Errorf("OPENCODE_CONFIG_CONTENT = %q", v)
	}
	if v, _ := envValue(env, "OPENCODE_CONFIG_DIR"); v != "/home/someone/opencode-dir" {
		t.Errorf("OPENCODE_CONFIG_DIR = %q", v)
	}
}

// TestAnUnpinnedOpenCodeRunWritesNoConfiguration: without a pin the launch
// is what it was: no configuration written, no version asked, the provider
// pin listed as not configured.
func TestAnUnpinnedOpenCodeRunWritesNoConfiguration(t *testing.T) {
	f := newFake(t, "ok", OpenCode)
	ans, _, err := newOpenCode("openrouter/"+orModel).Run(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.log, "version.env")); err == nil {
		t.Error("an unpinned run asked for the version")
	}
	if ans.PinConfigSHA256 != "" || ans.ProviderPin != nil {
		t.Errorf("an unpinned run recorded a pin: %+v", ans.Sides)
	}
	if reasons(ans.Unpinned)[oracle.AskedProviderPin] != oracle.NotConfigured {
		t.Error("an unpinned run does not list its provider pin as not configured")
	}
}

// TestAVersionThatDoesNotReadIsNotReported: a pinned run whose opencode
// answers its version with something else runs all the same and lists the
// version as not reported, never recording what it said.
func TestAVersionThatDoesNotReadIsNotReported(t *testing.T) {
	f := newFake(t, "badversion", OpenCode)
	pin := testPin
	o := newOpenCode("openrouter/" + orModel)
	o.pin = &pin
	ans, _, err := o.Run(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if ans.HarnessVersion != "" || reasons(ans.Unpinned)[oracle.AskedHarnessVersion] != oracle.NotReported {
		t.Fatalf("harness version %q, listed %q", ans.HarnessVersion, reasons(ans.Unpinned)[oracle.AskedHarnessVersion])
	}
}

// TestAPinnedRunNeverWritesIntoTheRepository: a temporary directory that
// resolves inside the repository the role runs in is refused before anything
// is written or launched.
func TestAPinnedRunNeverWritesIntoTheRepository(t *testing.T) {
	f := newFake(t, "ok", OpenCode)
	tmp := filepath.Join(f.repo, "tmp")
	if err := os.Mkdir(tmp, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", tmp)
	pin := testPin
	o := newOpenCode("openrouter/" + orModel)
	o.pin = &pin
	_, _, err := o.Run(context.Background(), f.request("ruthless-reviewer"))
	var fl *Failure
	if !errors.As(err, &fl) || fl.Reason != ReasonFailed {
		t.Fatalf("err = %v, want a failure", err)
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Fatalf("the run wrote into the repository: %v", entries)
	}
	if f.launched(OpenCode) {
		t.Fatal("opencode was launched")
	}
}

// TestTheRunnerConfigurationCarriesThePin: a machine that pins the model its
// opencode route names hands the runner the pin; a route to an unpinned
// model carries none.
func TestTheRunnerConfigurationCarriesThePin(t *testing.T) {
	pinned := `"oracle":{"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","key":"openrouter",` +
		`"models":["` + orModel + `","other/model"],"pins":{"` + orModel + `":{"order":["deepinfra"],"quantizations":["fp8"],` +
		`"canonical_slug":"meta-llama/llama-3.1-8b-instruct-20240723"}}}}}`
	c := mustLoad(t, `{`+pinned+`,"runner":{"opencode":{"model":"openrouter/`+orModel+`"}}}`, "")
	o, ok := c.adapter(c.runners[OpenCode]).(*OpenCodeCLI)
	if !ok || o.pin == nil || !slices.Equal(o.pin.Order, []string{"deepinfra"}) {
		t.Fatalf("adapter = %+v", o)
	}
	c = mustLoad(t, `{`+pinned+`,"runner":{"opencode":{"model":"openrouter/other/model"}}}`, "")
	if o := c.adapter(c.runners[OpenCode]).(*OpenCodeCLI); o.pin != nil {
		t.Fatalf("an unpinned model's runner carries a pin: %+v", o.pin)
	}
}
