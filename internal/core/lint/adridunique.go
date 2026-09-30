package lint

import (
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

// ruleADRIDUnique refuses two files in the ADR store that answer to one
// decision. It is the decision-side sibling of issue_id_unique,
// intent_lifecycle's id check and spec_id_unique, and shares their one
// validateIDUnique primitive.
const ruleADRIDUnique = "adr_id_unique"

// checkADRIDUnique flags every file in a set of ADR files that claim one id.
//
// Every ADR reader resolves an id to ONE file, so a second claimant is read
// first-wins or refused: an `0037-a.md` marked accepted beside a proposed
// `0037-x.md` would otherwise settle whatever waits on adr-37. A file claims an
// id twice over — by its filename's number, which is how the resolver routes,
// and by its frontmatter `id:`, which is how the readers confirm — and a
// collision through either is refused. Both id vintages (the 0001–0058
// ordinals and the minted stamp) and every spelling of a handle (padded,
// quoted, case-shifted) meet on the one canonical id recordid derives, so no
// spelling slips past as a second record.
//
// The store is listed by recordid.ADRFiles, the resolver's own scan, so the gate
// judges exactly the files a lookup can be handed. The store lies outside
// cfg.Roots' per-root rules, so the rule runs once. An absent store holds no
// decisions and is not an error; a present store that cannot be read is.
func checkADRIDUnique(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	type adrFile struct {
		abs    string
		claims []string
		fields map[string]fmField
	}
	var files []adrFile
	idFiles := map[string][]string{}

	rels, err := recordid.ADRFiles(repoRoot)
	if err != nil {
		return nil, err
	}
	for _, rel := range rels {
		abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
		content, err := readRepoAbs(repoRoot, abs, maxRepoFileBytes)
		if err != nil {
			return nil, err
		}
		fields := frontmatterFields(strings.Split(string(content), "\n"))
		byName := recordid.ADRFileID(filepath.Base(abs))
		claims := []string{byName}
		byID := recordid.CanonADRID(strings.Trim(strings.TrimSpace(fields["id"].value), `"'`))
		if byID != "" && byID != byName {
			claims = append(claims, byID)
		}
		for _, id := range claims {
			idFiles[id] = append(idFiles[id], abs)
		}
		files = append(files, adrFile{abs: abs, claims: claims, fields: fields})
	}

	var out []Finding
	for _, f := range files {
		rel := repoRel(repoRoot, f.abs)
		for _, id := range f.claims {
			out = append(out, validateIDUnique(repoRoot, rel, id, "decision", ruleADRIDUnique, cfg.Severity, f.fields, idFiles)...)
		}
	}
	return out, nil
}
