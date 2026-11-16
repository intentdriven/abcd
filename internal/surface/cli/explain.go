package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/cobra"
)

// explain.go — `abcd explain` (itd-2610301346085246, spc-2610301431075964
// step 2): the running binary's own short answer to "how do I call this
// command". An agent learned a verb from its plugin page, which ships with the
// plugin version the session loaded rather than with the binary it runs; on
// 2026-10-30 one composed an ideate verdict from a page at schema version 1
// and the binary refused it. This verb prints one command's facts from the one
// source of them (CommandFacts, surface.go), and one input's shape from the
// input registry (inputs.go), so what it says is what the binary it runs in
// accepts.
//
// It reads only: it builds a fresh command tree, never executed, and writes
// nothing.

// explainMaxLines is the most lines one command's text answer runs to. A
// command whose flags would run past it shows the first of them and one line
// saying how many more there are and that --json lists them all.
const explainMaxLines = 40

// newExplainCommand builds the `explain` verb.
func newExplainCommand(asJSON *bool) *cobra.Command {
	var input string
	cmd := &cobra.Command{
		Use: "explain [<verb> [<sub-verb>...]]",
		Long: "Print how to call one command, from the binary that runs it: its sentence, its usage\n" +
			"line, each flag with its type, its allowed values, its default and whether it is\n" +
			"required, the flags it inherits, each filled-in input it reads with that input's\n" +
			"schema version, its worked example and its sub-commands, in at most 40 lines.\n" +
			"A hidden command is answered too, marked (hidden) on its first line.\n\n" +
			"--input <flag> prints one input's schema version, its fields and an example that\n" +
			"passes the command's own check. --json prints the command's facts as JSON, or with\n" +
			"--input the input's shape.\n\n" +
			"Run it before filling in an input for a command: where it and a page disagree, the\n" +
			"binary is right, since a page may be older. Bare, it prints this help. It reads\n" +
			"only and writes nothing. Exit 2 on a refusal: a command path the tree does not have,\n" +
			"naming the nearest real path or the sub-commands where the path went wrong, or an\n" +
			"input the command does not read, naming the inputs it does.",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			const prefix = "abcd explain: "
			flag := strings.TrimLeft(input, "-")
			if len(args) == 0 {
				if flag != "" {
					return &exitError{Code: 2, Msg: prefix + "--input names an input of one command, so name the command first, " +
						"as in `abcd explain ideate record --input verdict-json`"}
				}
				return cmd.Help()
			}
			facts := CommandFacts()
			at, err := explainResolve(facts, args)
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + err.Error()}
			}
			hidden := explainHidden(facts, at.Path)
			if flag != "" {
				shape, err := explainInput(at, flag)
				if err != nil {
					return &exitError{Code: 2, Msg: prefix + err.Error()}
				}
				return render(cmd.OutOrStdout(), *asJSON, shape, func(w io.Writer) { renderExplainInput(w, shape, hidden) })
			}
			// The --json answer for one command is the per-command files'
			// document (spc-2610301431075964 approach 4) cut to the one
			// command, and written by the files' own encoder, with no HTML
			// escaping, so it is byte for byte that command's slice of its
			// file and a usage line's <file> reads as written.
			if *asJSON {
				data, err := EncodeCommandFacts(CommandFactsDocument{FormatVersion: surface.FactsFormatVersion, Commands: []surface.Facts{at}})
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(data)
				return err
			}
			for _, line := range explainLines(at, hidden, explainInherited(facts, at.Path)) {
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&input, "input", "",
		"print the schema version, the fields and an example of the input the command reads through this flag")
	return cmd
}

// explainFactsByPath indexes the facts by command path.
func explainFactsByPath(facts []surface.Facts) map[string]surface.Facts {
	by := make(map[string]surface.Facts, len(facts))
	for _, f := range facts {
		by[f.Path] = f
	}
	return by
}

// explainChildren names every sub-command of the command at path, hidden ones
// included, from the facts' paths.
func explainChildren(facts []surface.Facts, path string) []string {
	var out []string
	for _, f := range facts {
		if rest, ok := strings.CutPrefix(f.Path, path+" "); ok && !strings.Contains(rest, " ") {
			out = append(out, rest)
		}
	}
	return out
}

// explainResolve finds the command the words name, below the root. A first
// word that is the root's own name is read as the root, so `abcd explain abcd`
// answers for the board and `abcd explain abcd ideate` for ideate. A word that
// names no sub-command is refused with the nearest real path, or the
// sub-commands of the path the words reached; the refusal never repeats the
// word, which the error boundary would withhold in any case.
func explainResolve(facts []surface.Facts, words []string) (surface.Facts, error) {
	if len(facts) == 0 {
		return surface.Facts{}, fmt.Errorf("the command tree holds no command")
	}
	by := explainFactsByPath(facts)
	rootPath := facts[0].Path
	for _, f := range facts {
		if !strings.Contains(f.Path, " ") {
			rootPath = f.Path
			break
		}
	}
	if len(words) > 0 && words[0] == rootPath {
		words = words[1:]
	}
	path := rootPath
	for _, word := range words {
		next := path + " " + word
		if _, ok := by[next]; ok {
			path = next
			continue
		}
		children := explainChildren(facts, path)
		if len(children) == 0 {
			return surface.Facts{}, fmt.Errorf("`%s` takes no sub-command, so a word after it names nothing to explain; "+
				"run `abcd explain %s`", path, explainArgs(path))
		}
		best, bestDist := "", 3 // accept edit distances 1 and 2
		for _, name := range children {
			if d := levenshtein(word, name); d < bestDist {
				best, bestDist = name, d
			}
		}
		if best != "" {
			return surface.Facts{}, fmt.Errorf("`%s` has no sub-command by that name; the nearest real path is `%s %s`",
				path, path, best)
		}
		return surface.Facts{}, fmt.Errorf("`%s` has no sub-command by that name; %s", path, explainSubcommandList(by[path]))
	}
	return by[path], nil
}

// explainSubcommandList names a command's visible sub-commands.
func explainSubcommandList(f surface.Facts) string {
	if len(f.Subcommands) == 0 {
		return "it lists no visible sub-command"
	}
	return "its sub-commands are " + strings.Join(f.Subcommands, ", ")
}

// explainArgs is a command path as `abcd explain` takes it: the words below
// the root, or the root's own name for the root.
func explainArgs(path string) string {
	if _, rest, ok := strings.Cut(path, " "); ok {
		return rest
	}
	return path
}

// explainHidden reports whether the command at path, or any command above it,
// is hidden: a sub-command of the hidden hook plane is as hidden as its parent.
func explainHidden(facts []surface.Facts, path string) bool {
	by := explainFactsByPath(facts)
	words := strings.Fields(path)
	for i := 1; i <= len(words); i++ {
		if by[strings.Join(words[:i], " ")].Hidden {
			return true
		}
	}
	return false
}

// explainInheritedFlag is a persistent flag a command takes from a command
// above it.
type explainInheritedFlag struct {
	flag surface.FlagFact
	from string
}

// explainInherited is every visible persistent flag a command above path
// declares, nearest the root first, so the answer names the flags every
// sub-command takes (the root's --json among them) without the facts holding a
// second copy of them.
func explainInherited(facts []surface.Facts, path string) []explainInheritedFlag {
	by := explainFactsByPath(facts)
	words := strings.Fields(path)
	var out []explainInheritedFlag
	for i := 1; i < len(words); i++ {
		above := strings.Join(words[:i], " ")
		for _, fl := range by[above].Flags {
			if fl.Persistent && !fl.Hidden {
				out = append(out, explainInheritedFlag{flag: fl, from: above})
			}
		}
	}
	return out
}

// explainLines is one command's text answer, in the spec's order: the
// sentence, the usage line, the flags, the inputs, the example and the
// sub-commands, at most explainMaxLines lines.
func explainLines(f surface.Facts, hidden bool, inherited []explainInheritedFlag) []string {
	title := f.Path
	if hidden {
		title += " (hidden)"
	}
	if f.Sentence != "" {
		title += " — " + f.Sentence
	}
	head := []string{title, "Usage: " + f.Usage}

	var flags []string
	for _, fl := range f.Flags {
		if !fl.Hidden {
			flags = append(flags, "  "+explainFlagLine(fl))
		}
	}
	for _, in := range inherited {
		flags = append(flags, "  "+explainFlagLine(in.flag)+" (inherited from "+in.from+")")
	}

	var tail []string
	if len(f.Inputs) > 0 {
		tail = append(tail, "Inputs:")
		for _, in := range f.Inputs {
			tail = append(tail, "  "+explainInputLine(f.Path, in))
		}
	}
	if f.Example != "" {
		tail = append(tail, "Example:")
		for _, line := range strings.Split(f.Example, "\n") {
			tail = append(tail, "  "+strings.TrimSpace(line))
		}
	}
	if len(f.Subcommands) > 0 {
		call := "abcd explain " + explainArgs(f.Path) + " <sub-command>"
		if !strings.Contains(f.Path, " ") {
			call = "abcd explain <verb>"
		}
		tail = append(tail, fmt.Sprintf("Sub-commands: %s; `%s` explains one", strings.Join(f.Subcommands, ", "), call))
	}

	out := head
	if len(flags) > 0 {
		room := explainMaxLines - len(head) - len(tail) - 1 // the Flags: line
		if len(flags) > room {
			keep := max(room-1, 0) // one line says how many more
			more := len(flags) - keep
			flags = append(flags[:keep:keep], fmt.Sprintf("  and %d more; `abcd explain %s --json` lists every flag",
				more, explainArgs(f.Path)))
		}
		out = append(out, "Flags:")
		out = append(out, flags...)
	}
	return append(out, tail...)
}

// explainFlagLine is one flag: its spelling, its type, its closed set, its
// default and whether it is required. A boolean takes no value, so its type is
// said in parentheses rather than as a value placeholder.
func explainFlagLine(fl surface.FlagFact) string {
	var b strings.Builder
	if fl.Shorthand != "" {
		b.WriteString("-" + fl.Shorthand + ", ")
	}
	b.WriteString("--" + fl.Name)
	if fl.Type == surface.FlagBool {
		b.WriteString(" (bool)")
	} else {
		b.WriteString(" <" + fl.Type + ">")
	}
	if len(fl.Allowed) > 0 {
		b.WriteString(" [one of " + strings.Join(fl.Allowed, " | ") + "]")
	}
	if fl.Default != "" {
		b.WriteString(" (default " + fl.Default + ")")
	}
	if fl.Required {
		b.WriteString(" (required)")
	}
	return b.String()
}

// explainVersionText says which schema version an input is read at and where
// the document carries it.
func explainVersionText(version int, versionIn string) string {
	switch versionIn {
	case surface.VersionType:
		return fmt.Sprintf("schema version %d, in its _type tag", version)
	case surface.VersionNone:
		return fmt.Sprintf("schema_version %d (the input carries none)", version)
	default:
		return fmt.Sprintf("schema_version %d", version)
	}
}

// explainInputLine is one input a command reads. A filled-in JSON input names
// its schema version and the call that prints its shape; an interview's
// answers file is named alone, since the interview that asks prints its shape.
func explainInputLine(path string, in surface.InputFact) string {
	if in.Kind != surface.InputJSON {
		return "--" + in.Flag + ": " + in.Name
	}
	return fmt.Sprintf("--%s: %s, %s; abcd explain %s --input %s",
		in.Flag, in.Name, explainVersionText(in.SchemaVersion, in.VersionIn), explainArgs(path), in.Flag)
}

// explainInput is the registered shape of the input f reads through flag. An
// answers file, or a flag the command reads no input through, is refused,
// naming the inputs it does read.
func explainInput(f surface.Facts, flag string) (surface.InputShape, error) {
	var names []string
	for _, in := range f.Inputs {
		if in.Flag != flag {
			if in.Kind == surface.InputJSON {
				names = append(names, "--"+in.Flag)
			}
			continue
		}
		if in.Kind != surface.InputJSON {
			return surface.InputShape{}, fmt.Errorf("--%s is an interview's answers file, whose shape the interview "+
				"that asks the questions prints; it has no registered shape", in.Flag)
		}
		if shape, ok := inputFor(f.Path, flag); ok {
			return shape, nil
		}
	}
	if len(names) == 0 {
		return surface.InputShape{}, fmt.Errorf("`%s` reads no filled-in input, so --input names nothing for it", f.Path)
	}
	return surface.InputShape{}, fmt.Errorf("`%s` reads no input through that flag; the inputs it reads are %s",
		f.Path, strings.Join(names, ", "))
}

// renderExplainInput writes one input's shape: what it is, its schema version,
// its root, its fields and its example.
func renderExplainInput(w io.Writer, s surface.InputShape, hidden bool) {
	title := s.Path + " --" + s.Flag
	if hidden {
		title += " (hidden)"
	}
	fmt.Fprintf(w, "%s — %s, %s\n", title, s.Name, explainVersionText(s.SchemaVersion, s.VersionIn))
	if s.Root == surface.RootArray {
		fmt.Fprintln(w, "The document is a JSON array; each element has these fields:")
	} else {
		fmt.Fprintln(w, "The document is a JSON object with these fields:")
	}
	renderExplainFields(w, s.Fields, 1)
	fmt.Fprintln(w, "Example:")
	for _, line := range strings.Split(strings.TrimRight(s.Example, "\n"), "\n") {
		fmt.Fprintln(w, "  "+line)
	}
}

// renderExplainFields writes a field list, a nested object's fields indented
// under it.
func renderExplainFields(w io.Writer, fields []surface.Field, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, f := range fields {
		need := "optional"
		if f.Required {
			need = "required"
		}
		line := fmt.Sprintf("%s%s (%s, %s)", indent, f.Name, f.Type, need)
		if f.Doc != "" {
			line += ": " + f.Doc
		}
		fmt.Fprintln(w, line)
		renderExplainFields(w, f.Fields, depth+1)
	}
}
