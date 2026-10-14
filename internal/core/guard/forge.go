package guard

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The forge-text check (iss-2610142210511831). The text an agent hands the
// forge CLI for a pull request, an issue or a comment is public the moment it
// is posted, and the forge keeps the pre-edit revision of it, so a live
// agent-session URL or a tool's attribution footer in it cannot be taken back.
// The commit-msg gate judges commit messages only; nothing judged this text
// before it left but an agent re-reading what it had already posted.
//
// The shell guard already sees every command an agent runs, so it is where the
// check goes, and it is abcd itself that runs it: no binary is looked up, and
// none can be missing. It judges with scanner.CheckOutbound, the one definition
// of the outbound policy `abcd lint outbound` applies, never a second pattern
// list.
//
// It is the repository's choice and never abcd's default (abcd-fixes-what-it-
// adds): it runs only where `.abcd/config.json` carries `outbound.forge: true`,
// which `abcd ahoy install --outbound-forge` writes. That is a fact about the
// repository and the text, not about the command's words, so no Pattern can say
// it; the guard raises it under reserved ids of its own, as git-stash-shared-
// stack does.
//
// What it reads (first version): `gh pr create|new|edit|comment` and `gh issue
// create|new|edit|comment`, their --title/-t, --body/-b and --body-file/-F in
// every spelling (`--body x`, `--body=x`, `-b x`, `-bx`, `-b=x`), a value written
// out in full on the line, a `$(cat <<'EOF' … EOF)` the tokenizer reads as the
// document it prints, and a body file read through the guard's guarded read.
// What it does not see is named in commands/guard.md.

const (
	// forgeTextEntryID is the reserved id a posting command whose text breaks
	// the outbound policy is blocked under.
	forgeTextEntryID = "forge-text-outbound"

	// forgeUnreadEntryID is the reserved id for forge text the guard could not
	// judge: a body file it cannot read or place, or a scanner configuration
	// that does not load. A warn, never a block and never a silent pass.
	forgeUnreadEntryID = "forge-text-unread"

	familyForge = "forge text"

	// maxForgeTexts bounds the texts one check judges. Each builds a scanner
	// from the repository's configuration, so a line of thousands of posting
	// commands would cost thousands of them; past the bound the check warns
	// that it stopped (forge-text-unread). No posting command carries more
	// than three texts, and no everyday line holds more than a few commands.
	maxForgeTexts = 16

	// maxForgeReadBytes is the body files' own read budget for one check.
	// It is never the scripts' budget (maxScriptFiles, maxScriptReadBytes):
	// the forge check runs first, so drawing on that one let a line's body
	// files spend what a script it then runs needed, and a script the guard
	// would block was only warned about. The file count is bounded by
	// maxForgeTexts already.
	maxForgeReadBytes = 1 << 20

	// maxForgeConfigBytes caps the config.json the opt-in is read from, the cap
	// the registry load uses for .abcd/guard.json.
	maxForgeConfigBytes = maxGuardFileBytes
)

// ForgeTextOptedIn reports whether the repository at repoRoot opted into the
// forge-text check: `.abcd/config.json` carries `outbound.forge: true`, a JSON
// boolean. Anything else — no file, a file that does not parse, a symlinked
// .abcd or config.json, a value that is not true — is not opted in. It is the
// one reader of the key; `abcd ahoy` asks it too.
func ForgeTextOptedIn(repoRoot string) bool {
	if repoRoot == "" {
		return false
	}
	if di, err := os.Lstat(filepath.Join(repoRoot, ".abcd")); err != nil || di.Mode()&os.ModeSymlink != 0 {
		return false
	}
	data, err := fsutil.ReadGuarded(filepath.Join(repoRoot, ".abcd", "config.json"), maxForgeConfigBytes)
	if err != nil {
		return false
	}
	var cfg struct {
		Outbound struct {
			Forge bool `json:"forge"`
		} `json:"outbound"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return false
	}
	return cfg.Outbound.Forge
}

// forgeGate is the opt-in of the repository a registry was loaded for, read
// the first time a posting command asks, so a line with no gh in it costs no
// file read.
type forgeGate struct {
	root  string
	once  sync.Once
	armed bool
}

func newForgeGate(root string) *forgeGate { return &forgeGate{root: root} }

func (g *forgeGate) on() bool {
	g.once.Do(func() { g.armed = ForgeTextOptedIn(g.root) })
	return g.armed
}

// forgePost is one posting command on the line: the segment it stands in, the
// command as `gh <group> <verb>`, and the text values it hands the forge.
type forgePost struct {
	seg     int
	command string
	values  []forgeValue
}

// forgeValue is one value of --title, --body or --body-file. tok is the token
// index of a value written as its own word, -1 for one attached to its flag;
// fixed reports whether the line fixes the word (no substitution whose output
// is unknown, no variable).
type forgeValue struct {
	flag  string
	text  string
	tok   int
	fixed bool
}

// forgePosting reports whether `gh <group> <verb>` posts text to the forge.
func forgePosting(group, verb string) bool {
	if group != "pr" && group != "issue" {
		return false
	}
	switch verb {
	case "create", "new", "edit", "comment":
		return true
	}
	return false
}

// forgeWord is the text of token i as gh will receive it, when the line fixes
// it: a literal word, or a word whose substitutions print fixed text (a
// `$(cat <<'EOF' … EOF)`). fixed is false for the second, which has no token
// text a path can be resolved from.
func forgeWord(s segment, i int) (text string, fixed, ok bool) {
	tok := s.tokens[i]
	if !isUnknown(tok) && !strings.ContainsRune(tok, varMark) {
		return tok, true, true
	}
	lit, has := s.literal[i]
	if !has {
		return "", false, false
	}
	if lit.split {
		return strings.Join(lit.words, " "), false, true
	}
	return lit.text, false, true
}

// forgePostsOf returns the posting commands segment s runs.
func forgePostsOf(s segment, seg int) []forgePost {
	var out []forgePost
	for _, a := range sitesNamed(s, "gh") {
		args := s.tokens[a.idx+1:]
		readings, _ := operandReadings(args, nil, 2)
		for _, idx := range readings {
			if len(idx) < 2 {
				continue
			}
			group, _, ok1 := forgeWord(s, a.idx+1+idx[0])
			verb, _, ok2 := forgeWord(s, a.idx+1+idx[1])
			if !ok1 || !ok2 || !forgePosting(group, verb) {
				continue
			}
			out = append(out, forgePost{seg: seg, command: "gh " + group + " " + verb,
				values: forgeValues(s, a.idx+1+idx[1]+1)})
			break
		}
	}
	return out
}

// The text flags, by what each hands the forge.
const (
	forgeTitle    = "title"
	forgeBody     = "body"
	forgeBodyFile = "body file"
)

// forgeFlag names the text flag a word sets (forgeTitle, forgeBody,
// forgeBodyFile), and the value attached to it, if any: `--body`, `--body=x`,
// `-b`, `-bx` and `-b=x`.
func forgeFlag(w string) (name, val string, attached bool) {
	longs := map[string]string{"--title": forgeTitle, "--body": forgeBody, "--body-file": forgeBodyFile}
	if strings.HasPrefix(w, "--") {
		n, v, eq := strings.Cut(w, "=")
		if l, ok := longs[n]; ok {
			return l, v, eq
		}
		return "", "", false
	}
	if len(w) < 2 || w[0] != '-' {
		return "", "", false
	}
	shorts := map[byte]string{'t': forgeTitle, 'b': forgeBody, 'F': forgeBodyFile}
	l, ok := shorts[w[1]]
	if !ok {
		return "", "", false
	}
	if len(w) == 2 {
		return l, "", false
	}
	return l, strings.TrimPrefix(w[2:], "="), true
}

// forgeValues reads the text flags in the words from start on, up to `--`.
func forgeValues(s segment, start int) []forgeValue {
	var out []forgeValue
	for j := start; j < len(s.tokens); j++ {
		w, fixed, ok := forgeWord(s, j)
		if !ok {
			continue
		}
		if w == "--" {
			break
		}
		name, val, attached := forgeFlag(w)
		if name == "" {
			continue
		}
		if attached {
			out = append(out, forgeValue{flag: name, text: val, tok: -1, fixed: fixed})
			continue
		}
		if j+1 >= len(s.tokens) {
			break
		}
		j++
		v, vfixed, vok := forgeWord(s, j)
		if !vok {
			// A value the line does not fix (`--body "$BODY"`) is not seen.
			continue
		}
		out = append(out, forgeValue{flag: name, text: v, tok: j, fixed: vfixed})
	}
	return out
}

// forgeText is the check over one reading of a command line: every posting
// command's text values, judged when the repository opted in. recs holds the
// shell states each segment runs in, from which a body file's path resolves.
func (r Registry) forgeText(rc *readCtx, segs []segment, recs []scriptSeg) []payloadSignal {
	if r.forge == nil {
		return nil
	}
	var posts []forgePost
	for i, s := range segs {
		posts = append(posts, forgePostsOf(s, i)...)
	}
	if len(posts) == 0 || !r.forge.on() {
		return nil
	}
	var out []payloadSignal
	add := func(sig payloadSignal) {
		if !containsSignal(out, sig) {
			out = append(out, sig)
		}
	}
	judged, readBytes := 0, 0
	for _, p := range posts {
		for _, v := range p.values {
			if judged >= maxForgeTexts {
				add(payloadSignal{
					id: forgeUnreadEntryID, verdict: VerdictWarn, family: familyForge,
					reason: fmt.Sprintf("This line hands the forge more than the %d texts one check judges, so the guard stopped "+
						"and has not checked the rest for a live agent-session URL or a tool attribution footer.", maxForgeTexts),
					successor: "Post each pull request, issue or comment in a command of its own, so the guard checks what is posted.",
				})
				return out
			}
			judged++
			if v.flag == forgeBodyFile {
				r.forgeFile(rc, segs, recs, p, v, &readBytes, add)
				continue
			}
			r.forgeJudge(p.command, v.flag, v.text, add)
		}
	}
	return out
}

// forgeFile reads a --body-file the line names and judges it. `-` is standard
// input, which the guard does not read, and a path the line does not fix is
// not seen, as a body held in a variable is not; every other way the file goes
// unread warns.
//
// readBytes is the bytes this check has read for body files so far, on their
// own budget (maxForgeReadBytes), never the scripts'.
func (r Registry) forgeFile(rc *readCtx, segs []segment, recs []scriptSeg, p forgePost, v forgeValue, readBytes *int, add func(payloadSignal)) {
	if v.text == "-" || !v.fixed || strings.TrimSpace(v.text) == "" {
		return
	}
	s := segs[p.seg]
	if v.tok >= 0 && s.globAt(v.tok) {
		return
	}
	var paths []string
	unplaced := false
	for _, st := range recs[p.seg].states {
		var pth string
		var ok bool
		if v.tok >= 0 {
			pth, ok = st.resolveToken(rc, s, v.tok)
		} else {
			pth, ok = st.resolve(rc, v.text, false)
		}
		if !ok {
			unplaced = true
			continue
		}
		if !containsString(paths, pth) {
			paths = append(paths, pth)
		}
	}
	if unplaced {
		add(forgeUnreadSignal(p.command, "is relative to a directory this line leaves unknown"))
	}
	if len(paths) == 0 {
		return
	}
	var before []writeTarget
	for j := range segs {
		if j != p.seg && earlier(recs[j].order, recs[p.seg].order) {
			for _, st := range recs[j].states {
				before = append(before, writesOf(rc, segs[j], st)...)
			}
		}
	}
	for _, pth := range paths {
		if written, _ := writtenBefore(pth, before); written {
			add(forgeUnreadSignal(p.command, "is written earlier on this line, so the file the guard could read now is not the one gh will post"))
			continue
		}
		real, err := filepath.EvalSymlinks(pth)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				add(forgeUnreadSignal(p.command, "does not exist"))
			} else {
				add(forgeUnreadSignal(p.command, "cannot be resolved"))
			}
			continue
		}
		if *readBytes >= maxForgeReadBytes {
			add(forgeUnreadSignal(p.command, fmt.Sprintf("is past the %d KiB of body files one check reads", maxForgeReadBytes>>10)))
			continue
		}
		fr := rc.files.read(real)
		*readBytes += len(fr.data)
		tally(len(fr.data))
		switch {
		case fr.err != nil:
			add(forgeUnreadSignal(p.command, "cannot be read as a regular file (a pipe, a directory, or a file this account may not read)"))
		case fr.tooBig:
			add(forgeUnreadSignal(p.command, fmt.Sprintf("is over the %d KiB the guard reads", maxScriptBytes>>10)))
		default:
			r.forgeJudge(p.command, forgeBodyFile, string(fr.data), add)
		}
	}
}

// forgeJudge runs the outbound check over one text. The text is never carried
// into the verdict: only the kind and the line of what was found.
func (r Registry) forgeJudge(command, what, text string, add func(payloadSignal)) {
	text = strings.NewReplacer(unknownText, "", string(varMark), "").Replace(text)
	if strings.TrimSpace(text) == "" {
		return
	}
	findings, err := scanner.CheckOutbound(r.forge.root, text, "the "+what+" of "+command)
	if len(findings) > 0 {
		add(forgeLeakSignal(command, what, findings))
		return
	}
	if err != nil {
		add(payloadSignal{
			id: forgeUnreadEntryID, verdict: VerdictWarn, family: familyForge,
			reason: "The guard could not judge the " + what + " this `" + command + "` hands the forge: the repository's scanner " +
				"configuration (.abcd/config/pii.json) does not load, and judging with fewer detectors than it declares would " +
				"be a clearance nobody earned.",
			successor: "Repair .abcd/config/pii.json, or remove any session URL and tool attribution footer from the text yourself " +
				"before posting.",
		})
	}
}

// forgeKindDesc names a finding kind in plain words.
func forgeKindDesc(kind string) string {
	switch kind {
	case "harness:session_url":
		return "a live agent-session URL"
	case "harness:attribution_footer":
		return "a tool attribution footer"
	}
	return "an outbound-policy finding"
}

// forgeLeakSignal is the block. It names what was found and where, never the
// matched text: a refusal is shown to the person and handed to the agent, and
// reprinting the session URL would publish it into another log.
func forgeLeakSignal(command, what string, findings []scanner.Finding) payloadSignal {
	f := findings[0]
	found := fmt.Sprintf("%s on line %d", forgeKindDesc(f.Kind), f.Line)
	if n := len(findings); n > 1 {
		found = fmt.Sprintf("%d outbound-policy findings, the first %s", n, found)
	}
	return payloadSignal{
		id: forgeTextEntryID, verdict: VerdictBlock, family: familyForge,
		reason: "The " + what + " this `" + command + "` hands the forge carries " + found + ". This repository checks the " +
			"text it posts (outbound.forge in .abcd/config.json), and a forge keeps the pre-edit revision of whatever is " +
			"posted, so it cannot be taken back once sent.",
		successor: "Remove the session URL, and replace a tool attribution footer with the repository's own disclosure " +
			"trailer (Assisted-by: <Vendor>:<model-version>), then run the same command again.",
	}
}

// forgeUnreadSignal is the warn for a body file the guard did not read; why
// completes "The body file this `<command>` names …".
func forgeUnreadSignal(command, why string) payloadSignal {
	return payloadSignal{
		id: forgeUnreadEntryID, verdict: VerdictWarn, family: familyForge,
		reason: "The body file this `" + command + "` names " + why + ", so the guard has not checked it for a live " +
			"agent-session URL or a tool attribution footer.",
		successor: "Write the body file in a command of its own first and name it by a path the line fixes, or pass the " +
			"text with --body, so the guard checks what is posted.",
	}
}
