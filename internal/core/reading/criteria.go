package reading

// criteria.go reads the declared selection criteria off the committed discipline
// (itd-191; spc-2609020626039834, "The bundle, the kind, the entry, and the
// criteria").
//
// The criteria are never supplied at invocation — itd-191's gate states it, and
// there is no operand that could — so they have to come from the record. A
// comparative bundle without them characterises against nothing, which is why
// the assembly REFUSES rather than assembling a reading that would have to
// invent its own criteria; and the names are parsed rather than restated in Go,
// so amending the slate is an ordinary discipline amendment and a recorded
// change, exactly as that record says.

import (
	"fmt"
	"strings"
)

// CriteriaDiscipline is the record that declares the selection criteria. It is
// the one record the comparative position's disciplines row is narrowed to, and
// the one an entry cannot widen past.
const CriteriaDiscipline = "itd-191"

// criteriaSection is the heading whose bullets declare the slate. It is the
// discipline's own heading, and a discipline that renamed it would be refused
// here rather than read as declaring nothing.
const criteriaSection = "## The rule"

// criteriaSeparator is what divides a criterion's NAME from its gloss in the
// declaration. The discipline writes "Plausibility — the conjecture could work
// by a mechanism we can state." and the name is the text before the em dash.
const criteriaSeparator = "—"

// declaredCriteria parses the criteria the discipline declares out of the
// document the assembly is handing the reading.
//
// It reads the SAME BYTES the bundle carries rather than opening the record
// again. Two readers of one file are how the manifest comes to name a slate the
// reading was not given: the exclusion floor may have redacted the document
// between the two reads, and a criterion the reading never saw is one the ingest
// would then accept.
func declaredCriteria(doc string) ([]string, error) {
	body, ok := sectionBody(doc, criteriaSection)
	if !ok {
		return nil, fmt.Errorf("the criteria discipline %s carries no %q section, so the slate a "+
			"comparative reading characterises against cannot be read; the criteria are declared "+
			"by the record and never supplied at invocation (itd-191)", CriteriaDiscipline, criteriaSection)
	}
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		if idx := strings.Index(name, criteriaSeparator); idx >= 0 {
			name = strings.TrimSpace(name[:idx])
		}
		// A bullet with no name before the separator declares nothing, and a
		// duplicate declares nothing new. Neither is a criterion, and admitting
		// either would let the ingest's undeclared-criterion check pass on a
		// value the discipline does not actually declare.
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("the criteria discipline %s declares no criterion under %q; a "+
			"comparative reading characterises each candidate against each declared criterion, "+
			"and an empty slate is nothing to characterise against (itd-191)",
			CriteriaDiscipline, criteriaSection)
	}
	return out, nil
}

// sectionBody returns the text under a heading, down to the next heading of the
// same level or shallower.
//
// It is deliberately its own small scanner rather than projectField's: that
// function answers "what does this named field project to" over a record with
// frontmatter, and it is reached through a row's Fields. This answers a narrower
// question over a document already in hand, and giving it the projection's
// machinery would tie the criteria parse to a projection nothing applies here.
func sectionBody(doc, heading string) (string, bool) {
	level := len(heading) - len(strings.TrimLeft(heading, "#"))
	lines := strings.Split(strings.ReplaceAll(doc, "\r\n", "\n"), "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return "", false
	}
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		if n := len(trimmed) - len(strings.TrimLeft(trimmed, "#")); n <= level {
			return strings.Join(lines[start:i], "\n"), true
		}
	}
	return strings.Join(lines[start:], "\n"), true
}
