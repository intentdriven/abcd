package mdrecord

import (
	"math/rand/v2"
	"strings"
	"testing"
)

// costUnits are bodies of repeated units whose fences and comments sit one to
// three columns in, each one ReadPage asks which list item it is in.
var costUnits = []string{
	"  ```\n",
	"  <!--\n  -->\n",
	"x\n  ```\n",
	"   - a\n  - b\n - c\n  ```\n",
}

// TestReadPageStepsInLineWithTheBody: the walks ReadPage makes beyond Read's
// own pass visit each line a bounded number of times, so a body four times
// longer takes at most about four times the steps.
func TestReadPageStepsInLineWithTheBody(t *testing.T) {
	steps := 0
	stepHook = func() { steps++ }
	defer func() { stepHook = nil }()
	for _, unit := range costUnits {
		for _, n := range []int{500, 2000} {
			body := lines(strings.Repeat(unit, n))
			steps = 0
			ReadPage(body)
			if limit := 24 * len(body); steps > limit {
				t.Errorf("ReadPage over %d lines of %q took %d steps, want at most %d", len(body), unit, steps, limit)
			}
		}
	}
}

// itemContentColByWalk is the list-item answer read by walking up from the
// line, the reading listItems takes in one pass: the reference it is held to.
func itemContentColByWalk(lines []string, mask []uint8, at, indent int) (int, bool) {
	col0 := false
	for j := at - 1; j >= 0; j-- {
		ln := strings.TrimRight(lines[j], "\r")
		if strings.TrimSpace(ln) == "" {
			if col0 {
				return 0, false
			}
			continue
		}
		if mask[j] != 0 {
			col0 = false
			continue
		}
		w := indentWidth(ln)
		if w < indent {
			if c, ok := listContentCol(ln); ok {
				return c, true
			}
		}
		col0 = w == 0
	}
	return 0, false
}

// TestListItemsAnswersAsTheWalkUp: over bodies drawn from list markers at
// several indents, column-0 text, blank lines and fences, listItems gives the
// walk's answer for every line and indent, under each reading's mask.
func TestListItemsAnswersAsTheWalkUp(t *testing.T) {
	alphabet := []string{"", "x", "- a", "  - b", "1. c", "   * d", "    - e", "\t+ f", "-", "  text", "```", "  ```", "    ```", "<!--", "-->", "10.  g"}
	rng := rand.New(rand.NewPCG(1, 2))
	for n := 0; n < 400; n++ {
		body := make([]string, 1+rng.IntN(30))
		for i := range body {
			body[i] = alphabet[rng.IntN(len(alphabet))]
		}
		for _, rule := range []Rule{TopLevel, ListNested, listItem} {
			mask := Read(body, rule).Mask
			for indent := 1; indent <= 6; indent++ {
				items := &listItems{lines: body, mask: mask}
				for at := range body {
					wc, wok := itemContentColByWalk(body, mask, at, indent)
					gc, gok := items.contentCol(at, indent)
					if wc != gc || wok != gok {
						t.Fatalf("body %q, rule %d, line %d, indent %d: listItems = (%d, %v), the walk = (%d, %v)", body, rule, at, indent, gc, gok, wc, wok)
					}
				}
			}
		}
	}
}
