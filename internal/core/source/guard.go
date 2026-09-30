package source

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/banlist"
)

// BlockOwner names the generated block the corpus owns in a repository's private
// banlist, and heads every key in it: `sources/<key>/<field>`.
const BlockOwner = "sources"

// Projection is every confidential source's patterns, in bibliography order: the
// title and aliases always, the authors only where the entry opts in with
// ban_authors. A source is confidential by its FOLDER; a corpus where folder and
// entry disagree never reaches here (every caller requires consistency first).
func (c *Corpus) Projection() ([]banlist.KeyedPattern, error) {
	var out []banlist.KeyedPattern
	for _, e := range c.Entries {
		if c.Class(e.ID) != ClassConfidential {
			continue
		}
		pats, err := projectEntry(e)
		if err != nil {
			return nil, err
		}
		out = append(out, pats...)
	}
	return out, nil
}

// confidentialCount counts the sources under confidential/.
func (c *Corpus) confidentialCount() int {
	n := 0
	for _, e := range c.Entries {
		if c.Class(e.ID) == ClassConfidential {
			n++
		}
	}
	return n
}

// SyncResult is what a banlist sync did.
type SyncResult struct {
	// Sources counts the confidential sources projected.
	Sources int `json:"sources"`
	// Block is the private store's generated-block outcome.
	Block banlist.GeneratedResult `json:"block"`
}

// SyncOptions says how a banlist sync may treat the private store.
type SyncOptions struct {
	// Refresh is the pre-commit guard's mode: update a private store that already
	// exists and declares the keyed format, and never create one
	// (banlist.RefreshGeneratedBlock). Without it the sync is the person's by-hand
	// act, which creates the store when there is something to ban.
	Refresh bool
}

// SyncBanlist projects the corpus's confidential entries into repoRoot's untracked
// private banlist (the itd-74 private layer), as the generated block the corpus
// owns. Hand-written entries outside the block survive; a declassified source's
// strings leave it on the next sync. A corpus whose classes disagree is refused
// before anything is written, so the block already there keeps banning.
func SyncBanlist(corpus, repoRoot string, opts SyncOptions) (SyncResult, error) {
	c, err := loadCorpus(corpus)
	if err != nil {
		return SyncResult{}, err
	}
	if err := c.requireConsistent(); err != nil {
		return SyncResult{}, err
	}
	pats, err := c.Projection()
	if err != nil {
		return SyncResult{}, err
	}
	sync := banlist.SyncGeneratedBlock
	if opts.Refresh {
		sync = banlist.RefreshGeneratedBlock
	}
	block, err := sync(repoRoot, BlockOwner, pats)
	if err != nil {
		return SyncResult{}, err
	}
	return SyncResult{Sources: c.confidentialCount(), Block: block}, nil
}

// Finding is one place a scanned text names a confidential source: the source's
// key, which of its strings matched (title, alias-N, author-N), and where. It
// carries no matched text, so a report is safe to relay.
type Finding struct {
	Source string `json:"source"`
	Field  string `json:"field"`
	// Line is the 1-based line the match is on.
	Line int `json:"line"`
	// Offset is banlist.Hit's: the 0-based byte offset from the start of the WHOLE
	// scanned text (never a column in Line) to the start of the matched span, which
	// can be the one boundary byte before the phrase.
	Offset int `json:"offset"`
}

// CiteReport is a cite-check's outcome.
type CiteReport struct {
	// Sources counts the confidential sources checked against.
	Sources  int       `json:"sources"`
	Findings []Finding `json:"findings"`
}

// Clean reports whether the text named no confidential source.
func (r CiteReport) Clean() bool { return len(r.Findings) == 0 }

// CiteCheck scans text for every confidential source's projected strings through
// the private layer's matcher — the engine the pre-commit guard runs — and reports
// offenders by key, field and position only.
func CiteCheck(corpus string, text []byte) (CiteReport, error) {
	c, err := loadCorpus(corpus)
	if err != nil {
		return CiteReport{}, err
	}
	if err := c.requireConsistent(); err != nil {
		return CiteReport{}, err
	}
	pats, err := c.Projection()
	if err != nil {
		return CiteReport{}, err
	}
	hits, err := banlist.ScanText(pats, text)
	if err != nil {
		return CiteReport{}, err
	}
	rep := CiteReport{Sources: c.confidentialCount(), Findings: []Finding{}}
	for _, h := range hits {
		rest := strings.TrimPrefix(h.Key, BlockOwner+"/")
		key, field, _ := strings.Cut(rest, "/")
		rep.Findings = append(rep.Findings, Finding{Source: key, Field: field, Line: h.Line, Offset: h.Offset})
	}
	return rep, nil
}
