package site

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// A called workflow's `secrets` context is ONLY what its caller passes. A job inside
// one that declares `environment:` gets the environment APPLIED — GitHub even creates
// a deployment record for it — and still resolves none of that environment's secrets
// unless `inherit` unlocked the context first. So the property is not "the deploy job
// names its environment", which was always true; it is "every link in the call chain
// passes secrets", and a single missing link empties the context for everything below
// it.
//
// That is exactly how iss-2608231912566984 happened. release.yml's call to site.yml
// carried `secrets: inherit` from v0.6.3 onward, and auto-release.yml's call to
// release.yml carried nothing, so the chain ran empty and the production deploy failed
// on a credential that had never been conveyed — on v0.6.2, v0.6.3 and v0.6.5. Not
// v0.6.4: that release's render failed first and its deploy was skipped, so it never
// reached the credential. The failure is invisible until release day: nothing parses wrong, no job is
// skipped, the deployment record is created, and wrangler is simply handed an empty
// token at the last step of a release whose binaries are already public.
//
// Measured rather than reasoned. A canary environment secret was read through this
// exact two-level shape: a top-level job declaring the environment saw it SET; nested
// twice with no `secrets:` line, EMPTY; nested twice with `inherit` at the outer call,
// SET.
//
// The templates are checked beside the live workflows because a scaffolded repository
// inherits whatever shape they carry, and a managed repo hitting this would find it on
// its own first release with no record to read.
func TestReleaseChainPassesSecretsAtEveryLevel(t *testing.T) {
	for _, tc := range []struct {
		file   string
		called string
	}{
		{".github/workflows/auto-release.yml", "./.github/workflows/release.yml"},
		{".github/workflows/release.yml", "./.github/workflows/site.yml"},
		// The template names the workflow it calls by substitution: release.yml
		// for a plugin, the gate workflow for any other kind (spelt without spaces
		// so the call reads as one token).
		{"internal/core/launch/scaffold/templates/auto-release.yml.tmpl", "./.github/workflows/<%.ReleaseWorkflow%>"},
		{"internal/core/launch/scaffold/templates/release.yml.tmpl", "./.github/workflows/site.yml"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			lines := workflowLines(t, tc.file)
			if err := callPassesSecrets(lines, tc.called); err != nil {
				t.Errorf("%s: %v\n\nA called workflow resolves NO environment secret unless "+
					"every caller above it passes `secrets: inherit` — the environment applying "+
					"is not enough. Without this the production deploy fails after the binaries "+
					"are published (iss-2608231912566984).", tc.file, err)
			}
		})
	}
}

// usesLine matches a reusable-workflow call and captures its indentation.
var usesLine = regexp.MustCompile(`^( *)uses:\s*(\S+)\s*$`)

// callPassesSecrets finds the job calling `called` and reports whether that job also
// carries a sibling `secrets: inherit`. Hand-parsed for the same reason the input
// contract beside it is: this repository carries no YAML dependency, and the two keys
// read are fixed-indentation siblings.
func callPassesSecrets(lines []string, called string) error {
	for i, line := range lines {
		m := usesLine.FindStringSubmatch(line)
		if m == nil || m[2] != called {
			continue
		}
		indent := m[1]
		// Walk the remaining sibling keys of this job. A key at shallower
		// indentation ends the job, and anything deeper belongs to a sibling's
		// value rather than to the job itself.
		for j := i + 1; j < len(lines); j++ {
			raw := lines[j]
			if strings.TrimSpace(raw) == "" || strings.HasPrefix(strings.TrimSpace(raw), "#") {
				continue
			}
			key := yamlKeyLine.FindStringSubmatch(raw)
			if key == nil {
				continue
			}
			switch {
			case len(key[1]) < len(indent):
				return fmt.Errorf("the job calling %s ends without a `secrets:` key", called)
			case len(key[1]) > len(indent):
				continue
			case key[2] == "secrets":
				if got := strings.TrimSpace(strings.SplitN(raw, ":", 2)[1]); !strings.HasPrefix(got, "inherit") {
					return fmt.Errorf("the job calling %s passes `secrets: %s`, not `inherit`", called, got)
				}
				return nil
			}
		}
		return fmt.Errorf("the job calling %s reaches end of file without a `secrets:` key", called)
	}
	return fmt.Errorf("no job calls %s; this test is checking nothing", called)
}
