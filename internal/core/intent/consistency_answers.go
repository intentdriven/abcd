package intent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// consistency_answers.go keeps the answer a provider returned for one chunk of
// an overflowing consistency emit (itd-2609291923559186, A4), so an answer
// already paid for is not asked for again when a later send falls back to the
// host. Each answer sits under the local tier beside its request, keyed by the
// chunk's receipt, which binds the corpus, the limit and the chunk's place: a
// corpus that moved, or another limit, plans other receipts and never meets an
// answer kept for the old ones.
//
// The answer file is the payload exactly as the provider returned it, so a
// host ingest takes it as one of the set's --findings-json files unchanged;
// the route file beside it is the dispatch receipt of the call that produced
// it, opaque here, so a run that reuses the answer reports the call honestly.
// A kept answer is untrusted like any payload: it is reused only when it
// validates against its request and the corpus as they stand, and the ingest
// validates it again.

// ConsistencyAnswerPath is where chunk receipt rcp's kept answer sits,
// repo-relative.
func ConsistencyAnswerPath(rcp string) string {
	return filepath.ToSlash(filepath.Join(reviewsRelDir, rcp+".findings.json"))
}

func consistencyAnswerRoutePath(rcp string) string {
	return filepath.ToSlash(filepath.Join(reviewsRelDir, rcp+".route.json"))
}

// KeepConsistencyAnswer writes the payload a provider returned for chunk
// receipt rcp, and the receipt of the call (route, opaque JSON), under the
// local tier, the route first: an answer on disk always has its route beside
// it. It returns the answer's repo-relative path.
func KeepConsistencyAnswer(repoRoot, rcp string, payload, route []byte) (string, error) {
	if !rcpIDRe.MatchString(rcp) {
		return "", fmt.Errorf("intent: %q is not a receipt to keep an answer under", rcp)
	}
	if err := ensureRecordDir(repoRoot, reviewsRelDir); err != nil {
		return "", err
	}
	rel := ConsistencyAnswerPath(rcp)
	if err := fsutil.WriteFileAtomic(filepath.Join(repoRoot, filepath.FromSlash(consistencyAnswerRoutePath(rcp))), route, 0o644); err != nil {
		return "", fmt.Errorf("intent: keeping the route of %s's answer: %w", rcp, err)
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(repoRoot, filepath.FromSlash(rel)), payload, 0o644); err != nil {
		return "", fmt.Errorf("intent: keeping %s's answer at %s: %w", rcp, rel, err)
	}
	return rel, nil
}

// KeptConsistencyAnswers reads the answers kept for one emit's chunks, through
// one validation cache, so the corpus is assembled once however many it reads.
type KeptConsistencyAnswers struct {
	repoRoot string
	cache    *consistencyIngestCache
}

// NewKeptConsistencyAnswers reads kept answers under repoRoot.
func NewKeptConsistencyAnswers(repoRoot string) *KeptConsistencyAnswers {
	return &KeptConsistencyAnswers{repoRoot: repoRoot, cache: newConsistencyIngestCache()}
}

// Get returns chunk receipt rcp's kept answer and its route when both are on
// disk and the answer validates against its request and the corpus as they
// stand. ok is false with a nil error when nothing is kept; a kept answer that
// cannot be read or does not validate is ok false with the reason, and the
// caller sends the chunk again.
func (k *KeptConsistencyAnswers) Get(rcp string) (payload, route []byte, ok bool, err error) {
	path := filepath.Join(k.repoRoot, filepath.FromSlash(ConsistencyAnswerPath(rcp)))
	if _, err := os.Lstat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, nil, false, nil
	}
	payload, err = readVerdictFile(path)
	if err != nil {
		return nil, nil, false, err
	}
	route, err = readVerdictFile(filepath.Join(k.repoRoot, filepath.FromSlash(consistencyAnswerRoutePath(rcp))))
	if err != nil {
		return nil, nil, false, err
	}
	rv, err := validateConsistency(k.repoRoot, payload, k.cache)
	if err != nil {
		return nil, nil, false, err
	}
	if rv.ReceiptID != rcp {
		return nil, nil, false, fmt.Errorf("intent: the answer kept for %s answers %s", rcp, rv.ReceiptID)
	}
	return payload, route, true, nil
}
