package spec

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

// StepsHeading is the section a spec lists its steps under
// (itd-2609212103565953; adr-2609212115255771, decision 4). A step is the unit
// below a spec: one lane and one pull request. The loop's `implement step`
// uses the same word for the same thing.
const StepsHeading = "## Steps"

// Step is one entry of a spec's `## Steps` section: an ordered, independently
// landable piece of the spec, with its own footprint.
//
// The section is a numbered list. Each item's first line is the step's title;
// the lines indented beneath it are its body, where three keyed bullets are
// read:
//
//  1. The parser
//     - packages: internal/core/spec
//     - tests: the parser over a stepped and an unstepped spec
//     - landed: #123
//
// `landed:` names what landed the step (a pull request or a commit); a step
// without one, or with a null value, is not landed. `needs:` names the earlier
// steps a step waits for, or `none` (ruling DR6, spc-2609202134341288); a step
// without the line needs every step before it (ruling DR6b), so running beside
// earlier steps is an opt-in the step declares. Any other indented line is the
// author's and is carried verbatim wherever the step is copied, except the
// `needs:` line, which a remainder rewrites (CarryUnlanded).
type Step struct {
	// Number is the step's 1-based position in document order. The list's own
	// numerals are not trusted for order: a renderer renumbers them anyway.
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Packages string `json:"packages,omitempty"`
	Tests    string `json:"tests,omitempty"`
	Landed   string `json:"landed,omitempty"`
	// Needs are the earlier steps the step's `- needs:` line names, and
	// NeedsDeclared whether it has the line at all: a step without it needs
	// every step before it (Requires).
	Needs         []int `json:"needs,omitempty"`
	NeedsDeclared bool  `json:"needs_declared,omitempty"`
	// Implicit marks the one step of a spec that lists none: the whole spec,
	// built as one step (the intent's decision 2).
	Implicit bool `json:"implicit,omitempty"`

	// rawTitle is the item's text as the author wrote it (emphasis included),
	// and body the lines beneath it, both kept so a copy is the author's words.
	rawTitle string
	body     []string
}

// ErrUnclosedSpan is ParseSteps' refusal of a document holding a fenced block
// or HTML comment that nothing closes. It is its own error so a caller names
// the right remedy: close the opener, not rewrite the section.
var ErrUnclosedSpan = errors.New("unclosed span")

// ImplicitStepTitle is the title of the one step an unstepped spec is.
const ImplicitStepTitle = "the whole spec"

var (
	// stepItemRe is a top-level numbered list item: `1. Title` or `1) Title`.
	stepItemRe = regexp.MustCompile(`^([0-9]{1,4})[.)][ \t]+(\S.*)$`)
	// stepKeyRe is a keyed bullet indented under a step.
	stepKeyRe = regexp.MustCompile(`^[ \t]+[-*+][ \t]+(?i:(packages|tests|landed|needs))[ \t]*:[ \t]*(.*)$`)
	// guidanceRe is a whole-line italic paragraph, the shape of the minted
	// placeholder; before the first step it is guidance and is skipped.
	guidanceRe = regexp.MustCompile(`^_.*_$`)
)

// ParseSteps returns the steps a spec's `## Steps` section lists, in order.
// An absent section, an empty one, or one holding only italic guidance lists
// none and returns nil: the caller that wants the build's view asks Steps,
// which turns that into one implicit step.
//
// It fails closed. Anything in the section that is not a numbered step, a line
// indented under one, a blank line or leading guidance is an error naming the
// line — never a guess — because the remainder copy writes from this parse
// (unrecognized-input-never-writes). A second `## Steps` heading is an error
// too: which one is the section is undecidable. Which lines are live is
// mdrecord's answer: a heading inside a fenced block or an HTML comment
// elsewhere is an example, or parked, and is not the section; a fenced block or
// comment inside the section is refused, since a step inside one is not a step.
//
// A document holding a fence or comment that nothing closes is refused whole
// (ErrUnclosedSpan), naming the opener's line. The span runs to end of file,
// so a `## Steps` heading below it is masked and the section would read as
// absent — zero steps, no error — which a remainder copy would turn into a
// silent drop of every unlanded step. The opener is the fault, not the section.
func ParseSteps(content string) ([]Step, error) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if i, flag, ok := mdrecord.Unclosed(lines); ok {
		construct := "a fenced code block"
		if flag&mdrecord.MaskComment != 0 {
			construct = "an HTML comment (`<!--` with no `-->`)"
		}
		return nil, fmt.Errorf("spec: %w — the document leaves %s open at line %d, %q; it runs to end of file, so %s may be masked and cannot be read. Close it or remove it; the steps are not the fault",
			ErrUnclosedSpan, construct, i+1, strings.TrimSpace(lines[i]), StepsHeading)
	}
	mask := mdrecord.Mask(lines)
	start, end, err := stepsSection(lines, mask)
	if err != nil || start < 0 {
		return nil, err
	}
	var steps []Step
	for i := start; i < end; i++ {
		line := lines[i]
		if mask[i]&mdrecord.MaskFence != 0 {
			return nil, fmt.Errorf("spec: %s holds a fenced block (line %d) — a step inside one is an example, not a step", StepsHeading, i+1)
		}
		if mask[i]&mdrecord.MaskComment != 0 {
			return nil, fmt.Errorf("spec: %s holds an HTML comment (line %d) — a step inside one is parked, not a step", StepsHeading, i+1)
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if m := stepItemRe.FindStringSubmatch(line); m != nil {
			raw := strings.TrimSpace(m[2])
			title := strings.TrimSpace(strings.Trim(raw, "*_"))
			if title == "" {
				return nil, fmt.Errorf("spec: %s step %d (line %d) has no title", StepsHeading, len(steps)+1, i+1)
			}
			steps = append(steps, Step{Number: len(steps) + 1, Title: title, rawTitle: raw})
			continue
		}
		indented := line[0] == ' ' || line[0] == '\t'
		if indented && len(steps) > 0 {
			st := &steps[len(steps)-1]
			if m := stepKeyRe.FindStringSubmatch(line); m != nil {
				if err := st.setKey(strings.ToLower(m[1]), strings.TrimSpace(m[2]), i+1); err != nil {
					return nil, err
				}
			}
			st.body = append(st.body, line)
			continue
		}
		if len(steps) == 0 && guidanceRe.MatchString(trimmed) {
			continue
		}
		return nil, fmt.Errorf("spec: %s line %d is not a numbered step or a line indented under one: %q — list the steps as `1. <title>`, each with `- packages:`, `- tests:` and, once it lands, `- landed:` indented beneath it", StepsHeading, i+1, trimmed)
	}
	return steps, nil
}

// setKey records one keyed bullet on a step, refusing a key given twice.
func (s *Step) setKey(key, value string, line int) error {
	if key == "needs" {
		return s.setNeeds(value, line)
	}
	var dst *string
	switch key {
	case "packages":
		dst = &s.Packages
	case "tests":
		dst = &s.Tests
	case "landed":
		dst = &s.Landed
		if frontmatter.IsNull(value) {
			value = ""
		}
	}
	if *dst != "" {
		return fmt.Errorf("spec: %s step %d gives `%s:` twice (line %d)", StepsHeading, s.Number, key, line)
	}
	*dst = value
	return nil
}

// setNeeds records a step's `- needs:` line: `none`, or a comma-separated list
// of earlier steps by number. A need naming the step itself, a later step, or a
// step the spec does not list is refused naming the line.
func (s *Step) setNeeds(value string, line int) error {
	if s.NeedsDeclared {
		return fmt.Errorf("spec: %s step %d gives `needs:` twice (line %d)", StepsHeading, s.Number, line)
	}
	bad := func(why string) error {
		return fmt.Errorf("spec: %s step %d's `needs:` line (line %d) %s — write `- needs: none`, or the earlier steps it waits for by number (`- needs: 1, 3`)", StepsHeading, s.Number, line, why)
	}
	s.NeedsDeclared = true
	s.Needs = []int{}
	if strings.EqualFold(strings.TrimSpace(value), "none") {
		return nil
	}
	for _, f := range strings.Split(value, ",") {
		f = strings.TrimSpace(f)
		n, err := strconv.Atoi(f)
		switch {
		case err != nil || f == "" || f[0] == '+' || f[0] == '-':
			return bad(fmt.Sprintf("names %q, which is not a step number", f))
		case n == s.Number:
			return bad("names the step itself")
		case n > s.Number:
			return bad(fmt.Sprintf("names step %d, which comes after it", n))
		case n < 1:
			return bad(fmt.Sprintf("names step %d, which the spec does not list", n))
		case slices.Contains(s.Needs, n):
			return bad(fmt.Sprintf("names step %d twice", n))
		}
		s.Needs = append(s.Needs, n)
	}
	return nil
}

// Requires is the earlier steps a step waits for: the ones its `- needs:` line
// names, or, without the line, every step before it (ruling DR6b).
func (s Step) Requires() []int {
	if s.NeedsDeclared {
		return append([]int{}, s.Needs...)
	}
	out := []int{}
	for n := 1; n < s.Number; n++ {
		out = append(out, n)
	}
	return out
}

// NeedsRewrite is one `- needs:` line a remainder copy rewrote: the step as the
// remainder numbers it, and the line before and after.
type NeedsRewrite struct {
	Step   int    `json:"step"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// needsLineRe is a step's `- needs:` line, its indentation and marker kept.
var needsLineRe = regexp.MustCompile(`^([ \t]+[-*+][ \t]+)(?i:needs)[ \t]*:`)

// CarryUnlanded is the remainder copy's view of a spec's steps: the steps not
// marked landed, in document order, renumbered from one, every indented line
// carried verbatim but the `- needs:` line, which is rewritten against the
// remainder. A named step that landed is satisfied and leaves the list; one
// that did not is carried and renamed to its number in the remainder; a list
// left empty is written `- needs: none`, never removed, since an absent line
// means every earlier step (ruling DR6b). A step without the line stays
// without it. The rewrites are returned, before and after, in remainder order.
func CarryUnlanded(steps []Step) ([]Step, []NeedsRewrite) {
	carried := Unlanded(steps)
	renum := map[int]int{}
	for i, s := range carried {
		renum[s.Number] = i + 1
	}
	var rewrites []NeedsRewrite
	out := make([]Step, 0, len(carried))
	for i, s := range carried {
		s.Number = i + 1
		if s.NeedsDeclared {
			kept := []int{}
			for _, n := range s.Needs {
				if m, ok := renum[n]; ok {
					kept = append(kept, m)
				}
			}
			s.Needs = kept
			after := needsValue(kept)
			body := make([]string, len(s.body))
			for j, l := range s.body {
				m := needsLineRe.FindStringSubmatch(l)
				if m == nil {
					body[j] = l
					continue
				}
				nl := m[1] + "needs: " + after
				body[j] = nl
				if strings.TrimSpace(nl) != strings.TrimSpace(l) {
					rewrites = append(rewrites, NeedsRewrite{Step: s.Number, Before: strings.TrimSpace(l), After: strings.TrimSpace(nl)})
				}
			}
			s.body = body
		}
		out = append(out, s)
	}
	return out, rewrites
}

// Steps is the build's view of a spec: the steps it lists, or — when it lists
// none — one implicit step, the whole spec. It refuses what ParseSteps refuses.
func Steps(content string) ([]Step, error) {
	steps, err := ParseSteps(content)
	if err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return []Step{{Number: 1, Title: ImplicitStepTitle, Implicit: true}}, nil
	}
	return steps, nil
}

// ReadSteps reads a stored spec's listed steps (ParseSteps over its file),
// through the store's trust-boundary reader.
func ReadSteps(repoRoot string, sp Spec) ([]Step, error) {
	data, err := readRepoFile(filepath.Join(repoRoot, sp.Path), sp.Path)
	if err != nil {
		return nil, err
	}
	steps, err := ParseSteps(string(data))
	if err != nil {
		return nil, fmt.Errorf("%w (in %s)", err, sp.Path)
	}
	return steps, nil
}

// Unlanded returns the steps not marked landed, in order. An implicit step is
// never returned: it names the spec, not a piece of it.
func Unlanded(steps []Step) []Step {
	var out []Step
	for _, s := range steps {
		if s.Landed == "" && !s.Implicit {
			out = append(out, s)
		}
	}
	return out
}

// renderSteps renders steps as the body of a `## Steps` section, renumbered
// from one, each carrying its title and the lines beneath it as the author
// wrote them. ParseSteps reads the result back as the same steps.
func renderSteps(steps []Step) string {
	var b strings.Builder
	n := 0
	for _, s := range steps {
		if s.Implicit {
			continue
		}
		n++
		title := s.rawTitle
		if title == "" {
			title = s.Title
		}
		fmt.Fprintf(&b, "%d. %s\n", n, title)
		if s.body != nil {
			for _, l := range s.body {
				b.WriteString(l + "\n")
			}
			continue
		}
		for _, kv := range [][2]string{{"packages", s.Packages}, {"tests", s.Tests}, {"landed", s.Landed}} {
			if kv[1] != "" {
				fmt.Fprintf(&b, "   - %s: %s\n", kv[0], kv[1])
			}
		}
		if s.NeedsDeclared {
			fmt.Fprintf(&b, "   - needs: %s\n", needsValue(s.Needs))
		}
	}
	return b.String()
}

// needsValue is a needs list as its line writes it.
func needsValue(needs []int) string {
	if len(needs) == 0 {
		return "none"
	}
	parts := make([]string, len(needs))
	for i, n := range needs {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ", ")
}

// stepsSection finds the `## Steps` section: the body lines [start, end), or
// start < 0 when the spec has none. The section runs to the next live `#` or
// `##` heading; a deeper heading stays inside it, where ParseSteps refuses it.
// A masked line (fenced or commented, per mdrecord.Mask) neither opens the
// section, nor ends it, nor counts as a second heading.
func stepsSection(lines []string, mask []uint8) (start, end int, err error) {
	start, end = -1, len(lines)
	for i, line := range lines {
		if mask[i] != 0 {
			continue
		}
		if strings.TrimRight(line, " \t") == StepsHeading {
			if start >= 0 {
				return -1, 0, fmt.Errorf("spec: more than one %s heading (line %d) — which one is the section is undecidable", StepsHeading, i+1)
			}
			start = i + 1
			continue
		}
		if start >= 0 && end == len(lines) && (strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ")) {
			end = i
		}
	}
	if start < 0 {
		return -1, 0, nil
	}
	return start, end, nil
}
