package guard

import "strings"

// A setting handed to git on its own command line can do what a flag does, and
// the entries read flags: `git -c remote.origin.mirror=true push` mirrors as
// `git push --mirror` does. So, as expandGitAliases does for an alias, the
// command is also read with the setting spelled as the flag or the refspec it
// stands for, and the entries that already exist match it.

// gitSetting is one configuration key a git command line sets, folded, with its
// value.
type gitSetting struct {
	key, value string
}

// rewriteGitSettings returns args with what settings stand for spelled out
// directly after operand 0, in every reading of which word that is, or nothing
// when no setting bears on the subcommand.
func rewriteGitSettings(args []string, globs []bool, settings []gitSetting, valueFlags []string) []aliasRewrite {
	if len(settings) == 0 {
		return nil
	}
	var out []aliasRewrite
	for _, i := range firstOperands(args, valueFlags) {
		glob := i < len(globs) && globs[i]
		var words []string
		for _, set := range settings {
			switch {
			case operandIs(args[i], "push", glob):
				words = append(words, pushSettingWords(set)...)
			case operandIs(args[i], "gc", glob):
				words = append(words, gcSettingWords(set)...)
			}
		}
		if len(words) == 0 {
			continue
		}
		next := make([]string, 0, len(args)+len(words))
		next = append(next, args[:i+1]...)
		next = append(next, words...)
		next = append(next, args[i+1:]...)
		var nextGlob []bool
		if globs != nil {
			nextGlob = append(nextGlob, globAtRange(globs, 0, i+1)...)
			nextGlob = append(nextGlob, make([]bool, len(words))...)
			nextGlob = append(nextGlob, globAtRange(globs, i+1, len(args))...)
		}
		out = append(out, aliasRewrite{args: next, globs: nextGlob})
	}
	return out
}

// pushSettingWords is what one setting adds to a push: the refspecs of
// remote.<name>.push, whose `+` forces, and the --mirror of remote.<name>.mirror.
func pushSettingWords(set gitSetting) []string {
	if !strings.HasPrefix(set.key, "remote.") {
		return nil
	}
	switch {
	case strings.HasSuffix(set.key, ".push") && set.value != "":
		return []string{set.value}
	case strings.HasSuffix(set.key, ".mirror") && !gitBoolFalse(set.value):
		return []string{"--mirror"}
	}
	return nil
}

// gcSettingWords is what one setting adds to a gc: how long it keeps what
// nothing reaches, and the reflog entries a commit is found again from, read as
// the --prune that says the same. `never` keeps them all.
func gcSettingWords(set gitSetting) []string {
	if !strings.HasPrefix(set.key, "gc.") {
		return nil
	}
	switch {
	case set.key == "gc.pruneexpire",
		strings.HasSuffix(set.key, ".reflogexpire"),
		strings.HasSuffix(set.key, ".reflogexpireunreachable"):
		if v := strings.ToLower(set.value); v == "never" || v == "false" {
			return nil
		}
		return []string{"--prune=" + set.value}
	}
	return nil
}

// gitBoolFalse reports whether a config value is one git reads as false. A
// value the line does not fix is not.
func gitBoolFalse(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "false", "no", "off", "0":
		return true
	}
	return false
}
