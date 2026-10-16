package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Setting up live sessions at repository setup (spc-2610141709430146 step
// 6): the team's own GitHub App, from a manifest.

func TestAppManifestPrefillsTheTeamsApp(t *testing.T) {
	// The cloud's web sign-in returns to the relay's own callback, so the
	// manifest names it only once the relay's address is known; a local
	// session signs in by the device flow and needs none.
	for _, c := range []struct {
		where, relay string
		callbacks    []string
	}{
		{"local", "", nil},
		{"local", "https://brief-relay.onrender.com", nil},
		{"cloud", "", nil},
		{"cloud", "https://brief-relay.onrender.com", []string{"https://brief-relay.onrender.com/auth/callback"}},
		{"both", "https://brief-relay.onrender.com", []string{"https://brief-relay.onrender.com/auth/callback"}},
	} {
		m, err := NewAppManifest("acme/brief", c.where, "http://127.0.0.1:4321/created", c.relay)
		if err != nil {
			t.Fatalf("%s: %v", c.where, err)
		}
		raw, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if got["url"] != "https://github.com/acme/brief" || got["redirect_url"] != "http://127.0.0.1:4321/created" || got["public"] != false {
			t.Errorf("%s: manifest = %s", c.where, raw)
		}
		perms, _ := got["default_permissions"].(map[string]any)
		if len(perms) != 3 || perms["contents"] != "write" || perms["pull_requests"] != "write" || perms["metadata"] != "read" {
			t.Errorf("%s: permissions = %v", c.where, perms)
		}
		if _, hook := got["hook_attributes"]; hook {
			t.Errorf("%s: the manifest asks for a webhook: %s", c.where, raw)
		}
		cbs, _ := got["callback_urls"].([]any)
		if len(cbs) != len(c.callbacks) || (len(cbs) == 1 && cbs[0] != c.callbacks[0]) {
			t.Errorf("%s: callback urls = %v, want %v", c.where, cbs, c.callbacks)
		}
		name, _ := got["name"].(string)
		if name == "" || len(name) > 34 {
			t.Errorf("%s: name %q is not a GitHub App name", c.where, name)
		}
	}
	if _, err := NewAppManifest("acme/brief", "anywhere", "http://127.0.0.1:1/created", ""); err == nil {
		t.Error("an unknown where was accepted")
	}
	if _, err := NewAppManifest("acme", "local", "http://127.0.0.1:1/created", ""); err == nil {
		t.Error("a repository without a name was accepted")
	}
	m, _ := NewAppManifest("a-very-long-organisation-name/an-even-longer-repository-name", "local", "http://127.0.0.1:1/created", "")
	if len(m.Name) > 34 {
		t.Errorf("name %q is longer than GitHub allows", m.Name)
	}
}

// The callback is the relay's own: its origin and /auth/callback, the path
// the relay answers GitHub's return on. A relay address that is not an https
// origin names none.
func TestTheCallbackIsTheRelays(t *testing.T) {
	for in, want := range map[string]string{
		"https://brief-relay.onrender.com":      "https://brief-relay.onrender.com/auth/callback",
		"https://Brief-Relay.onrender.com/":     "https://brief-relay.onrender.com/auth/callback",
		"https://relay.example.org:8443":        "https://relay.example.org:8443/auth/callback",
		"http://brief-relay.onrender.com":       "",
		"https://brief-relay.onrender.com/x":    "",
		"https://brief-relay.onrender.com/?a=1": "",
		"":                                      "",
	} {
		if got := RelayCallbackURL(in); got != want {
			t.Errorf("RelayCallbackURL(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := NewAppManifest("acme/brief", "cloud", "http://127.0.0.1:1/created", "http://insecure.example"); err == nil {
		t.Error("a relay address that is not an https origin was taken for the callback")
	}
}

func TestAppPagesAreGitHubs(t *testing.T) {
	if got := ManifestFormURL("acme", true, "s1"); got != "https://github.com/organizations/acme/settings/apps/new?state=s1" {
		t.Errorf("organisation form = %q", got)
	}
	if got := ManifestFormURL("octo", false, "s1"); got != "https://github.com/settings/apps/new?state=s1" {
		t.Errorf("personal form = %q", got)
	}
	org := CreatedApp{Slug: "acme-brief", OwnerLogin: "acme", OwnerType: "Organization"}
	if got := org.SettingsURL(); got != "https://github.com/organizations/acme/settings/apps/acme-brief" {
		t.Errorf("organisation settings = %q", got)
	}
	user := CreatedApp{Slug: "octo-brief", OwnerLogin: "octo", OwnerType: "User"}
	if got := user.SettingsURL(); got != "https://github.com/settings/apps/octo-brief" {
		t.Errorf("personal settings = %q", got)
	}
	if got := user.InstallURL(); got != "https://github.com/apps/octo-brief/installations/new" {
		t.Errorf("install = %q", got)
	}
}

// TestConversionKeepsOnlyThePublicFields: the manifest conversion answers
// with the app's client secret, webhook secret and private key; none of them
// survives the read.
func TestConversionKeepsOnlyThePublicFields(t *testing.T) {
	body := `{"id": 1, "slug": "acme-brief", "client_id": "Iv23liAbC", "client_secret": "SECRET-ONE",
		"webhook_secret": "SECRET-TWO", "pem": "-----BEGIN RSA PRIVATE KEY-----SECRET-THREE",
		"html_url": "https://github.com/apps/acme-brief", "owner": {"login": "acme", "type": "Organization"}}`
	app, err := ParseConversion([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if app.ClientID != "Iv23liAbC" || app.Slug != "acme-brief" || app.OwnerLogin != "acme" || app.OwnerType != "Organization" {
		t.Fatalf("app = %+v", app)
	}
	if dump, _ := json.Marshal(app); strings.Contains(string(dump), "SECRET") {
		t.Fatalf("a secret survived the read: %s", dump)
	}
	for _, bad := range []string{`{}`, `{"client_id": "bad id!", "slug": "x"}`, `{"client_id": "Iv1", "slug": "../x"}`, `not json`} {
		if _, err := ParseConversion([]byte(bad)); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
}

// TestWriteAppFileKeepsOnlyTheClientID: the file holds the client id and
// nothing else, reads back through ReadAppClientID, and is written through
// the checkout's root.
func TestWriteAppFileKeepsOnlyTheClientID(t *testing.T) {
	root := t.TempDir()
	if err := WriteAppFile(root, "Iv23liAbC"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".abcd", "relay", AppFile))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil || len(got) != 1 || got["client_id"] != "Iv23liAbC" {
		t.Fatalf("%s = %s (%v)", AppFile, raw, err)
	}
	if id, err := ReadAppClientID(root); err != nil || id != "Iv23liAbC" {
		t.Fatalf("read back %q %v", id, err)
	}
	if err := WriteAppFile(root, "not an id!"); err == nil {
		t.Error("a malformed client id was written")
	}
	elsewhere := t.TempDir()
	linked := t.TempDir()
	if err := os.MkdirAll(filepath.Join(linked, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(linked, ".abcd", "relay")); err != nil {
		t.Fatal(err)
	}
	if err := WriteAppFile(linked, "Iv23liAbC"); err == nil {
		t.Error("wrote through a symlinked .abcd/relay")
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
		t.Error("a file landed outside the checkout")
	}
}
