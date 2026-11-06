package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// render_sanitize_test.go — iss-264. record-lint's standalone binary prints each
// finding as `file:line: [SEV RuleID] message`. File carries a path and Message
// carries link targets, both lifted from the linted (possibly hostile-clone)
// tree; only the two error exits were path-scrubbed. A control byte in either
// field replays a terminal escape into CI logs, and an absolute File leaks the
// developer's local layout (iss-29). The findings line must mask control runes
// (termsafe.Sanitize) and strip the repo root / home, mirroring the main abcd
// CLI renderer (cli.go:496) and the error exits. Attack runes are built
// numerically so this file carries none of them.
func TestRenderFindingMasksAndScrubs(t *testing.T) {
	root := t.TempDir()
	// A finding whose File is ABSOLUTE (under root) and carries a control byte in
	// the path tail, and whose Message carries an ESC + RLO override.
	poisonedFile := filepath.Join(root, "docs", "note\x1b.md")
	f := lint.Finding{
		File:     poisonedFile,
		Line:     7,
		RuleID:   "stray_root_docs",
		Severity: "blocker",
		Message:  "link \x1btarget‮ points outside the tree",
	}

	got := renderFinding(f, root)

	for _, r := range []rune{0x1b, 0x202e} {
		if strings.ContainsRune(got, r) {
			t.Errorf("iss-264: findings line carries a raw U+%04X attack rune; it must be masked like cli.go:496\nline: %q", r, got)
		}
	}
	if strings.Contains(got, root) {
		t.Errorf("iss-264: findings line leaks the absolute repo root %q (iss-29: no absolute path in machine output)\nline: %q", root, got)
	}
	// The line structure and the legitimate content survive.
	if !strings.Contains(got, "docs/note") || !strings.Contains(got, ":7:") ||
		!strings.Contains(got, "stray_root_docs") || !strings.Contains(got, "link") {
		t.Fatalf("iss-264: scrubbing must not drop the finding's structure/content:\n%q", got)
	}
}

// TestSlugNamesMasksAConfigValueInItsErrorLine: the cap's refusal quotes what the
// repository's config holds, so a control byte there reaches the terminal unless
// the error line masks it as renderFinding does. The attack rune is written as a
// JSON escape, so this file carries none.
func TestSlugNamesMasksAConfigValueInItsErrorLine(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := `{"records": {"slug_cap": "\u001b[31mred", "x\u001b[31my": 1}}`
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	names := filepath.Join(t.TempDir(), "names")
	if err := os.WriteFile(names, []byte(".abcd/work/decisions/dcn-2611050000000001-a.md\x00"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stderr
	os.Stderr = w
	code := runSlugNames(names, root)
	os.Stderr = saved
	w.Close()
	line, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if code != 2 {
		t.Fatalf("a refused cap cannot judge: exit %d\n%q", code, line)
	}
	if !strings.Contains(string(line), "record-lint:") {
		t.Fatalf("the error line is missing: %q", line)
	}
	if strings.ContainsRune(string(line), 0x1b) {
		t.Errorf("the error line carries a raw escape: %q", line)
	}
}

// TestARefusedCapIsMaskedOnAWholeRun: the same refusal, reached through the rule on
// a whole run of the linter rather than through -slug-names.
func TestARefusedCapIsMaskedOnAWholeRun(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "record-lint")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("building the linter: %v\n%s", err, out)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"config.json":      `{"records": {"x\u001b[31my": 1}}`,
		"record-lint.json": `{"rules": {"record_slug_cap": {"enabled": true, "severity": "blocker"}}}`,
	} {
		if err := os.WriteFile(filepath.Join(root, ".abcd", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := exec.Command(bin, "-root", root)
	var stderr bytes.Buffer
	run.Stderr = &stderr
	if err := run.Run(); err == nil {
		t.Fatalf("a refused cap ran: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "record-lint:") || !strings.Contains(stderr.String(), "records.x") {
		t.Fatalf("the refusal is not the cap's: %q", stderr.String())
	}
	if strings.ContainsRune(stderr.String(), 0x1b) {
		t.Errorf("the error line carries a raw escape: %q", stderr.String())
	}
}
