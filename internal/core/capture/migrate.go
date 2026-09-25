package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The promote join's retired spellings and their successors (itd-4 AC3). The
// ledger half's pair lives in issueschema.Retired, where the reader and the
// record gate read it; the intent half's in core/intent, beside the key it
// replaced. They are named once more here only as the two ends of one rename.
const (
	retiredForwardKey = "promoted_to"
	forwardKey        = "related_intents"
)

// MigrateRequest is the input to Migrate. Apply false reports what would change
// and writes nothing.
type MigrateRequest struct {
	RepoRoot   string
	IssuesRoot string
	Apply      bool
}

// MigrateChange is one record the migration rewrites (or would rewrite).
type MigrateChange struct {
	ID   string `json:"id"`
	Path string `json:"path"` // repo-relative
	// Retired is the retired line the record carries, `key: value`, or empty
	// when the record carries none and changes only because the OTHER end of a
	// join it belongs to did.
	Retired string `json:"retired,omitempty"`
	// Key is the successor list the record carries after the change, and List
	// its full value then.
	Key  string   `json:"key"`
	List []string `json:"list"`
}

// MigrateResult reports a migration run.
type MigrateResult struct {
	Applied bool `json:"applied"`
	// Scanned counts the ledger records and intents read.
	Scanned int             `json:"scanned"`
	Changes []MigrateChange `json:"changes"`
	// Notes are joins the migration completed from one end only because the
	// other end names a record this tree does not hold. The drift check reports
	// the same joins as dangling; the migration writes what it can.
	Notes []string `json:"notes,omitempty"`
}

// migrateRecord is one record the migration read: where it is, which list key
// it carries the join in, that list, and the retired value if any.
type migrateRecord struct {
	id, abs, rel  string
	content       string
	key, retired  string
	list          []string
	retiredValue  string
	intentHalf    bool
	additions     []string
	retiredIsHead bool // intent half: the retired source goes FIRST in the list
}

// Migrate rewrites the promote join's retired back-links into the names itd-4
// AC3 gives them, across the issue ledger, the readings store and the intent
// store: `promoted_to: itd-M` on a ledger record becomes a member of its
// `related_intents`, and `promoted_from: <id>` on an intent becomes the first
// member of its `related_issues`.
//
// "Promoted" is the two-sided PAIR under the new names (see promotedInto), where
// the old scheme let either end say it alone. So a join an older abcd wrote from
// one end only — a link-mode promote of a hand-filed draft stamped the ledger
// half alone — is completed from the other end as well, or the migrated tree
// would stop saying the record was promoted at all.
//
// Nothing else moves: a loose `related_intents` entry is kept where it is, and a
// record carrying no retired key and joined to nothing that did is left
// byte-identical. The run is idempotent. It holds the ledger lock for the whole
// apply, so no verb writes the ledger between the read and the rewrite.
func Migrate(req MigrateRequest) (MigrateResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return MigrateResult{}, err
	}
	var res MigrateResult
	run := func() error {
		records, err := migrateScan(repoRoot, issuesRoot)
		if err != nil {
			return err
		}
		res.Scanned = len(records)
		res.Changes, res.Notes = migratePlan(records)
		if !req.Apply {
			return nil
		}
		for _, r := range records {
			if r.retired == "" && len(r.additions) == 0 {
				continue
			}
			updated, err := migrateRewrite(r)
			if err != nil {
				return fmt.Errorf("migrate %s: %w", r.rel, err)
			}
			// Resolved inside an os.Root (iss-2609012037143368): the ledger's own
			// containment base for a ledger record, the checkout for an intent.
			base := ledgerBase(repoRoot, issuesRoot)
			if !fsutil.PathWithin(r.abs, issuesRoot, false) {
				base = repoRoot
			}
			if err := writeContained(base, r.abs, []byte(updated)); err != nil {
				return fmt.Errorf("migrate %s: %w", r.rel, err)
			}
		}
		res.Applied = true
		return nil
	}
	if req.Apply {
		if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
			return MigrateResult{}, err
		}
		err = withLedgerLock(repoRoot, issuesRoot, run)
	} else {
		err = run()
	}
	if err != nil {
		return MigrateResult{}, err
	}
	if res.Changes == nil {
		res.Changes = []MigrateChange{}
	}
	return res, nil
}

// migrateScan reads every record that can carry either half of the join. It
// reads RAW frontmatter rather than through the ledger reader, because the
// reader refuses exactly the records this exists to repair.
func migrateScan(repoRoot, issuesRoot string) ([]*migrateRecord, error) {
	var out []*migrateRecord
	ledgerRecord := func(abs, prefix string) error {
		content, err := readRecordGuarded(abs)
		if err != nil {
			return err
		}
		fm, _, err := parseFrontmatterAndBody(content)
		if err != nil {
			return fmt.Errorf("%s: %w", fsutil.RepoRel(repoRoot, abs), err)
		}
		id := asString(fm["id"])
		if !strings.HasPrefix(id, prefix+"-") {
			return nil
		}
		r := &migrateRecord{
			id: id, abs: abs, rel: fsutil.RepoRel(repoRoot, abs), content: content,
			key: forwardKey, list: asStrList(fm[forwardKey]),
		}
		if v, ok := fm[retiredForwardKey]; ok {
			r.retired, r.retiredValue = retiredForwardKey, strings.TrimSpace(asString(v))
		}
		out = append(out, r)
		return nil
	}
	for _, dir := range issueschema.StatusDirs {
		paths, err := mdFiles(filepath.Join(issuesRoot, dir))
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			if err := ledgerRecord(p, "iss"); err != nil {
				return nil, err
			}
		}
	}
	readingsRoot := filepath.Join(issuesRoot, issueschema.ReadingsDir)
	runs, err := os.ReadDir(readingsRoot)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	for _, e := range runs {
		if !e.IsDir() {
			continue
		}
		paths, err := mdFiles(filepath.Join(readingsRoot, e.Name()))
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			if err := ledgerRecord(p, issueschema.ReadingItemFamily); err != nil {
				return nil, err
			}
		}
	}
	for _, rel := range intentStoreRelDirs() {
		paths, err := mdFiles(filepath.Join(repoRoot, rel))
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			content, err := readRecordGuarded(p)
			if err != nil {
				return nil, err
			}
			fields := frontmatter.Fields(strings.Split(content, "\n"))
			id := fields["id"].Value
			if !reItdID.MatchString(id) {
				continue
			}
			r := &migrateRecord{
				id: id, abs: p, rel: fsutil.RepoRel(repoRoot, p), content: content,
				key: intent.RelatedIssuesKey, intentHalf: true,
			}
			if f, ok := fields[intent.RelatedIssuesKey]; ok && !frontmatter.IsNull(f.Value) {
				r.list = frontmatter.StringList(f.Value)
			}
			if f, ok := fields[intent.RetiredRelatedIssuesKey]; ok {
				r.retired, r.retiredValue = intent.RetiredRelatedIssuesKey, strings.Trim(f.Value, `"'`)
			}
			out = append(out, r)
		}
	}
	return out, nil
}

// migratePlan works out each record's additions: the retired value on the
// record itself, and the other end of every join a retired key on some OTHER
// record declares.
func migratePlan(records []*migrateRecord) ([]MigrateChange, []string) {
	byID := map[string]*migrateRecord{}
	for _, r := range records {
		byID[r.id] = r
	}
	var notes []string
	add := func(r *migrateRecord, id string) {
		for _, have := range append(append([]string{}, r.list...), r.additions...) {
			if have == id {
				return
			}
		}
		r.additions = append(r.additions, id)
	}
	for _, r := range records {
		if r.retired == "" || r.retiredValue == "" || frontmatter.IsNull(r.retiredValue) {
			continue
		}
		if r.intentHalf {
			r.retiredIsHead = true
		}
		add(r, r.retiredValue)
		other, ok := byID[r.retiredValue]
		if !ok {
			notes = append(notes, fmt.Sprintf("%s names %s in `%s`, which this tree does not hold; only %s's half was written",
				r.id, r.retiredValue, r.retired, r.id))
			continue
		}
		add(other, r.id)
	}
	var changes []MigrateChange
	for _, r := range records {
		if r.retired == "" && len(r.additions) == 0 {
			continue
		}
		c := MigrateChange{ID: r.id, Path: r.rel, Key: r.key, List: r.finalList()}
		if r.retired != "" {
			c.Retired = r.retired + ": " + r.retiredValue
		}
		changes = append(changes, c)
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	sort.Strings(notes)
	return changes, notes
}

// finalList is the successor list the record carries after the migration. On
// the intent half the retired source goes FIRST: it is the record the intent
// was promoted from, and the first entry is where that record sits.
func (r *migrateRecord) finalList() []string {
	if !r.retiredIsHead {
		return append(append([]string{}, r.list...), r.additions...)
	}
	out := []string{r.retiredValue}
	for _, id := range append(append([]string{}, r.list...), r.additions...) {
		if id != r.retiredValue {
			out = append(out, id)
		}
	}
	return out
}

// migrateRewrite renders a record's new bytes: the retired line becomes the
// successor list in place when the record carries no successor line yet (so a
// diff shows a rename, not a move), and otherwise the successor line is
// rewritten and the retired line dropped. Every other byte is preserved.
func migrateRewrite(r *migrateRecord) (string, error) {
	content := r.content
	if r.retired != "" && !hasTopLevelKey(content, r.key) {
		renamed, err := renameTopLevelKey(content, r.retired, r.key)
		if err != nil {
			return "", err
		}
		content = renamed
	}
	content, err := setListField(content, r.key, r.finalList())
	if err != nil {
		return "", err
	}
	if r.retired != "" && hasTopLevelKey(content, r.retired) {
		if content, err = setListField(content, r.retired, nil); err != nil {
			return "", err
		}
	}
	return content, nil
}

// hasTopLevelKey reports whether content's frontmatter carries key at column 0.
func hasTopLevelKey(content, key string) bool {
	_, ok := frontmatter.Fields(strings.Split(content, "\n"))[key]
	return ok
}

// renameTopLevelKey rewrites the key of content's top-level frontmatter line
// `old:` to `new:`, leaving its value and every other byte as they were.
func renameTopLevelKey(content, old, new string) (string, error) {
	lines := splitKeepEnds(content)
	openIdx, closeIdx, err := frontmatterBounds(lines)
	if err != nil {
		return "", err
	}
	for k := openIdx + 1; k < closeIdx; k++ {
		if strings.HasPrefix(lines[k], old+":") {
			lines[k] = new + ":" + strings.TrimPrefix(lines[k], old+":")
			return strings.Join(lines, ""), nil
		}
	}
	return content, nil
}

// mdFiles lists the markdown files directly in dir, sorted; an absent dir is
// no files.
func mdFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out, nil
}
