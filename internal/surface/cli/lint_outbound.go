package cli

// The front door onto scanner.CheckOutbound: `abcd lint outbound`.
//
// WHY THIS DOOR EXISTS. The outbound policy has two halves — a live agent-session
// URL and a tool's own attribution footer — and until this verb landed only the
// FOOTER half was gated deterministically, by scripts/check-attribution.sh's
// GENERATED_RE over a pull request's commit range and body. The session-URL half
// was gated nowhere: not in that script, not in a hook, not in Go. It reached
// three commit messages and two pull-request bodies of a managed public repo
// before anyone noticed (iss-2609061438431625), and a merged commit message comes
// out only by rewriting a protected branch.
//
// WHY IT IS NOT A REGEX IN THE SHELL GATE, which is where the footer half lives
// and would have been the cheaper edit. The session-URL detector is not a
// pattern; it is a pattern plus an OPACITY CLASSIFIER
// (scanner.hasOpaqueSessionID), and that classifier is a conjunction — a UUID, or
// a token carrying BOTH a digit and an upper-case letter, or a long lower-case hex
// run — which POSIX ERE cannot express. Grep can only have the pattern without the
// classifier, and the pattern without the classifier flags every page written
// about session handling, including this repository's own research notes. So the
// choice was never "shell or Go", it was "the real policy in Go, or a weaker
// policy in shell". The shell gate calls this verb instead, and there is still one
// definition of the class.
//
// WHY IT REFUSES AND NEVER REWRITES. ScrubOutbound is the rewrite direction and it
// is right for a routine sanitising text it is about to post. This judges text a
// person already wrote and a forge may already hold, where an edit made on the
// author's behalf is not a remedy. CheckOutbound has no text return at all, so
// this door structurally cannot become a rewriter.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// maxOutboundBytes caps the artefact read from stdin. A pull-request body and a
// commit message are both far under this; the cap exists so a pipe that never
// ends cannot hold the gate open.
const maxOutboundBytes = 1 << 20 // 1 MiB

// outboundReport is the --json shape. `findings` uses scanner.Finding's own
// marshaller, which masks the matched span before it is serialised — the point
// being that a CI log on a public repository is public text too, so the gate must
// not republish the leak it is reporting.
type outboundReport struct {
	Label    string            `json:"label"`
	Findings []scanner.Finding `json:"findings"`
	Policy   string            `json:"policy"`
}

// newLintOutboundCommand wires `abcd lint outbound [FILE]`.
//
// It sits under `lint` rather than at the top level because adr-40's vocabulary
// puts a verb that applies rules about FORM in the lint bucket, and this applies
// the outbound policy's form rules to one artefact. The parent's own subject is
// this repository; the sub-verb's subject is a piece of text the caller hands it.
// That difference is why it takes its scanner configuration from --root
// explicitly rather than inheriting the parent's.
func newLintOutboundCommand(asJSON *bool) *cobra.Command {
	var label, rootDir string
	cmd := &cobra.Command{
		Use: "outbound [FILE]",
		Long: "Judge one outbound artefact — a commit message, a pull-request body, an issue, a\n" +
			"comment, a release note — against abcd's outbound policy: never a live\n" +
			"agent-session URL, never a tool's own attribution footer.\n\n" +
			"Reads FILE, or standard input when FILE is absent or `-`. It REPORTS and REFUSES;\n" +
			"it never rewrites the text it was given, because the text belongs to whoever\n" +
			"wrote it. Exit 0 clean, 1 the artefact is refused, 2 the check could not run.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, err := outboundArtefact(cmd, args)
			if err != nil {
				return &exitError{Code: 2, Msg: "lint outbound: " + scrubPaths(err)}
			}
			root, err := outboundScanRoot(rootDir)
			if err != nil {
				return &exitError{Code: 2, Msg: "lint outbound: " + scrubPaths(err)}
			}

			findings, checkErr := scanner.CheckOutbound(root, text, label)

			// A degraded or unreadable scanner configuration is a check that never
			// ran, and it exits 2 rather than 1: a caller keying on "1 means the
			// text is bad" must not read "the gate was broken" as a verdict on the
			// text. CheckOutbound reports no findings on that path, which is what
			// tells the two apart here.
			if checkErr != nil && len(findings) == 0 {
				return &exitError{Code: 2, Msg: "lint outbound: " + scrubPaths(checkErr)}
			}

			if *asJSON {
				report := outboundReport{Label: label, Findings: findings, Policy: scanner.OutboundPolicy}
				if report.Findings == nil {
					report.Findings = []scanner.Finding{}
				}
				out, err := json.Marshal(report)
				if err != nil {
					return &exitError{Code: 2, Msg: "lint outbound: " + scrubPaths(err)}
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
			} else {
				renderOutboundHuman(cmd.OutOrStdout(), label, findings)
			}

			if len(findings) > 0 {
				// The report is already rendered, so the code propagates with an
				// empty message and main prints nothing more.
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&label, "label", "outbound-artefact",
		"what the artefact is (commit-message, pr-body, issue, comment) — it names the artefact in the report")
	cmd.Flags().StringVar(&rootDir, "root", "",
		"repo root supplying the scanner configuration (default: current working directory)")
	return cmd
}

// outboundArtefact resolves the text to judge: the positional file, or stdin when
// there is none or it is `-`.
//
// An EMPTY artefact is a fault, not a pass. "There was nothing to check" and
// "what I checked was clean" must never look the same to a gate's caller — that
// equivalence is how a misrouted pipe becomes a green tick.
func outboundArtefact(cmd *cobra.Command, args []string) (string, error) {
	if len(args) == 1 && args[0] != "-" {
		info, err := os.Stat(args[0])
		if err != nil {
			return "", fmt.Errorf("cannot read the artefact: %w", err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("%s is a directory, not an artefact", args[0])
		}
		if info.Size() > maxOutboundBytes {
			return "", fmt.Errorf("the artefact is larger than %d bytes; it was not checked", maxOutboundBytes)
		}
		raw, err := os.ReadFile(args[0])
		if err != nil {
			return "", fmt.Errorf("cannot read the artefact: %w", err)
		}
		if strings.TrimSpace(string(raw)) == "" {
			return "", fmt.Errorf("%s is empty; nothing was checked", args[0])
		}
		return string(raw), nil
	}
	// Read one byte past the cap so an overflow is detectable. Truncating to the
	// cap and answering on the prefix is the quietest way to hand out a clearance
	// nobody earned: the tail is exactly where a harness appends.
	raw, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), maxOutboundBytes+1))
	if err != nil {
		return "", fmt.Errorf("reading the artefact from stdin failed: %w", err)
	}
	if len(raw) > maxOutboundBytes {
		return "", fmt.Errorf("the artefact is larger than %d bytes; it was not checked", maxOutboundBytes)
	}
	if strings.TrimSpace(string(raw)) == "" {
		return "", fmt.Errorf("no artefact to check: pass a FILE or pipe the text on stdin")
	}
	return string(raw), nil
}

// outboundScanRoot resolves the directory whose .abcd/config/pii.json configures
// the scan. It is validated before the scan for the reason `abcd lint` validates
// its own root: a missing directory is read by the config loader as "no override",
// which would silently downgrade a configured repo to the built-in set.
func outboundScanRoot(rootDir string) (string, error) {
	dir := rootDir
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = cwd
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		shown := rootDir
		if shown == "" {
			shown = abs
		}
		return "", fmt.Errorf("%s is not a directory", shown)
	}
	return abs, nil
}

// renderOutboundHuman writes the verdict.
//
// It deliberately does NOT echo the matched span. The finding's location and kind
// are what an author needs in order to find and delete the line; reprinting the
// session URL would publish a live handle on a run into a CI log, which on a
// public repository is itself public text — the gate would then leak the thing it
// exists to catch.
func renderOutboundHuman(w io.Writer, label string, findings []scanner.Finding) {
	safeLabel := termsafe.Sanitize(label)
	if len(findings) == 0 {
		fmt.Fprintf(w, "abcd lint outbound — ✓ %s carries no session URL and no tool attribution footer\n", safeLabel)
		return
	}
	fmt.Fprintf(w, "abcd lint outbound — ✗ %s breaks the outbound policy\n", safeLabel)
	for _, f := range findings {
		fmt.Fprintf(w, "  ✗ line %d, column %d — %s\n", f.Line, f.Column, termsafe.Sanitize(f.Kind))
		if f.Suggested != "" {
			fmt.Fprintf(w, "      fix: %s\n", termsafe.Sanitize(f.Suggested))
		}
	}
	fmt.Fprintln(w, "  policy: "+scanner.OutboundPolicy)
}
