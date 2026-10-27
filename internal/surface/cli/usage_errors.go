package cli

// usage_errors.go — the one formatter for the usage errors the command
// framework raises. cobra and pflag word an unknown sub-command, an unreadable
// flag value, an unknown shorthand cluster and a malformed flag by repeating
// the caller's text. abcd's own refusals name what was refused by its position
// or its flag and list what is accepted, without repeating it; these do the
// same. Only the text changes: exit codes, the --json
// object's shape and help output are untouched.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// applyUsageFlagErrors wraps the flag-error function of every command in the
// tree so the function below it — whichever refusal an earlier pass chose —
// receives describeFlagError's text in place of the parser's. It runs after
// every pass that sets a flag-error function and before markFlagParseErrors,
// which must stay outermost. cobra's own help verb, built inside Execute,
// inherits the root's, so it is covered too.
func applyUsageFlagErrors(c *cobra.Command) {
	inner := c.FlagErrorFunc()
	c.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return inner(cmd, describeFlagError(err))
	})
	for _, sub := range c.Commands() {
		applyUsageFlagErrors(sub)
	}
}

// describeFlagError rewords a flag-parse error so it names the flag and never
// the text that came with it:
//
//   - an unreadable value names the flag and the type it takes; the value and
//     the converter's message (strconv repeats the value) are dropped;
//   - an unknown shorthand names the one letter, not the cluster it came in;
//   - a shorthand that needs a value names the letter and its flag, not the
//     cluster;
//   - a malformed flag (`---x`, `--=x`) says what is malformed, not the text.
//
// An unknown long flag keeps the parser's line: it is a flag name, and the
// retired-flag, help-verb and stale-binary answers read it. Anything else
// passes through unchanged.
func describeFlagError(err error) error {
	var invalid *pflag.InvalidValueError
	if errors.As(err, &invalid) {
		f := invalid.GetFlag()
		return fmt.Errorf("invalid argument for %q flag: it takes a value of type %s, and the one given does not parse as one (the value is not repeated here)",
			flagDisplayName(f), f.Value.Type())
	}
	var missing *pflag.NotExistError
	if errors.As(err, &missing) && missing.GetSpecifiedShortnames() != "" {
		return fmt.Errorf("unknown shorthand flag: %q (the letters around it are not repeated here)", firstRune(missing.GetSpecifiedName()))
	}
	var needs *pflag.ValueRequiredError
	if errors.As(err, &needs) && needs.GetSpecifiedShortnames() != "" {
		msg := fmt.Sprintf("flag needs an argument: %q", firstRune(needs.GetSpecifiedName()))
		if f := needs.GetFlag(); f != nil {
			msg += fmt.Sprintf(" (--%s, of type %s)", f.Name, f.Value.Type())
		}
		return errors.New(msg)
	}
	var syntax *pflag.InvalidSyntaxError
	if errors.As(err, &syntax) {
		return errors.New("bad flag syntax: an argument begins with `---` or `--=`, which names no flag (its text is not repeated here); a value that starts with a dash goes after `--`")
	}
	return err
}

// flagDisplayName spells a flag the way pflag's own refusal does.
func flagDisplayName(f *pflag.Flag) string {
	if f.Shorthand != "" && f.ShorthandDeprecated == "" {
		return "-" + f.Shorthand + ", --" + f.Name
	}
	return "--" + f.Name
}

func firstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

// unknownCommandLineRe matches cobra's unknown-command wording wherever it
// stands in a refusal: alone, at the head of the requirement aggregate, before
// the hook plane's second line, or ahead of the update verb's own explanation.
// Both operands are Go-quoted, so the pattern bounds them exactly.
var unknownCommandLineRe = regexp.MustCompile(`unknown command ("(?:[^"\\]|\\.)*") for ("(?:[^"\\]|\\.)*")`)

// withholdOperands rewrites cobra's unknown-command wording in msg so it names
// the command and lists its sub-commands, and says which argument was refused
// by its position instead of repeating it. Run calls it after staleUsageNote
// has read the operand out of the original wording. cobra always refuses the
// first positional (NoArgs and its legacy check read args[0], and so does the
// update verb's own validator), so the position is stated as the first.
func withholdOperands(root *cobra.Command, msg string) string {
	return unknownCommandLineRe.ReplaceAllStringFunc(msg, func(line string) string {
		m := unknownCommandLineRe.FindStringSubmatch(line)
		path, err := strconv.Unquote(m[2])
		if err != nil {
			return "unknown command (the first argument, not repeated here)"
		}
		return fmt.Sprintf("unknown command for %q (the first argument, not repeated here; %s)", path, subCommandList(root, path))
	})
}

// subCommandList names the sub-commands of the command at path, or says it
// takes none.
func subCommandList(root *cobra.Command, path string) string {
	fields := strings.Fields(path)
	cmd := root
	if len(fields) > 0 && fields[0] == root.Name() {
		cmd = findByPath(root, fields[1:])
	}
	var names []string
	if cmd != nil {
		for _, sub := range cmd.Commands() {
			if sub.IsAvailableCommand() {
				names = append(names, sub.Name())
			}
		}
	}
	if len(names) == 0 {
		return "it takes no sub-command"
	}
	return "its sub-commands are " + strings.Join(names, ", ")
}

// pathErrorCause is err without the path an *fs.PathError carries — the bare
// cause ("no such file or directory", "is a directory") — for a refusal that
// names the path by its role rather than by what the caller typed.
func pathErrorCause(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// minWithheldLen is the shortest argument the boundary pass withholds. A
// shorter one is a count, a letter or a short word whose every occurrence in a
// refusal would otherwise be rewritten.
const minWithheldLen = 8

// minWithheldPrefix is the shortest leading part of a longer argument the
// pass withholds where a refusal cut the argument short before repeating it.
// The sites that cut keep forty bytes or more.
const minWithheldPrefix = 24

// maxPrefixScan bounds how much of an argument the prefix search walks back
// over, so a long argument costs a bounded number of searches.
const maxPrefixScan = 512

// plainWordRe is a lower-case word or hyphenated words: the shape of a verb,
// a closed-set value or a word of the refusal's own text. A part derived from
// an argument (a prefix, a base name, the value after an `=`) is not withheld
// on its own when it has this shape, since it would rewrite the refusal's own
// words; the whole argument still is.
var plainWordRe = regexp.MustCompile(`^[a-z]+(?:-[a-z]+)*$`)

// typedArgument is one argument the caller typed whose text the pass
// withholds, and its 1-based position among the arguments.
type typedArgument struct {
	text string
	pos  int
}

// withholdArguments is the one redaction pass at the error boundary: Run calls
// it on the finished refusal, before the line goes to stderr or into the --json
// error object, so a refusal some verb words by repeating what it was passed
// still does not carry it. Each argument the caller typed (a positional, a
// flag's value, whether after an `=`, attached to a shorthand or standing as
// the next argument, and anything after `--`) is replaced by its position
// wherever it stands in msg: as typed; as a refusal renders it (quoted with %q
// or %+q, or masked for the terminal); as a path a refusal shortened to its
// working-directory or home form, to a relative path or to its base name; as
// the value after its last `=`; and as its leading part, where a refusal cut it
// short. Three kinds are left as they are: an argument shorter than
// minWithheldLen, the name of a command in the tree, and a well-formed record
// id (recordid.CitedIDRe), which a refusal names so the caller can find the
// record. The per-site wording names what was refused; this pass is the
// backstop for a site that still repeats it.
func withholdArguments(root *cobra.Command, args []string, msg string) string {
	ops := typedArguments(root, args)
	// Longest first, so an argument that holds a shorter one is withheld whole.
	slices.SortStableFunc(ops, func(a, b typedArgument) int { return len(b.text) - len(a.text) })
	for _, op := range ops {
		stand := fmt.Sprintf("(argument %d, not repeated here)", op.pos)
		for _, form := range renderedForms(op.text) {
			msg = replaceWhole(msg, form, stand)
		}
		msg = withholdPrefix(msg, op.text, stand)
	}
	return msg
}

// typedArguments collects the arguments withholdArguments withholds. A flag's
// name is never collected; whether the argument after a flag is its value is
// read from the flag itself, so a value that begins with a dash is still
// collected. An unknown flag's argument is left to the parser, which refuses
// it naming the flag alone.
func typedArguments(root *cobra.Command, args []string) []typedArgument {
	cmd := root
	if found, _, err := root.Find(args); err == nil && found != nil {
		cmd = found
	}
	lookup := func(name string) *pflag.Flag {
		if f := cmd.Flags().Lookup(name); f != nil {
			return f
		}
		return cmd.InheritedFlags().Lookup(name)
	}
	shorthand := func(c byte) *pflag.Flag {
		if f := cmd.Flags().ShorthandLookup(string(c)); f != nil {
			return f
		}
		return cmd.InheritedFlags().ShorthandLookup(string(c))
	}
	takesValue := func(f *pflag.Flag) bool { return f != nil && f.NoOptDefVal == "" }
	commands := commandNames(root)
	var ops []typedArgument
	add := func(v string, i int) {
		if len(v) < minWithheldLen || commands[v] || recordid.CitedIDRe.MatchString(v) {
			return
		}
		ops = append(ops, typedArgument{text: v, pos: i + 1})
	}
	afterTerminator := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case afterTerminator:
			add(a, i)
		case a == "--":
			afterTerminator = true
		case strings.HasPrefix(a, "--"):
			name, val, hasValue := strings.Cut(a[2:], "=")
			if hasValue {
				add(val, i)
			} else if takesValue(lookup(name)) && i+1 < len(args) {
				i++
				add(args[i], i)
			}
		case len(a) > 1 && a[0] == '-':
			// A shorthand cluster: letters until the first that takes a
			// value, whose value is the rest of the argument or the next one.
			for k := 1; k < len(a); k++ {
				f := shorthand(a[k])
				if f == nil {
					break
				}
				if takesValue(f) {
					if rest := strings.TrimPrefix(a[k+1:], "="); rest != "" {
						add(rest, i)
					} else if i+1 < len(args) {
						i++
						add(args[i], i)
					}
					break
				}
			}
		default:
			add(a, i)
		}
	}
	return ops
}

// renderedForms is v and each form a refusal may carry it in, longest first:
// quoted by %q or %+q, masked for the terminal, shortened as a path, and the
// parts a refusal names on their own (the base name, the value after the last
// `=`). A derived part shorter than minWithheldLen, or shaped like a plain
// word, is left out.
func renderedForms(v string) []string {
	forms := []string{v}
	seen := map[string]bool{v: true}
	keep := func(f string, derived bool) {
		if seen[f] || len(f) < minWithheldLen || (derived && plainWordRe.MatchString(f)) {
			return
		}
		seen[f] = true
		forms = append(forms, f)
	}
	if q := strconv.Quote(v); len(q) >= 2 {
		keep(q[1:len(q)-1], false)
	}
	if q := strconv.QuoteToASCII(v); len(q) >= 2 {
		keep(q[1:len(q)-1], false)
	}
	keep(termsafe.Sanitize(v), false)
	if abs, err := filepath.Abs(v); err == nil {
		keep(fsutil.RedactHome(abs), false)
		if cwd, err := os.Getwd(); err == nil {
			keep(fsutil.RedactRoot(abs, cwd, "."), false)
			if rel, err := filepath.Rel(cwd, abs); err == nil {
				keep(rel, true)
			}
		}
	}
	if k := strings.LastIndexAny(v, `/\`); k >= 0 {
		keep(v[k+1:], true)
	}
	if k := strings.LastIndex(v, "="); k >= 0 {
		keep(v[k+1:], true)
	}
	slices.SortStableFunc(forms, func(a, b string) int { return len(b) - len(a) })
	return forms
}

// withholdPrefix replaces the longest leading part of v, at least
// minWithheldPrefix bytes and shorter than v, that still stands in msg: what
// a refusal that cut v short before repeating it carries.
func withholdPrefix(msg, v, stand string) string {
	longest := min(len(v)-1, maxPrefixScan)
	for n := longest; n >= minWithheldPrefix; n-- {
		p := v[:n]
		if !utf8.ValidString(p) || plainWordRe.MatchString(p) {
			continue
		}
		if strings.Contains(msg, p) {
			return strings.ReplaceAll(msg, p, stand)
		}
	}
	return msg
}

// replaceWhole replaces each occurrence of old in s. A short old made of word
// characters alone (under twelve bytes of letters, digits and underscores) is
// replaced only where it is not part of a longer word, so it cannot rewrite
// the inside of one; any other old is replaced wherever it stands.
func replaceWhole(s, old, repl string) string {
	if len(old) >= 12 || strings.IndexFunc(old, func(r rune) bool { return !isWordRune(r) }) >= 0 {
		return strings.ReplaceAll(s, old, repl)
	}
	var b strings.Builder
	for {
		i := strings.Index(s, old)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		end := i + len(old)
		before, _ := utf8.DecodeLastRuneInString(s[:i])
		after, _ := utf8.DecodeRuneInString(s[end:])
		if (i > 0 && isWordRune(before)) || (end < len(s) && isWordRune(after)) {
			b.WriteString(s[:i+1])
			s = s[i+1:]
			continue
		}
		b.WriteString(s[:i])
		b.WriteString(repl)
		s = s[end:]
	}
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// commandNames is the set of every command name and alias in the tree, which
// the boundary pass never withholds: a refusal names its own verb.
func commandNames(root *cobra.Command) map[string]bool {
	names := map[string]bool{}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		names[c.Name()] = true
		for _, a := range c.Aliases {
			names[a] = true
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return names
}
