package session

import (
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
	res, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.14.1", Fetch: rel})
	if err != nil {
		t.Fatal(err)
	}
	if res.Digest != testDigest || res.Asset != "abcd-linux-amd64" || !strings.HasSuffix(res.Source, "/v0.14.1/checksums.txt") {
		t.Fatalf("result = %+v", res)
	}
	if strings.Join(rel.asked, ",") != "v0.14.1/checksums.txt" {
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
	files := RenderBlueprint("v0.14.1", testDigest)
	docker := string(files["Dockerfile"])
	for _, want := range []string{
		"https://github.com/intentdriven/abcd/releases/download/v0.14.1/abcd-linux-amd64",
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
	if _, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.14.0", Fetch: goodRelease()}); err != nil {
		t.Fatal(err)
	}
	// Unedited: the same release is unchanged, a new release replaces it.
	res, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.14.0", Fetch: goodRelease()})
	if err != nil || res.Files[0].State != "unchanged" || res.Files[1].State != "unchanged" {
		t.Fatalf("rerun: %+v %v, want both unchanged", res, err)
	}
	res, err = WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.14.1", Fetch: goodRelease()})
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
	_, err = WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.14.2", Fetch: goodRelease()})
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
	if _, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.14.2", Fetch: goodRelease()}); !errors.As(err, &ref) {
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
		{"no linux binary", "v0.14.1", &fakeRelease{found: true, checksums: "1111111111111111111111111111111111111111111111111111111111111111  abcd-darwin-arm64\n"}, "names no abcd-linux-amd64"},
		{"garbage", "v0.14.1", &fakeRelease{found: true, checksums: "not a manifest\n"}, "malformed"},
		{"the network", "v0.14.1", &fakeRelease{err: errors.New("offline")}, "offline"},
	} {
		root := t.TempDir()
		_, err := WriteBlueprint(BlueprintRequest{Root: root, Release: c.release, Fetch: c.fetch})
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
	if _, err := WriteBlueprint(BlueprintRequest{Root: root, Release: "v0.14.1", Fetch: goodRelease()}); err == nil {
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
