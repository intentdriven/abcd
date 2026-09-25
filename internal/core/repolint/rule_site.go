package repolint

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/site"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// siteGates runs the website's gates (adr-47 decision 3) as part of bare `abcd
// lint`, so the one lint runs every target that judges the repository
// (itd-2609212130136102): the docs through docs-currency, the identity through
// identity-positioning, the outbound policy over committed files through
// privacy-hygiene, and the site here. `abcd lint site` runs the same gates on
// their own, over a directory the caller names.
//
// It is Where-gated on the repository declaring a site composition: a repo with
// no .abcd/site.json has no site to gate, and the skip is listed as every
// inapplicable rule is.
//
// The lint writes nothing in the repository, so the site is rendered into a
// fresh temporary directory outside it, checked there, and removed. A site
// that cannot be rendered is a finding rather than an aborted lint: the other
// rules' results are not taken down with it, and a composition that does not
// render is exactly what a gate over the site exists to say.
//
// Every finding is warn severity, as docs-currency's are: the authoritative
// release gate is `abcd lint site`, which exits 1 on any failure, and
// re-raising the same failures as errors here would double-gate one check.
type siteGates struct{}

func (siteGates) Meta() RuleMeta {
	return RuleMeta{
		ID:       "site-gates",
		Severity: SeverityWarn,
		Fix:      "run `abcd lint site` for the grouped report, and fix each finding at its source",
		PolicyInfo: "the website is a rendered surface of the repository (adr-47); its gates hold every published " +
			"string to a source in the record, and a site that fails one is not publishable",
	}
}

// Where: only a repo that declares a site composition.
func (siteGates) Where(ctx Context) bool {
	present, err := fsutil.Exists(filepath.Join(ctx.RepoRoot, filepath.FromSlash(site.ManifestRelPath)))
	return err == nil && present
}

func (siteGates) Eval(ctx Context) ([]Finding, error) {
	tmp, err := os.MkdirTemp("", "abcd-lint-site-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	res, err := site.Check(site.CheckRequest{RepoRoot: ctx.RepoRoot, OutDir: filepath.Join(tmp, "site")})
	if err != nil {
		return []Finding{{
			RuleID:   "site-gates",
			Severity: SeverityWarn,
			File:     site.ManifestRelPath,
			Message:  "the site could not be rendered and checked: " + cleanErr(err),
		}}, nil
	}
	out := make([]Finding, 0, len(res.Findings))
	for _, f := range res.Findings {
		// Where is the rendered page, which lives in the temporary directory;
		// the source span, when the finding has one, is the repository file
		// to fix, so it is the finding's File.
		msg := fmt.Sprintf("[%s] %s: %s", f.Check, f.Where, f.Detail)
		out = append(out, Finding{
			RuleID:   "site-gates",
			Severity: SeverityWarn,
			File:     f.Source,
			Message:  msg,
		})
	}
	return out, nil
}
