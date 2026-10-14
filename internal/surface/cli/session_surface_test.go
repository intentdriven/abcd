package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	coresession "github.com/intentdriven/abcd/internal/core/session"
)

// sessionDigest is the checksum the fake release publishes.
const sessionDigest = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

// fakeSessionRelease serves a release's checksums.txt from memory: no test
// reaches the network.
type fakeSessionRelease struct{ asked []string }

func (f *fakeSessionRelease) FetchReleaseAsset(tag, name string) ([]byte, string, bool, error) {
	f.asked = append(f.asked, tag+"/"+name)
	url := "https://github.com/intentdriven/abcd/releases/download/" + tag + "/" + name
	if tag == "v9.9.9" {
		return nil, url, false, nil
	}
	return []byte(sessionDigest + "  abcd-linux-amd64\n"), url, true, nil
}

func swapSessionRelease(t *testing.T) *fakeSessionRelease {
	t.Helper()
	f := &fakeSessionRelease{}
	old := newSessionReleaseFetcher
	newSessionReleaseFetcher = func() (coresession.AssetFetcher, error) { return f, nil }
	t.Cleanup(func() { newSessionReleaseFetcher = old })
	return f
}

// managedSessionRepo is a checkout abcd manages, as the working directory.
func managedSessionRepo(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := filepath.Join(home, "repo")
	gitInitAt(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("<!-- BEGIN ABCD -->\n<!-- END ABCD -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	return repo
}

func TestSessionBlueprintWritesThePinnedFiles(t *testing.T) {
	repo := managedSessionRepo(t)
	rel := swapSessionRelease(t)
	out := string(runCLI(t, "session", "blueprint", "--release", "v0.14.1"))
	for _, want := range []string{".abcd/relay/Dockerfile", ".abcd/relay/render.yaml", "v0.14.1", sessionDigest, "checksums.txt", "Blueprint path"} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not say %q:\n%s", want, out)
		}
	}
	if strings.Contains(strings.ToLower(out), "funnel") {
		t.Errorf("output mentions Funnel:\n%s", out)
	}
	docker, err := os.ReadFile(filepath.Join(repo, ".abcd", "relay", "Dockerfile"))
	if err != nil || !bytes.Contains(docker, []byte(sessionDigest+"  /usr/local/bin/abcd")) {
		t.Fatalf("Dockerfile: %v\n%s", err, docker)
	}
	if strings.Join(rel.asked, ",") != "v0.14.1/checksums.txt" {
		t.Fatalf("fetched %v", rel.asked)
	}

	// --json puts the result alone on stdout; the fetch is named on stderr.
	root := NewRootCommand()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"session", "blueprint", "--release", "v0.14.1", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var res coresession.BlueprintResult
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("%v\n%s", err, stdout.String())
	}
	if !strings.Contains(stderr.String(), "checksums.txt") {
		t.Errorf("stderr does not name the fetch: %q", stderr.String())
	}
	if res.Digest != sessionDigest || len(res.Files) != 2 || res.Files[0].State != "unchanged" {
		t.Fatalf("--json = %+v", res)
	}
}

func TestSessionBlueprintRefusals(t *testing.T) {
	repo := managedSessionRepo(t)
	swapSessionRelease(t)

	// This test binary is no release, so the release must be named.
	out, err := runCLIStdinErr(t, "", "session", "blueprint")
	if exitCode(err) != 2 || !strings.Contains(err.Error(), "--release") {
		t.Fatalf("no --release on a dev build: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"session", "blueprint", "--release", "latest"},
		{"session", "blueprint", "--release", "v9.9.9"},
	} {
		if _, err := runCLIStdinErr(t, "", args...); exitCode(err) != 2 {
			t.Errorf("%v: %v, want exit 2", args, err)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd")); err == nil {
		t.Fatal("a refusal created .abcd/")
	}

	runCLI(t, "session", "blueprint", "--release", "v0.14.1")
	yaml := filepath.Join(repo, ".abcd", "relay", "render.yaml")
	if err := os.WriteFile(yaml, []byte("services: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = runCLIStdinErr(t, "", "session", "blueprint", "--release", "v0.14.2")
	if exitCode(err) != 2 || !strings.Contains(err.Error(), "render.yaml") {
		t.Fatalf("an edited render.yaml: %v, want exit 2 naming it", err)
	}

	// Outside a checkout abcd manages.
	plain := t.TempDir()
	gitInitAt(t, plain)
	t.Chdir(plain)
	if _, err := runCLIStdinErr(t, "", "session", "blueprint", "--release", "v0.14.1"); exitCode(err) != 2 {
		t.Fatalf("an unmanaged checkout: %v, want exit 2", err)
	}
}

// syncBuffer is a writer a running server and a test read at once.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// startServe runs `abcd session serve` with args until the test ends, and
// returns its output so far once it is listening.
func startServe(t *testing.T, args ...string) (*syncBuffer, func() error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	root := NewRootCommand()
	out := &syncBuffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs(append([]string{"session", "serve"}, args...))
	done := make(chan error, 1)
	go func() { done <- root.ExecuteContext(ctx) }()
	stop := func() error {
		cancel()
		select {
		case err := <-done:
			return err
		case <-time.After(10 * time.Second):
			return errors.New("serve did not stop")
		}
	}
	t.Cleanup(func() { _ = stop() })
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), "http://") {
		select {
		case err := <-done:
			t.Fatalf("serve ended: %v\n%s", err, out.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("serve never said where it listens:\n%s", out.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	return out, stop
}

func TestSessionServeListensOnLoopbackAndStops(t *testing.T) {
	t.Setenv("ABCD_SESSION_GITHUB_CLIENT_SECRET", "test-secret-value-never-printed")
	out, stop := startServe(t, "--listen", "127.0.0.1:0", "--client-id", "Iv1.publicid", "--json")
	var started struct {
		URL      string `json:"url"`
		Loopback bool   `json:"loopback"`
		SignIn   bool   `json:"sign_in"`
		Hint     string `json:"hint"`
	}
	if err := json.NewDecoder(strings.NewReader(out.String())).Decode(&started); err != nil {
		t.Fatalf("the start line is not JSON: %v\n%s", err, out.String())
	}
	if !started.Loopback || !started.SignIn || !strings.HasPrefix(started.URL, "http://127.0.0.1:") {
		t.Fatalf("start = %+v", started)
	}
	if !strings.Contains(started.Hint, "tailscale serve") || strings.Contains(strings.ToLower(started.Hint), "funnel") {
		t.Fatalf("hint = %q, want Serve and never Funnel", started.Hint)
	}
	resp, err := http.Get(started.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz = %d", resp.StatusCode)
	}
	if err := stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if strings.Contains(out.String(), "test-secret-value-never-printed") {
		t.Fatalf("the secret reached the output:\n%s", out.String())
	}
	if _, err := http.Get(started.URL + "/healthz"); err == nil {
		t.Fatal("the relay still answers after it stopped")
	}
}

// TestSessionServeOnLoopbackCountsTailnetVisitorsApart: serve on loopback
// is the host's own machine behind Tailscale Serve, so the wrong codes of one
// tailnet address shut out that address and not every visitor Serve proxies.
func TestSessionServeOnLoopbackCountsTailnetVisitorsApart(t *testing.T) {
	out, _ := startServe(t, "--listen", "127.0.0.1:0", "--json")
	var started struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(strings.NewReader(out.String())).Decode(&started); err != nil {
		t.Fatalf("the start line is not JSON: %v\n%s", err, out.String())
	}
	wsURL := "ws" + strings.TrimPrefix(started.URL, "http") + "/session"
	// join tries a wrong code once on ws and returns the error code the
	// relay answers; dial connects as Serve proxies the visitor at addr.
	join := func(ws *websocket.Conn) string {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ws.Write(ctx, websocket.MessageText, []byte(`{"type":"join","code":"ZZZZZZ"}`)); err != nil {
			t.Fatalf("write: %v", err)
		}
		_, data, err := ws.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		code, _ := m["code"].(string)
		return code
	}
	dial := func(addr string) *websocket.Conn {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ws, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: http.Header{
			"Origin": {"https://abcdesign.app"}, "X-Forwarded-For": {addr},
		}})
		if err != nil {
			t.Fatalf("dial as %s: %v", addr, err)
		}
		t.Cleanup(func() { _ = ws.CloseNow() })
		return ws
	}
	mal := dial("100.101.102.103")
	for i := 0; i < coresession.DefaultLimits().MaxFailedJoins; i++ {
		if got := join(mal); got != "bad-code" {
			t.Fatalf("wrong code %d: %q, want bad-code", i, got)
		}
	}
	if got := join(dial("100.101.102.103")); got != "too-many-tries" {
		t.Fatalf("the address past its limit: %q, want too-many-tries", got)
	}
	if got := join(dial("100.64.0.9")); got != "bad-code" {
		t.Fatalf("another tailnet address: %q, want bad-code, not held up by the first", got)
	}
}

func TestSessionServeTextNamesTheTailnetRoute(t *testing.T) {
	t.Setenv("ABCD_SESSION_GITHUB_CLIENT_SECRET", "")
	out, stop := startServe(t, "--listen", "127.0.0.1:0")
	_ = stop()
	text := out.String()
	for _, want := range []string{"loopback", "tailscale serve --bg", "sign-in is not set up"} {
		if !strings.Contains(text, want) {
			t.Errorf("output does not say %q:\n%s", want, text)
		}
	}
	if strings.Contains(strings.ToLower(text), "funnel") {
		t.Errorf("output mentions Funnel:\n%s", text)
	}
}

func TestSessionServeRefusals(t *testing.T) {
	for _, args := range [][]string{
		{"session", "serve", "--listen", "nonsense"},
		{"session", "serve", "--listen", "example.com:80"},
		{"session", "serve", "--listen", "127.0.0.1:99999"},
		{"session", "serve", "--origin", "*"},
		{"session", "serve", "extra"},
	} {
		if _, err := runCLIStdinErr(t, "", args...); exitCode(err) != 2 {
			t.Errorf("%v: %v, want exit 2", args, err)
		}
	}
}

func TestSessionPageWritesThePageAndRefusesAForeignDirectory(t *testing.T) {
	out := filepath.Join(t.TempDir(), "page")
	text := string(runCLI(t, "session", "page", "--out", out))
	for _, want := range []string{"wrote 6 files", "index.html", "404.html", "vendor/collab.js", "/<owner>/<repo>/brief/"} {
		if !strings.Contains(text, want) {
			t.Errorf("output does not say %q:\n%s", want, text)
		}
	}
	var res struct {
		Out      string   `json:"out"`
		Files    []string `json:"files"`
		Replaced bool     `json:"replaced"`
	}
	if err := json.Unmarshal(runCLI(t, "session", "page", "--out", out, "--json"), &res); err != nil {
		t.Fatal(err)
	}
	if !res.Replaced || len(res.Files) != 6 || res.Out != out {
		t.Fatalf("--json = %+v", res)
	}

	foreign := t.TempDir()
	if err := os.WriteFile(filepath.Join(foreign, "notes.md"), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runCLIStdinErr(t, "", "session", "page", "--out", foreign)
	if exitCode(err) != 2 || !strings.Contains(err.Error(), "notes.md") || !strings.Contains(err.Error(), "nothing written") {
		t.Fatalf("a foreign --out: %v, want exit 2 naming notes.md", err)
	}
	if entries, _ := os.ReadDir(foreign); len(entries) != 1 {
		t.Fatalf("a refusal wrote into the directory: %v", entries)
	}
	if _, err := runCLIStdinErr(t, "", "session", "page"); exitCode(err) != 2 {
		t.Fatalf("no --out: %v, want exit 2", err)
	}
}
