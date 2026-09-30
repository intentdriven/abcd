package source

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// AddRequest registers one source. Class is required and has no default: the
// classification is declared once, at ingestion, and a forgotten flag must not file
// a confidential document as public.
type AddRequest struct {
	Corpus string
	Key    string
	Title  string
	// Type is the CSL item type; "document" when empty.
	Type  string
	Class string
	// Permission is the permission_status; by class when empty (confidential:
	// no-public-citation, public: citable).
	Permission string
	Authors    []Name
	Year       int
	Venue      string
	URL        string
	Keywords   []string
	Aliases    []string
	BanAuthors bool
	// Original is the document to store (any format); Text is its extracted text.
	// A Markdown or plain-text original is its own text.
	Original string
	Text     string
}

// AddResult is what a registration or a declassification did, named by key.
type AddResult struct {
	Key        string   `json:"key"`
	Class      string   `json:"class"`
	Permission string   `json:"permission_status"`
	Folder     string   `json:"folder"`
	Files      []string `json:"files"`
}

// maxDocumentBytes caps a stored original or text (trust boundary).
const maxDocumentBytes = 512 << 20

var (
	cslTypeRe = regexp.MustCompile(`^[a-z][a-z_-]{0,39}$`)
	extRe     = regexp.MustCompile(`^\.[a-z0-9]{1,8}$`)
)

// textExts are originals that are their own extracted text.
var textExts = []string{".md", ".markdown", ".txt"}

// oneLine reports whether s is non-empty after trimming and holds no line break.
func oneLine(s string) bool {
	return strings.TrimSpace(s) != "" && !strings.ContainsAny(s, "\r\n")
}

// Add registers a document: the CSL-JSON entry with its custom block, the original
// and its extracted text under <class>/<key>/, committed in the corpus repository.
// Every check runs before the first write, and no refusal quotes a title, alias or
// author.
func Add(req AddRequest) (AddResult, error) {
	c, err := loadCorpus(req.Corpus)
	if err != nil {
		return AddResult{}, err
	}
	entry, text, ext, err := validateAdd(c, req)
	if err != nil {
		return AddResult{}, err
	}

	res := AddResult{Key: entry.ID, Class: req.Class, Permission: entry.Custom.PermissionStatus,
		Folder: req.Class + "/" + entry.ID, Files: []string{}}
	err = withLock(c.Dir, func() error {
		// Re-read under the lock: a concurrent add of the same key loses here.
		fresh, err := loadCorpus(c.Dir)
		if err != nil {
			return err
		}
		if _, dup := fresh.Lookup(entry.ID); dup || len(fresh.folders[entry.ID]) > 0 {
			return fmt.Errorf("%w: %q", ErrDuplicateSource, entry.ID)
		}
		before, err := os.ReadFile(filepath.Join(c.Dir, SourcesFile))
		if err != nil {
			return err
		}
		folder := filepath.Join(c.Dir, req.Class, entry.ID)
		rollback := func() {
			_ = os.RemoveAll(folder)
			_ = fsutil.WriteFileAtomic(filepath.Join(c.Dir, SourcesFile), before, 0o600)
			_, _ = corpusGit(c.Dir, "reset", "-q", "--", SourcesFile, res.Folder)
		}
		if err := os.MkdirAll(folder, 0o700); err != nil {
			return fmt.Errorf("%w: cannot create %s: %v", ErrCorpusInvalid, res.Folder, err)
		}
		if req.Original != "" {
			name := "original" + ext
			if err := copyFile(req.Original, filepath.Join(folder, name)); err != nil {
				rollback()
				return err
			}
			res.Files = append(res.Files, res.Folder+"/"+name)
		}
		body := "---\nkey: " + entry.ID + "\n---\n\n" + text
		if text != "" && !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		if err := fsutil.WriteFileAtomic(filepath.Join(folder, TextFile), []byte(body), 0o600); err != nil {
			rollback()
			return err
		}
		res.Files = append(res.Files, res.Folder+"/"+TextFile)
		raw, err := json.Marshal(entry)
		if err != nil {
			rollback()
			return err
		}
		if err := writeSources(c.Dir, append(fresh.raw, raw)); err != nil {
			rollback()
			return err
		}
		if err := commit(c.Dir, "source: add "+entry.ID+" ("+req.Class+")", SourcesFile, res.Folder); err != nil {
			rollback()
			return err
		}
		return nil
	})
	if err != nil {
		return AddResult{}, err
	}
	return res, nil
}

// validateAdd checks a registration and builds its entry, the text to store, and
// the original's extension.
func validateAdd(c *Corpus, req AddRequest) (Entry, string, string, error) {
	bad := func(format string, a ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrInvalidEntry}, a...)...)
	}
	if !keyRe.MatchString(req.Key) {
		return Entry{}, "", "", bad("the key is not a source key (lowercase letters, digits, '.', '_' or '-', at most 64; the value is withheld)")
	}
	if req.Class != ClassConfidential && req.Class != ClassPublic {
		return Entry{}, "", "", bad("declare the class — confidential or public; it is decided once, here, and never defaulted")
	}
	if !oneLine(req.Title) {
		return Entry{}, "", "", bad("the title is empty or spans lines")
	}
	typ := req.Type
	if typ == "" {
		typ = "document"
	}
	if !cslTypeRe.MatchString(typ) {
		return Entry{}, "", "", bad("the CSL type %q is not a CSL type name", typ)
	}
	conf := req.Class == ClassConfidential
	perm := req.Permission
	if perm == "" {
		perm = PermissionCitable
		if conf {
			perm = PermissionNoPublicCitation
		}
	}
	if !validPermission(perm) {
		return Entry{}, "", "", bad("permission status %q is not one of %s", perm, strings.Join(Permissions, ", "))
	}
	if conf && perm == PermissionCitable {
		return Entry{}, "", "", bad("a confidential source cannot be citable; register it public, or declassify it when it is published")
	}
	for i, a := range req.Aliases {
		if !oneLine(a) {
			return Entry{}, "", "", bad("alias %d is empty or spans lines", i+1)
		}
	}
	for i, k := range req.Keywords {
		if !oneLine(k) {
			return Entry{}, "", "", bad("keyword %d is empty or spans lines", i+1)
		}
	}
	for i, n := range req.Authors {
		if nameText(n) == "" || strings.ContainsAny(n.Family+n.Given+n.Literal, "\r\n") {
			return Entry{}, "", "", bad("author %d is empty or spans lines", i+1)
		}
	}
	if req.URL != "" && !oneLine(req.URL) {
		return Entry{}, "", "", bad("the URL spans lines")
	}
	if req.Venue != "" && !oneLine(req.Venue) {
		return Entry{}, "", "", bad("the venue spans lines")
	}
	if req.Year != 0 && (req.Year < 1000 || req.Year > 9999) {
		return Entry{}, "", "", bad("the year %d is not a four-digit year", req.Year)
	}
	if req.BanAuthors && !conf {
		return Entry{}, "", "", bad("ban-authors applies to a confidential source only")
	}

	entry := Entry{ID: req.Key, Type: typ, Title: strings.TrimSpace(req.Title), Author: req.Authors,
		ContainerTitle: req.Venue, URL: req.URL,
		Custom: Custom{Confidential: conf, PermissionStatus: perm, Keywords: trimAll(req.Keywords),
			Aliases: trimAll(req.Aliases), BanAuthors: req.BanAuthors}}
	if req.Year != 0 {
		entry.Issued = &Date{DateParts: [][]int{{req.Year}}}
	}
	if conf {
		// Every string the ban will carry must project now: a phrase the guard cannot
		// hold would otherwise surface at the next sync as a refusal of the whole corpus.
		if _, err := projectEntry(entry); err != nil {
			return Entry{}, "", "", err
		}
		if field := identifyingKey(entry); field != "" {
			return Entry{}, "", "", fmt.Errorf("%w: the key contains the source's %s (the key is withheld here for that reason), and the key is the one handle every refusal and scan prints; choose an opaque key such as conf2026a",
				ErrIdentifyingKey, field)
		}
	}
	if _, dup := c.Lookup(req.Key); dup || len(c.folders[req.Key]) > 0 {
		return Entry{}, "", "", fmt.Errorf("%w: %q", ErrDuplicateSource, req.Key)
	}

	var text, ext string
	if req.Original != "" {
		fi, err := os.Stat(req.Original)
		if err != nil || !fi.Mode().IsRegular() {
			return Entry{}, "", "", bad("the document is not a readable regular file")
		}
		if fi.Size() > maxDocumentBytes {
			return Entry{}, "", "", bad("the document is over %d bytes", maxDocumentBytes)
		}
		if e := strings.ToLower(filepath.Ext(req.Original)); extRe.MatchString(e) {
			ext = e
		}
		entry.Custom.File = "original" + ext
	}
	switch {
	case req.Text != "":
		b, err := fsutil.ReadGuarded(req.Text, maxDocumentBytes)
		if err != nil {
			return Entry{}, "", "", bad("the extracted text is not a readable regular file")
		}
		text = string(b)
	case req.Original != "" && slices.Contains(textExts, ext):
		b, err := fsutil.ReadGuarded(req.Original, maxDocumentBytes)
		if err != nil {
			return Entry{}, "", "", bad("the document cannot be read as text")
		}
		text = string(b)
	case req.Original != "":
		return Entry{}, "", "", bad("a %q original needs its extracted text (pass the text file); abcd converts nothing itself", ext)
	case req.URL == "":
		return Entry{}, "", "", bad("nothing to store: give the document, its extracted text, or its URL")
	}
	return entry, text, ext, nil
}

// trimAll trims each value.
func trimAll(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.TrimSpace(s)
	}
	return out
}

// nameText is a CSL name as one string.
func nameText(n Name) string {
	if n.Literal != "" {
		return strings.TrimSpace(n.Literal)
	}
	return strings.TrimSpace(strings.TrimSpace(n.Given) + " " + strings.TrimSpace(n.Family))
}

// fold lowercases s and keeps only its letters and digits, for the key check.
func fold(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// identifyingKey names the field a confidential entry's key would reveal, or "".
// The key is printed by the guard and by cite-check, so it must not carry an alias,
// the title or one of its longer words, or an author's name.
func identifyingKey(e Entry) string {
	key := fold(e.ID)
	has := func(s string, min int) bool {
		f := fold(s)
		return len([]rune(f)) >= min && strings.Contains(key, f)
	}
	for _, a := range e.Custom.Aliases {
		if has(a, 3) {
			return "alias"
		}
	}
	if has(e.Title, 3) {
		return "title"
	}
	for _, w := range strings.Fields(e.Title) {
		if has(w, 5) {
			return "title"
		}
	}
	for _, n := range e.Author {
		for _, part := range []string{n.Family, n.Literal} {
			if has(part, 3) {
				return "author"
			}
		}
	}
	return ""
}

// copyFile copies a document into the corpus, creating the destination exclusively
// at 0600.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("%w: the document cannot be opened", ErrInvalidEntry)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, io.LimitReader(in, maxDocumentBytes)); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// Declassify is the visible move of a published confidential source: the folder
// moves confidential/<key> → public/<key> by `git mv`, the entry's confidential flag
// and permission_status follow, and the corpus commits both as one change. The next
// banlist sync drops the key's strings, and its ledger lines become flippable when
// the permission set here grants citation (citable, the default).
func Declassify(corpus, key, permission string) (AddResult, error) {
	c, err := loadCorpus(corpus)
	if err != nil {
		return AddResult{}, err
	}
	e, ok := c.Lookup(key)
	if !ok {
		return AddResult{}, fmt.Errorf("%w: %q", ErrUnknownSource, key)
	}
	if c.Class(key) != ClassConfidential || !e.Custom.Confidential {
		return AddResult{}, fmt.Errorf("%w: %q is not a confidential source in confidential/", ErrInvalidEntry, key)
	}
	if permission == "" {
		permission = PermissionCitable
	}
	if !validPermission(permission) {
		return AddResult{}, fmt.Errorf("%w: permission status %q is not one of %s", ErrInvalidEntry, permission, strings.Join(Permissions, ", "))
	}
	res := AddResult{Key: key, Class: ClassPublic, Permission: permission, Folder: ClassPublic + "/" + key, Files: []string{}}
	err = withLock(c.Dir, func() error {
		fresh, err := loadCorpus(c.Dir)
		if err != nil {
			return err
		}
		idx := -1
		for i, en := range fresh.Entries {
			if en.ID == key {
				idx = i
			}
		}
		if idx < 0 || fresh.Class(key) != ClassConfidential {
			return fmt.Errorf("%w: %q changed underneath the declassification", ErrInvalidEntry, key)
		}
		var m map[string]any
		if err := json.Unmarshal(fresh.raw[idx], &m); err != nil {
			return fmt.Errorf("%w: the entry does not read as a JSON object", ErrCorpusInvalid)
		}
		custom, _ := m["custom"].(map[string]any)
		if custom == nil {
			custom = map[string]any{}
		}
		custom["confidential"] = false
		custom["permission_status"] = permission
		m["custom"] = custom
		raw, err := json.Marshal(m)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(c.Dir, ClassPublic), 0o700); err != nil {
			return err
		}
		from, to := ClassConfidential+"/"+key, ClassPublic+"/"+key
		if _, err := corpusGit(c.Dir, "mv", "--", from, to); err != nil {
			return fmt.Errorf("%w: moving the folder failed (is it committed?): %v", ErrCorpusInvalid, err)
		}
		fresh.raw[idx] = raw
		if err := writeSources(c.Dir, fresh.raw); err != nil {
			return err
		}
		// git mv staged the rename already; the old path no longer exists to name.
		return commit(c.Dir, "source: declassify "+key+" ("+permission+")", SourcesFile, to)
	})
	if err != nil {
		return AddResult{}, err
	}
	return res, nil
}

// projectEntry is one confidential entry's patterns: title and aliases always,
// authors only under ban_authors.
func projectEntry(e Entry) ([]banlist.KeyedPattern, error) {
	var out []banlist.KeyedPattern
	add := func(suffix, field string, phrases ...string) error {
		p, err := banlist.PhrasePattern(phrases...)
		if err != nil {
			return fmt.Errorf("%w: %s of %q cannot be banned: %v", ErrInvalidEntry, field, e.ID, err)
		}
		out = append(out, banlist.KeyedPattern{Key: "sources/" + e.ID + "/" + suffix, Pattern: p})
		return nil
	}
	if err := add("title", "the title", e.Title); err != nil {
		return nil, err
	}
	for i, a := range e.Custom.Aliases {
		if err := add(fmt.Sprintf("alias-%d", i+1), fmt.Sprintf("alias %d", i+1), a); err != nil {
			return nil, err
		}
	}
	if e.Custom.BanAuthors {
		for i, n := range e.Author {
			phrases := []string{nameText(n)}
			if n.Literal == "" && n.Given != "" && n.Family != "" {
				phrases = append(phrases, strings.TrimSpace(n.Family)+", "+strings.TrimSpace(n.Given))
			}
			if err := add(fmt.Sprintf("author-%d", i+1), fmt.Sprintf("author %d", i+1), phrases...); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
