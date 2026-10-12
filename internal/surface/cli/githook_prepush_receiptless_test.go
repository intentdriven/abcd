package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The pre-push hook's receipt-less prefixes are a declared table, one row per
// prefix: the prefix, the shape check a pushed range under it must pass, the
// record that rules it, and the act that ends a ref there (named when a delete
// is refused). A second prefix is coming (cloud sessions' work-in-progress
// branches, itd-2610120738075547), so the exemption is held here to a shape
// every row must have, rather than to the one prefix it was written for.

// receiptlessTableRe finds the table in the hook, and receiptlessRowRe each
// quoted row inside it.
var (
	receiptlessTableRe = regexp.MustCompile(`(?ms)^receiptless_prefixes=\(\n(.*?)^\)`)
	receiptlessRowRe   = regexp.MustCompile(`(?m)^\s*"([^"]*)"\s*$`)
)

// committedHook reads the committed pre-push hook from the checkout the test
// runs in, and its root.
func committedHook(t *testing.T) (root, hook string) {
	t.Helper()
	c := newPrePushCase(t)
	data, err := os.ReadFile(filepath.Join(c.dir, ".githooks", "pre-push"))
	if err != nil {
		t.Fatal(err)
	}
	top, err := c.run(".", "git", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("not in a git checkout: %s", top)
	}
	return strings.TrimSpace(top), string(data)
}

// receiptlessRows is the hook's table, each row split into its four fields.
func receiptlessRows(t *testing.T, hook string) [][]string {
	t.Helper()
	m := receiptlessTableRe.FindStringSubmatch(hook)
	if m == nil {
		t.Fatalf("the pre-push hook declares no receiptless_prefixes table")
	}
	var rows [][]string
	for _, r := range receiptlessRowRe.FindAllStringSubmatch(m[1], -1) {
		rows = append(rows, strings.Split(r[1], "|"))
	}
	return rows
}

// TestPrePushReceiptlessPrefixesAreDeclaredWithARulingAndAShapeCheck: every
// row names a prefix ending in a slash, a shape check the hook defines, a
// ruling record that exists, and the act that ends a ref there; the
// reservation prefix is the only row.
func TestPrePushReceiptlessPrefixesAreDeclaredWithARulingAndAShapeCheck(t *testing.T) {
	root, hook := committedHook(t)
	rows := receiptlessRows(t, hook)
	if len(rows) != 1 || len(rows[0]) != 4 || rows[0][0] != "abcd-reserved/" {
		t.Fatalf("the receipt-less table is %q; want the reservation prefix as its only row, in four fields", rows)
	}
	for _, row := range rows {
		prefix, shape, ruling, end := row[0], row[1], row[2], row[3]
		if !strings.HasSuffix(prefix, "/") || strings.ContainsAny(prefix, "*?[ ") {
			t.Errorf("prefix %q is not a plain branch prefix ending in a slash", prefix)
		}
		if shape == "" || !regexp.MustCompile(`(?m)^`+regexp.QuoteMeta(shape)+`\(\) \{`).MatchString(hook) {
			t.Errorf("prefix %q names the shape check %q, which the hook does not define", prefix, shape)
		}
		// The check runs through its own arm in shape_check, never under a
		// name read from the table.
		if !regexp.MustCompile(`(?m)^\s+`+regexp.QuoteMeta(shape)+`\) `).MatchString(hook) {
			t.Errorf("prefix %q names the shape check %q, which shape_check has no arm for", prefix, shape)
		}
		id := strings.TrimPrefix(ruling, "adr-")
		matches, _ := filepath.Glob(filepath.Join(root, ".abcd", "development", "decisions", "adrs", id+"-*.md"))
		if !strings.HasPrefix(ruling, "adr-") || len(matches) != 1 {
			t.Errorf("prefix %q names the ruling %q, which is no ADR in the decisions store", prefix, ruling)
		}
		if !strings.Contains(end, "<name>") {
			t.Errorf("prefix %q names %q as the act that ends a ref there, without the <name> it ends", prefix, end)
		}
	}
}

// TestPrePushAPrefixWithNoShapeCheckIsNeverReceiptless: a row whose shape
// check is empty, or names a function the hook does not define, exempts
// nothing: a reservation pushed under it asks for a receipt.
func TestPrePushAPrefixWithNoShapeCheckIsNeverReceiptless(t *testing.T) {
	_, hook := committedHook(t)
	rows := receiptlessRows(t, hook)
	if len(rows) == 0 || len(rows[0]) != 4 {
		t.Fatalf("the receipt-less table is %q", rows)
	}
	row := strings.Join(rows[0], "|")
	for name, shape := range map[string]string{"empty": "", "undefined": "no_such_shape_check"} {
		t.Run(name, func(t *testing.T) {
			c := newPrePushCase(t)
			broken := strings.Join([]string{rows[0][0], shape, rows[0][2], rows[0][3]}, "|")
			c.write(".githooks/pre-push", strings.Replace(hook, `"`+row+`"`, `"`+broken+`"`, 1), 0o755)
			first := c.emptyTreeCommit("", "abcd reservation: iss-2609020716570699")
			out, err := c.push("origin", first+":"+reservedRef)
			if err == nil || !strings.Contains(out, "no passing preflight") {
				t.Fatalf("a prefix with %s shape check let a push through without a receipt (err %v)\n%s", name, err, out)
			}
		})
	}
}

// TestPrePushAnUndeclaredPrefixStillNeedsAReceipt: only a declared prefix is
// receipt-less. An empty-tree commit pushed under a prefix the table does not
// name, or one that merely begins like the reservation prefix, needs a receipt
// as any push does.
func TestPrePushAnUndeclaredPrefixStillNeedsAReceipt(t *testing.T) {
	for _, ref := range []string{
		"refs/heads/claude/abcd-wip/iss-2609020716570699",
		"refs/heads/abcd-reservedx/iss-2609020716570699",
		"refs/tags/abcd-reserved/iss-2609020716570699",
	} {
		t.Run(ref, func(t *testing.T) {
			c := newPrePushCase(t)
			first := c.emptyTreeCommit("", "abcd reservation: iss-2609020716570699")
			out, err := c.push("origin", first+":"+ref)
			if err == nil || !strings.Contains(out, "no passing preflight") {
				t.Fatalf("an empty-tree commit under an undeclared prefix passed without a receipt (err %v)\n%s", err, out)
			}
		})
	}
}
