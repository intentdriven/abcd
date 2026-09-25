package lab

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// harvestMarker opens every harvest the verb assembles. A harvest.md without
// it was written by hand and is never overwritten.
const harvestMarker = "<!-- assembled by abcd lab harvest; rerunning it replaces this file -->"

// The harvest's sections, in the lifeboat's section shape.
var harvestSections = []string{"Intention", "Method", "Findings — what worked", "Findings — open", "Candidates", "Coverage"}

// Candidate is a product finding the harvest lists for filing through capture.
type Candidate struct {
	Finding     string `json:"finding"`
	Title       string `json:"title"`
	FoundDuring string `json:"found_during"`
	Refutation  string `json:"refutation,omitempty"`
	Command     string `json:"command"`
}

// Gap is a finding the harvest cannot cite from the lab's own records.
type Gap struct {
	Finding string `json:"finding"`
	Reason  string `json:"reason"`
}

// SectionCoverage is one harvest section's grounding, in the lifeboat's
// grounded / partial / blank vocabulary.
type SectionCoverage struct {
	Section string `json:"section"`
	Status  string `json:"status"`
	Note    string `json:"note"`
}

// Harvested is the harvest's result.
type Harvested struct {
	ID         string            `json:"id"`
	Written    bool              `json:"written"`
	Artefact   string            `json:"artefact"`
	Sections   []string          `json:"sections"`
	Probes     []string          `json:"probes"`
	Candidates []Candidate       `json:"candidates"`
	Amendments []string          `json:"amendments"`
	Gaps       []Gap             `json:"gaps"`
	Coverage   []SectionCoverage `json:"coverage"`
	Halted     []string          `json:"halted"`
}

// Harvest assembles the lab's harvest in the lifeboat's section shape —
// intention, method, what worked, what is open, candidates, coverage — from the
// INTENTION, the findings log and the probe records, citing each finding's probe
// records by path. Its product findings are listed as capture candidates, each
// with the capture line that files it, found during this lab.
//
// No claim outlives its input: a finding that cites no probe record or evidence
// file, or cites one that is missing or incomplete, is a gap, and a harvest with
// a gap is refused and writes nothing. A halted lab can still be harvested —
// halting is what ends a lab's mutation, and its gate findings lead the harvest.
func Harvest(repoRoot, id string) (Harvested, error) {
	l, err := open(repoRoot, id)
	if err != nil {
		return Harvested{}, err
	}
	defer l.close()
	res := Harvested{ID: id, Artefact: l.display(harvestName), Sections: harvestSections,
		Probes: []string{}, Candidates: []Candidate{}, Amendments: []string{}, Gaps: []Gap{}, Halted: l.haltedGates()}

	existing, err := l.readDoc(harvestName)
	if err != nil {
		return Harvested{}, err
	}
	if strings.TrimSpace(existing) != "" && !strings.HasPrefix(existing, harvestMarker) {
		return Harvested{}, fmt.Errorf("%w: %s was written by hand; the verb never overwrites it (move it aside to assemble one)", ErrRefused, l.display(harvestName))
	}
	intention, err := l.readDoc(intentionName)
	if err != nil {
		return Harvested{}, err
	}
	fdoc, err := l.readDoc(findingsName)
	if err != nil {
		return Harvested{}, err
	}
	findings := parseFindings(fdoc)

	probes := map[string]probeState{}
	for _, n := range l.probeNames() {
		probes[n] = l.probe(n)
		res.Probes = append(res.Probes, n)
	}
	for _, f := range findings {
		if len(f.Probes) == 0 && len(f.Evidence) == 0 {
			res.Gaps = append(res.Gaps, Gap{Finding: f.ID, Reason: "cites no probe record and no evidence file"})
			continue
		}
		for _, p := range f.Probes {
			ps, ok := probes[p]
			if !ok {
				ps = l.probe(p)
			}
			if !ps.Complete {
				res.Gaps = append(res.Gaps, Gap{Finding: f.ID, Reason: fmt.Sprintf("probe %s is incomplete: %s", termsafe.Sanitize(clip(p)), strings.Join(ps.Missing, ", "))})
			}
		}
		for _, ev := range f.Evidence {
			if !fsutil.ValidRelPath(ev) {
				res.Gaps = append(res.Gaps, Gap{Finding: f.ID, Reason: "evidence " + termsafe.Sanitize(clip(ev)) + " is not a lab-relative path"})
				continue
			}
			if fi, err := l.root.Lstat(ev); err != nil || !fi.Mode().IsRegular() {
				res.Gaps = append(res.Gaps, Gap{Finding: f.ID, Reason: "evidence " + termsafe.Sanitize(clip(ev)) + " is not a file in the lab"})
			}
		}
	}

	found := "lab-" + strings.TrimPrefix(id, "lab-")
	for _, f := range findings {
		switch f.Kind {
		case KindProduct:
			res.Candidates = append(res.Candidates, Candidate{
				Finding: f.ID, Title: f.Title, FoundDuring: found, Refutation: f.Refutation,
				Command: "abcd capture " + shellQuote(f.Title) + " --found-during " + shellQuote(found),
			})
		case KindProcedure:
			res.Amendments = append(res.Amendments, f.ID+" "+f.Title)
		}
	}
	res.Coverage = coverage(intention, findings, probes, res)
	if len(res.Gaps) > 0 {
		return res, fmt.Errorf("%w: %d finding citation(s) cannot be verified from the lab's records; nothing was written", ErrHalted, len(res.Gaps))
	}
	if err := l.writeDoc(harvestName, harvestDoc(l.entry, intention, findings, probes, res)); err != nil {
		return Harvested{}, err
	}
	res.Written = true
	return res, nil
}

// coverage grades each section the way the lifeboat grades a brief section.
func coverage(intention string, findings []Finding, probes map[string]probeState, res Harvested) []SectionCoverage {
	grade := func(ok, some bool) string {
		switch {
		case ok:
			return "grounded"
		case some:
			return "partial"
		}
		return "blank"
	}
	q := section(intention, "Question")
	h := section(intention, "Hypothesis")
	complete := 0
	for _, p := range probes {
		if p.Complete {
			complete++
		}
	}
	var worked, open int
	for _, f := range findings {
		if f.Status == "worked" {
			worked++
		} else {
			open++
		}
	}
	unrefuted := 0
	for _, c := range res.Candidates {
		if c.Refutation == "" {
			unrefuted++
		}
	}
	return []SectionCoverage{
		{"Intention", grade(q != "" && h != "", q != ""), "the question and the hypothesis from INTENTION.md"},
		{"Method", grade(len(probes) > 0 && complete == len(probes), complete > 0), fmt.Sprintf("%d of %d probe record(s) complete", complete, len(probes))},
		{"Findings — what worked", grade(worked > 0, false), fmt.Sprintf("%d finding(s)", worked)},
		{"Findings — open", grade(open > 0, false), fmt.Sprintf("%d finding(s)", open)},
		{"Candidates", grade(len(res.Candidates) > 0 && unrefuted == 0, len(res.Candidates) > 0), fmt.Sprintf("%d capture candidate(s), %d without a refutation", len(res.Candidates), unrefuted)},
		{"Coverage", "grounded", "this table"},
	}
}

// section returns a level-two section's body from a markdown document, with the
// scaffold's italic placeholder treated as empty.
func section(doc, name string) string {
	lines := strings.Split(doc, "\n")
	var body []string
	in := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if in {
				break
			}
			in = strings.TrimSpace(strings.TrimPrefix(line, "## ")) == name
			continue
		}
		if in {
			body = append(body, line)
		}
	}
	s := strings.TrimSpace(strings.Join(body, "\n"))
	if strings.HasPrefix(s, "_") && strings.HasSuffix(s, "_") && !strings.Contains(s, "\n") {
		return ""
	}
	return s
}

// harvestDoc renders the harvest.
func harvestDoc(e Entry, intention string, findings []Finding, probes map[string]probeState, res Harvested) string {
	var b strings.Builder
	b.WriteString(harvestMarker + "\n")
	fmt.Fprintf(&b, "# HARVEST — %s\n\n", e.ID)
	fmt.Fprintf(&b, "%s · pin %s · minted %s · assembled %s\n\n", e.ID, short(e.Pin), e.Created, stamp())

	fmt.Fprintf(&b, "## 1. %s\n\n%s\n\n", harvestSections[0], e.Question)
	if h := section(intention, "Hypothesis"); h != "" {
		fmt.Fprintf(&b, "Hypothesis: %s\n\n", h)
	}

	fmt.Fprintf(&b, "## 2. %s\n\n", harvestSections[1])
	if len(res.Probes) == 0 {
		b.WriteString("No probe records.\n\n")
	}
	for _, n := range res.Probes {
		p := probes[n]
		state := fmt.Sprintf("exit %d", p.Exit)
		if !p.Complete {
			state = "incomplete: " + strings.Join(p.Missing, ", ")
		}
		fmt.Fprintf(&b, "- `state/probes/%s/` (%s)\n", n, state)
	}
	if len(res.Probes) > 0 {
		b.WriteString("\n")
	}
	if len(res.Halted) > 0 {
		fmt.Fprintf(&b, "The lab is halted by its %s.\n\n", strings.Join(res.Halted, " and "))
	}

	writeFindings := func(title string, keep func(Finding) bool) {
		fmt.Fprintf(&b, "## %s\n\n", title)
		n := 0
		// Gate findings lead: a refusal that halted the lab is the headline.
		for _, gateFirst := range []bool{true, false} {
			for _, f := range findings {
				if !keep(f) || (f.Kind == KindGate) != gateFirst {
					continue
				}
				n++
				fmt.Fprintf(&b, "- **%s %s** (%s)", f.ID, f.Title, f.Kind)
				var cites []string
				for _, p := range f.Probes {
					cites = append(cites, "`state/probes/"+p+"/`")
				}
				for _, ev := range f.Evidence {
					cites = append(cites, "`"+ev+"`")
				}
				fmt.Fprintf(&b, " — cites %s\n", strings.Join(cites, ", "))
				if f.Body != "" {
					for _, line := range strings.Split(f.Body, "\n") {
						fmt.Fprintf(&b, "  %s\n", line)
					}
				}
			}
		}
		if n == 0 {
			b.WriteString("None.\n")
		}
		b.WriteString("\n")
	}
	writeFindings("3. "+harvestSections[2], func(f Finding) bool { return f.Status == "worked" })
	writeFindings("4. "+harvestSections[3], func(f Finding) bool { return f.Status != "worked" })

	fmt.Fprintf(&b, "## 5. %s\n\n", harvestSections[4])
	b.WriteString("Capture candidates (product findings). Each is filed through capture, found\nduring this lab, with the pin in its text; nothing here is filed by the harvest.\n\n")
	if len(res.Candidates) == 0 {
		b.WriteString("None.\n")
	}
	for _, c := range res.Candidates {
		ref := "refutation owed: read the source or record that could kill it before filing"
		if c.Refutation != "" {
			ref = "refutation: " + c.Refutation
		}
		fmt.Fprintf(&b, "- %s %s — %s\n  `%s`\n", c.Finding, c.Title, ref, c.Command)
	}
	b.WriteString("\nAmendment candidates (procedure findings), for amendments.md:\n\n")
	if len(res.Amendments) == 0 {
		b.WriteString("None.\n")
	}
	for _, a := range res.Amendments {
		fmt.Fprintf(&b, "- %s\n", a)
	}

	fmt.Fprintf(&b, "\n## 6. %s\n\n| Section | Status | Basis |\n| --- | --- | --- |\n", harvestSections[5])
	for _, c := range res.Coverage {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", c.Section, c.Status, c.Note)
	}
	b.WriteString("\nCost: this harvest records what the lab's records hold and claims no token or\nmoney cost the lab did not measure.\n")
	return b.String()
}

// shellQuote quotes s for a POSIX shell, single-quoted.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
