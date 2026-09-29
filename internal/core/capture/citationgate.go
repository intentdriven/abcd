package capture

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// citationgate.go holds the two ledger ingests that copy host-delegated prose
// into lint-bound records — the consistency pass's findings and a reading's
// items — to record-lint's prose_citation_resolves before they write
// (iss-2609261835118276). It is the gate the verdict ingest asks
// (iss-2609231036448320), reached through the one registration the front doors
// make (intent.SetProseCitationGate), because this package cannot import
// core/lint: lint's tests import this package.
//
// It behaves exactly as the verdict ingest does. It REFUSES rather than
// sanitises, names every id it refuses, and nothing is written: a payload has
// no way to say an id is illustrative, and deciding what the host meant is not
// the ingest's to do. A repository whose record-lint does not arm the rule over
// the store refuses nothing, and a front door that registered no gate is
// refused outright.
//
// The question is asked of the text as the ingest hands it to the writer,
// before the redactor runs: the redactor replaces secret spans and never
// introduces an id, so the only difference it could make is an id inside a
// redacted span, which this refuses where the gate would not — the safe side.

// citedPart is one piece of host text bound for a record, as the record will
// carry it, and whose words it is, for the refusal.
type citedPart struct {
	what string
	text string
}

// refuseUnresolvedCitations asks the gate ONCE of every part, bound for records
// under dirRel (the repo-relative directory they land in), and refuses naming
// each part that cites an id the gate refuses and the ids it cites. The parts
// are asked as one text, so each must begin outside a fence exactly as it will
// in its record: a frontmatter line always does, and a body is asked after
// its own record's frontmatter and nothing else.
func refuseUnresolvedCitations(repoRoot, dirRel string, parts []citedPart) error {
	var b strings.Builder
	var owner []int // owner[i] is the part holding line i+1
	for i, p := range parts {
		text := strings.TrimSuffix(strings.ReplaceAll(p.text, "\r\n", "\n"), "\n")
		for _, line := range strings.Split(text, "\n") {
			b.WriteString(line)
			b.WriteByte('\n')
			owner = append(owner, i)
		}
	}
	// The gate judges a record by the store its path lies in; the record's own
	// name is not minted yet and does not bear on the answer.
	rel := filepath.ToSlash(filepath.Join(dirRel, "pending.md"))
	cites, err := intent.UnresolvedProseCitations(repoRoot, rel, b.String())
	if errors.Is(err, intent.ErrNoProseCitationGate) {
		return errors.New("capture: no prose-citation gate is registered, so the ingested text cannot be checked " +
			"against the record gate its records must pass; refusing to ingest (nothing written)")
	}
	if err != nil {
		return fmt.Errorf("capture: the prose-citation check over the ingested text: %w", err)
	}
	if len(cites) == 0 {
		return nil
	}
	byPart := map[int][]string{}
	var order []int
	for _, c := range cites {
		if c.Line < 1 || c.Line > len(owner) {
			continue
		}
		i := owner[c.Line-1]
		if _, seen := byPart[i]; !seen {
			order = append(order, i)
		}
		if !containsString(byPart[i], c.ID) {
			byPart[i] = append(byPart[i], c.ID)
		}
	}
	named := make([]string, 0, len(order))
	for _, i := range order {
		named = append(named, parts[i].what+" cites "+strings.Join(byPart[i], ", "))
	}
	return fmt.Errorf("%w: %s — a record id that names no record in this repository; record-lint's "+
		"prose_citation_resolves refuses a record under %s that carries one, and an ingest has no way to mark "+
		"an id illustrative. Re-word the prose to describe the record rather than cite an id that does not "+
		"exist, and ingest again (nothing written)", ErrUnresolvedCitation, strings.Join(named, "; "), dirRel)
}

// ErrUnresolvedCitation is the refusal of host prose citing a record id that
// names no record.
var ErrUnresolvedCitation = errors.New("capture: unresolved record citation")

// repoRelDir is dir relative to repoRoot as a slash path; a directory outside
// the repository (a custom issues root) keeps its own path, which no store of
// the repository's record-lint holds.
func repoRelDir(repoRoot, dir string) string {
	if rel, err := filepath.Rel(repoRoot, dir); err == nil && filepath.IsLocal(rel) {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(dir)
}

// CheckReadingCitations holds every free-text value of a reading's items — the
// pattern and each body field, the host's own words — to the record gate the
// item's record must pass, and refuses naming the item, the field and the ids.
// Each value is asked as the frontmatter line its record carries it on (the
// writer renders every value on one line), so what is judged is what the gate
// will read. IngestReading asks it before it writes anything; the reading
// ingest asks it too, before its orphan sweep and its stage, so a refusal there
// leaves the tree exactly as it found it.
func CheckReadingCitations(req IngestReadingRequest) error {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return err
	}
	dirRel := repoRelDir(repoRoot, filepath.Join(issuesRoot, issueschema.ReadingsDir, req.Run))
	var parts []citedPart
	for i, item := range req.Items {
		keys := make([]string, 0, len(item.Body))
		for k := range item.Body {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		values := []kv{{"pattern", item.Pattern}}
		for _, k := range keys {
			values = append(values, kv{k, item.Body[k]})
		}
		for _, v := range values {
			line, err := yamlScalar(v.val)
			if err != nil {
				// A value the writer cannot render is refused by the writer, on
				// its own terms; it is not this check's to answer.
				continue
			}
			parts = append(parts, citedPart{what: fmt.Sprintf("reading item %d's %s", i+1, v.key), text: v.key + ": " + line})
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return refuseUnresolvedCitations(repoRoot, dirRel, parts)
}
