package scaffold

import (
	"os/exec"
	"strings"
	"testing"
)

// TestReleaseShapeChecksTheTagBeforeBuilding pins the release job's tag
// shape-check (iss-2608261041218890). TAG is inputs.tag or github.ref_name,
// and the Abcd build runs `make build VERSION="${TAG}"`, which Make splices
// textually into a quoted -ldflags recipe: a legal v* ref carrying a double
// quote closes the quoting and runs a command. site.yml already refuses any
// value that is not a vX.Y.Z tag; the release job now makes the same check,
// with the same [[ =~ ]] pattern, before anything consumes the tag.
//
// The step's own script is executed against good and hostile tags, in both
// profiles, and must precede every step that builds or publishes.
func TestReleaseShapeChecksTheTagBeforeBuilding(t *testing.T) {
	const stepName = "- name: Refuse a tag that is not a vX.Y.Z release tag"
	for _, subs := range []Substitutions{AbcdSubstitutions(), BareSubstitutions("main")} {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatal(err)
		}
		rel := jobSection(t, string(rendered.ReleaseYML), "release")
		check := indexOf(t, rel, stepName, "release")
		for _, consumer := range []string{"make build", "go build", "gh release create"} {
			if at := strings.Index(rel, consumer); at >= 0 && at < check {
				t.Errorf("%q runs before the tag shape-check (Abcd=%v)", consumer, subs.Abcd)
			}
		}

		script := runBlock(t, rel[check:])
		for _, c := range []struct {
			tag string
			ok  bool
		}{
			{"v1.2.3", true},
			{"v0.10.0-rc.1", true},
			{"v1.2.3+build.7", true},
			{`v9.9.9";echo INJECTED;x="`, false},
			{"v1.2.3\nforged=1", false},
			{"1.2.3", false},
			{"v1.2", false},
			{"main", false},
		} {
			cmd := exec.Command("bash", "-c", script)
			cmd.Env = []string{"TAG=" + c.tag, "PATH=/usr/bin:/bin"}
			out, err := cmd.CombinedOutput()
			if got := err == nil; got != c.ok {
				t.Errorf("tag %q: accepted=%v, want %v (Abcd=%v)\n%s", c.tag, got, c.ok, subs.Abcd, out)
			}
			// The refusal echoes the tag, so only a line that IS the payload's
			// output shows it ran.
			if strings.Contains("\n"+string(out), "\nINJECTED\n") {
				t.Errorf("tag %q executed its payload (Abcd=%v)", c.tag, subs.Abcd)
			}
		}
	}
}

// runBlock returns the dedented `run: |` body of the first step in s.
func runBlock(t *testing.T, s string) string {
	t.Helper()
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) != "run: |" {
			continue
		}
		key := len(l) - len(strings.TrimLeft(l, " "))
		var body []string
		indent := -1
		for _, b := range lines[i+1:] {
			if strings.TrimSpace(b) == "" {
				body = append(body, "")
				continue
			}
			ind := len(b) - len(strings.TrimLeft(b, " "))
			if ind <= key {
				break
			}
			if indent < 0 {
				indent = ind
			}
			body = append(body, b[indent:])
		}
		return strings.Join(body, "\n")
	}
	t.Fatal("no `run: |` block after the shape-check step")
	return ""
}
