package lab

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// probeNameRe is a probe's name: it becomes a directory, so it is held to a
// narrow shape before it touches a path.
var probeNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// The files one probe record holds. record.md names the artefact the probe
// observed; the other five are the probe's input, command line, exit status and
// output, written before any harvest may cite it.
var probeFiles = []string{"input", "argv", "exit", "stdout", "stderr"}

// Probe is one probe record's scaffold.
type Probe struct {
	Lab      string   `json:"lab"`
	Name     string   `json:"name"`
	Dir      string   `json:"dir"`
	Files    []string `json:"files"`
	Artefact string   `json:"artefact"`
	Fill     string   `json:"fill"`
}

// Record scaffolds one probe record under the lab's state/probes/: the five
// capture files, empty, and a record.md naming the artefact the probe observes
// (the work binary's vintage and sha256, when there is one). It refuses on a
// halted lab: a lab halted by a gate records nothing more until the gate passes,
// which is what keeps a refusal from being adapted around.
func Record(repoRoot, id, name string) (Probe, error) {
	if !probeNameRe.MatchString(name) {
		return Probe{}, fmt.Errorf("%w: %q is not a probe name (lower-case letters, digits and hyphens, at most 64)", ErrRefused, clip(name))
	}
	l, err := open(repoRoot, id)
	if err != nil {
		return Probe{}, err
	}
	defer l.close()
	if gates := l.haltedGates(); len(gates) > 0 {
		h, _ := l.readHalt(gates[0])
		return Probe{}, fmt.Errorf("%w: the lab is halted by its %s (finding %s); a halted lab records nothing more until that gate passes again", ErrHalted, strings.Join(gates, " and "), h.Finding)
	}
	dir := probesDir + "/" + name
	if err := l.root.Mkdir(dir, storeDirPerm); err != nil {
		if errors.Is(err, os.ErrExist) {
			return Probe{}, fmt.Errorf("%w: the probe %s is already recorded; a probe record is never overwritten, so name the re-run as a new probe", ErrRefused, name)
		}
		return Probe{}, fmt.Errorf("cannot scaffold the probe: %v", redact(err, l.store.home))
	}
	artefact := "none: bin/abcd is absent"
	if fi, err := l.root.Lstat(workBinary); err == nil && fi.Mode().IsRegular() {
		if cur, hash, err := l.binaryIdentity(workBinary); err == nil {
			rev := cur.Revision
			if rev == "" {
				rev = "unknown"
			}
			artefact = "bin/abcd vintage " + rev + " sha256 " + hash
		}
	}
	p := Probe{Lab: id, Name: name, Dir: l.display(dir), Files: []string{}, Artefact: artefact}
	for _, f := range probeFiles {
		if err := fsutil.CreateExclusiveIn(l.root, dir+"/"+f, nil, fileMode); err != nil {
			return Probe{}, fmt.Errorf("cannot scaffold %s: %v", f, redact(err, l.store.home))
		}
		p.Files = append(p.Files, l.display(dir+"/"+f))
	}
	rec := "---\nprobe: " + name + "\nlab: " + id + "\ncreated: " + stamp() + "\nartefact: " + artefact + "\n---\n\n" +
		"# Probe " + name + "\n\n" +
		"## Observation\n\n_What the output shows, stated so the files beside this one can refute it._\n\n" +
		"Private-tier input is never stored: write a redaction note into `input` instead —\n" +
		"shape, size, hash and the command line, marked `redacted-private`.\n"
	if err := fsutil.CreateExclusiveIn(l.root, dir+"/record.md", []byte(rec), fileMode); err != nil {
		return Probe{}, fmt.Errorf("cannot scaffold record.md: %v", redact(err, l.store.home))
	}
	p.Files = append(p.Files, l.display(dir+"/record.md"))
	p.Fill = "printf '%s\\n' \"<command>\" > argv; <command> < input > stdout 2> stderr; echo $? > exit"
	return p, nil
}

// probeState is one probe record as the harvest judges it.
type probeState struct {
	Name     string
	Complete bool
	Exit     int
	Missing  []string
}

// probeNames lists the recorded probes, sorted.
func (l *lab) probeNames() []string {
	f, err := l.root.Open(probesDir)
	if err != nil {
		return nil
	}
	defer f.Close()
	ents, err := f.ReadDir(-1)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() && probeNameRe.MatchString(e.Name()) {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// probe reads one probe record's completeness: argv and exit written, the exit
// a number, and every capture file present.
func (l *lab) probe(name string) probeState {
	ps := probeState{Name: name}
	if !probeNameRe.MatchString(name) {
		ps.Missing = []string{"the record"}
		return ps
	}
	dir := probesDir + "/" + name
	if fi, err := l.root.Lstat(dir); err != nil || !fi.IsDir() {
		ps.Missing = []string{"the record"}
		return ps
	}
	for _, f := range append(append([]string{}, probeFiles...), "record.md") {
		if fi, err := l.root.Lstat(dir + "/" + f); err != nil || !fi.Mode().IsRegular() {
			ps.Missing = append(ps.Missing, f)
		}
	}
	argv, _ := fsutil.ReadGuardedInRoot(l.root, dir+"/argv", 64<<10)
	if strings.TrimSpace(string(argv)) == "" {
		ps.Missing = append(ps.Missing, "argv (empty)")
	}
	exit, _ := fsutil.ReadGuardedInRoot(l.root, dir+"/exit", 64)
	code, err := strconv.Atoi(strings.TrimSpace(string(exit)))
	if err != nil {
		ps.Missing = append(ps.Missing, "exit (not a status)")
	}
	ps.Exit = code
	ps.Complete = len(ps.Missing) == 0
	return ps
}
