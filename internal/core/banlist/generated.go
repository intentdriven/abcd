package banlist

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The generated block (itd-76, spc-31). A caller that DERIVES private entries from
// somewhere else — the sources corpus projects its confidential entries' titles,
// aliases and opted-in authors — owns one fenced block of the private store and
// regenerates it whole, while every hand-written or verb-added line outside the
// fence survives. The block's lines are ordinary keyed entries, so the committed
// guard enforces them with no change to its parser: the fence is two comment lines
// only this file's writer gives meaning to.
//
// Matching is this package's, not the caller's. The phrase-to-pattern projection
// (PhrasePattern) and the scan over arbitrary text (ScanText) both live here and
// both drive the guard's own engine, so what the pre-commit guard refuses and what
// a pre-share scan reports cannot disagree: there is one matcher, and it is grep.

// KeyedPattern is one entry a caller hands the private layer: a non-secret key and
// the POSIX ERE the guard's engine enforces. The pattern never reaches any output.
type KeyedPattern struct {
	Key     string
	Pattern string
}

// Hit is one match found by ScanText: the key, the 1-based line, and the byte
// offset of the matched span in the scanned text. The span may begin one byte
// before the phrase itself, where the leading boundary consumed a non-alphanumeric
// neighbour. There is deliberately no field for the matched text: the type is the
// redaction, so a scan's report is safe to relay.
type Hit struct {
	Key    string `json:"key"`
	Line   int    `json:"line"`
	Offset int    `json:"offset"`
}

// GeneratedResult is the outcome of a generated-block sync.
type GeneratedResult struct {
	// Path is the private store, repo-relative.
	Path string `json:"path"`
	// Owner names the block synced.
	Owner string `json:"owner"`
	// Entries counts the lines the block now holds.
	Entries int `json:"entries"`
	// Created reports that the store did not exist and this sync created it.
	Created bool `json:"created"`
	// Changed reports that the store's bytes changed; an idempotent sync is false.
	Changed bool `json:"changed"`
}

// The fence. A line whose ASCII-trimmed text begins with the begin prefix and the
// owner opens the block; the end prefix and the owner close it. They are comments,
// so every reader of the store that does not know about blocks reads them as such.
const (
	generatedBeginPrefix = "# >>> abcd-generated: "
	generatedEndPrefix   = "# <<< abcd-generated: "
)

// minPhraseAlnum is the fewest letters or digits a projected phrase may carry. A
// shorter fragment matches inside ordinary names and words (iss-2609212142568782),
// and a ban that refuses legitimate text is one somebody switches off.
const minPhraseAlnum = 3

// unicodeSpaces are the non-ASCII members of Unicode's space-separator category,
// accepted between the words of a phrase. The engine runs in the C locale, where
// [[:space:]] is ASCII-only, so without these one non-breaking space between two
// words evades the ban (the class iss-2608311306535485 recorded against RE2).
var unicodeSpaces = []rune{0x00A0, 0x1680, 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005,
	0x2006, 0x2007, 0x2008, 0x2009, 0x200A, 0x202F, 0x205F, 0x3000}

// PhrasePattern projects one or more literal phrases into ONE pattern the private
// layer's engine (POSIX ERE, `grep -iE` under LC_ALL=C) enforces. It is the only
// place a literal becomes a pattern, so the guard and every scan read the same one.
//
//   - Literal: every ERE metacharacter is neutralised, as a one-byte bracket
//     expression where POSIX makes that portable and as `\^` for the one it does
//     not. A title is never an expression.
//   - Whitespace-flexible: words are split on Unicode white space and joined by one
//     or more ASCII spaces OR non-ASCII space separators, so a no-break or thin
//     space between two words is still the phrase.
//   - Case-insensitive, Unicode-aware: the engine folds ASCII itself; a non-ASCII
//     letter is written as the alternation of its simple case-fold orbit, which the
//     byte-oriented C locale cannot fold.
//   - Bounded by NEIGHBOURS, never by `\b`: the match needs a line edge or a byte
//     that is not an ASCII letter or digit on either side. `\b` is undefined in POSIX
//     ERE, and RE2's ASCII-only `\b` finds no boundary before a non-ASCII initial at
//     the start of a word — a silent miss. The neighbour test errs the other way:
//     a non-ASCII letter beside the phrase counts as a boundary, so the ban can
//     match more than the phrase, never less.
//
// A phrase with fewer than three letters or digits is refused, as is one that is
// not valid UTF-8. The error never quotes the phrase.
func PhrasePattern(phrases ...string) (string, error) {
	if len(phrases) == 0 {
		return "", fmt.Errorf("%w: no phrase to project", ErrInvalidPattern)
	}
	alts := make([]string, 0, len(phrases))
	for i, ph := range phrases {
		if !utf8.ValidString(ph) || strings.IndexByte(ph, 0) >= 0 {
			return "", fmt.Errorf("%w: phrase %d is not valid UTF-8 text (the value is withheld)", ErrInvalidPattern, i+1)
		}
		alnum := 0
		for _, r := range ph {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				alnum++
			}
		}
		if alnum < minPhraseAlnum {
			return "", fmt.Errorf("%w: phrase %d has fewer than %d letters or digits and would match ordinary text (the value is withheld)",
				ErrInvalidPattern, i+1, minPhraseAlnum)
		}
		words := strings.FieldsFunc(ph, unicode.IsSpace)
		enc := make([]string, len(words))
		for j, w := range words {
			enc[j] = literalWord(w)
		}
		alts = append(alts, strings.Join(enc, phraseSeparator()))
	}
	return "(^|[^[:alnum:]])(" + strings.Join(alts, "|") + ")([^[:alnum:]]|$)", nil
}

// phraseSeparator is the between-words expression: one or more white-space runes.
func phraseSeparator() string {
	parts := []string{"[[:space:]]"}
	for _, r := range unicodeSpaces {
		parts = append(parts, string(r))
	}
	return "(" + strings.Join(parts, "|") + ")+"
}

// literalWord encodes one word as a literal for the engine.
func literalWord(w string) string {
	var b strings.Builder
	for _, r := range w {
		switch {
		case r == '^':
			b.WriteString(`\^`)
		case r == '\\':
			b.WriteString(`[\]`)
		case strings.ContainsRune(".[]()*+?{}|$", r):
			b.WriteString("[" + string(r) + "]")
		case r < utf8.RuneSelf:
			b.WriteRune(r)
		default:
			b.WriteString(foldOrbit(r))
		}
	}
	return b.String()
}

// foldOrbit writes a non-ASCII rune as the alternation of its simple case-fold
// orbit ("(É|é)"), or as itself when it has no other case.
func foldOrbit(r rune) string {
	orbit := []string{string(r)}
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		orbit = append(orbit, string(f))
	}
	if len(orbit) == 1 {
		return orbit[0]
	}
	return "(" + strings.Join(orbit, "|") + ")"
}

// ScanText reports every place the given patterns match text, by key, line and
// byte offset, through the guard's own engine: `grep -inobaE` under LC_ALL=C, each
// pattern on STDIN (never argv), against the text in a 0600 temporary file removed
// afterwards. It is the read-only twin of the pre-commit guard, so a text this scan
// calls clean is a text the guard would pass, line for line.
//
// No grep is ErrNoEngine; a pattern the engine refuses is ErrInvalidPattern naming
// the key. Neither the pattern nor any matched byte is returned or quoted.
func ScanText(patterns []KeyedPattern, text []byte) ([]Hit, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	grepBinOnce.Do(func() { grepBin, _ = exec.LookPath("grep") })
	if grepBin == "" {
		return nil, ErrNoEngine
	}
	f, err := os.CreateTemp("", "abcd-scan-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := f.Write(text); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	var hits []Hit
	for _, p := range patterns {
		cmd := exec.Command(grepBin, "-inobaE", "-f", "-", "--", f.Name())
		cmd.Env = append(os.Environ(), "LC_ALL=C")
		cmd.Stdin = strings.NewReader(p.Pattern + "\n")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = nil // grep's diagnostics quote the expression
		err := cmd.Run()
		var exit *exec.ExitError
		switch {
		case err == nil:
		case errors.As(err, &exit) && exit.ExitCode() == 1:
			continue
		case errors.As(err, &exit):
			return nil, fmt.Errorf("%w for key %q: the guard's grep refuses it (the value is withheld)", ErrInvalidPattern, p.Key)
		default:
			return nil, fmt.Errorf("%w: %v", ErrNoEngine, err)
		}
		sc := bufio.NewScanner(&stdout)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			lineField, rest, ok := strings.Cut(sc.Text(), ":")
			if !ok {
				continue
			}
			offField, _, ok := strings.Cut(rest, ":")
			if !ok {
				continue
			}
			line, lerr := strconv.Atoi(lineField)
			off, oerr := strconv.Atoi(offField)
			if lerr != nil || oerr != nil {
				continue
			}
			hits = append(hits, Hit{Key: p.Key, Line: line, Offset: off})
		}
	}
	return hits, nil
}

// SyncGeneratedBlock regenerates the block owner owns in the private store from
// entries, and touches nothing outside it.
//
// Every key must sit in the owner's namespace (`<owner>/…`), be unique, and not be
// carried by a hand-written line outside the block; every pattern must be one the
// guard's engine accepts as written and must round-trip through the store's format.
// All of that is checked before anything is written, and no refusal quotes a
// pattern. The store's own contract holds as for AddPrivate: a legacy store with
// entries is refused rather than reinterpreted, the store must be gitignored, the
// write is contained and atomic at 0600, and the load-modify-write runs under the
// store's lock.
//
// An absent store with nothing to project stays absent — a machine that has not
// opted in is not opted in by an empty sync. An empty projection over an existing
// block empties the block and keeps its fence. A second sync of the same entries
// writes nothing (Changed is false).
func SyncGeneratedBlock(repoRoot, owner string, entries []KeyedPattern) (GeneratedResult, error) {
	res := GeneratedResult{Path: PrivateRelPath, Owner: owner}
	if !validKey(owner) || strings.Contains(owner, "/") {
		return res, fmt.Errorf("%w: block owner %q", ErrInvalidKey, owner)
	}
	want := map[string]bool{}
	for _, e := range entries {
		if !validKey(e.Key) || !strings.HasPrefix(e.Key, owner+"/") {
			return res, fmt.Errorf("%w: %q is outside the %q block's namespace (want %s/…)", ErrInvalidKey, e.Key, owner, owner)
		}
		if want[e.Key] {
			return res, fmt.Errorf("%w: %q appears twice in the projection", ErrDuplicateKey, e.Key)
		}
		want[e.Key] = true
		if strings.IndexByte(e.Pattern, 0) >= 0 {
			return res, fmt.Errorf("%w for key %q: contains a NUL byte (the value is withheld)", ErrInvalidPattern, e.Key)
		}
		if fault, byEngine := checkPattern(e.Pattern); fault != faultNone {
			return res, fmt.Errorf("%w for key %q: %s (the value is withheld)", ErrInvalidPattern, e.Key, patternRefusal(fault, byEngine))
		}
		if !composedLineRoundTrips(e.Key, e.Pattern) {
			return res, fmt.Errorf("%w for key %q: the composed entry does not round-trip (the value is withheld)", ErrInvalidPattern, e.Key)
		}
	}
	res.Entries = len(entries)

	// Nothing to project and no store: nothing to do, and no tier to create.
	if _, err := readPrivate(repoRoot); errors.Is(err, ErrNoStore) && len(entries) == 0 {
		return res, nil
	}
	if err := requireIgnoredStore(repoRoot); err != nil {
		return res, err
	}

	err := withPrivateLock(repoRoot, func() error {
		data, err := readPrivate(repoRoot)
		switch {
		case err == nil:
		case errors.Is(err, ErrNoStore):
			if len(entries) == 0 {
				return nil
			}
			data = []byte(privateHeader)
			res.Created = true
		default:
			return err
		}
		parsed, keyed, perr := parse(data)
		if perr != nil {
			return perr
		}
		if !keyed && len(parsed) > 0 {
			return legacyStoreRefusal("sync a generated block into")
		}
		body := string(data)
		if !keyed {
			body = privateFormatDecl + "\n" + body
		}

		lines := strings.Split(body, "\n")
		begin, end, ferr := findBlock(lines, owner)
		if ferr != nil {
			return ferr
		}
		// Line numbers in parsed are 1-based over the ORIGINAL data; a prepended
		// declaration shifts them by one, and only an entryless store gets one, so
		// the collision check below reads the parse of the body actually written.
		reparsed, _, perr := parse([]byte(body))
		if perr != nil {
			return perr
		}
		for _, e := range reparsed {
			inBlock := begin >= 0 && e.line-1 > begin && e.line-1 < end
			if !inBlock && !e.unparsed && want[e.key] {
				return fmt.Errorf("%w: %q is already a hand-written entry outside the generated block; remove it there first", ErrDuplicateKey, e.key)
			}
		}

		block := renderBlock(owner, entries)
		var out []string
		switch {
		case begin >= 0:
			out = append(out, lines[:begin]...)
			out = append(out, block...)
			out = append(out, lines[end+1:]...)
		case len(entries) == 0:
			return nil
		default:
			trimmed := strings.TrimRight(body, "\n")
			out = append(strings.Split(trimmed, "\n"), "")
			out = append(out, block...)
			out = append(out, "")
		}
		next := strings.Join(out, "\n")
		if next == string(data) {
			return nil
		}
		if err := writePrivateStore(repoRoot, []byte(next)); err != nil {
			return err
		}
		res.Changed = true
		return nil
	})
	if err != nil {
		return GeneratedResult{Path: PrivateRelPath, Owner: owner}, err
	}
	return res, nil
}

// findBlock locates owner's fence: the 0-based begin and end line indexes, or -1,-1
// when there is none. Anything but exactly one begin followed by exactly one end is
// a malformed store, refused by line number.
func findBlock(lines []string, owner string) (begin, end int, err error) {
	begin, end = -1, -1
	bm := generatedBeginPrefix + owner
	em := generatedEndPrefix + owner
	for i, raw := range lines {
		l := trimLead(trimTrail(raw))
		switch {
		case isMarker(l, bm):
			if begin >= 0 {
				return -1, -1, fmt.Errorf("%w: %s opens the %q generated block twice (lines %d and %d); remove one fence and re-run",
					ErrMalformedStore, PrivateRelPath, owner, begin+1, i+1)
			}
			begin = i
		case isMarker(l, em):
			if end >= 0 {
				return -1, -1, fmt.Errorf("%w: %s closes the %q generated block twice (lines %d and %d); remove one fence and re-run",
					ErrMalformedStore, PrivateRelPath, owner, end+1, i+1)
			}
			end = i
		}
	}
	switch {
	case begin < 0 && end < 0:
		return -1, -1, nil
	case begin < 0 || end < 0 || end < begin:
		return -1, -1, fmt.Errorf("%w: %s holds a broken %q generated block fence (an opening line needs one closing line after it); repair or remove the fence and re-run",
			ErrMalformedStore, PrivateRelPath, owner)
	}
	return begin, end, nil
}

// isMarker reports whether a trimmed line is the marker: the prefix, then a space or
// the end of the line (so owner "sources" never claims "sources2"'s fence).
func isMarker(line, marker string) bool {
	if !strings.HasPrefix(line, marker) {
		return false
	}
	rest := line[len(marker):]
	return rest == "" || rest[0] == ' '
}

// renderBlock is the fence and its entries.
func renderBlock(owner string, entries []KeyedPattern) []string {
	out := []string{
		generatedBeginPrefix + owner + " >>>",
		"# Generated by abcd; every line between these fences is rewritten on each sync.",
		"# Hand-written entries belong outside them.",
	}
	for _, e := range entries {
		out = append(out, e.Key+" "+e.Pattern)
	}
	return append(out, generatedEndPrefix+owner+" <<<")
}
