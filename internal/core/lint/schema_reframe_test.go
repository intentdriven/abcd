package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// The reframe record (spc-2609020626048705) is held by the committed-tree gate
// to the shape its writer produces, so a record written by hand is judged as a
// written one is.

const (
	fpA = "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb"
	fpB = "3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d"
	fpC = "2e7d2c03a9507ae265ecf5b5356885a53393a2029d241394997265a1a25aefc6"
	fpD = "18ac3e7343f016890c510e93f935261169d9e3f565436429830faf0934f4f8e4"
)

func reframeSchemaConfig() Config {
	stores := admissionStores()
	stores["rfm"] = "work/issues/reframes"
	return Config{
		Roots: []string{"rec"},
		Rules: map[string]RuleConfig{
			ruleRecordSchema: {Enabled: true, Severity: severityBlocker, RecordStores: stores},
		},
	}
}

// reframeRecord renders a reframe record from its lines, one key per line.
func reframeRecord(lines ...string) string {
	return "---\n" + strings.Join(lines, "\n") + "\n---\n\n"
}

func reframeHead(id, occasion string) []string {
	return []string{
		"schema_version: 1", "id: " + id, "occasioned_by: " + occasion,
		"construal_before: " + fpA, "glossary_before: " + fpB, "scope_before: " + fpC,
		"grounds: the detection reading sent the researcher back to the frame",
	}
}

func reframeAfter(changed string) []string {
	return []string{"construal_after: " + fpD, "glossary_after: " + fpB, "scope_after: " + fpC, "changed: " + changed}
}

func TestRecordSchemaJudgesAReframeRecord(t *testing.T) {
	root := admissionCorpus(t)
	dir := "work/issues/reframes/"
	// Controls: an open record, and a complete one naming what moved. A rule
	// watched only failing is a rule that might refuse everything.
	writeFile(t, root, dir+"rfm-1.md", reframeRecord(reframeHead("rfm-1", "rdi-2")...))
	writeFile(t, root, dir+"rfm-2.md", reframeRecord(append(reframeHead("rfm-2", "rdi-2"), reframeAfter(`["construal"]`)...)...))

	cases := map[string]struct {
		lines []string
		want  string
	}{
		"rfm-10.md": {replace(reframeHead("rfm-10", "rdi-2"), "grounds: ", "grounds:"), "'grounds'"},
		"rfm-11.md": {drop(reframeHead("rfm-11", "rdi-2"), "glossary_before"), "'glossary_before'"},
		"rfm-12.md": {append(reframeHead("rfm-12", "rdi-2"), "construal_text: the old frame"), "unknown frontmatter property 'construal_text'"},
		"rfm-13.md": {replace(reframeHead("rfm-13", "rdi-2"), "scope_before: ", "scope_before: sha256:beef"), "scope_before"},
		"rfm-14.md": {append(reframeHead("rfm-14", "rdi-2"), reframeAfter(`["construal", "frame"]`)...), "'frame'"},
		"rfm-15.md": {reframeHead("rfm-15", "rdi-9999"), "rdi-9999"},
		"rfm-16.md": {reframeHead("rfm-16", "adm-2"), "not a handle of"},
		"rfm-17.md": {append(reframeHead("rfm-17", "rdi-2"), "construal_after: "+fpD), "after half"},
		"rfm-18.md": {append(reframeHead("rfm-18", "rdi-2"), reframeAfter(`[]`)...), "changed"},
	}
	for name, c := range cases {
		writeFile(t, root, dir+name, reframeRecord(c.lines...))
	}

	fs, err := Lint(reframeSchemaConfig(), root)
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range cases {
		if !findingWith(fs, filepath.Join("work", "issues", "reframes", name), ruleRecordSchema, c.want) {
			t.Errorf("%s: want a record_schema finding naming %q: %+v", name, c.want, fs)
		}
	}
	for _, ok := range []string{"rfm-1.md", "rfm-2.md"} {
		for _, f := range fs {
			if f.RuleID == ruleRecordSchema && strings.HasSuffix(f.File, ok) {
				t.Errorf("the well-formed control %s drew a finding: %+v", ok, f)
			}
		}
	}
}

// replace swaps the line beginning with prefix for with.
func replace(lines []string, prefix, with string) []string {
	out := append([]string{}, lines...)
	for i, l := range out {
		if strings.HasPrefix(l, prefix) {
			out[i] = with
		}
	}
	return out
}

// drop removes the line for key.
func drop(lines []string, key string) []string {
	var out []string
	for _, l := range lines {
		if !strings.HasPrefix(l, key+":") {
			out = append(out, l)
		}
	}
	return out
}
