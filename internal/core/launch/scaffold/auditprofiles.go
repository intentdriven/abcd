package scaffold

import (
	"os"
	"path/filepath"
	"sort"
)

// AuditProfiles is every profile the scaffold's templates render, by name,
// each with a fact set that exercises its guarded regions: abcd's own, a bare
// managed repository with and without derived CI checks and semantic gates,
// and the gate profile with and without an own release workflow. The in-repo
// workflow audit (TestScaffoldedWorkflowsPassTheWorkflowAudit) and CI's zizmor
// audit of the rendered profiles (cmd/scaffold-render, iss-2609251939472371)
// read this one set, so neither can audit a profile the other does not.
func AuditProfiles() map[string]Substitutions {
	semantic := BareSubstitutions("main")
	semantic.SemanticGates = []string{"docs-currency-reviewer"}
	withChecks := BareSubstitutions("trunk")
	withChecks.CIChecks = []string{"Unit tests (linux)", "lint"}
	return map[string]Substitutions{
		"abcd":           AbcdSubstitutions(),
		"bare":           BareSubstitutions("main"),
		"bare+ci-checks": withChecks,
		"bare+semantic":  semantic,
		"gate":           GateSubstitutions("main", ""),
		"gate+own":       GateSubstitutions("main", ".github/workflows/release.yml"),
	}
}

// WriteAuditProfiles renders every AuditProfiles profile and writes its two
// workflows under dir/<profile>/.github/workflows/, the layout a workflow
// auditor discovers, returning the files written in a stable order. dir is a
// scratch directory the caller owns; nothing outside it is written.
func WriteAuditProfiles(dir string) ([]string, error) {
	profiles := AuditProfiles()
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	var written []string
	for _, name := range names {
		r, err := Render(profiles[name])
		if err != nil {
			return written, err
		}
		wf := filepath.Join(dir, name, ".github", "workflows")
		if err := os.MkdirAll(wf, 0o755); err != nil {
			return written, err
		}
		for _, f := range []struct {
			name string
			body []byte
		}{{"release.yml", r.ReleaseYML}, {"auto-release.yml", r.AutoReleaseYML}} {
			p := filepath.Join(wf, f.name)
			if err := os.WriteFile(p, f.body, 0o644); err != nil {
				return written, err
			}
			written = append(written, p)
		}
	}
	return written, nil
}
