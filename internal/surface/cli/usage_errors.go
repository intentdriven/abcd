package cli

// usage_errors.go — the one formatter for the usage errors the command
// framework raises. cobra and pflag word an unknown sub-command, an unreadable
// flag value, an unknown shorthand cluster and a malformed flag by repeating
// the caller's text. A value a script passed through a variable, which its own
// log shows unexpanded, would then be printed expanded on stderr and in the
// --json error object, where CI logs and session transcripts keep it. abcd's
// own refusals name what was refused by its position or its flag and list what
// is accepted; these do the same. Only the text changes: exit codes, the --json
// object's shape and help output are untouched.

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
