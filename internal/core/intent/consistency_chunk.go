package intent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
)

// consistency_chunk.go — A4 of itd-2609291923559186: the emit measures the
// assembled corpus against the request limit, and a corpus larger than it is
// reviewed in chunks.
//
// The measure is the byte length of the corpus file the reviewer would read
// (consistencyCorpusText for the scope), so there is no tokenizer and no
// estimate. A corpus within the limit is emitted as it always was: one request,
// one corpus file, and nothing about an overflow anywhere.
//
// An overflowing corpus is planned deterministically, from the corpus and the
// limit alone, so the ingest recomputes the very chunk a payload answers:
//
//   - The whole-corpus run groups the documents in path order, filling each
//     group up to half of what a chunk leaves once its header is counted, and
//     writes one chunk for every pair of groups (one chunk when there is one
//     group). Every pair of documents therefore shares at least one chunk, and
//     a contradiction between two documents is not lost at a chunk boundary.
//   - A scoped run pins its scope's documents into every chunk and pairs them
//     with each group of the rest, the groups sized to what the scope leaves
//     free: one chunk per group. A scope that cannot fit a chunk by itself is
//     refused.
//   - A document larger than a group may hold is reviewed in no chunk, and the
//     emit and the report name it, with its size.
//
// Every pair of documents is owned by one chunk, which alone reports it
// (owner), so one ingest never files one contradiction twice; a chunk's
// prompt names the sets whose pairs it leaves to another chunk.
//
// Each chunk is an ordinary request and corpus pair. Its manifest lists only
// its own documents, so the rubric's rule that each end is a manifest document
// confines a finding to the chunk with no change to the rubric. Its prompt
// names the chunk, the emit's receipt (the unchunked corpus's) and the limit,
// so prompt_hash binds all three.

// consistencyChunkInfo marks an assembled corpus as one chunk of an
// overflowing emit.
type consistencyChunkInfo struct {
	Index, Count int
	EmitReceipt  string // the unchunked corpus's receipt
	LimitBytes   int
	WholeDocs    int        // documents in the whole corpus
	NotOwned     [][]string // sets whose pairs among themselves another chunk owns
}

// consistencyPlan is the chunk plan for one overflowing corpus.
type consistencyPlan struct {
	CorpusBytes int
	Limit       int
	Groups      int
	Chunks      []consistencyCorpus // each with Chunk set, in index order
	Receipts    []string            // Chunks[i]'s receipt
	Unreviewed  []ConsistencyUnreviewed
	Owners      consistencyOwners
}

// consistencyOwners is each reviewed document's group in a plan, -1 for a
// pinned scope document, from which the chunk owning a pair follows.
type consistencyOwners struct {
	Group  map[string]int
	Groups int
	Scoped bool
}

// owner is the 1-based chunk that owns the pair (a, b): a cross-group pair's
// only chunk; group i's own pairs chunk (i, i+1), the last group's chunk
// (g-2, g-1); in a scoped run, the scope's own pairs chunk 1 and the rest
// their group's chunk. 0 is a pair no chunk holds.
func (o consistencyOwners) owner(a, b string) int {
	ga, okA := o.Group[a]
	gb, okB := o.Group[b]
	switch {
	case !okA || !okB:
		return 0
	case o.Scoped && ga < 0 && gb < 0:
		return 1
	case o.Scoped && (ga < 0 || gb < 0 || ga == gb):
		return max(ga, gb) + 1
	case o.Scoped:
		return 0
	case o.Groups == 1:
		return 1
	}
	i, j := min(ga, gb), max(ga, gb)
	if i == j && i < o.Groups-1 {
		j = i + 1
	} else if i == j {
		i = o.Groups - 2
	}
	return i*(2*o.Groups-i-1)/2 + j - i
}

// overflow is the plan as the emit and the report state it.
func (p consistencyPlan) overflow() *ConsistencyOverflow {
	return &ConsistencyOverflow{
		CorpusBytes: p.CorpusBytes, LimitBytes: p.Limit, Groups: p.Groups, Chunks: len(p.Chunks),
		Unreviewed: append([]ConsistencyUnreviewed{}, p.Unreviewed...),
	}
}

// chunkList names every chunk's receipt, request, corpus and size.
func (p consistencyPlan) chunkList() []ConsistencyChunk {
	out := make([]ConsistencyChunk, len(p.Chunks))
	for i, ch := range p.Chunks {
		rcp := p.Receipts[i]
		out[i] = ConsistencyChunk{
			Index: i + 1, ReceiptID: rcp, Documents: len(ch.Docs),
			RequestPath: filepath.ToSlash(filepath.Join(reviewsRelDir, rcp+".request.md")),
			CorpusPath:  filepath.ToSlash(filepath.Join(reviewsRelDir, rcp+".corpus.md")),
		}
	}
	return out
}

// consistencyChunkReceipt is chunk i of n's receipt: derived from the scope,
// the whole corpus's digest, the request limit and the chunk's place, so a
// re-emit over an unchanged corpus and limit reuses it, and one at another
// limit never overwrites a chunk set in flight, even when it plans as many
// chunks.
func consistencyChunkReceipt(scope, corpusDigest string, limit, i, n int) string {
	h := sha256.Sum256([]byte("consistency|" + scope + "|" + corpusDigest + "|limit " + strconv.Itoa(limit) +
		"|chunk " + strconv.Itoa(i) + " of " + strconv.Itoa(n)))
	return "rcp-" + hex.EncodeToString(h[:])[:12]
}

// corpusDigest is the digest over every document's path and digest, the one
// assembleConsistency states for the whole corpus and a chunk states for its
// own documents.
func corpusDigest(docs []consistencyDoc) string {
	h := sha256.New()
	h.Write([]byte("abcd/intent-consistency-corpus/v1\n"))
	for _, d := range docs {
		fmt.Fprintf(h, "%s\x00%s\n", d.Path, d.Digest)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// framedBytes is what a document takes in a chunk's corpus file: its manifest
// line, numbered as wide as the whole corpus could number it, its BEGIN and
// END lines and its text.
func framedBytes(d consistencyDoc, widestIndex int) int {
	return len(manifestLine(widestIndex, d)) + len(documentFrame(d))
}

// planConsistencyChunks plans the chunks of corpus c, whose unchunked corpus
// file (receipt emitRcp) is corpusBytes long and over limit. Every number it
// sizes against is an upper bound on what the chunk renders, and the emit
// checks each rendered chunk against the limit all the same.
func planConsistencyChunks(c consistencyCorpus, emitRcp string, corpusBytes, limit int) (consistencyPlan, error) {
	n := len(c.Docs)
	// A chunk's header, with every number in it as wide as it can be: no chunk
	// holds more documents than the corpus, and no plan has more chunks than
	// the corpus has pairs of documents.
	wide := n*n + 1
	overhead := len(consistencyCorpusHeader(emitRcp, n, n, n, c.Digest, &consistencyChunkInfo{
		Index: wide, Count: wide, EmitReceipt: emitRcp, LimitBytes: limit, WholeDocs: n,
	}))
	pinned := map[string]bool{}
	if c.ScopePath != "" {
		pinned[c.ScopePath] = true
	}
	var pins, rest []consistencyDoc
	pinBytes := 0
	for _, d := range c.Docs {
		if pinned[d.Path] {
			pins = append(pins, d)
			pinBytes += framedBytes(d, n)
		} else {
			rest = append(rest, d)
		}
	}
	var budget int
	if len(pins) > 0 {
		budget = limit - overhead - pinBytes
		if budget <= 0 {
			return consistencyPlan{}, fmt.Errorf("intent: the scope %s is %d bytes as the corpus frames it, and beside a chunk's %d-byte header it leaves no room "+
				"under the request limit of %d bytes for any other document; raise consistency.max_request_bytes (at most %d)",
				c.Scope, pinBytes, overhead, limit, MaxConsistencyRequestBytes)
		}
	} else {
		budget = (limit - overhead) / 2
	}
	p := consistencyPlan{CorpusBytes: corpusBytes, Limit: limit, Unreviewed: []ConsistencyUnreviewed{}}
	if budget <= 0 {
		return consistencyPlan{}, fmt.Errorf("intent: the request limit of %d bytes leaves no room for a document beside a chunk's %d-byte header; "+
			"raise consistency.max_request_bytes", limit, overhead)
	}
	var groups [][]consistencyDoc
	fill := budget + 1 // no open group yet
	for _, d := range rest {
		size := framedBytes(d, n)
		if size > budget {
			p.Unreviewed = append(p.Unreviewed, ConsistencyUnreviewed{Path: d.Path, Bytes: size})
			continue
		}
		if fill+size > budget {
			groups = append(groups, nil)
			fill = 0
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], d)
		fill += size
	}
	if len(groups) == 0 {
		return consistencyPlan{}, fmt.Errorf("intent: no document of the corpus fits a chunk under the request limit of %d bytes "+
			"(each is larger than the %d bytes a chunk leaves it); raise consistency.max_request_bytes", limit, budget)
	}
	p.Groups = len(groups)
	p.Owners = consistencyOwners{Group: map[string]int{}, Groups: len(groups), Scoped: len(pins) > 0}
	for _, d := range pins {
		p.Owners.Group[d.Path] = -1
	}
	for gi, g := range groups {
		for _, d := range g {
			p.Owners.Group[d.Path] = gi
		}
	}

	// parts is each chunk's sets of documents: the pins, then its groups.
	var parts [][][]consistencyDoc
	switch {
	case len(pins) > 0:
		for _, g := range groups {
			parts = append(parts, [][]consistencyDoc{pins, g})
		}
	case len(groups) == 1:
		parts = append(parts, [][]consistencyDoc{groups[0]})
	default:
		for i := range groups {
			for j := i + 1; j < len(groups); j++ {
				parts = append(parts, [][]consistencyDoc{groups[i], groups[j]})
			}
		}
	}
	for i, part := range parts {
		var docs []consistencyDoc
		var notOwned [][]string
		for _, set := range part {
			docs = append(docs, set...)
			if p.Owners.owner(set[0].Path, set[0].Path) != i+1 {
				var paths []string
				for _, d := range set {
					paths = append(paths, d.Path)
				}
				notOwned = append(notOwned, paths)
			}
		}
		sort.Slice(docs, func(a, b int) bool { return docs[a].Path < docs[b].Path })
		ch := consistencyCorpus{
			Scope: c.Scope, ScopePath: c.ScopePath, Docs: docs, Digest: corpusDigest(docs),
			Chunk: &consistencyChunkInfo{Index: i + 1, Count: len(parts), EmitReceipt: emitRcp, LimitBytes: limit, WholeDocs: n, NotOwned: notOwned},
		}
		for _, d := range docs {
			if d.IntentID == "" {
				ch.Brief++
			} else {
				ch.Intents++
			}
		}
		p.Chunks = append(p.Chunks, ch)
		p.Receipts = append(p.Receipts, consistencyChunkReceipt(c.Scope, c.Digest, limit, i+1, len(parts)))
	}
	return p, nil
}
