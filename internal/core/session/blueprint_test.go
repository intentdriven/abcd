package session

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var updateBlueprint = flag.Bool("update-blueprint", false, "rewrite the blueprint goldens under testdata from this run's output")

// testDigest is the checksum the fake release publishes for the linux/amd64
// binary.
const testDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// fakeRelease is a release's assets, served without a network.
type fakeRelease struct {
	checksums string
	found     bool
	err       error
	asked     []string
}

func (f *fakeRelease) FetchReleaseAsset(tag, name string) ([]byte, string, bool, error) {
	f.asked = append(f.asked, tag+"/"+name)
	url := "https://github.com/intentdriven/abcd/releases/download/" + tag + "/" + name
	if f.err != nil {
		return nil, url, false, f.err
	}
	if !f.found {
		return nil, url, false, nil
	}
	return []byte(f.checksums), url, true, nil
}

// testRepos is the repository the test blueprints serve.
var testRepos = []string{"acme/brief"}

func goodRelease() *fakeRelease {
	return &fakeRelease{found: true, checksums: "" +
		"1111111111111111111111111111111111111111111111111111111111111111  abcd-darwin-arm64\n" +
		testDigest + "  abcd-linux-amd64\n" +
		"2222222222222222222222222222222222222222222222222222222222222222  abcd-linux-arm64\n"}
}

func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", "blueprint", name)
	if *updateBlueprint {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update-blueprint to write it)", err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s differs from its golden:\n--- got\n%s\n--- want\n%s", name, got, want)
	}
}

func TestBlueprintMatchesItsGoldens(t *testing.T) {
	root := t.TempDir()
	rel := goodRelease()
	res, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.1", Fetch: rel, Repos: testRepos})
	if err != nil {
		t.Fatal(err)
	}
	if res.Digest != testDigest || res.Asset != "abcd-linux-amd64" || !strings.HasSuffix(res.Source, "/v0.17.1/checksums.txt") {
		t.Fatalf("result = %+v", res)
	}
	if strings.Join(rel.asked, ",") != "v0.17.1/checksums.txt" {
		t.Fatalf("fetched %v, want the release's checksums alone", rel.asked)
	}
	for _, f := range []string{"Dockerfile", "render.yaml"} {
		data, err := os.ReadFile(filepath.Join(root, ".abcd", "relay", f))
		if err != nil {
			t.Fatal(err)
		}
		golden(t, f+".golden", data)
	}
	for _, f := range res.Files {
		if f.State != "written" {
			t.Fatalf("a first write reported %+v", f)
		}
	}
}

func TestBlueprintPinsTheReleaseAndItsChecksum(t *testing.T) {
	files := renderBlueprint("v0.17.1", testDigest, testRepos, nil)
	docker := string(files["Dockerfile"])
	for _, want := range []string{
		"https://github.com/intentdriven/abcd/releases/download/v0.17.1/abcd-linux-amd64",
		testDigest + "  /usr/local/bin/abcd\" | sha256sum -c -",
		"session serve --listen \\\"0.0.0.0:${PORT}\\\" --behind-proxy",
		"--platform=linux/amd64",
	} {
		if !strings.Contains(docker, want) {
			t.Errorf("Dockerfile lacks %q", want)
		}
	}
	if strings.Contains(docker, "ARG ") {
		t.Error("the pin is a build argument, which a build could override")
	}
	// The base image is pinned by its multi-arch index digest, so a tag moved
	// on the registry cannot change what the relay is built on.
	if !regexp.MustCompile(`(?m)^FROM --platform=linux/amd64 alpine:3\.22@sha256:[0-9a-f]{64}$`).MatchString(docker) {
		t.Errorf("the base image is not pinned by digest:\n%s", docker)
	}
	if !strings.Contains(docker, "alpine:3.22@"+alpineDigest) {
		t.Errorf("the Dockerfile does not pin %s", alpineDigest)
	}
	yaml := string(files["render.yaml"])
	for _, want := range []string{
		"runtime: docker", "plan: free", "dockerfilePath: ./.abcd/relay/Dockerfile", "healthCheckPath: /healthz",
		"- key: ABCD_SESSION_GITHUB_CLIENT_SECRET\n        sync: false",
		"- key: ABCD_SESSION_REPOS\n        value: \"acme/brief\"",
	} {
		if !strings.Contains(yaml, want) {
			t.Errorf("render.yaml lacks %q", want)
		}
	}
	if strings.Count(yaml, "- type: web") != 1 {
		t.Error("render.yaml declares other than one web service")
	}
	if strings.Contains(strings.ToLower(yaml), "funnel") || strings.Contains(strings.ToLower(docker), "funnel") {
		t.Error("the blueprint mentions Funnel")
	}
}

func TestBlueprintRefusesToOverwriteAnEditedFile(t *testing.T) {
	root := t.TempDir()
	if _, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.0", Fetch: goodRelease(), Repos: testRepos}); err != nil {
		t.Fatal(err)
	}
	// Unedited: the same release is unchanged, a new release replaces it.
	res, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.0", Fetch: goodRelease(), Repos: testRepos})
	if err != nil || res.Files[0].State != "unchanged" || res.Files[1].State != "unchanged" {
		t.Fatalf("rerun: %+v %v, want both unchanged", res, err)
	}
	res, err = WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.1", Fetch: goodRelease(), Repos: testRepos})
	if err != nil || res.Files[0].State != "replaced" {
		t.Fatalf("new release: %+v %v, want replaced", res, err)
	}

	// Edited: refused, and neither file is touched.
	docker := filepath.Join(root, ".abcd", "relay", "Dockerfile")
	yaml := filepath.Join(root, ".abcd", "relay", "render.yaml")
	edited := []byte(strings.Replace(string(mustRead(t, yaml)), "plan: free", "plan: starter", 1))
	if err := os.WriteFile(yaml, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	before := mustRead(t, docker)
	_, err = WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.2", Fetch: goodRelease(), Repos: testRepos})
	var ref *BlueprintRefusal
	if !errors.As(err, &ref) || len(ref.Edited) != 1 || ref.Edited[0] != ".abcd/relay/render.yaml" {
		t.Fatalf("err = %v, want a refusal naming render.yaml", err)
	}
	if string(mustRead(t, yaml)) != string(edited) || string(mustRead(t, docker)) != string(before) {
		t.Fatal("a refused write changed a file")
	}

	// A file abcd did not write is never overwritten either.
	if err := os.WriteFile(docker, []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(yaml)
	if _, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.2", Fetch: goodRelease(), Repos: testRepos}); !errors.As(err, &ref) {
		t.Fatalf("a foreign Dockerfile: err = %v, want a refusal", err)
	}
}

func TestBlueprintRefusals(t *testing.T) {
	for _, c := range []struct {
		name    string
		release string
		fetch   *fakeRelease
		want    string
	}{
		{"not a tag", "latest", goodRelease(), "not a release tag"},
		{"a path in the tag", "v1.0.0/../x", goodRelease(), "not a release tag"},
		{"no release", "v9.9.9", &fakeRelease{}, "publishes no checksums.txt"},
		{"no linux binary", "v0.17.1", &fakeRelease{found: true, checksums: "1111111111111111111111111111111111111111111111111111111111111111  abcd-darwin-arm64\n"}, "names no abcd-linux-amd64"},
		{"garbage", "v0.17.1", &fakeRelease{found: true, checksums: "not a manifest\n"}, "malformed"},
		{"the network", "v0.17.1", &fakeRelease{err: errors.New("offline")}, "offline"},
	} {
		root := t.TempDir()
		_, err := WriteBlueprint(BlueprintRequest{Root: root, Release: c.release, Fetch: c.fetch, Repos: testRepos})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
		if _, statErr := os.Stat(filepath.Join(root, ".abcd")); statErr == nil {
			t.Errorf("%s: a refusal created .abcd/", c.name)
		}
	}
}

func TestBlueprintRefusesASymlinkedDirectory(t *testing.T) {
	root := t.TempDir()
	elsewhere := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(root, ".abcd", "relay")); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.1", Fetch: goodRelease(), Repos: testRepos}); err == nil {
		t.Fatal("wrote through a symlinked .abcd/relay")
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
		t.Fatal("a file landed outside the checkout")
	}
}

func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestBlueprintRefusesWithoutARepositoryToServe: the relay it deploys
// listens beyond loopback, so it must name what it serves.
func TestBlueprintRefusesWithoutARepositoryToServe(t *testing.T) {
	for _, repos := range [][]string{nil, {"acme"}, {"acme/brief\"\nx: y"}} {
		root := t.TempDir()
		if _, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.1", Fetch: goodRelease(), Repos: repos}); err == nil {
			t.Errorf("repos %q accepted", repos)
		}
		if _, err := os.Stat(filepath.Join(root, ".abcd")); err == nil {
			t.Errorf("repos %q: a refusal wrote .abcd/", repos)
		}
	}
}

// TestBlueprintNamesThePageOriginsAndTheRelayOrigin: render.yaml names the
// page origins the relay admits, https://abcdesign.app unless others are
// given, and no sign-in app type, since the relay signs in through a GitHub
// App alone; with a relay origin, relays.json names it as the repository's
// trusted relay, stamped like the other files, and without one no
// relays.json is written.
func TestBlueprintNamesThePageOriginsAndTheRelayOrigin(t *testing.T) {
	files := renderBlueprint("v0.17.1", testDigest, testRepos, nil)
	yaml := string(files["render.yaml"])
	if !strings.Contains(yaml, "- key: ABCD_SESSION_ORIGINS\n        value: \"https://abcdesign.app\"") {
		t.Errorf("render.yaml does not name the page origin:\n%s", yaml)
	}
	if strings.Contains(yaml, "APP_TYPE") || strings.Contains(yaml, "oauth") {
		t.Errorf("render.yaml still names a sign-in app type:\n%s", yaml)
	}
	files = renderBlueprint("v0.17.1", testDigest, testRepos, []string{"https://pages.example.org", "https://abcdesign.app"})
	if !strings.Contains(string(files["render.yaml"]), "value: \"https://pages.example.org,https://abcdesign.app\"") {
		t.Errorf("render.yaml does not name the given origins:\n%s", files["render.yaml"])
	}
	if _, err := WriteBlueprint(BlueprintRequest{Root: t.TempDir(), Release: "v0.17.1", Fetch: goodRelease(), Repos: testRepos, Origins: []string{"http://pages.example.org"}}); err == nil {
		t.Error("an http page origin was written into the blueprint")
	}

	root := t.TempDir()
	res, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.1", Fetch: goodRelease(), Repos: testRepos})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 2 {
		t.Fatalf("without a relay origin: %+v", res.Files)
	}
	if _, err := os.Stat(filepath.Join(root, ".abcd", "relay", "relays.json")); err == nil {
		t.Fatal("relays.json was written without a relay origin")
	}

	res, err = WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.1", Fetch: goodRelease(), Repos: testRepos, RelayOrigin: "https://Relay.Example.org/"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 3 || res.Files[2].Path != ".abcd/relay/relays.json" || res.Files[2].State != "written" {
		t.Fatalf("with a relay origin: %+v", res.Files)
	}
	data, err := os.ReadFile(filepath.Join(root, ".abcd", "relay", "relays.json"))
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "relays.json.golden", data)
	var got struct {
		Relays []string `json:"relays"`
	}
	if err := json.Unmarshal(data, &got); err != nil || len(got.Relays) != 1 || got.Relays[0] != "https://relay.example.org" {
		t.Fatalf("relays.json = %v %s", err, data)
	}

	// Untouched, it is replaced; edited, it is refused and nothing written.
	if _, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.2", Fetch: goodRelease(), Repos: testRepos, RelayOrigin: "https://other.example.org"}); err != nil {
		t.Fatalf("replacing an untouched relays.json: %v", err)
	}
	path := filepath.Join(root, ".abcd", "relay", "relays.json")
	if err := os.WriteFile(path, []byte(`{"relays": ["https://evil.example"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var ref *BlueprintRefusal
	if _, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.17.2", Fetch: goodRelease(), Repos: testRepos, RelayOrigin: "https://relay.example.org"}); !errors.As(err, &ref) {
		t.Fatalf("an edited relays.json: %v, want a refusal", err)
	}
}

func TestRelayOriginsAreReadStrictly(t *testing.T) {
	for in, want := range map[string]string{
		"https://relay.example.org":         "https://relay.example.org",
		"https://Relay.Example.org/":        "https://relay.example.org",
		"https://relay.example.org:8443":    "https://relay.example.org:8443",
		"https://abcd-session.onrender.com": "https://abcd-session.onrender.com",
	} {
		if got, err := NormalizeRelayOrigin(in); err != nil || got != want {
			t.Errorf("%q = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{
		"", "http://relay.example.org", "relay.example.org", "https://relay.example.org/path",
		"https://user@relay.example.org", "https://relay.example.org?x=1", "https://*.example.org",
		"https://relay.example.org#f", "https://re\"lay.example.org", "wss://relay.example.org",
	} {
		if _, err := NormalizeRelayOrigin(in); err == nil {
			t.Errorf("%q accepted", in)
		}
	}
}

// TestBlueprintRefusesAReleaseWithoutTheSessionVerb: a release older than the
// first to carry `abcd session` would build a relay container that exits at
// once, so the blueprint refuses it before reading anything; the release the
// running abcd is counts as carrying it, whatever the constant says (final
// review, docs item 1).
func TestBlueprintRefusesAReleaseWithoutTheSessionVerb(t *testing.T) {
	if firstSessionRelease != "v0.17.0" {
		t.Fatalf("firstSessionRelease = %s; the release cut sets it to the first release carrying abcd session", firstSessionRelease)
	}
	for _, c := range []struct {
		release, running string
		ok               bool
	}{
		{"v0.14.1", "", false},
		{"v0.16.9", "dev", false},
		{"v0.17.0-rc.1", "", true},
		{"v0.17.0", "", true},
		{"v1.0.0", "", true},
		{"v0.15.2", "v0.15.2", true},
		{"v0.15.1", "v0.15.2", false},
	} {
		rel := goodRelease()
		_, err := WriteBlueprint(BlueprintRequest{Root: t.TempDir(), Release: c.release, Running: c.running, Fetch: rel, Repos: testRepos})
		if c.ok && err != nil {
			t.Errorf("%s (running %q): %v, want it pinned", c.release, c.running, err)
		}
		if !c.ok && (err == nil || !strings.Contains(err.Error(), "abcd session") || len(rel.asked) != 0) {
			t.Errorf("%s (running %q): %v, fetched %v; want a refusal naming abcd session, before any fetch", c.release, c.running, err, rel.asked)
		}
	}
}
