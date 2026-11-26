package guard

import (
	"path"
	"strings"
)

// recordActs are the git subcommands that make a commit or send one: the shell
// commands that count as a sub-agent's write (spc-2611251757228271 step 2).
var recordActs = map[string]bool{
	"commit": true, "merge": true, "rebase": true, "cherry-pick": true,
	"revert": true, "am": true, "pull": true, "push": true,
}

// functionAct is the act RecordActs reads for a function in a form the guard
// does not read (functionFormSignal): its body is not seen here, so it is
// read as an act.
const functionAct = "function"

// RecordActs reads command for the git commands that make a commit or send
// one, in the order written. It refuses nothing and decides nothing. A line the
// tokenizer cannot split is read as no act, as is a command run through a
// wrapper, a shell string or a script, which is not seen here. A function in a
// form the guard does not read is functionAct.
func RecordActs(command string) []string {
	segs, err := tokenize(command)
	if err != nil {
		return nil
	}
	var out []string
	for _, s := range segs {
		if _, read := commandWordsRead(s.tokens); s.functionQuoted || !read {
			out = append(out, functionAct)
			continue
		}
		if act := recordActIn(s.tokens); act != "" {
			out = append(out, act)
		}
	}
	return out
}

// recordActIn is the held git subcommand of one command, or "". The subcommand
// is taken by position, after the words commandWords steps over and git's global
// options with their values. A word a substitution or a variable helps to write is allowed
// as the value of such an option and anywhere after the subcommand, where it
// changes nothing read here (the message of `git commit -m "$(cat …)"`); as the
// program, an option or the subcommand it fixes no act, and none is read.
func recordActIn(toks []string) string {
	toks = commandWords(toks)
	if len(toks) == 0 || isUnknown(toks[0]) || path.Base(toks[0]) != "git" {
		return ""
	}
	i := 1
	for i < len(toks) && strings.HasPrefix(toks[i], "-") {
		if isUnknown(toks[i]) {
			return ""
		}
		if gitValueOptions[toks[i]] {
			i++ // its value, whatever prints it
		}
		i++
	}
	if i >= len(toks) || isUnknown(toks[i]) || !recordActs[toks[i]] {
		return ""
	}
	return toks[i]
}
