package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// The OpenAI-compatible API adapter's front door (itd-2609081951381895):
// `abcd ahoy --providers` explains the adapter and the homes a key may live
// in, and `abcd ahoy connect` verifies a provider with one call and writes its
// block and its key. No test reaches a network: the provider is a fake on
// this machine's loopback address.

const connectKey = "sk-or-v1-00112233445566778899-not-a-real-key"

// fakeProvider answers every chat completion with reply at code, counting
// calls and remembering the Authorization header.
func fakeProvider(t *testing.T, code int, reply string) (base string, calls *atomic.Int32, auth *atomic.Value) {
	t.Helper()
	calls, auth = &atomic.Int32{}, &atomic.Value{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		auth.Store(r.Header.Get("Authorization"))
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/api/v1", calls, auth
}

func completionReply(model string) string {
	b, _ := json.Marshal(map[string]any{"model": model,
		"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": "ok"}}}})
	return string(b)
}

// TestAhoyProvidersExplainsWithNothingConfigured is criteria 6 and 8's
// explanation: what an aggregator is and what abcd would use it for, that
// everything works on the host without one, the three homes with the keychain
// recommended in prose, and the command that sets one up.
func TestAhoyProvidersExplainsWithNothingConfigured(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	out, err := runCLIErr(t, "ahoy", "--providers")
	if err != nil {
		t.Fatalf("ahoy --providers: %v\n%s", err, out)
	}
	for _, want := range []string{
		"aggregator", "decision models", "every delegated step runs on the host",
		"none configured", "anthropic/* (bundled)",
		"The platform keychain is the safest home", "abcd ahoy connect",
		"--home abcd",
	} {
		if !strings.Contains(string(out), want) {
			t.Errorf("ahoy --providers does not say %q:\n%s", want, out)
		}
	}
	jout, err := runCLIErr(t, "ahoy", "--providers", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Explanation string            `json:"explanation"`
		Providers   []json.RawMessage `json:"providers"`
		Homes       []string          `json:"homes"`
	}
	if err := json.Unmarshal(jout, &v); err != nil {
		t.Fatalf("--json: %v\n%s", err, jout)
	}
	if v.Explanation == "" || v.Providers == nil || len(v.Providers) != 0 || len(v.Homes) != 4 {
		t.Fatalf("--json = %s", jout)
	}
}

// TestAhoyConnectVerifiesThenWrites is criterion 8's write, end to end
// through the front door: the key arrives on stdin, one verification call is
// made with it, the block and the key are written under ~/.abcd/ and nowhere
// else, the output names what was written and never the key, and the board
// reads the provider back with its key set.
func TestAhoyConnectVerifiesThenWrites(t *testing.T) {
	hermeticEnv(t)
	repo := t.TempDir()
	t.Chdir(repo)
	base, calls, auth := fakeProvider(t, 200, completionReply("typesafe/jev-1.13-20260915"))
	out, err := runCLIStdinErr(t, connectKey+"\n", "ahoy", "connect", "openrouter",
		"--base-url", base, "--model", "typesafe/jev-1.13", "--home", "abcd")
	if err != nil {
		t.Fatalf("ahoy connect: %v\n%s", err, out)
	}
	if strings.Contains(string(out), connectKey) {
		t.Fatal("the key reached the output")
	}
	if calls.Load() != 1 || auth.Load() != "Bearer "+connectKey {
		t.Fatalf("verification: %d call(s), auth %v", calls.Load(), auth.Load())
	}
	for _, want := range []string{"typesafe/jev-1.13-20260915", "~/.abcd/credentials.json", "~/.abcd/config.json", "spc-2609251028149555"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("ahoy connect does not say %q:\n%s", want, out)
		}
	}
	home := os.Getenv("HOME")
	for _, name := range []string{"config.json", "credentials.json"} {
		fi, err := os.Lstat(filepath.Join(home, ".abcd", name))
		if err != nil || fi.Mode().Perm() != 0o600 {
			t.Fatalf("~/.abcd/%s: %v", name, err)
		}
	}
	if entries, _ := os.ReadDir(repo); len(entries) != 0 {
		t.Fatalf("ahoy connect wrote into the working directory: %v", entries)
	}
	board, err := runCLIErr(t, "ahoy", "--providers")
	if err != nil {
		t.Fatalf("ahoy --providers: %v\n%s", err, board)
	}
	if !strings.Contains(string(board), "openrouter") || !strings.Contains(string(board), "key openrouter (set)") ||
		strings.Contains(string(board), connectKey) {
		t.Fatalf("board after connect:\n%s", board)
	}
}

// TestAhoyConnectRefusals: a deferred home names the credential store, an
// absent key and a refused verification each exit non-zero, write nothing
// and never print the key.
func TestAhoyConnectRefusals(t *testing.T) {
	cases := []struct {
		name, stdin, home string
		code              int
		reply             string
		wantCalls         int32
		want              string
	}{
		{"keychain deferred", connectKey, "keychain", 200, completionReply("m"), 0, "itd-2609221017023290"},
		{"external deferred", connectKey, "external", 200, completionReply("m"), 0, "itd-2609221017023290"},
		{"no key on stdin", "", "abcd", 200, completionReply("m"), 0, "stdin"},
		{"provider refuses the key", connectKey, "abcd", 401, `{"error":{"message":"bad key ` + connectKey + `"}}`, 1, "nothing was written"},
		{"provider does not list the model", connectKey, "abcd", 404, `{"error":{"message":"No endpoints found"}}`, 1, "HTTP 404"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hermeticEnv(t)
			t.Chdir(t.TempDir())
			base, calls, _ := fakeProvider(t, tc.code, tc.reply)
			out, err := runCLIStdinErr(t, tc.stdin, "ahoy", "connect", "openrouter",
				"--base-url", base, "--model", "typesafe/jev-1.13", "--home", tc.home)
			if err == nil {
				t.Fatalf("ahoy connect succeeded:\n%s", out)
			}
			msg := string(out) + err.Error()
			if strings.Contains(msg, connectKey) {
				t.Fatal("the key reached the output or the error")
			}
			if !strings.Contains(msg, tc.want) {
				t.Fatalf("refusal does not say %q:\n%s", tc.want, msg)
			}
			if n := calls.Load(); n != tc.wantCalls {
				t.Fatalf("%d call(s), want %d", n, tc.wantCalls)
			}
			if _, statErr := os.Lstat(filepath.Join(os.Getenv("HOME"), ".abcd", "credentials.json")); statErr == nil {
				t.Fatal("a refused setup stored the key")
			}
		})
	}
}

// TestAhoyConnectRefusesBareNamingTheExplanation: without a provider name the
// verb does nothing and names the read that explains it.
func TestAhoyConnectRefusesBareNamingTheExplanation(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	out, err := runCLIErr(t, "ahoy", "connect")
	if err == nil || !strings.Contains(string(out)+err.Error(), "abcd ahoy --providers") {
		t.Fatalf("bare ahoy connect = %v\n%s", err, out)
	}
}

// TestBareAhoyNamesTheProviderAdapter is criterion 6 at the bare board: a
// repository on a machine with no provider names the optional adapter, that
// every step runs on the host, and where the explanation is.
func TestBareAhoyNamesTheProviderAdapter(t *testing.T) {
	hermeticEnv(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	out, err := runCLIErr(t, "ahoy")
	if err != nil {
		t.Fatalf("ahoy: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "provider:    none configured (optional); every delegated step runs on the host") ||
		!strings.Contains(string(out), "abcd ahoy --providers") {
		t.Fatalf("bare ahoy does not name the provider adapter:\n%s", out)
	}
}
