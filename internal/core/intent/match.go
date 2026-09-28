package intent

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/core/record/match"
)

// match.go is the intent store's half of the filing-time match
// (itd-2609212137116617): the text every intent offers as a candidate, and the
// quoted-text create that matches a new draft against the record before it is
// written. The score is internal/core/record/match's; the ledger gathers the
// candidate set, because the ledger reads intents and this store does not read
// the ledger.

// MatchText is one intent's comparable text: its H1 and its press release.
// A press release that is still the seed note a create wrote is left out, since
// every promoted draft carries the same words there.
type MatchText struct {
	ID           string
	Title        string
	PressRelease string
}

var (
	titleH1Re      = regexp.MustCompile(`^#\s+(.+?)\s*$`)
	pressReleaseRe = regexp.MustCompile(`^##\s+Press Release\s*$`)
	seedReduceRe   = regexp.MustCompile(`[>_*\s]+`)
)

// MatchTexts reads the title and press release of every intent in every
// bucket. A store no reader can load is an error, which the match reports as
// its reason for comparing nothing, never as a refusal of the write.
func MatchTexts(repoRoot string) ([]MatchText, error) {
	c, err := Load(repoRoot)
	if err != nil {
		return nil, err
	}
	out := make([]MatchText, 0, len(c.Intents))
	for _, it := range c.Intents {
		data, err := readRepoFile(filepath.Join(repoRoot, it.Path), it.Path)
		if err != nil {
			return nil, err
		}
		out = append(out, matchTextOf(it.ID, string(data)))
	}
	return out, nil
}

func matchTextOf(id, content string) MatchText {
	lines := strings.Split(content, "\n")
	// The body starts after the frontmatter's closing delimiter.
	start := 0
	if len(lines) > 0 && frontmatter.IsDelimiter(frontmatter.TrimBOM(lines[0])) {
		for i := 1; i < len(lines); i++ {
			if frontmatter.IsDelimiter(lines[i]) {
				start = i + 1
				break
			}
		}
	}
	body := lines[start:]
	mt := MatchText{ID: id}
	mask := mdrecord.Mask(body)
	for i, ln := range body {
		if mask[i] != 0 {
			continue
		}
		if m := titleH1Re.FindStringSubmatch(strings.TrimRight(ln, "\r")); m != nil {
			mt.Title = m[1]
			break
		}
	}
	if s, e, ok := mdrecord.SectionLineRangeIn(body, mask, pressReleaseRe); ok {
		press := strings.TrimSpace(strings.Join(body[s:e], "\n"))
		if !IsSeedNote(strings.TrimSpace(seedReduceRe.ReplaceAllString(press, " "))) {
			mt.PressRelease = press
		}
	}
	return mt
}

// Matcher is a create's request to be matched before it is written: the
// threshold, and the candidate set, which the caller's reader gathers and the
// create calls under the store's mint lock.
type Matcher struct {
	Threshold  float64
	Candidates func() ([]match.Candidate, error)
}

// Created is a quoted-text create's result: the draft, and the match's
// outcome when one was asked for.
type Created struct {
	Intent
	Match *match.Outcome `json:"match,omitempty"`
}

// CreateFromTextMatched is CreateFromText with the filing-time match: the new
// draft's title and press release are compared with the candidates under the
// mint lock, and each likely double is written onto the draft as a
// `duplicates:` or `refines:` link. The match never refuses the create: a
// candidate set that cannot be read is reported on the outcome and the draft
// is written unlinked. A nil matcher is CreateFromText exactly.
func CreateFromTextMatched(repoRoot, text string, opts TextOptions, m *Matcher) (Created, error) {
	return createFromText(repoRoot, text, opts, m)
}

// runMatch runs m over text, turning a candidate read that fails into an
// outcome that says so rather than an error.
func runMatch(m *Matcher, text string) *match.Outcome {
	if match.Short(text) {
		o := match.Rank(text, nil, m.Threshold)
		return &o
	}
	cands, err := m.Candidates()
	if err != nil {
		o := match.Unread(m.Threshold, err)
		return &o
	}
	o := match.Rank(text, cands, m.Threshold)
	return &o
}
