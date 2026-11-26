package guard

import (
	"path/filepath"
	"slices"
	"strings"
)

// How a writer's words are read where its options decide which word is a
// destination: as GNU getopt_long reads them (options anywhere before `--`, a
// short option that takes a value ending its cluster, a long option named in
// full or by a prefix that names no other), with a second reading where
// another platform's command reads a letter otherwise.

// Long option kinds.
const (
	optFlag     = iota // takes no value
	optValue           // takes one, after `=` or as the next word
	optOptional        // takes one only after `=`
)

// optSpec is one command's options: the short letters that take a value, the
// letters whose value word another platform's command reads as an operand
// (macOS's cp and install read -S as a plain flag), the letters macOS's
// command reads with a value (mac), and its long options by name, without the
// dashes.
type optSpec struct {
	values, maybeOperand, mac string
	long                      map[string]int
	// optional names the letters that take a value only glued on, ending
	// their cluster (git am's -S<key>), and negate reads `--no-<name>` as
	// name's negation, as git's parse-options does.
	optional string
	negate   bool
}

// GNU coreutils 9 long options of the writers, and GNU patch's.
var (
	cpOpts = optSpec{values: "St", maybeOperand: "S", long: map[string]int{
		"archive": optFlag, "attributes-only": optFlag, "backup": optOptional, "copy-contents": optFlag,
		"debug": optFlag, "dereference": optFlag, "force": optFlag, "interactive": optFlag, "link": optFlag,
		"no-clobber": optFlag, "no-dereference": optFlag, "no-preserve": optValue, "no-target-directory": optFlag,
		"one-file-system": optFlag, "parents": optFlag, "preserve": optOptional, "recursive": optFlag,
		"reflink": optOptional, "remove-destination": optFlag, "sparse": optValue, "strip-trailing-slashes": optFlag,
		"suffix": optValue, "symbolic-link": optFlag, "target-directory": optValue, "update": optOptional,
		"verbose": optFlag, "keep-directory-symlink": optFlag, "context": optOptional, "help": optFlag, "version": optFlag,
	}}
	mvOpts = optSpec{values: "St", long: map[string]int{
		"backup": optOptional, "context": optFlag, "debug": optFlag, "exchange": optFlag, "force": optFlag,
		"interactive": optFlag, "no-clobber": optFlag, "no-copy": optFlag, "no-target-directory": optFlag,
		"strip-trailing-slashes": optFlag, "suffix": optValue, "target-directory": optValue, "update": optOptional,
		"verbose": optFlag, "help": optFlag, "version": optFlag,
	}}
	installOpts = optSpec{values: "Stmog", maybeOperand: "S", mac: "BDfghlMmNoT", long: map[string]int{
		"backup": optOptional, "compare": optFlag, "debug": optFlag, "directory": optFlag, "group": optValue,
		"mode": optValue, "owner": optValue, "preserve-timestamps": optFlag, "strip": optFlag,
		"strip-program": optValue, "suffix": optValue, "target-directory": optValue, "no-target-directory": optFlag,
		"verbose": optFlag, "preserve-context": optFlag, "context": optOptional, "help": optFlag, "version": optFlag,
	}}
	lnOpts = optSpec{values: "St", long: map[string]int{
		"backup": optOptional, "directory": optFlag, "force": optFlag, "interactive": optFlag, "logical": optFlag,
		"no-dereference": optFlag, "no-target-directory": optFlag, "physical": optFlag, "relative": optFlag,
		"symbolic": optFlag, "suffix": optValue, "target-directory": optValue, "verbose": optFlag,
		"help": optFlag, "version": optFlag,
	}}
	// patchOpts are GNU patch 2.7's options and Apple patch's (-x, -C).
	patchOpts = optSpec{values: "BDdFgioprVxYz", long: map[string]int{
		"backup": optFlag, "backup-if-mismatch": optFlag, "no-backup-if-mismatch": optFlag, "basename-prefix": optValue,
		"binary": optFlag, "batch": optFlag, "check": optFlag, "context": optFlag, "debug": optValue, "directory": optValue,
		"dry-run": optFlag, "ed": optFlag, "follow-symlinks": optFlag, "force": optFlag, "forward": optFlag,
		"fuzz": optValue, "get": optValue, "help": optFlag, "ifdef": optValue, "ignore-whitespace": optFlag,
		"input": optValue, "merge": optOptional, "normal": optFlag, "posix": optFlag, "prefix": optValue,
		"quiet": optFlag, "quoting-style": optValue, "read-only": optValue, "reject-file": optValue,
		"reject-format": optValue, "remove-empty-files": optFlag, "reverse": optFlag, "set-time": optFlag,
		"set-utc": optFlag, "silent": optFlag, "strip": optValue, "suffix": optValue, "unified": optFlag,
		"verbose": optFlag, "version": optFlag, "version-control": optValue, "output": optValue,
	}}
)

// optVal is where an option's value stands: word idx after its first skip
// bytes; ok is false where the line gives none.
type optVal struct {
	idx, skip int
	ok        bool
}

// optRead is a writer's words read by its optSpec: its operands in order, the
// words another platform's command reads as operands too (maybe), each short
// option and each long option given with their values, and whether an option
// word the line does not fix stands among them.
type optRead struct {
	operands, maybe []int
	short           map[byte][]optVal
	long            map[string][]optVal
	unsure          bool
	// unknown marks an option word that names no option of the command, or
	// more than one, which the command refuses; end is the index of the
	// `--` that ends the options, 0 where none does.
	unknown bool
	end     int
	// fails marks a reading as macOS's command reads the words in which it
	// refuses an option it does not have before it copies anything.
	fails bool
}

// has reports whether the short option c or the long option name was given.
func (r optRead) has(c byte, name string) bool {
	return len(r.short[c]) > 0 || len(r.long[name]) > 0
}

// last is the value the later of -c and --name was last given, and whether one was.
func (r optRead) last(c byte, name string) (optVal, bool) {
	var best optVal
	found := false
	for _, vs := range [][]optVal{r.short[c], r.long[name]} {
		for _, v := range vs {
			if !found || v.idx > best.idx {
				best, found = v, true
			}
		}
	}
	return best, found
}

// resolveLong is the long option name names as getopt_long resolves it: the
// option of that exact name, else the one option it begins; "" where it names
// none or more than one, which the command refuses.
func (o optSpec) resolveLong(name string) string {
	if _, ok := o.long[name]; ok {
		return name
	}
	found := ""
	for full := range o.long {
		if strings.HasPrefix(full, name) {
			if found != "" {
				return ""
			}
			found = full
		}
	}
	return found
}

// readOptsEach is each reading of s's words from index from by o: as GNU's
// command reads them, options anywhere among the operands, and, where it
// differs (always, for a command whose letters macOS reads with other
// values), as macOS's reads them (readOptsStop).
func readOptsEach(s segment, from int, o optSpec) []optRead {
	gnu, bsd := readOpts(s, from, o), readOptsStop(s, from, o, true)
	if bsd.fails || o.mac == "" && slices.Equal(gnu.operands, bsd.operands) {
		return []optRead{gnu}
	}
	return []optRead{gnu, bsd}
}

// readOpts reads s's words from index from by o, as GNU's getopt does.
func readOpts(s segment, from int, o optSpec) optRead {
	return readOptsStop(s, from, o, false)
}

// readOptsStop is readOpts, or, when stop is set, as macOS's command reads
// them: every word from the first operand on an operand, the letters o.mac
// names taking a value and every other letter none. macOS's cp, mv, install
// and ln have no long option, no -t, and no -T but install's, so a reading
// that meets one before the first operand fails, as the command does before
// it copies anything.
func readOptsStop(s segment, from int, o optSpec, stop bool) optRead {
	r := optRead{short: map[byte][]optVal{}, long: map[string][]optVal{}}
	toks := s.tokens
	opts := true
	for i := from; i < len(toks); i++ {
		t := toks[i]
		switch {
		case opts && t == "--":
			opts, r.end = false, i
		case opts && strings.HasPrefix(t, "--"):
			name, _, glued := strings.Cut(t[2:], "=")
			if isUnknown(name) || strings.ContainsRune(name, varMark) {
				// the name may be any option's, -t's among them
				r.unsure = true
			}
			if stop {
				r.fails = true
			}
			full := o.resolveLong(name)
			if full == "" && o.negate && strings.HasPrefix(name, "no-") && !glued {
				if f := o.resolveLong(name[len("no-"):]); f != "" {
					r.long["no-"+f] = append(r.long["no-"+f], optVal{})
					continue
				}
			}
			if full == "" {
				r.unknown = true
				continue
			}
			v := optVal{}
			switch {
			case glued:
				v = optVal{idx: i, skip: len("--") + len(name) + 1, ok: true}
			case o.long[full] == optValue:
				if i+1 < len(toks) {
					v = optVal{idx: i + 1, ok: true}
				}
				i++
			}
			r.long[full] = append(r.long[full], v)
		case opts && strings.HasPrefix(t, "-") && t != "-":
			values := o.values
			if stop {
				values = o.mac
			}
			for j := 1; j < len(t); j++ {
				c := t[j]
				if c == unknownMark || c == varMark {
					// a letter no line fixes may be any option's, -t's
					// among them; a value glued to a letter before it is
					// read before this one is reached
					r.unsure = true
					break
				}
				if stop && (c == 't' || c == 'T') && strings.IndexByte(values, c) < 0 {
					r.fails = true
					continue
				}
				if !stop && strings.IndexByte(o.optional, c) >= 0 {
					v := optVal{}
					if j+1 < len(t) {
						v = optVal{idx: i, skip: j + 1, ok: true}
					}
					r.short[c] = append(r.short[c], v)
					break
				}
				if strings.IndexByte(values, c) < 0 {
					r.short[c] = append(r.short[c], optVal{})
					continue
				}
				v := optVal{}
				switch {
				case j+1 < len(t):
					v = optVal{idx: i, skip: j + 1, ok: true}
				case i+1 < len(toks):
					v = optVal{idx: i + 1, ok: true}
					if !stop && strings.IndexByte(o.maybeOperand, c) >= 0 {
						r.maybe = append(r.maybe, i+1)
					}
					i++
				default:
					i++
				}
				r.short[c] = append(r.short[c], v)
				break
			}
		default:
			if stop {
				opts = false
			}
			r.operands = append(r.operands, i)
		}
	}
	return r
}

// only reports whether every option r read is one of the letters short or
// one of the long options long (a negation spelled "no-<name>"), with no
// option word the command does not have and none the line does not fix: an
// allowlist, so any other option voids what it permits.
func (r optRead) only(short string, long ...string) bool {
	if r.unsure || r.unknown {
		return false
	}
	for c := range r.short {
		if strings.IndexByte(short, c) < 0 {
			return false
		}
	}
	for name := range r.long {
		if !slices.Contains(long, name) {
			return false
		}
	}
	return true
}

// readings is each list of operands r can stand for: as read, and with the
// words another platform reads as operands among them.
func (r optRead) readings() [][]int {
	out := [][]int{r.operands}
	if len(r.maybe) == 0 {
		return out
	}
	merged := append(append([]int(nil), r.operands...), r.maybe...)
	slices.Sort(merged)
	return append(out, merged)
}

// eachSources is copySources under each of reads, every operand any of
// them reads as leaving where it is, once and in order.
func eachSources(reads []optRead) []int {
	var out []int
	for _, r := range reads {
		for _, op := range r.readings() {
			out = append(out, copySources(r, op)...)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// valueOf is the path value v carries, as a write.
func (w writerArgs) valueOf(v optVal) writeTarget {
	switch {
	case !v.ok:
		return unplacedWrite("")
	case v.skip == 0:
		return w.at(v.idx)
	}
	return w.valueAt(v.idx, v.skip)
}

// copySpec is the optSpec of cp, mv, install or ln.
func copySpec(name string) optSpec {
	switch name {
	case "mv":
		return mvOpts
	case "install":
		return installOpts
	case "ln":
		return lnOpts
	}
	return cpOpts
}

// copySources is the operands of a move, copy or link read by r that leave
// where they are or are linked to: every operand when -t names the
// destination, else every one but the last.
func copySources(r optRead, operands []int) []int {
	if _, ok := r.last('t', "target-directory"); ok || len(operands) == 0 {
		return operands
	}
	return operands[:len(operands)-1]
}

// xargsView is s as the command at site receives its words when an xargs
// before it runs it: each word holding xargs's replacement string (-I, -J,
// -i, --replace) with that string read as a name no line fixes, and with no
// replacement string a word no line fixes after the command's own, since
// xargs adds what it reads there.
func xargsView(s segment, site int) segment {
	x, ok := xargsBefore(s, site)
	if !ok {
		return s
	}
	toks := append([]string(nil), s.tokens...)
	repl, ok, _, unsure := xargsReplacement(s, x+1, site)
	switch {
	case unsure:
		for i := site + 1; i < len(toks); i++ {
			toks[i] = unknownText
		}
	case ok:
		for i := site + 1; i < len(toks); i++ {
			toks[i] = strings.ReplaceAll(toks[i], repl, unknownText)
		}
	default:
		toks = append(toks, unknownText)
	}
	v := s
	v.tokens = toks
	return v
}

// xargsReplacement is the replacement string xargs's options between from and
// site name: -I and -J take one, glued or the next word, and -i and --replace
// (or a start of it) take one glued on, `{}` by default, each also last in a cluster of flags
// (`-tI%`). ok is false where none is named; all says -J named it, which puts
// every word xargs reads where the string stands. unsure says the line does not
// fix the string, or an option word, which may name one, so any word of the
// command may hold it.
func xargsReplacement(s segment, from, site int) (repl string, ok, all, unsure bool) {
	for i := from; i < site; i++ {
		t := s.tokens[i]
		if isUnknown(t) || strings.ContainsRune(t, varMark) {
			unsure = true
			continue
		}
		// GNU xargs reads any start of --replace as it (`--repl=%`): no other
		// of its long options starts --r.
		if key, val, _ := strings.Cut(t, "="); len(key) > 2 && strings.HasPrefix("--replace", key) {
			repl, ok, all = val, true, false
			continue
		}
		if len(t) < 2 || t[0] != '-' || t[1] == '-' {
			continue
		}
		// A cluster of flags ends at the first that takes a value, which is
		// the rest of the word, or for -I and -J the next word.
		k := 1
		for k < len(t) && strings.IndexByte(xargsFlagLetters, t[k]) >= 0 {
			k++
		}
		if k == len(t) {
			continue
		}
		switch t[k] {
		case 'I', 'J':
			repl, ok, all = t[k+1:], true, t[k] == 'J'
			if repl == "" && i+1 < site {
				repl = s.tokens[i+1]
				i++
				unsure = unsure || unfixed(repl)
			}
		case 'i':
			repl, ok, all = t[k+1:], true, false
		}
	}
	if ok && repl == "" {
		repl = "{}"
	}
	return repl, ok, all, unsure
}

// xargsFlagLetters are the short options of GNU and BSD xargs that take no value.
const xargsFlagLetters = "0oprtx"

// tarValueLetters are the short options GNU tar and bsdtar give a value;
// bsdtar alone gives -s one (a substitution), which GNU tar reads as a flag.
const (
	tarValueLetters = "bCfFgHIKLNTVX"
	bsdtarValue     = "s"
)

// tarRead is a tar command's words as GNU tar and bsdtar read them: whether it
// extracts, whether it hands every member to standard output or a program
// instead of writing it (toProgram, under both readings), whether it keeps a
// member's absolute or parent path (absolute, under either), and where each
// -C or --directory value stands.
type tarRead struct {
	extract, toProgram, absolute bool
	dirs                         []optVal
}

// readTar reads s's words from index from: a first word without a dash holds
// option letters in the old style, each taking its value from the words after
// it in turn; a word with one dash is a cluster, which a letter taking a
// value ends; a long option is matched by the prefixes that name it.
func readTar(s segment, from int) tarRead {
	var r tarRead
	toProgram := [2]bool{}
	toks := s.tokens
	for reading, values := range []string{tarValueLetters, tarValueLetters + bsdtarValue} {
		next := from + 1
		take := func() optVal {
			if next >= len(toks) {
				return optVal{}
			}
			v := optVal{idx: next, ok: true}
			next++
			return v
		}
		letter := func(c byte, v optVal) {
			switch c {
			case 'x':
				r.extract = true
			case 'O':
				toProgram[reading] = true
			case 'P':
				r.absolute = true
			case 'C':
				if reading == 0 {
					r.dirs = append(r.dirs, v)
				}
			}
		}
		for i := from; i < len(toks); i++ {
			t := toks[i]
			switch {
			case i > from && i < next:
				// a value the old-style letters took
			case isUnknown(t) || strings.ContainsRune(t, varMark):
				// a word no line fixes may be -P
				r.absolute = true
			case t == "--":
				i = len(toks)
			case i == from && !strings.HasPrefix(t, "-"):
				for j := 0; j < len(t); j++ {
					v := optVal{}
					if strings.IndexByte(values, t[j]) >= 0 {
						v = take()
					}
					letter(t[j], v)
				}
			case strings.HasPrefix(t, "--"):
				name, _, glued := strings.Cut(t, "=")
				longValue := func() optVal {
					if glued {
						return optVal{idx: i, skip: len(name) + 1, ok: true}
					}
					if i+1 < len(toks) {
						i++
						return optVal{idx: i, ok: true}
					}
					return optVal{}
				}
				at := func(full string, min int) bool { return len(name) >= min && strings.HasPrefix(full, name) }
				switch {
				case at("--extract", len("--ex")) || at("--get", len("--get")):
					r.extract = true
				case at("--absolute-names", len("--ab")) || at("--absolute-paths", len("--ab")) || at("--insecure", len("--ins")):
					r.absolute = true
				case name == "--to-stdout" || at("--to-stdout", len("--to-s")) || at("--to-command", len("--to-c")):
					toProgram[reading] = true
					if !glued && at("--to-command", len("--to-c")) {
						i++
					}
				case at("--directory", len("--di")) || name == "--cd":
					v := longValue()
					if reading == 0 {
						r.dirs = append(r.dirs, v)
					}
				}
			case strings.HasPrefix(t, "-") && len(t) > 1:
				for j := 1; j < len(t); j++ {
					c := t[j]
					if strings.IndexByte(values, c) < 0 {
						letter(c, optVal{})
						continue
					}
					v := optVal{}
					switch {
					case j+1 < len(t):
						v = optVal{idx: i, skip: j + 1, ok: true}
					case i+1 < len(toks):
						i++
						v = optVal{idx: i, ok: true}
					}
					letter(c, v)
					break
				}
			}
		}
	}
	r.toProgram = toProgram[0] && toProgram[1]
	return r
}

// tarTargets is what an extracting tar writes: a tree of names no line fixes
// in the directory -C names or the one it runs in, and anywhere at all where
// it keeps absolute and parent paths; nothing where every member goes to
// standard output or a program. Two -C values place each other, so the line
// does not place where the second leads.
func (w writerArgs) tarTargets() []writeTarget {
	r := readTar(w.s, w.from)
	switch {
	case !r.extract:
		return nil
	case r.absolute:
		return []writeTarget{anywhere()}
	case r.toProgram:
		return []writeTarget{w.dirTarget(nil)}
	case len(r.dirs) > 1:
		return []writeTarget{unplacedWrite("")}
	case len(r.dirs) == 1:
		return []writeTarget{asTreeIn(w.valueOf(r.dirs[0]))}
	}
	return []writeTarget{asTreeIn(w.dirTarget(nil))}
}

// unzipReading are the letters with which unzip lists, tests, prints or shows
// an archive's comment and writes no member.
const unzipReading = "lptvzcZ"

// unzipTargets is what unzip writes: a tree of names no line fixes in the
// directory -d names (glued on, or the next word, after any cluster of
// letters) or the one it runs in, and anywhere at all with -: (members keep
// their parent paths); nothing where it only lists, tests or prints. The
// words after -x are names left out, read until the next option.
func (w writerArgs) unzipTargets() []writeTarget {
	toks := w.s.tokens
	var dir *optVal
	absolute, reading := false, false
	for i := w.from; i < len(toks); i++ {
		t := toks[i]
		if !strings.HasPrefix(t, "-") || len(t) < 2 {
			continue
		}
		for j := 1; j < len(t); j++ {
			c := t[j]
			switch {
			case c == ':':
				absolute = true
			case c == 'P':
				// the password: the rest of the word, else the next one
				if j+1 == len(t) {
					i++
				}
				j = len(t)
			case strings.IndexByte(unzipReading, c) >= 0:
				reading = true
			case c == 'd':
				v := optVal{}
				switch {
				case j+1 < len(t):
					v = optVal{idx: i, skip: j + 1, ok: true}
				case i+1 < len(toks):
					i++
					v = optVal{idx: i, ok: true}
				}
				dir = &v
				j = len(t)
			case c == 'x':
				for i+1 < len(toks) && !strings.HasPrefix(toks[i+1], "-") {
					i++
				}
				j = len(t)
			}
		}
	}
	switch {
	case reading:
		return nil
	case absolute:
		return []writeTarget{anywhere()}
	case dir != nil:
		return []writeTarget{asTreeIn(w.valueOf(*dir))}
	}
	return []writeTarget{asTreeIn(w.dirTarget(nil))}
}

// anywhere is a tree of names no line fixes that may land at any path.
func anywhere() writeTarget {
	return writeTarget{path: string(filepath.Separator), ok: true, wrote: true, tree: true, in: true}
}
