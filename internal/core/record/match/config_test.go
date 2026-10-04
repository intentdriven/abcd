package match

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// roots lays a checkout and a home under one temp dir, writing the repository
// and machine config files when given (0o600, which the machine layer's
// declaration guard admits).
func roots(t *testing.T, repoJSON, machineJSON string) layered.Roots {
	t.Helper()
	base := t.TempDir()
	r := layered.Roots{Repo: filepath.Join(base, "repo"), Home: filepath.Join(base, "home")}
	put := func(p, body string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(r.Repo, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(r.Home, 0o700); err != nil {
		t.Fatal(err)
	}
	if repoJSON != "" {
		put(filepath.Join(r.Repo, ".abcd", "config.json"), repoJSON)
	}
	if machineJSON != "" {
		put(abcdhome.Path(r.Home, "config.json"), machineJSON)
	}
	return r
}

func TestConfigDefaultsWhenNothingIsConfigured(t *testing.T) {
	c, err := LoadConfig(roots(t, `{"docs":{"target":"agents_md"}}`, ""))
	if err != nil {
		t.Fatal(err)
	}
	if c.Threshold != DefaultThreshold || !reflect.DeepEqual(c.Fields, DefaultFields) {
		t.Fatalf("config = %+v, want the bundled defaults", c)
	}
	if c.ThresholdOrigin != "bundled" || c.FieldsOrigin != "bundled" {
		t.Fatalf("origins = %q / %q, want bundled", c.ThresholdOrigin, c.FieldsOrigin)
	}
}

func TestConfigReadsThroughTheLayers(t *testing.T) {
	c, err := LoadConfig(roots(t,
		`{"match":{"threshold":0.75}}`,
		`{"match":{"threshold":0.9,"fields":["issue.body"]}}`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Threshold != 0.75 || c.ThresholdOrigin != ".abcd/config.json" {
		t.Fatalf("threshold %v from %q, want 0.75 from the repo file", c.Threshold, c.ThresholdOrigin)
	}
	if !reflect.DeepEqual(c.Fields, []string{"issue.body"}) || c.FieldsOrigin != "~/.abcd/config.json" {
		t.Fatalf("fields %v from %q, want [issue.body] from the machine file", c.Fields, c.FieldsOrigin)
	}
	if !c.Compares(FieldIssueBody) || c.Compares(FieldIntentTitle) {
		t.Fatalf("Compares disagrees with fields %v", c.Fields)
	}
}

func TestConfigRefusesWhatWouldSilentlyDoLess(t *testing.T) {
	cases := map[string]string{
		`{"match":{"treshold":0.7}}`:                          "match.treshold",
		`{"match":{"threshold":0}}`:                           "threshold",
		`{"match":{"threshold":1.5}}`:                         "threshold",
		`{"match":{"threshold":"high"}}`:                      "threshold",
		`{"match":{"fields":[]}}`:                             "fields",
		`{"match":{"fields":["issue.title"]}}`:                "issue.title",
		`{"match":{"fields":["issue.body","issue.body"]}}`:    "twice",
		`{"match":{"fields":["intent.title","intent.body"]}}`: "intent.body",
	}
	for repo, want := range cases {
		_, err := LoadConfig(roots(t, repo, ""))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want a refusal naming %q", repo, err, want)
		}
	}
}
