package cli

import (
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// `abcd session setup` (spc-2610141709430146 step 6): the team's own GitHub
// App from a pre-filled manifest, the device-flow tick for local sessions,
// Render's Deploy button for the cloud, and the relay's address recorded.

// fakeManifestAPI answers as GitHub's API does: the owner lookup and the
// manifest conversion, whose answer carries secrets.
func fakeManifestAPI(t *testing.T) (*[]string, func()) {
	t.Helper()
	return fakeManifestAPIOwnedBy(t, `{"login": "acme", "type": "Organization"}`)
}

// fakeManifestAPIOwnedBy is fakeManifestAPI with the converted app's owner
// given, so a test can bring back an app some other account created.
func fakeManifestAPIOwnedBy(t *testing.T, owner string) (*[]string, func()) {
	t.Helper()
	var mu sync.Mutex
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		asked = append(asked, r.Method+" "+r.URL.Path)
		mu.Unlock()
		switch {
		case r.URL.Path == "/users/acme":
			io.WriteString(w, `{"login": "acme", "type": "Organization"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/app-manifests/code123/conversions":
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{"id": 7, "slug": "acme-brief", "client_id": "Iv23liFromGitHub", "client_secret": "SECRET-ONE",
				"webhook_secret": "SECRET-TWO", "pem": "SECRET-THREE", "owner": `+owner+`}`)
		default:
			http.NotFound(w, r)
		}
	}))
	oldAPI, oldVisit := sessionGitHubAPI, sessionSetupVisit
	sessionGitHubAPI = srv.URL
	// The browser: it opens the page, and GitHub, once the person clicks
	// Create, sends it back with a code and the page's state.
	sessionSetupVisit = func(page string) {
		go func() {
			resp, err := http.Get(page)
			if err != nil {
				t.Errorf("open %s: %v", page, err)
				return
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			m := regexp.MustCompile(`action="([^"]+)"`).FindStringSubmatch(string(body))
			if m == nil {
				t.Errorf("no form on the page:\n%s", body)
				return
			}
			action, _ := url.Parse(html.UnescapeString(m[1]))
			back, err := http.Get(strings.TrimSuffix(page, "/") + "/created?code=code123&state=" + url.QueryEscape(action.Query().Get("state")))
			if err != nil {
				t.Errorf("redirect back: %v", err)
				return
			}
			back.Body.Close()
		}()
	}
	return &asked, func() {
		sessionGitHubAPI, sessionSetupVisit = oldAPI, oldVisit
		srv.Close()
	}
}

// sessionConfig gives the managed checkout the config its answer is kept in.
func sessionConfig(t *testing.T, repo, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSessionSetupLocalCreatesTheAppAndSaysToTickDeviceFlow(t *testing.T) {
	repo := managedSessionRepo(t)
	sessionConfig(t, repo, `{"meta": {"schema_version": 1}}`)
	asked, done := fakeManifestAPI(t)
	defer done()
	out, err := runCLIStdinErr(t, "", "session", "setup", "--where", "local")
	if err != nil {
		t.Fatalf("setup: %v\n%s", err, out)
	}
	text := string(out)
	if strings.Contains(text, "SECRET-") {
		t.Fatalf("a secret reached the output:\n%s", text)
	}
	id, err := os.ReadFile(filepath.Join(repo, ".abcd", "relay", "github-app.json"))
	if err != nil || strings.Contains(string(id), "SECRET") || !strings.Contains(string(id), `"client_id": "Iv23liFromGitHub"`) {
		t.Fatalf("github-app.json (%v):\n%s", err, id)
	}
	for _, want := range []string{
		"http://127.0.0.1:",
		"https://github.com/organizations/acme/settings/apps/acme-brief",
		"Enable Device Flow",
		"https://github.com/apps/acme-brief/installations/new",
		"abcd session start",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output does not say %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "render.com") {
		t.Errorf("a local-only setup offers Render:\n%s", text)
	}
	if _, err := os.Stat(filepath.Join(repo, "render.yaml")); err == nil {
		t.Error("a local-only setup wrote a Blueprint")
	}
	cfg, _ := os.ReadFile(filepath.Join(repo, ".abcd", "config.json"))
	if !strings.Contains(string(cfg), `"live": "local"`) {
		t.Errorf("the answer is not recorded:\n%s", cfg)
	}
	if len(*asked) != 2 {
		t.Errorf("asked GitHub %v", *asked)
	}

	// Run again: the app is recorded, so nothing is created twice.
	*asked = nil
	out, err = runCLIStdinErr(t, "", "session", "setup")
	if err != nil {
		t.Fatalf("again: %v\n%s", err, out)
	}
	if len(*asked) != 0 || !strings.Contains(string(out), "already recorded") {
		t.Errorf("a second run asked GitHub %v:\n%s", *asked, out)
	}
}

func TestSessionSetupCloudOffersTheDeployButton(t *testing.T) {
	repo := managedSessionRepo(t)
	sessionConfig(t, repo, `{"meta": {"schema_version": 1}, "sessions": {"live": "cloud"}}`)
	swapSessionRelease(t)
	_, done := fakeManifestAPI(t)
	defer done()
	out, err := runCLIStdinErr(t, "", "session", "setup", "--release", "v0.17.1")
	if err != nil {
		t.Fatalf("setup: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{
		"https://render.com/deploy?repo=https://github.com/acme/brief",
		"Generate a new client secret",
		"ABCD_SESSION_GITHUB_CLIENT_SECRET",
		"Leave Enable Device Flow off",
		"--relay-origin",
		"https://github.com/apps/acme-brief/installations/new",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output does not say %q:\n%s", want, text)
		}
	}
	yaml, err := os.ReadFile(filepath.Join(repo, "render.yaml"))
	if err != nil || !strings.Contains(string(yaml), `value: "Iv23liFromGitHub"`) || strings.Contains(string(yaml), "SECRET-") {
		t.Fatalf("render.yaml (%v):\n%s", err, yaml)
	}

	out, err = runCLIStdinErr(t, "", "session", "setup", "--release", "v0.17.1", "--relay-origin", "https://abcd-relay.onrender.com")
	if err != nil {
		t.Fatalf("--relay-origin: %v\n%s", err, out)
	}
	relays, err := os.ReadFile(filepath.Join(repo, ".abcd", "relay", "relays.json"))
	if err != nil || !strings.Contains(string(relays), "https://abcd-relay.onrender.com") {
		t.Fatalf("relays.json (%v):\n%s", err, relays)
	}
	// The relay's address known, setup names its callback exactly.
	if !strings.Contains(string(out), "https://abcd-relay.onrender.com/auth/callback") {
		t.Errorf("--relay-origin: output does not name the relay's callback:\n%s", out)
	}
}

func TestSessionSetupRefusals(t *testing.T) {
	repo := managedSessionRepo(t)
	sessionConfig(t, repo, `{"meta": {"schema_version": 1}}`)
	_, done := fakeManifestAPI(t)
	defer done()
	if _, err := runCLIStdinErr(t, "", "session", "setup"); exitCode(err) != 2 || !strings.Contains(err.Error(), "--where") {
		t.Errorf("nothing says where: %v, want exit 2 naming --where", err)
	}
	if _, err := runCLIStdinErr(t, "", "session", "setup", "--where", "moon"); exitCode(err) != 2 {
		t.Errorf("an unknown where: %v", err)
	}
	if _, err := runCLIStdinErr(t, "", "session", "setup", "--where", "local", "--relay-origin", "https://x.example.org"); exitCode(err) != 2 {
		t.Errorf("a relay address for local sessions: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd", "relay")); err == nil {
		t.Error("a refusal wrote .abcd/relay/")
	}
	// No refusal changed the recorded answer.
	cfg, _ := os.ReadFile(filepath.Join(repo, ".abcd", "config.json"))
	var m map[string]any
	if json.Unmarshal(cfg, &m) != nil || m["sessions"] != nil {
		t.Errorf("a refusal recorded an answer:\n%s", cfg)
	}
	_ = time.Second
}

// TestSessionSetupChecksTheReleaseBeforeCreatingTheApp: for the cloud, the
// release's checksums are read before GitHub is asked for anything, so a
// release that cannot be pinned refuses with no app created on GitHub and
// nothing written.
func TestSessionSetupChecksTheReleaseBeforeCreatingTheApp(t *testing.T) {
	repo := managedSessionRepo(t)
	sessionConfig(t, repo, `{"meta": {"schema_version": 1}, "sessions": {"live": "both"}}`)
	swapSessionRelease(t)
	asked, done := fakeManifestAPI(t)
	defer done()
	_, err := runCLIStdinErr(t, "", "session", "setup", "--release", "v9.9.9")
	if exitCode(err) != 2 || !strings.Contains(err.Error(), "checksums") {
		t.Fatalf("a release with no checksums: %v, want exit 2 naming them", err)
	}
	if len(*asked) != 0 {
		t.Errorf("GitHub was asked %v before the release was checked", *asked)
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd", "relay")); err == nil {
		t.Error("a refusal wrote .abcd/relay/")
	}
}

// TestSessionSetupWithARecordedAppNamesTheCallbackForTheCloud: the cloud's
// web sign-in returns to the relay's own callback, which the team's app must
// list. With the relay's address, setup names the exact callback to add on
// the app's settings page; before it, it says the callback is the relay's
// address with /auth/callback, to add once Render shows it. The session
// page's own address is never the callback.
func TestSessionSetupWithARecordedAppNamesTheCallbackForTheCloud(t *testing.T) {
	repo := managedSessionRepo(t)
	sessionConfig(t, repo, `{"meta": {"schema_version": 1}}`)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "relay"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "relay", "github-app.json"), []byte(`{"client_id": "Iv23liRecorded"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	swapSessionRelease(t)
	asked, done := fakeManifestAPI(t)
	defer done()
	const callback = "https://brief-relay.onrender.com/auth/callback"
	const page = "https://abcdesign.app/acme/brief/brief/"
	for _, where := range []string{"cloud", "both"} {
		out, err := runCLIStdinErr(t, "", "session", "setup", "--where", where, "--release", "v0.17.1", "--relay-origin", "https://brief-relay.onrender.com")
		if err != nil {
			t.Fatalf("%s: %v\n%s", where, err, out)
		}
		text := string(out)
		if !strings.Contains(text, "already recorded") || !strings.Contains(text, callback) || !strings.Contains(text, "Callback URL") {
			t.Errorf("%s: output does not name the callback URL %s to add:\n%s", where, callback, text)
		}
		if !strings.Contains(text, "wildcard") {
			t.Errorf("%s: output does not say to leave wildcard matching off:\n%s", where, text)
		}
	}
	// Before the relay is deployed, the callback is named by its shape.
	if err := os.Remove(filepath.Join(repo, ".abcd", "relay", "relays.json")); err != nil {
		t.Fatal(err)
	}
	out, err := runCLIStdinErr(t, "", "session", "setup", "--where", "cloud", "--release", "v0.17.1")
	if err != nil {
		t.Fatalf("cloud, no relay yet: %v\n%s", err, out)
	}
	if text := string(out); !strings.Contains(text, "https://<your relay>.onrender.com/auth/callback") {
		t.Errorf("cloud, no relay yet: output does not say the callback is the relay's:\n%s", text)
	}
	out, err = runCLIStdinErr(t, "", "session", "setup", "--where", "local")
	if err != nil {
		t.Fatalf("local: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "/auth/callback") {
		t.Errorf("a local-only setup names a callback URL:\n%s", out)
	}
	if len(*asked) != 0 {
		t.Errorf("GitHub was asked %v with the app already recorded", *asked)
	}
}

// TestSessionSetupWithARecordedAppSaysToRemoveThePageCallback: an earlier
// setup told people to list the session page's own address as the app's
// callback URL. The sign-in now returns to the relay's own callback, so an
// app recorded before is told to remove the page's address if it is listed,
// wherever sessions may run; the page's address is named only in that step,
// never as one to add. An app created now never listed it, and is told
// nothing about it.
func TestSessionSetupWithARecordedAppSaysToRemoveThePageCallback(t *testing.T) {
	repo := managedSessionRepo(t)
	sessionConfig(t, repo, `{"meta": {"schema_version": 1}}`)
	swapSessionRelease(t)
	asked, done := fakeManifestAPI(t)
	defer done()
	const page = "https://abcdesign.app/acme/brief/brief/"
	steps := func(args ...string) []string {
		t.Helper()
		out, err := runCLIStdinErr(t, "", append([]string{"--json", "session", "setup"}, args...)...)
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
		var res struct {
			Steps []string `json:"steps"`
		}
		// The setup page's address is printed before the result, while the
		// app is being created.
		body := string(out)
		if i := strings.Index(body, "\n{"); i >= 0 && !strings.HasPrefix(body, "{") {
			body = body[i+1:]
		}
		if err := json.Unmarshal([]byte(body), &res); err != nil {
			t.Fatalf("%v: not JSON: %v\n%s", args, err, out)
		}
		return res.Steps
	}
	naming := func(steps []string) []string {
		var out []string
		for _, s := range steps {
			if strings.Contains(s, page) {
				out = append(out, s)
			}
		}
		return out
	}
	// Created now: the manifest never listed the page, so nothing is said.
	if got := naming(steps("--where", "local")); len(got) != 0 {
		t.Errorf("an app created now is told about the page's address: %q", got)
	}
	if len(*asked) == 0 {
		t.Fatal("the app was not created")
	}
	*asked = nil
	for _, args := range [][]string{
		{"--where", "local"},
		{"--where", "cloud", "--release", "v0.17.1"},
		{"--where", "both", "--release", "v0.17.1", "--relay-origin", "https://brief-relay.onrender.com"},
	} {
		got := naming(steps(args...))
		if len(got) != 1 || !strings.Contains(got[0], "remove") || !strings.Contains(got[0], "Callback URL") || strings.Contains(got[0], "add ") {
			t.Errorf("%v: steps naming the page's address = %q, want one saying to remove it", args, got)
		}
	}
	if len(*asked) != 0 {
		t.Errorf("GitHub was asked %v with the app already recorded", *asked)
	}
}

// TestSessionSetupRefusedAfterTheChoiceRecordsNothing: a refusal says
// "(nothing written)", so where sessions may run is recorded only once the
// app is: a refusal on the way leaves .abcd/config.json as it was.
func TestSessionSetupRefusedAfterTheChoiceRecordsNothing(t *testing.T) {
	repo := managedSessionRepo(t)
	sessionConfig(t, repo, `{"meta": {"schema_version": 1}}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	old := sessionGitHubAPI
	sessionGitHubAPI = srv.URL
	defer func() { sessionGitHubAPI = old }()
	_, err := runCLIStdinErr(t, "", "session", "setup", "--where", "local")
	if exitCode(err) != 2 || !strings.Contains(err.Error(), "nothing written") {
		t.Fatalf("GitHub unavailable: %v, want exit 2 saying nothing was written", err)
	}
	cfg, _ := os.ReadFile(filepath.Join(repo, ".abcd", "config.json"))
	var m map[string]any
	if json.Unmarshal(cfg, &m) != nil || m["sessions"] != nil {
		t.Errorf("a refusal saying nothing was written recorded an answer:\n%s", cfg)
	}
}

// TestSessionSetupRefusesAnAppAnotherAccountOwns: for an organisation the
// form posts to the organisation's own page, so an app that comes back owned
// by any other account is not the one the person created there, and nothing
// is recorded.
func TestSessionSetupRefusesAnAppAnotherAccountOwns(t *testing.T) {
	repo := managedSessionRepo(t)
	sessionConfig(t, repo, `{"meta": {"schema_version": 1}}`)
	_, done := fakeManifestAPIOwnedBy(t, `{"login": "another-account", "type": "User"}`)
	defer done()
	out, err := runCLIStdinErr(t, "", "session", "setup", "--where", "local")
	if err == nil {
		t.Fatalf("setup recorded an app another account owns:\n%s", out)
	}
	said := string(out) + err.Error()
	if !strings.Contains(said, "another-account") || !strings.Contains(said, "organisation acme") {
		t.Errorf("the refusal does not name both owners:\n%s", said)
	}
	if _, serr := os.Stat(filepath.Join(repo, ".abcd", "relay", "github-app.json")); !os.IsNotExist(serr) {
		t.Errorf("github-app.json was written for another account's app (%v)", serr)
	}
}
