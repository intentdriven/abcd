package credential_test

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// The guided connect against this package's key homes (spc-2610031241482088,
// G3, G4 and G8): the guide runs here, in the credential store's external
// test package, because only here can the keychain home be pointed at the
// fake (the locateKeychain seam) and every reach for it counted.

// guideService is a keyless stand-in on loopback: it lists two models and
// answers a completion, and counts what it was sent.
func guideService(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/models"):
			_, _ = io.WriteString(w, `{"data":[{"id":"vendor/coder-large"},{"id":"vendor/coder-small"}]}`)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/chat/completions"):
			var req struct {
				Model string `json:"model"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			b, _ := json.Marshal(map[string]any{"model": req.Model,
				"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": "ok"}}}})
			_, _ = w.Write(b)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// guideRoots is a home that already holds a connection, a stored key and
// the index, so "byte-identical" has bytes to hold.
func guideRoots(t *testing.T) layered.Roots {
	t.Helper()
	r := layered.Roots{Home: filepath.Join(t.TempDir(), "h"), Repo: filepath.Join(t.TempDir(), "r")}
	for _, d := range []string{filepath.Join(r.Home, ".abcd"), r.Repo} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	cfg := `{"oracle":{"api":{"alpha":{"base_url":"https://alpha.example.com/v1","key":"alpha","models":["vendor/coder-small"]}}}}`
	if err := os.WriteFile(filepath.Join(r.Home, ".abcd", "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := credential.SetMachine(r.Home, "alpha", "alpha-value-not-a-key"); err != nil {
		t.Fatal(err)
	}
	return r
}

// snapshot is every file under each dir, path to bytes.
func snapshot(t *testing.T, dirs ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, d := range dirs {
		_ = filepath.WalkDir(d, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return nil
			}
			b, rerr := os.ReadFile(p)
			if rerr != nil {
				t.Fatal(rerr)
			}
			out[p] = string(b)
			return nil
		})
	}
	return out
}

// runGuide drives the guide to its end the way the page does, the resume
// object round-tripped as JSON, and returns every turn's JSON and the end.
func runGuide(t *testing.T, roots layered.Roots, base string, answers ...string) ([]string, *oracle.GuideDone) {
	t.Helper()
	turn, err := oracle.Guide(context.Background(), oracle.GuideRequest{Roots: roots, BaseURL: base})
	if err != nil {
		t.Fatal(err)
	}
	var said []string
	for i := 0; ; i++ {
		raw, err := json.Marshal(turn)
		if err != nil {
			t.Fatal(err)
		}
		said = append(said, string(raw))
		if turn.Done != nil || i == len(answers) {
			return said, turn.Done
		}
		var back struct {
			Resume oracle.GuideState `json:"resume"`
		}
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatal(err)
		}
		a := answers[i]
		if turn, err = oracle.Guide(context.Background(), oracle.GuideRequest{Roots: roots, Resume: &back.Resume, Answer: &a}); err != nil {
			t.Fatalf("answer %q: %v", a, err)
		}
	}
}

var guidePaths = map[string][]string{
	oracle.KeyHomeNone:     {"lookup", "vendor/coder-large", "none"},
	oracle.KeyHomeExternal: {"lookup", "vendor/coder-large", "key", "external", "LOCAL_API_KEY"},
	oracle.KeyHomeABCD:     {"lookup", "vendor/coder-large", "key", "abcd"},
	oracle.KeyHomeKeychain: {"lookup", "vendor/coder-large", "key", "keychain"},
}

// TestGuideWritesNothing (G3): for each home, a guided run to its end
// against a keyless stand-in leaves the fake home's ~/.abcd, the index, the
// abcd credential file and the fake keychain byte-identical, never reaches
// for the keychain, and ends in exactly one command.
func TestGuideWritesNothing(t *testing.T) {
	for home, answers := range guidePaths {
		t.Run(home, func(t *testing.T) {
			chain := credential.WithFakeKeychain(t, credential.KeychainMacOS)
			reaches := credential.CountKeychainReaches(t)
			roots := guideRoots(t)
			srv := guideService(t)
			before := snapshot(t, roots.Home, roots.Repo, chain)
			said, done := runGuide(t, roots, srv.URL+"/v1", answers...)
			if done == nil || strings.Count(strings.Join(said, "\n"), `"command":`) != 1 {
				t.Fatalf("the guide did not end in exactly one command: %q", said)
			}
			if !strings.Contains(done.Command, "--home "+home) {
				t.Fatalf("the command %q is not for the %s home", done.Command, home)
			}
			if after := snapshot(t, roots.Home, roots.Repo, chain); !reflect.DeepEqual(before, after) {
				t.Fatalf("the guide changed the home:\nbefore %v\nafter  %v", before, after)
			}
			if n := reaches.Load(); n != 0 {
				t.Fatalf("the guide reached for the keychain %d time(s)", n)
			}
		})
	}
}

// TestGuideNeverOffersAHandSavedItem (G8): with an item saved by hand in the
// fake keychain under the service's name, outside abcd's index, the home
// question offers exactly the three homes and decide later, no turn carries
// the item, the guide never reaches for the keychain, and the printed
// command carries no key: run without one, it is refused before anything is
// read or written, so the item is never adopted.
func TestGuideNeverOffersAHandSavedItem(t *testing.T) {
	chain := credential.WithFakeKeychain(t, credential.KeychainMacOS)
	const handSaved = "hand-saved-item-value-for-local"
	if err := os.WriteFile(filepath.Join(chain, "item-local"), []byte(handSaved), 0o600); err != nil {
		t.Fatal(err)
	}
	reaches := credential.CountKeychainReaches(t)
	roots := guideRoots(t)
	srv := guideService(t)
	said, done := runGuide(t, roots, srv.URL+"/v1", guidePaths[oracle.KeyHomeKeychain]...)
	var home struct {
		Ask struct {
			Questions []struct {
				ID      string `json:"id"`
				Options []struct {
					Value string `json:"value"`
				} `json:"options"`
				Later struct {
					Label string `json:"label"`
				} `json:"later"`
			} `json:"questions"`
		} `json:"ask"`
	}
	found := false
	for _, s := range said {
		if strings.Contains(s, handSaved) {
			t.Fatalf("a turn carries the hand-saved item: %s", s)
		}
		if err := json.Unmarshal([]byte(s), &home); err == nil && len(home.Ask.Questions) == 1 && home.Ask.Questions[0].ID == oracle.GuideQHome {
			found = true
			var got []string
			for _, o := range home.Ask.Questions[0].Options {
				got = append(got, o.Value)
			}
			if !reflect.DeepEqual(got, []string{"external", "abcd", "keychain"}) || home.Ask.Questions[0].Later.Label != "Decide later" {
				t.Fatalf("the home question offers %q and %q", got, home.Ask.Questions[0].Later.Label)
			}
		}
	}
	if !found || done == nil {
		t.Fatalf("the run never asked where the key lives, or never ended: %q", said)
	}
	if n := reaches.Load(); n != 0 {
		t.Fatalf("the guide reached for the keychain %d time(s)", n)
	}
	_, err := oracle.Connect(context.Background(), oracle.ConnectRequest{Roots: roots, Provider: done.Provider,
		BaseURL: srv.URL + "/v1", Models: []string{"vendor/coder-large"}, Home: done.Home})
	if err == nil || !strings.Contains(err.Error(), "none was given") {
		t.Fatalf("the command run without a key = %v; want it refused, the hand-saved item never adopted", err)
	}
	if b, _ := os.ReadFile(filepath.Join(chain, "item-local")); string(b) != handSaved {
		t.Fatal("the hand-saved item was changed")
	}
}

// TestGuideKeychainWritesAreTheRunsWrites (G4, the keychain home, which only
// this package can fake): the paths the done turn lists are the paths a run
// of its command reports, in order.
func TestGuideKeychainWritesAreTheRunsWrites(t *testing.T) {
	credential.WithFakeKeychain(t, credential.KeychainMacOS)
	roots := guideRoots(t)
	srv := guideService(t)
	_, done := runGuide(t, roots, srv.URL+"/v1", guidePaths[oracle.KeyHomeKeychain]...)
	if done == nil {
		t.Fatal("no done turn")
	}
	res, err := oracle.Connect(context.Background(), oracle.ConnectRequest{Roots: roots, Provider: done.Provider,
		BaseURL: srv.URL + "/v1", Models: []string{"vendor/coder-large"}, Home: done.Home, Key: "pasted-value-not-a-key"})
	if err != nil {
		t.Fatalf("running the command: %v", err)
	}
	if !reflect.DeepEqual(res.Wrote, done.Writes) {
		t.Fatalf("the run wrote %q; the guide listed %q", res.Wrote, done.Writes)
	}
}
