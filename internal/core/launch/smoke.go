package launch

// smoke.go — the installability smoke: the ASSERTION tier over the surface
// installsurface.go resolves.
//
// The light tier asserts the three things itd-67 names: both manifests parse,
// the marketplace source resolves, and every declared path the payload is
// responsible for is actually carried. It asserts nothing about what those files
// CONTAIN — that is itd-66's deep tier, which consumes the identical resolved
// list and adds import/frontmatter/help assertions in an isolated subprocess.
// Because the tiers share the resolution step, the upgrade replaces the
// assertions here and leaves the surface definition untouched.

import "fmt"

// SmokeTier names how deep an installability check went, so a report never
// implies more assurance than it earned.
type SmokeTier string

// SmokeTierLight is manifest-parse + source-resolve + declared-path-exists.
const SmokeTierLight SmokeTier = "light"

// Finding kinds. They are stable strings because an operator greps them and a
// gate summary counts them.
const (
	findingManifestUnreadable  = "manifest-unreadable"
	findingSourceUnresolved    = "source-unresolved"
	findingNameMismatch        = "plugin-name-mismatch"
	findingMissingPath         = "missing-declared-path"
	findingArchivePinMalformed = "archive-pin-malformed"
)

// SmokeFinding is one reason the payload would not install.
type SmokeFinding struct {
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	Detail string `json:"detail"`
}

// SmokeReport is one installability check.
type SmokeReport struct {
	Tier    SmokeTier      `json:"tier"`
	OK      bool           `json:"ok"`
	Surface InstallSurface `json:"surface"`
	// Checked counts the assertions actually made, so a pass over a payload that
	// declared nothing is visibly vacuous rather than reassuring.
	Checked  int            `json:"checked"`
	Findings []SmokeFinding `json:"findings,omitempty"`
}

// SmokeLight runs the light installability tier over a payload.
//
// It never returns an error: an unreadable manifest is the most serious FINDING
// it can make, and reporting it as a finding keeps the gate's output shape the
// same whether the payload is perfect or unparseable — the dry-run preview
// depends on always having a report to render.
func SmokeLight(tree PayloadTree) SmokeReport {
	report := SmokeReport{Tier: SmokeTierLight}

	surface, err := ResolveInstallSurface(tree)
	if err != nil {
		report.Findings = append(report.Findings, SmokeFinding{
			Kind: findingManifestUnreadable, Detail: err.Error(),
		})
		return report
	}
	report.Surface = surface

	for _, mp := range surface.Marketplace {
		switch mp.SourceKind {
		case SourceMissing:
			report.Checked++
			report.Findings = append(report.Findings, SmokeFinding{
				Kind:   findingSourceUnresolved,
				Path:   mp.Name,
				Detail: fmt.Sprintf("marketplace plugin %q declares no source", mp.Name),
			})
			continue
		case SourceExternal:
			// A remote source is resolvable only over the network. An offline
			// gate records it and does NOT count it as checked — a Checked total
			// that included unasserted entries would overstate the assurance.
			continue
		case SourceArchive:
			// The pin's SHAPE is judged here, offline; whether its digest is the
			// digest of this payload's archive is the release gate's judgement
			// (VerifyArchivePin), which re-renders the archive to answer it.
			report.Checked++
			if err := validatePin(*mp.Pin); err != nil {
				report.Findings = append(report.Findings, SmokeFinding{
					Kind:   findingArchivePinMalformed,
					Path:   mp.Name,
					Detail: fmt.Sprintf("marketplace plugin %q: %v — the harness would refuse to install it", mp.Name, err),
				})
			}
		}
		report.Checked++
		manifest := joinPayloadPath(mp.Root, pluginManifestFile)
		if !tree.Has(manifest) {
			report.Findings = append(report.Findings, SmokeFinding{
				Kind:   findingSourceUnresolved,
				Path:   manifest,
				Detail: fmt.Sprintf("marketplace plugin %q sources %q, which carries no plugin manifest", mp.Name, mp.Source),
			})
			continue
		}
		named, err := readManifest(tree, manifest)
		if err != nil {
			report.Findings = append(report.Findings, SmokeFinding{
				Kind: findingSourceUnresolved, Path: manifest, Detail: err.Error(),
			})
			continue
		}
		if name, _ := named["name"].(string); name != mp.Name {
			report.Findings = append(report.Findings, SmokeFinding{
				Kind:   findingNameMismatch,
				Path:   manifest,
				Detail: fmt.Sprintf("marketplace lists %q but the sourced manifest is named %q — the install id would not resolve", mp.Name, name),
			})
		}
	}

	for _, e := range surface.Entries {
		if e.Requirement != RequirePayload {
			continue
		}
		report.Checked++
		if tree.Has(e.Path) {
			continue
		}
		detail := fmt.Sprintf("declared %s %q is not in the payload", e.Kind, e.Path)
		if e.DeclaredAs != "" {
			detail += fmt.Sprintf(" (declared as %q)", e.DeclaredAs)
		}
		report.Findings = append(report.Findings, SmokeFinding{
			Kind: findingMissingPath, Path: e.Path, Detail: detail,
		})
	}

	report.OK = len(report.Findings) == 0
	return report
}

// joinPayloadPath joins a payload-relative plugin root with a path beneath it.
func joinPayloadPath(root, rel string) string {
	if root == "" {
		return rel
	}
	return root + "/" + rel
}

// smokeRefusals renders a failed smoke as the refusal lines a launch gate
// reports, so an operator reads WHICH path broke rather than a count.
func smokeRefusals(report SmokeReport) []string {
	var out []string
	for _, f := range report.Findings {
		out = append(out, "installability smoke: "+f.Detail)
	}
	return out
}

// smokeDetail summarises the smoke for the gate line.
func smokeDetail(report SmokeReport) string {
	if report.OK {
		return "checked " + itoa(report.Checked) + " declared paths, 0 findings"
	}
	return "checked " + itoa(report.Checked) + " declared paths, " + itoa(len(report.Findings)) + " findings"
}
