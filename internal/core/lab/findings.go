package lab

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Finding is one entry of a lab's findings log.
type Finding struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Kind       string   `json:"kind"`
	Status     string   `json:"status"`
	Probes     []string `json:"probes"`
	Evidence   []string `json:"evidence"`
	Refutation string   `json:"refutation,omitempty"`
	Gate       string   `json:"gate,omitempty"`
	Body       string   `json:"body,omitempty"`
}

// The finding kinds the harvest sorts on.
const (
	KindProduct   = "product"
	KindProcedure = "procedure"
	KindResult    = "result"
	KindGate      = "gate"
)

var findingHeadRe = regexp.MustCompile(`^## (F-([0-9]+))\s+(.*)$`)
var fieldRe = regexp.MustCompile(`^(?:- )?(kind|status|probes|evidence|refutation|gate):\s*(.*)$`)

// parseFindings reads the findings log. Text before the first finding heading
// is the scaffold's explanation and is not a finding.
func parseFindings(doc string) []Finding {
	var out []Finding
	var cur *Finding
	var body []string
	flush := func() {
		if cur == nil {
			return
		}
		cur.Body = strings.TrimSpace(strings.Join(body, "\n"))
		if cur.Kind == "" {
			cur.Kind = KindResult
		}
		if cur.Status == "" {
			cur.Status = "open"
		}
		out = append(out, *cur)
		cur, body = nil, nil
	}
	for _, line := range strings.Split(doc, "\n") {
		line = strings.TrimRight(line, "\r")
		if m := findingHeadRe.FindStringSubmatch(line); m != nil {
			flush()
			cur = &Finding{ID: m[1], Title: strings.TrimSpace(m[3]), Probes: []string{}, Evidence: []string{}}
			continue
		}
		if cur == nil {
			continue
		}
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "# ") {
			flush()
			continue
		}
		if m := fieldRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil && len(body) == 0 {
			v := strings.TrimSpace(m[2])
			switch m[1] {
			case "kind":
				cur.Kind = strings.ToLower(v)
			case "status":
				cur.Status = strings.ToLower(v)
			case "probes":
				cur.Probes = splitList(v)
			case "evidence":
				cur.Evidence = splitList(v)
			case "refutation":
				cur.Refutation = v
			case "gate":
				cur.Gate = v
			}
			continue
		}
		if strings.TrimSpace(line) == "" && len(body) == 0 {
			continue
		}
		body = append(body, line)
	}
	flush()
	return out
}

// splitList reads a comma-separated field, dropping empties and backticks.
func splitList(v string) []string {
	out := []string{}
	for _, p := range strings.Split(v, ",") {
		p = strings.Trim(strings.TrimSpace(p), "`")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// nextFindingID is one past the highest finding number in the log.
func nextFindingID(fs []Finding) string {
	max := 0
	for _, f := range fs {
		if n, err := strconv.Atoi(strings.TrimPrefix(f.ID, "F-")); err == nil && n > max {
			max = n
		}
	}
	return "F-" + strconv.Itoa(max+1)
}

// halt is a standing refusal of one of the lab's own gates.
type halt struct {
	Gate    string   `json:"gate"`
	Finding string   `json:"finding"`
	Checks  []string `json:"checks"`
	At      string   `json:"at"`
}

func haltName(gate string) string { return "state/halt-" + gate + ".json" }

// haltedGates names the gates holding the lab halted, in order.
func (l *lab) haltedGates() []string {
	out := []string{}
	for _, g := range []string{"preflight", "sweep"} {
		if _, err := l.root.Lstat(haltName(g)); err == nil {
			out = append(out, g)
		}
	}
	return out
}

// haltAndRecord is the lab rule a gate refusal enforces: the refusal is written
// into the findings log as a gate finding and the lab is marked halted on that
// gate, so no probe is recorded until the gate passes again. A refusal that
// repeats a standing halt on the same checks reuses its finding rather than
// logging it twice.
func (l *lab) haltAndRecord(gate string, failed []string, title, detail, evidence string) (string, error) {
	sort.Strings(failed)
	if prev, ok := l.readHalt(gate); ok && strings.Join(prev.Checks, ",") == strings.Join(failed, ",") {
		return prev.Finding, nil
	}
	doc, err := l.readDoc(findingsName)
	if err != nil {
		return "", err
	}
	id := nextFindingID(parseFindings(doc))
	var b strings.Builder
	b.WriteString(strings.TrimRight(doc, "\n"))
	fmt.Fprintf(&b, "\n\n## %s %s\n\n", id, title)
	fmt.Fprintf(&b, "- kind: %s\n- status: open\n- gate: %s/%s\n- evidence: %s\n\n", KindGate, gate, strings.Join(failed, ","), evidence)
	fmt.Fprintf(&b, "%s\n\nHalted at %s. The lab stops here and records the refusal rather than\nadapting around it; the halt lifts only when the %s passes again.\n", detail, stamp(), gate)
	if err := l.writeDoc(findingsName, b.String()); err != nil {
		return "", err
	}
	h, err := json.Marshal(halt{Gate: gate, Finding: id, Checks: failed, At: stamp()})
	if err != nil {
		return "", err
	}
	if err := l.writeDoc(haltName(gate), string(h)+"\n"); err != nil {
		return "", err
	}
	return id, nil
}

// readHalt reads a standing halt, if any.
func (l *lab) readHalt(gate string) (halt, bool) {
	data, err := fsutil.ReadGuardedInRoot(l.root, haltName(gate), 64<<10)
	if err != nil {
		return halt{}, false
	}
	var h halt
	if json.Unmarshal(data, &h) != nil {
		return halt{Gate: gate}, true
	}
	return h, true
}

// lift removes a gate's halt once the gate passes. The finding stays in the log.
func (l *lab) lift(gate string) (string, error) {
	h, ok := l.readHalt(gate)
	if !ok {
		return "", nil
	}
	if err := l.root.Remove(haltName(gate)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("cannot lift the %s halt: %v", gate, redact(err, l.store.home))
	}
	return h.Finding, nil
}
