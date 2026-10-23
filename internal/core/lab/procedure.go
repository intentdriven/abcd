package lab

// procedure.go is the seam a lab procedure drives a lab through
// (itd-2610161733316518 decision 4): a run that is a lab's own work, such as a
// model-matrix run, opens the lab once, is refused while any of the lab's gates
// holds it halted, and writes its records only under the lab's state/matrix/,
// each file once. It never reaches the lab's other documents, so the
// procedure's records cannot rewrite a preflight, a probe or a finding.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// procedureDir is the one directory of a lab home a procedure writes under.
const procedureDir = stateDir + "/matrix"

// maxStateBytes bounds one procedure record read back.
const maxStateBytes = maxDocBytes

// stateSegRe is one segment of a procedure record's path: it becomes a
// directory or file name, so it is held to a narrow shape first.
var stateSegRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// Procedure is one lab opened by a procedure that drives it.
type Procedure struct {
	l *lab
}

// OpenProcedure opens lab id of the repository at repoRoot for a procedure. It
// refuses a lab a gate holds halted (ErrHalted), the same refusal Record gives,
// and a lab whose preflight has not passed (ErrRefused): a procedure runs only
// in a lab whose isolation was proved.
func OpenProcedure(repoRoot, id string) (*Procedure, error) {
	l, err := open(repoRoot, id)
	if err != nil {
		return nil, err
	}
	p := &Procedure{l: l}
	if err := p.admit(); err != nil {
		l.close()
		return nil, err
	}
	data, err := fsutil.ReadGuardedInRoot(l.root, preflightJSON, maxDocBytes)
	var pf Preflighted
	if err != nil || json.Unmarshal(data, &pf) != nil || !pf.Passed {
		l.close()
		return nil, fmt.Errorf("%w: the lab %s has no passing preflight; run `abcd lab preflight %s` first", ErrRefused, id, id)
	}
	return p, nil
}

// admit refuses while a gate holds the lab halted.
func (p *Procedure) admit() error {
	if gates := p.l.haltedGates(); len(gates) > 0 {
		h, _ := p.l.readHalt(gates[0])
		return fmt.Errorf("%w: the lab is halted by its %s (finding %s); a halted lab records nothing more until that gate passes again", ErrHalted, strings.Join(gates, " and "), h.Finding)
	}
	return nil
}

// Close releases the lab.
func (p *Procedure) Close() { p.l.close() }

// Entry is the lab's registry line: its id, root commit and pin.
func (p *Procedure) Entry() Entry { return p.l.entry }

// WorkBinary is the work binary's identity as the lab's first passing
// preflight pinned it; ok is false where none was pinned (a lab of a
// repository that is not abcd's own has none).
func (p *Procedure) WorkBinary() (sha string, revision string, ok bool) {
	data, err := fsutil.ReadGuardedInRoot(p.l.root, workBinPinName, 64<<10)
	if err != nil {
		return "", "", false
	}
	var pin workBinPin
	if json.Unmarshal(data, &pin) != nil || pin.SHA256 == "" {
		return "", "", false
	}
	return pin.SHA256, pin.Revision, true
}

// stateRel holds rel to a procedure record's shape and returns its path in the
// lab home.
func stateRel(rel string) (string, error) {
	if rel == "" || strings.Contains(rel, `\`) || path.Clean(rel) != rel {
		return "", fmt.Errorf("%w: %q is not a procedure record path", ErrRefused, clip(rel))
	}
	for _, seg := range strings.Split(rel, "/") {
		if !stateSegRe.MatchString(seg) || strings.Contains(seg, "..") {
			return "", fmt.Errorf("%w: %q is not a procedure record path (segments of letters, digits, '.', '_' and '-')", ErrRefused, clip(rel))
		}
	}
	return procedureDir + "/" + rel, nil
}

// WriteState writes one procedure record under the lab's state/matrix/,
// creating its directories, and never replaces one already written.
func (p *Procedure) WriteState(rel string, data []byte) error {
	full, err := stateRel(rel)
	if err != nil {
		return err
	}
	if err := p.admit(); err != nil {
		return err
	}
	if err := p.l.root.MkdirAll(path.Dir(full), storeDirPerm); err != nil {
		return fmt.Errorf("cannot create %s: %v", p.l.display(path.Dir(full)), redact(err, p.l.store.home))
	}
	if err := fsutil.CreateExclusiveIn(p.l.root, full, data, fileMode); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %s is already written; a procedure record is never overwritten", ErrRefused, p.l.display(full))
		}
		return fmt.Errorf("cannot write %s: %v", p.l.display(full), redact(err, p.l.store.home))
	}
	return nil
}

// ReadState reads one procedure record back.
func (p *Procedure) ReadState(rel string) ([]byte, error) {
	full, err := stateRel(rel)
	if err != nil {
		return nil, err
	}
	data, err := fsutil.ReadGuardedInRoot(p.l.root, full, maxStateBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s is not written: %w", p.l.display(full), os.ErrNotExist)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %v", p.l.display(full), redact(err, p.l.store.home))
	}
	return data, nil
}

// BinaryIdentity is a binary's build vintage and its sha256, read without
// running it. The preflight pins a lab's work binary through it, and a
// procedure pins the binary that drives it the same way.
func BinaryIdentity(f interface {
	io.Reader
	io.ReaderAt
}) (vintage.Current, string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return vintage.Current{}, "", err
	}
	cur, err := vintageOf(f)
	return cur, hex.EncodeToString(h.Sum(nil)), err
}
