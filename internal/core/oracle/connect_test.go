package oracle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/credential"
)

// The setup's write (criterion 8, the abcd-only home): verify with one call,
// then write the key into the owner-only store and the provider block into
// the machine's configuration, and nothing into the repository.

func connectReq(f *fx, base string) ConnectRequest {
	return ConnectRequest{
		Roots:    f.roots,
		Provider: "openrouter",
		BaseURL:  base,
		Models:   []string{"typesafe/jev-1.13"},
		Home:     KeyHomeABCD,
		Key:      callKey,
		Timeout:  5 * time.Second,
	}
}

func machineFile(f *fx, name string) string { return filepath.Join(f.roots.Home, ".abcd", name) }

func TestConnectVerifiesThenWritesTheBlockAndTheKey(t *testing.T) {
	p := newProvFake(t, 200, chat("typesafe/jev-1.13-20260915", "ok"))
	f := newFx(t)
	f.machineConfig(`{"pace":{"work_minutes":90}}`)
	res, err := Connect(context.Background(), connectReq(f, p.base()))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if n := p.calls.Load(); n != 1 {
		t.Fatalf("verification made %d calls, want one", n)
	}
	if p.auth.Load() != "Bearer "+callKey {
		t.Fatal("the verification call did not carry the key")
	}
	want := CallRecord{Provider: "openrouter", ModelAsked: "typesafe/jev-1.13", ModelReported: "typesafe/jev-1.13-20260915"}
	if res.Verified != want || res.KeyName != "openrouter" || res.KeyHome != KeyHomeABCD {
		t.Fatalf("result = %+v", res)
	}
	if !reflect.DeepEqual(res.Wrote, []string{credential.StorePath, "~/.abcd/config.json"}) {
		t.Fatalf("wrote = %v", res.Wrote)
	}
	for _, name := range []string{"config.json", credential.StoreFileName} {
		fi, err := os.Lstat(machineFile(f, name))
		if err != nil || fi.Mode().Perm() != 0o600 {
			t.Fatalf("%s: %v, mode %v; want 0600", name, err, fi)
		}
	}
	// The configuration reads back: the block, and the pace key it did not own.
	c := f.loadAPI()
	got, ok := c.Provider("openrouter")
	if !ok || got.BaseURL != p.base() || got.Key != "openrouter" || !reflect.DeepEqual(got.Models, []string{"typesafe/jev-1.13"}) {
		t.Fatalf("provider read back = %+v, %v", got, ok)
	}
	raw, _ := os.ReadFile(machineFile(f, "config.json"))
	if !strings.Contains(string(raw), `"work_minutes": 90`) {
		t.Fatalf("the machine configuration lost a key it did not own:\n%s", raw)
	}
	if strings.Contains(string(raw), callKey) {
		t.Fatal("the key was written into the configuration")
	}
	if v, err := credential.Machine(f.roots.Home).Resolve("openrouter"); err != nil || v != callKey {
		t.Fatal("the key does not resolve by name after the setup")
	}
	enc, _ := json.Marshal(res)
	if strings.Contains(string(enc), callKey) {
		t.Fatal("the result carries the key")
	}
	// Nothing in the repository.
	entries, _ := os.ReadDir(f.roots.Repo)
	if len(entries) != 0 {
		t.Fatalf("the setup wrote into the repository: %v", entries)
	}
}

// TestConnectWritesNothingWhenVerificationFails: a key the provider refuses,
// or a model it does not list, leaves the machine exactly as it was.
func TestConnectWritesNothingWhenVerificationFails(t *testing.T) {
	for _, tc := range []struct {
		code  int
		reply string
	}{
		{401, `{"error":{"message":"No auth credentials found ` + callKey + `"}}`},
		{404, `{"error":{"message":"No endpoints found for typesafe/jev-1.13."}}`},
		{200, chat("anthropic/claude-opus-4", "ok")},
	} {
		p := newProvFake(t, tc.code, tc.reply)
		f := newFx(t)
		_, err := Connect(context.Background(), connectReq(f, p.base()))
		if err == nil {
			t.Fatalf("HTTP %d: Connect succeeded", tc.code)
		}
		if strings.Contains(err.Error(), callKey) {
			t.Fatalf("HTTP %d: the refusal carries the key", tc.code)
		}
		if !strings.Contains(err.Error(), "nothing was written") {
			t.Fatalf("HTTP %d: err = %v", tc.code, err)
		}
		for _, name := range []string{"config.json", credential.StoreFileName} {
			if _, statErr := os.Lstat(machineFile(f, name)); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("HTTP %d: %s was written", tc.code, name)
			}
		}
	}
}

// TestConnectDefersTheOtherHomes: the environment-variable and keychain homes
// are the credential store's (itd-2609221017023290); asked for, they are
// refused naming it, before any call and any write.
func TestConnectDefersTheOtherHomes(t *testing.T) {
	for _, home := range []string{KeyHomeExternal, KeyHomeKeychain} {
		p := newProvFake(t, 200, chat("m", "ok"))
		f := newFx(t)
		req := connectReq(f, p.base())
		req.Home = home
		_, err := Connect(context.Background(), req)
		if err == nil || !strings.Contains(err.Error(), "itd-2609221017023290") {
			t.Fatalf("%s: err = %v, want the deferral named", home, err)
		}
		if p.calls.Load() != 0 {
			t.Fatalf("%s: a call was made", home)
		}
		if _, statErr := os.Lstat(filepath.Join(f.roots.Home, ".abcd")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("%s: something was written", home)
		}
	}
}

// TestConnectRefusesBeforeAnyCall: every fault the read would refuse is
// refused here first, with no call made.
func TestConnectRefusesBeforeAnyCall(t *testing.T) {
	cases := map[string]func(*ConnectRequest){
		"denied model":        func(r *ConnectRequest) { r.Models = []string{"anthropic/claude-opus-4"} },
		"no models":           func(r *ConnectRequest) { r.Models = nil },
		"bad model":           func(r *ConnectRequest) { r.Models = []string{"a b"} },
		"duplicate model":     func(r *ConnectRequest) { r.Models = []string{"m/x", "m/x"} },
		"harness":             func(r *ConnectRequest) { r.Provider = Harness },
		"bad provider name":   func(r *ConnectRequest) { r.Provider = "Open Router" },
		"plain http":          func(r *ConnectRequest) { r.BaseURL = "http://api.example.com/v1" },
		"abcd home no key":    func(r *ConnectRequest) { r.Key = "" },
		"none home with key":  func(r *ConnectRequest) { r.Home = KeyHomeNone },
		"unknown home":        func(r *ConnectRequest) { r.Home = "vault" },
		"bad key name":        func(r *ConnectRequest) { r.KeyName = "../x" },
		"key with a new line": func(r *ConnectRequest) { r.Key = callKey + "\nmore" },
	}
	for name, mutate := range cases {
		p := newProvFake(t, 200, chat("m", "ok"))
		f := newFx(t)
		req := connectReq(f, p.base())
		mutate(&req)
		_, err := Connect(context.Background(), req)
		if err == nil {
			t.Errorf("%s: Connect succeeded", name)
			continue
		}
		if strings.Contains(err.Error(), callKey) {
			t.Errorf("%s: the refusal carries the key", name)
		}
		if p.calls.Load() != 0 {
			t.Errorf("%s: a call was made", name)
		}
	}
}

// TestConnectRefusesAProviderAlreadyConfigured: a block is never replaced
// unasked; the person edits ~/.abcd/config.json to change one.
func TestConnectRefusesAProviderAlreadyConfigured(t *testing.T) {
	p := newProvFake(t, 200, chat("m", "ok"))
	f := newFx(t)
	f.machineConfig(`{"oracle":{"api":{"openrouter":{"base_url":"https://openrouter.ai/api/v1","models":["typesafe/jev-1.13"]}}}}`)
	_, err := Connect(context.Background(), connectReq(f, p.base()))
	if err == nil || !strings.Contains(err.Error(), "already configured") {
		t.Fatalf("err = %v", err)
	}
	if p.calls.Load() != 0 {
		t.Fatal("a call was made")
	}
}

// TestConnectRefusesAKeyNameHoldingAnotherValue: the check runs before the
// call, so a person is not billed for a setup that cannot store its key.
func TestConnectRefusesAKeyNameHoldingAnotherValue(t *testing.T) {
	p := newProvFake(t, 200, chat("m", "ok"))
	f := newFx(t)
	if _, err := credential.SetMachine(f.roots.Home, "openrouter", "another-value-0123"); err != nil {
		t.Fatal(err)
	}
	_, err := Connect(context.Background(), connectReq(f, p.base()))
	if err == nil || strings.Contains(err.Error(), "another-value") || strings.Contains(err.Error(), callKey) {
		t.Fatalf("err = %v", err)
	}
	if p.calls.Load() != 0 {
		t.Fatal("a call was made")
	}
}

// TestConnectToALocalServerNeedsNoKey: the none home sends no key and stores
// none, for a local OpenAI-compatible server.
func TestConnectToALocalServerNeedsNoKey(t *testing.T) {
	p := newProvFake(t, 200, chat("local-model", "ok"))
	f := newFx(t)
	req := connectReq(f, p.base())
	req.Provider, req.Home, req.Key = "desk", KeyHomeNone, ""
	res, err := Connect(context.Background(), req)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if a := p.auth.Load(); a != "" {
		t.Fatalf("Authorization = %v", a)
	}
	if !reflect.DeepEqual(res.Wrote, []string{"~/.abcd/config.json"}) || res.KeyName != "" {
		t.Fatalf("result = %+v", res)
	}
	if got, _ := f.loadAPI().Provider("desk"); got.Key != "" {
		t.Fatalf("a key name was recorded for a keyless provider: %+v", got)
	}
}

// TestConcurrentConnectsKeepEveryKeyAndBlock: two setups that overlap must
// not lose each other's key or provider block while each reports it wrote
// them. Every connect's key resolves and every block reads back.
func TestConcurrentConnectsKeepEveryKeyAndBlock(t *testing.T) {
	const connects = 12
	p := newProvFake(t, 200, chat("typesafe/jev-1.13", "ok"))
	for round := 0; round < 3; round++ {
		f := newFx(t)
		var wg sync.WaitGroup
		errs := make(chan error, connects)
		start := make(chan struct{})
		for i := 0; i < connects; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				req := connectReq(f, p.base())
				req.Provider = fmt.Sprintf("provider-%02d", i)
				req.Key = fmt.Sprintf("throwaway-key-%02d", i)
				<-start
				_, err := Connect(context.Background(), req)
				errs <- err
			}(i)
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("round %d: Connect: %v", round, err)
			}
		}
		c := f.loadAPI()
		for i := 0; i < connects; i++ {
			name := fmt.Sprintf("provider-%02d", i)
			if _, ok := c.Provider(name); !ok {
				t.Fatalf("round %d: the block for %s was lost by a concurrent connect", round, name)
			}
			if v, err := credential.Machine(f.roots.Home).Resolve(name); err != nil || v != fmt.Sprintf("throwaway-key-%02d", i) {
				t.Fatalf("round %d: the key for %s was lost by a concurrent connect (%v)", round, name, err)
			}
		}
	}
}

// TestConcurrentConnectsOfOneProviderWriteOneBlock: two setups of the same
// provider both pass the check made before the call; the write re-checks
// under the lock, so the second is refused rather than replacing the first.
func TestConcurrentConnectsOfOneProviderWriteOneBlock(t *testing.T) {
	const connects = 8
	p := newProvFake(t, 200, chat("typesafe/jev-1.13", "ok"))
	f := newFx(t)
	var wg sync.WaitGroup
	var won atomic.Int32
	start := make(chan struct{})
	for i := 0; i < connects; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := connectReq(f, p.base())
			req.Home, req.Key = KeyHomeNone, ""
			<-start
			if _, err := Connect(context.Background(), req); err == nil {
				won.Add(1)
			} else if !strings.Contains(err.Error(), "already configured") {
				t.Errorf("Connect: %v, want a refusal naming the block already configured", err)
			}
		}()
	}
	close(start)
	wg.Wait()
	if n := won.Load(); n != 1 {
		t.Fatalf("%d concurrent setups of one provider reported success; want exactly one", n)
	}
}

// TestConnectNamesAnUnsafeConfigLockRatherThanContention: a lock beside
// ~/.abcd/config.json that is a symlink is refused, and the refusal says so;
// it is not the contention message, because retrying cannot cure a symlink.
// The symlink's target is never created and no block is written.
func TestConnectNamesAnUnsafeConfigLockRatherThanContention(t *testing.T) {
	p := newProvFake(t, 200, chat("local-model", "ok"))
	f := newFx(t)
	dir := filepath.Join(f.roots.Home, ".abcd")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.Symlink(target, filepath.Join(dir, configLockFileName)); err != nil {
		t.Fatal(err)
	}
	req := connectReq(f, p.base())
	req.Provider, req.Home, req.Key = "desk", KeyHomeNone, ""
	_, err := Connect(context.Background(), req)
	if err == nil {
		t.Fatal("Connect succeeded through a symlinked lock")
	}
	msg := err.Error()
	if strings.Contains(msg, "retry") || strings.Contains(msg, "another abcd") {
		t.Fatalf("err = %v, want the unsafe lock named, not contention", err)
	}
	if !strings.Contains(msg, "~/.abcd/"+configLockFileName) || !strings.Contains(msg, "not a regular file") {
		t.Fatalf("err = %v, want it to name the lock and that it is not a regular file", err)
	}
	if strings.Contains(msg, f.roots.Home) {
		t.Fatalf("err = %v carries the home path", err)
	}
	for _, name := range []string{target, machineFile(f, "config.json")} {
		if _, statErr := os.Lstat(name); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("%s was created", name)
		}
	}
}

// TestTheProviderBlockWriteRefusesAConfigNamingAKeyTwice: the write re-reads
// config.json under its lock, and that read is the one the rewrite trusts, so it
// is held to the check LoadAPI makes: a key named twice, or two spellings of
// one, is refused and the file is left as it stands, never collapsed
// last-wins and rewritten without the other spelling (iss-2609261312108500).
func TestTheProviderBlockWriteRefusesAConfigNamingAKeyTwice(t *testing.T) {
	for name, body := range map[string]string{
		"repeat at the top":        `{"pace": 1, "pace": 2}`,
		"case twin under oracle":   `{"oracle": {"api": {}, "API": {"x": {}}}}`,
		"repeat inside a provider": `{"oracle": {"api": {"a": {"base_url": "https://one.example.com", "base_url": "https://two.example.com"}}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			dir := filepath.Join(home, ".abcd")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(dir, "config.json")
			if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			err := writeProviderBlock(home, "desk", map[string]any{"base_url": "http://127.0.0.1:1"})
			if err == nil {
				t.Fatal("the provider block was written over a config naming a key twice")
			}
			if raw, _ := os.ReadFile(p); string(raw) != body {
				t.Fatalf("config.json was rewritten:\n%s", raw)
			}
		})
	}
}
