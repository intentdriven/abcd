package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// staleusage.go — the named refusal for an unknown command or flag on a binary
// that can prove it is stale (iss-2608230943088357).
//
// The plugin surface (commands/*.md) and the plugin binary ship from one
// release, but they drift: a plugin update lands a newer surface before the
// bootstrap re-provisions the binary, a cached root goes stale, a PATH copy
// outlives the root it was copied from. A page then tells the reader to run a
// verb the binary predates, and cobra answers `unknown command "update"` or
// `unknown flag: --yes` — the shape of a malformed invocation, not of a stale
// install, and nothing points at the plugin update or the next rung of the
// resolution ladder.
//
// staleUsageNote adds one sentence to that error, derived only from what the
// binary can prove on disk (adr-38: no network on any path the user did not
// ask to reach the network). Three sources, in order of strength:
//
//  1. The command surface beside the plugin root documents the very verb (or
//     flag) the binary refused. That is direct evidence the surface is newer
//     than the binary, and the remedy follows where the binary sits: a source
//     checkout is rebuilt, a plugin-root binary is replaced by a plugin update,
//     a PATH copy takes `abcd update`.
//  2. Otherwise, the disk-only vintage comparison the `version` verb already
//     renders: behind the checkout tip (dogfood) or differing from the release
//     the plugin cache pinned.
//  3. Otherwise nothing — cobra's line stands alone, byte-for-byte.
//
// The exit code, the stdout/stderr split, and the JSON envelope are untouched;
// only the text gains a line, the way hookPlaneSkewNote already does on the
// hook plane.

// osExecutable is os.Executable, overridable so a test can place the running
// binary inside or outside a plugin root without re-execing itself.
var osExecutable = os.Executable

// unknownCommandRe matches cobra's unknown-command line: the token and the
// command path, both rendered with %q.
var unknownCommandRe = regexp.MustCompile(`^unknown command ("(?:[^"\\]|\\.)*") for ("(?:[^"\\]|\\.)*")`)

// unknownFlagRe matches pflag's unknown-flag line. The shorthand form
// (`unknown shorthand flag: 'x' in -xyz`) is left alone: a one-letter flag is
// not evidence a page could be searched for.
var unknownFlagRe = regexp.MustCompile(`^unknown flag: (--\S+)`)

// verbShape and flagShape are the only token shapes that ever reach the
// filesystem or a regexp: a verb token is joined into a path under commands/,
// so anything else — a path fragment, a capital, a dot — is not looked up.
var (
	verbShape = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	flagShape = regexp.MustCompile(`^--[a-z][a-z0-9-]*$`)
)

// maxCommandPageBytes caps a command-page read; the pages are a few KiB.
const maxCommandPageBytes = 256 << 10

// usageSkew is what the user typed that the binary could not parse: the
// command path the binary does know, then either the verb it does not or the
// flag it does not.
type usageSkew struct {
	known []string // the command path below the root the binary resolved
	verb  string   // the unknown verb under known; "" for a flag-only skew
	flag  string   // the unknown flag, consulted only when verb is ""
}

// staleUsageNote is the sentence appended to a usage error, or "" when the
// error is not an unknown command/flag or the disk proves nothing.
func staleUsageNote(root *cobra.Command, args []string, msg string) string {
	skew, ok := classifyUsageError(root, args, msg)
	if !ok {
		return ""
	}
	pluginRoot, rootOK := ahoy.ResolvePluginRoot()
	inRoot := rootOK && executableUnder(pluginRoot)
	if rootOK {
		if what, documented := surfaceDocuments(pluginRoot, skew); documented {
			switch {
			case inRoot && isSourceCheckout(pluginRoot):
				return fmt.Sprintf("this binary predates %s its command surface documents — the plugin root is a source checkout, so rebuild it with `make build`", what)
			case inRoot:
				return fmt.Sprintf("this binary predates %s its plugin surface documents — %s", what, update.RemedyPluginUpdate)
			default:
				return fmt.Sprintf("this binary predates %s the plugin surface it was provisioned from documents — this PATH copy is stale; run `abcd update`", what)
			}
		}
	}
	// No page names what was typed (a typo, or a surface no older than the
	// binary): fall back to the vintage the `version` verb renders, git-only and
	// disk-only. A fresh or undeterminable vintage says nothing.
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	vin := ahoy.Vintage(cwd)
	if vin.Report.Outcome != vintage.Stale {
		return ""
	}
	if vin.Source == ahoy.VintageSourceCheckoutTip {
		// Ancestry-guarded, so the direction may be claimed (vintage.go).
		return fmt.Sprintf("this binary was built from commit %s but this checkout is at %s — it is behind its own source, so rebuild it with `make build`",
			shortSHA(vin.Report.Current), shortSHA(vin.Report.Expected))
	}
	// A version/pin comparison is string equality, so it is non-directional:
	// name both values, claim neither is ahead. The tag comes from a file the
	// bootstrap wrote, unchecked upstream — sanitise it like skew.go does.
	remedy := "this PATH copy may be stale; run `abcd update`"
	if inRoot {
		remedy = update.RemedyPluginUpdate
	}
	return fmt.Sprintf("this binary is %s and the plugin cache pinned release %s — the two differ, so %s",
		termsafe.Sanitize(vin.Report.Current), termsafe.Sanitize(vin.Report.Expected), remedy)
}

// classifyUsageError reads the skew out of a cobra usage error. An unknown
// command names both the token and the path in its own line, so that is
// parsed. An unknown flag names only the flag: cobra parses flags BEFORE it
// validates positionals, so `abcd update --yes` on a binary without `update`
// fails on `--yes` and the verb the reader actually wanted never appears —
// the positionals are walked against the tree to recover it.
func classifyUsageError(root *cobra.Command, args []string, msg string) (usageSkew, bool) {
	if m := unknownCommandRe.FindStringSubmatch(msg); m != nil {
		tok, err := strconv.Unquote(m[1])
		if err != nil {
			return usageSkew{}, false
		}
		path, err := strconv.Unquote(m[2])
		if err != nil {
			return usageSkew{}, false
		}
		fields := strings.Fields(path)
		if len(fields) == 0 || fields[0] != root.Name() {
			return usageSkew{}, false
		}
		return usageSkew{known: fields[1:], verb: tok}, true
	}
	if m := unknownFlagRe.FindStringSubmatch(msg); m != nil {
		known, verb := walkKnownPath(root, args)
		return usageSkew{known: known, verb: verb, flag: m[1]}, true
	}
	return usageSkew{}, false
}

// walkKnownPath descends the tree along the positional tokens of args and
// returns the path the binary knows plus the first token it does not. A token
// under a leaf is an argument, never an unknown verb; a flag value that looks
// positional can only ever make the lookup fail safe (no page is named for it).
func walkKnownPath(root *cobra.Command, args []string) (known []string, unknown string) {
	cur := root
	for _, tok := range args {
		if strings.HasPrefix(tok, "-") {
			continue
		}
		if !cur.HasSubCommands() {
			return known, ""
		}
		next := subcommandNamed(cur, tok)
		if next == nil {
			return known, tok
		}
		known = append(known, tok)
		cur = next
	}
	return known, ""
}

// subcommandNamed is findByPath's one-level form that also honours aliases,
// which cobra resolves and findByPath (built for construction-time lookups)
// deliberately does not.
func subcommandNamed(cur *cobra.Command, tok string) *cobra.Command {
	for _, sub := range cur.Commands() {
		if sub.Name() == tok || sub.HasAlias(tok) {
			return sub
		}
	}
	return nil
}

// surfaceDocuments reports whether the command surface at pluginRoot documents
// what the binary refused, and how to name it. A top-level verb is documented
// by its own page, commands/<verb>.md. A sub-verb is documented when its
// top-level page hands it over as an invocation in any of the ladder's
// spellings (`"${CLAUDE_PLUGIN_ROOT}/abcd" capture resolve`, `abcd capture
// resolve`, `/abcd:capture resolve`); a flag when the page names it literally.
// Prose that merely contains the word is not evidence: "status" appears in
// most pages without `<verb> status` being a command.
func surfaceDocuments(pluginRoot string, s usageSkew) (what string, ok bool) {
	for _, k := range s.known {
		if !verbShape.MatchString(k) {
			return "", false
		}
	}
	if s.verb != "" {
		if !verbShape.MatchString(s.verb) {
			return "", false
		}
		full := strings.Join(append(append([]string{}, s.known...), s.verb), " ")
		what = "the `" + full + "` command"
		if len(s.known) == 0 {
			return what, isRegularFile(commandPagePath(pluginRoot, s.verb))
		}
		body := readCommandPage(pluginRoot, s.known[0])
		if body == "" {
			return "", false
		}
		re := regexp.MustCompile(`abcd["':]?\s*` + regexp.QuoteMeta(full) + `(?:[^A-Za-z0-9-]|$)`)
		return what, re.MatchString(body)
	}
	if !flagShape.MatchString(s.flag) {
		return "", false
	}
	page := "abcd"
	if len(s.known) > 0 {
		page = s.known[0]
	}
	body := readCommandPage(pluginRoot, page)
	if body == "" {
		return "", false
	}
	re := regexp.MustCompile(`(?:^|[^A-Za-z0-9-])` + regexp.QuoteMeta(s.flag) + `(?:[^A-Za-z0-9-]|$)`)
	what = "the `" + s.flag + "` flag of `" + strings.Join(append([]string{"abcd"}, s.known...), " ") + "`"
	return what, re.MatchString(body)
}

func commandPagePath(pluginRoot, verb string) string {
	return filepath.Join(pluginRoot, "commands", verb+".md")
}

// readCommandPage returns the page's text, or "" when it is absent, unreadable,
// or larger than a command page can be.
func readCommandPage(pluginRoot, verb string) string {
	data, err := fsutil.ReadGuarded(commandPagePath(pluginRoot, verb), maxCommandPageBytes)
	if err != nil {
		return ""
	}
	return string(data)
}

func isRegularFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

// isSourceCheckout reports whether a plugin root is abcd's own source checkout:
// the curated payload carries no cmd/, so its presence is the tell the
// resolution ladder itself relies on (internal/core/launch/commandladder_test.go).
func isSourceCheckout(pluginRoot string) bool {
	return isRegularFile(filepath.Join(pluginRoot, "cmd", "abcd", "main.go"))
}

// executableUnder reports whether the running binary sits inside root — beside
// its hooks/ (a provisioned plugin root) or below it (the dogfood layout's
// repo-root link into bin/). Both sides are canonicalised, the executable by
// its directory so a path the seam names but the disk lacks still compares.
func executableUnder(root string) bool {
	exe, err := osExecutable()
	if err != nil {
		return false
	}
	dir, base := filepath.Split(exe)
	exe = filepath.Join(canonicalPath(dir), base)
	return strings.HasPrefix(exe, canonicalPath(root)+string(filepath.Separator))
}

func canonicalPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}
