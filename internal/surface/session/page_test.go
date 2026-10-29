package session

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/surface/session/web"
)

// tailnetHost stands for the host computer's tailnet name in these tests.
const tailnetHost = "host.example.ts.net"

// localFixture is a relay as `abcd session start` runs it: on loopback,
// serving its own page, admitting only its tailnet name and loopback.
func localFixture(t *testing.T) *fixture {
	t.Helper()
	f, _ := deviceFixture(t, func(c *Config) {
		c.LoopbackOnly = true
		c.ServePage = true
		c.Hosts = []string{tailnetHost, "127.0.0.1:8787"}
		c.Origins = []string{"https://" + tailnetHost, "http://127.0.0.1:8787"}
	})
	return f
}

// get asks the relay for path as a browser reaching it as host would.
func get(t *testing.T, f *fixture, host, path string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, f.http.URL+path, nil)
	req.Host = host
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, string(data)
}

// TestALocalSessionServesItsOwnPage: the relay serves the page's embedded
// files from a fixed route table, the HTML only at the brief of a repository
// it serves and carrying the marker that tells the page its relay is its own
// origin; and the page is never framed (adr-2610161400542916).
func TestALocalSessionServesItsOwnPage(t *testing.T) {
	f := localFixture(t)
	resp, body := get(t, f, tailnetHost, "/acme/brief/brief/")
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("the brief: %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if strings.Count(body, web.RelayMarker) != 1 {
		t.Fatalf("the page does not carry the relay marker once:\n%s", body)
	}
	if resp.Header.Get("X-Frame-Options") != "DENY" || !strings.Contains(resp.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Errorf("the page may be framed: %v", resp.Header)
	}
	for path, kind := range map[string]string{
		"/session.js": "text/javascript", "/session.css": "text/css", "/vendor/collab.js": "text/javascript", "/vendor/LICENSES.txt": "text/plain",
	} {
		resp, body := get(t, f, tailnetHost, path)
		if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), kind) || body == "" {
			t.Errorf("%s: %d %s", path, resp.StatusCode, resp.Header.Get("Content-Type"))
		}
	}
	if resp, _ := get(t, f, tailnetHost, "/"); resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/acme/brief/brief/" {
		t.Errorf("/: %d %s, want a redirect to the brief", resp.StatusCode, resp.Header.Get("Location"))
	}
	for _, path := range []string{"/other/repo/brief/", "/acme/brief/", "/../session.go", "/vendor/collab.entry.js", "/index.html", "/404.html", "/acme/brief/brief/x"} {
		if resp, _ := get(t, f, tailnetHost, path); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", path, resp.StatusCode)
		}
	}
	// The relay's own routes still answer.
	if resp, body := get(t, f, "127.0.0.1:8787", "/config"); resp.StatusCode != http.StatusOK || !strings.Contains(body, `"device"`) {
		t.Errorf("/config: %d %s", resp.StatusCode, body)
	}
}

// TestALocalSessionAnswersOnlyItsOwnNames: any other host name is refused
// before anything is read, which is the DNS-rebinding guard, and only the
// tailnet page and loopback are admitted as origins.
func TestALocalSessionAnswersOnlyItsOwnNames(t *testing.T) {
	f := localFixture(t)
	for _, host := range []string{"evil.example", "evil.example:8787", "localhost.evil.example", ""} {
		for _, path := range []string{"/acme/brief/brief/", "/config", "/healthz", "/session.js"} {
			if resp, body := get(t, f, host, path); resp.StatusCode != http.StatusMisdirectedRequest || strings.Contains(body, "client_id") {
				t.Errorf("host %q %s: %d, want 421", host, path, resp.StatusCode)
			}
		}
	}
	for _, o := range []string{"https://abcdesign.app", "http://localhost:9999", "https://evil.example"} {
		req, _ := http.NewRequest(http.MethodPost, f.http.URL+"/auth/device", strings.NewReader(`{}`))
		req.Host = tailnetHost
		req.Header.Set("Origin", o)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("origin %s: %d, want 403", o, resp.StatusCode)
		}
	}
	// The tailnet page itself is admitted.
	req, _ := http.NewRequest(http.MethodPost, f.http.URL+"/auth/device", strings.NewReader(`{}`))
	req.Host = tailnetHost
	req.Header.Set("Origin", "https://"+tailnetHost)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("the tailnet page's device sign-in: %d, want 200", resp.StatusCode)
	}
}

// TestACloudRelayServesNoPage: without ServePage the relay serves no page
// file, as before: the page is the static one on abcdesign.app.
func TestACloudRelayServesNoPage(t *testing.T) {
	f := newFixture(t, nil)
	for _, path := range []string{"/acme/brief/brief/", "/session.js", "/vendor/collab.js"} {
		resp, err := http.Get(f.http.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", path, resp.StatusCode)
		}
	}
}
