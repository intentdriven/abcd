package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `abcd ahoy credential` is the credential store's walkthrough at its front
// door (itd-2609221017023290): bare with a name it explains and writes
// nothing; with --home it verifies with the reading adapter's own call and
// only then stores. No test here chooses the keychain home, which would reach
// the real keychain; the store's own tests run it against a fake.

// providerNamingKey writes a machine configuration whose provider names the
// credential openrouter, served by a fake on the loopback address.
func providerNamingKey(t *testing.T, base string) {
	t.Helper()
	dir := filepath.Join(os.Getenv("HOME"), ".abcd")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := `{"oracle":{"api":{"openrouter":{"base_url":"` + base + `","key":"openrouter","models":["typesafe/jev-1.13"]}}}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
}

// abcdHomeEntries lists every path under ~/.abcd, so a test can prove a verb
// wrote nothing there. The hermetic environment seeds ~/.abcd itself (the
// cache attestation), so the absence of the directory proves nothing.
func abcdHomeEntries(t *testing.T) string {
	t.Helper()
	root := filepath.Join(os.Getenv("HOME"), ".abcd")
	var names []string
	err := filepath.WalkDir(root, func(p string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		names = append(names, rel)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return strings.Join(names, "\n")
}

func TestAhoyCredentialExplainsAndWritesNothing(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	before := abcdHomeEntries(t)
	out, err := runCLIErr(t, "ahoy", "credential", "hosting.cloudflare")
	if err != nil {
		t.Fatalf("ahoy credential: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{"unlocks", "Without it", "The platform keychain is the home abcd recommends", "--home"} {
		if !strings.Contains(text, want) {
			t.Errorf("the explanation does not say %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "unlocks") > strings.Index(text, "The platform keychain") {
		t.Fatal("the homes come before what the credential unlocks")
	}
	if after := abcdHomeEntries(t); after != before {
		t.Fatalf("the explanation wrote under ~/.abcd:\nbefore: %q\nafter:  %q", before, after)
	}
	if _, err := runCLIErr(t, "ahoy", "credential", "no.such.credential"); err == nil {
		t.Fatal("a name no adapter reads was explained")
	}
}

// TestAhoyCredentialVerifiesThenStores: the provider's own call is made with
// the key from stdin, then the key is stored; the output and the result name
// the home and never the key; the board then reads it as set in that home.
func TestAhoyCredentialVerifiesThenStores(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	base, calls, auth := fakeProvider(t, 200, completionReply("typesafe/jev-1.13-20260915"))
	providerNamingKey(t, base)
	out, err := runCLIStdinErr(t, connectKey+"\n", "ahoy", "credential", "openrouter", "--home", "abcd", "--json")
	if err != nil {
		t.Fatalf("ahoy credential: %v\n%s", err, out)
	}
	if strings.Contains(string(out), connectKey) {
		t.Fatal("the key reached the output")
	}
	if calls.Load() != 1 || auth.Load() != "Bearer "+connectKey {
		t.Fatalf("verification: %d call(s)", calls.Load())
	}
	var res struct {
		Name     string   `json:"name"`
		Home     string   `json:"home"`
		Verified bool     `json:"verified"`
		Wrote    []string `json:"wrote"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("--json: %v\n%s", err, out)
	}
	if res.Name != "openrouter" || res.Home != "abcd" || !res.Verified || len(res.Wrote) != 1 || res.Wrote[0] != "~/.abcd/credentials.json" {
		t.Fatalf("result = %+v", res)
	}
	board, err := runCLIErr(t, "ahoy", "--providers")
	if err != nil || !strings.Contains(string(board), "key openrouter (set, abcd home)") {
		t.Fatalf("board: %v\n%s", err, board)
	}
	list, err := runCLIErr(t, "ahoy", "credential")
	if err != nil || !strings.Contains(string(list), "openrouter: set, abcd home") || !strings.Contains(string(list), "hosting.cloudflare: not set") {
		t.Fatalf("list: %v\n%s", err, list)
	}
	if strings.Contains(string(list), connectKey) {
		t.Fatal("the list carries the key")
	}
}

// TestAhoyCredentialKeepsAPointerInTheExternalHome: --env stores where the key
// is and never the key.
func TestAhoyCredentialKeepsAPointerInTheExternalHome(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	base, calls, _ := fakeProvider(t, 200, completionReply("typesafe/jev-1.13"))
	providerNamingKey(t, base)
	t.Setenv("ABCD_TEST_PROVIDER_KEY", connectKey)
	out, err := runCLIErr(t, "ahoy", "credential", "openrouter", "--home", "external", "--env", "ABCD_TEST_PROVIDER_KEY")
	if err != nil {
		t.Fatalf("ahoy credential: %v\n%s", err, out)
	}
	if calls.Load() != 1 || strings.Contains(string(out), connectKey) {
		t.Fatalf("%d call(s); output:\n%s", calls.Load(), out)
	}
	home := os.Getenv("HOME")
	if _, err := os.Lstat(filepath.Join(home, ".abcd", "credentials.json")); err == nil {
		t.Fatal("the external home wrote the abcd-only store")
	}
	raw, err := os.ReadFile(filepath.Join(home, ".abcd", "credential-homes.json"))
	if err != nil || strings.Contains(string(raw), connectKey) || !strings.Contains(string(raw), "ABCD_TEST_PROVIDER_KEY") {
		t.Fatalf("index: %v\n%s", err, raw)
	}
}

// TestAhoyCredentialRefusesAFailedVerification: a key the provider refuses is
// never stored, and the refusal never carries it.
func TestAhoyCredentialRefusesAFailedVerification(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	base, _, _ := fakeProvider(t, 401, `{"error":{"message":"bad key `+connectKey+`"}}`)
	providerNamingKey(t, base)
	out, err := runCLIStdinErr(t, connectKey, "ahoy", "credential", "openrouter", "--home", "abcd")
	if err == nil {
		t.Fatalf("a refused key was stored:\n%s", out)
	}
	if strings.Contains(string(out)+err.Error(), connectKey) {
		t.Fatal("the refusal carries the key")
	}
	if _, statErr := os.Lstat(filepath.Join(os.Getenv("HOME"), ".abcd", "credentials.json")); statErr == nil {
		t.Fatal("a refused key was stored")
	}
}

// TestARepositoryRouteToAKeyedProviderIsSkippedWithAWarning is ruling CD2 of
// 2026-09-29 at the front doors that read the provider configuration: a
// repository's route to a provider that holds a key no longer refuses the
// command. The route is skipped with one warning on stderr naming the route
// and why, and the command does its work.
func TestARepositoryRouteToAKeyedProviderIsSkippedWithAWarning(t *testing.T) {
	hermeticEnv(t)
	providerNamingKey(t, "https://openrouter.ai/api/v1")
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"),
		[]byte(`{"oracle":{"roles":{"scribe":"openrouter/typesafe/jev-1.13"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	root := NewRootCommand()
	root.SetArgs([]string{"ahoy", "credential"})
	var so, se bytes.Buffer
	root.SetOut(&so)
	root.SetErr(&se)
	if err := root.Execute(); err != nil {
		t.Fatalf("ahoy credential refused over one repository route: %v\n%s%s", err, so.String(), se.String())
	}
	if !strings.Contains(so.String(), "openrouter") {
		t.Errorf("ahoy credential did not list the provider's credential:\n%s", so.String())
	}
	warn := se.String()
	if n := strings.Count(warn, "holds a key"); n != 1 {
		t.Fatalf("stderr carries %d keyed-route warning(s), want one:\n%s", n, warn)
	}
	for _, want := range []string{"oracle.roles.scribe", "openrouter/typesafe/jev-1.13", "skipped", "~/.abcd/config.json"} {
		if !strings.Contains(warn, want) {
			t.Errorf("the warning does not name %q:\n%s", want, warn)
		}
	}
}
