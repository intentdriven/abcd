package web

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// constLine is the named top-level const declaration of session.js.
func constLine(t *testing.T, js, name string) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^const ` + regexp.QuoteMeta(name) + ` = .*;$`).FindString(js)
	if m == "" {
		t.Fatalf("session.js has no const %s", name)
	}
	return m
}

// runPage runs script under node and decodes the JSON it prints last.
func runPage(t *testing.T, script string, into any) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	path := filepath.Join(t.TempDir(), "page.mjs")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := exec.Command(node, path).CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, got)
	}
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), into); err != nil {
		t.Fatalf("node said %s", got)
	}
}

// A device sign-in the relay refuses for too many under way says so, and one
// the person stops is ended on the relay.
func TestADeviceSignInSaysWhenTooManyAreUnderWayAndEndsWhenStopped(t *testing.T) {
	js := pageScript(t)
	script := `
const WORDS = new Proxy({}, { get: (_, k) => String(k) });
const toasts = [];
function toast(t) { toasts.push(t); }
const requests = [];
let answer = { status: 429, body: { error: "rate-limited" } };
globalThis.fetch = async (url, init) => { requests.push({ url, body: JSON.parse(init.body), keepalive: !!init.keepalive }); return { status: answer.status, json: async () => answer.body }; };
let anchor = { state: "self" };
const session = { relay: "http://127.0.0.1:4000" };
const ui = { device: { hidden: true }, deviceCode: {}, deviceOpen: { focus() {} } };
let device = null;
function pollDevice() {}
` + functionBody(t, js, "relayPost") + "\n" + functionBody(t, js, "startDeviceSignIn") + "\n" + functionBody(t, js, "stopDeviceSignIn") + `
const out = {};
(async () => {
  await startDeviceSignIn();
  answer = { status: 503, body: { error: "busy" } };
  await startDeviceSignIn();
  out.toasts = toasts.slice();
  device = { handle: "h".repeat(22), relay: session.relay, timer: 0 };
  requests.length = 0;
  stopDeviceSignIn(true);
  out.stopped = requests.slice();
  device = { handle: "h".repeat(22), relay: session.relay, timer: 0 };
  requests.length = 0;
  stopDeviceSignIn();
  out.ended = requests.length;
  console.log(JSON.stringify(out));
})();
`
	var out struct {
		Toasts  []string
		Stopped []struct {
			URL       string
			Body      map[string]string
			Keepalive bool
		}
		Ended int
	}
	runPage(t, script, &out)
	if len(out.Toasts) != 2 || out.Toasts[0] != "signInBusy" || out.Toasts[1] != "signInBusy" {
		t.Errorf("too many sign-ins under way: toasts %v, want signInBusy for each", out.Toasts)
	}
	if len(out.Stopped) != 1 || out.Stopped[0].URL != "http://127.0.0.1:4000/auth/device/cancel" || out.Stopped[0].Body["handle"] != strings.Repeat("h", 22) || !out.Stopped[0].Keepalive {
		t.Errorf("a stopped sign-in: %+v, want the relay told to end it", out.Stopped)
	}
	if out.Ended != 0 {
		t.Errorf("a sign-in that ended by itself was cancelled on the relay too (%d requests)", out.Ended)
	}
	if !strings.Contains(js, `signInBusy: "`) {
		t.Error("the page has no words for too many sign-ins under way")
	}
}
