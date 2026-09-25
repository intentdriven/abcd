package history

// Ingest: bringing transcripts that are already on disk into the corpus.
//
// This is the recovery path for everything the hooks never saw — sessions that
// ran before capture existed, sessions whose stop event was cancelled, a
// machine whose store was rebuilt. It reads files nobody redacted and writes
// records through the unchanged, fail-closed Capture.
//
// TWO THINGS ARE LOAD-BEARING HERE, and both are about which repository owns a
// transcript.
//
// The DESTINATION is an operand, never derived from the working directory. The
// scanner is built from dest.RepoRoot and from nothing else, so a repository's
// own pii.json and gitleaks.json govern its own transcripts and can never be
// applied to another repository's. Getting that wrong is a privacy fault rather
// than a misfiling, and the working directory is exactly the wrong authority:
// an operator recovering a backlog is not standing in the repository the
// transcripts belong to. The destination is also a PAIR — a root and a store
// key — and Destination.verify proves the two name one repository, so the seam
// defends its own invariant rather than trusting a caller to derive both halves
// from a single detection (iss-2609091911060345).
//
// The OWNER is resolved from the cwd recorded INSIDE the transcript lines, and
// never by decoding the harness's project-directory name. That name is not
// reversible to a filesystem path — a directory named `foo-bar` and a path
// `foo/bar` mangle identically — so decoding it is a guess wearing a
// resolution's clothes.
//
// And the owner is resolved for the SESSION before it is resolved for the file.
// A sub-agent handed its own worktree records that worktree as its cwd, and the
// harness removes the worktree when the agent stops, so a per-file resolution
// orphans exactly the isolated implementation lanes — the transcripts worth the
// most — while their parent's cwd sits one file away, resolving cleanly. Every
// sub-agent transcript carries the spawning session's id, so the session is
// what gets placed, and the file's own cwd is only the fallback. "Orphan"
// therefore means the SESSION cannot be placed, not the file
// (iss-2609090723023943).

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// transcriptSuffix is the extension a directory walk accepts. An explicitly
// named file is taken as given: naming a file IS the operator's assertion that
// it is a transcript, and refusing it on its extension would be this package
// deciding it knows better.
const transcriptSuffix = ".jsonl"

// ingestDefaultDepth bounds a directory walk. Deep enough for a per-project,
// per-session, per-agent nesting with room to spare; bounded so a source
// pointed at a home directory cannot turn into a filesystem crawl.
const ingestDefaultDepth = 8

// Skip reasons. A skip is a decision, and each of these names which one.
const (
	SkipOwnedElsewhere = "owned-elsewhere"
	SkipAmbiguousOwner = "ambiguous-owner"
	SkipNoSession      = "no-session-id"
	SkipAmbiguousAgent = "ambiguous-agent-id"
)

// resolveRootSHA maps a recorded working directory to the root-commit SHA of
// the repository containing it. A session run in a worktree resolves to the
// repository the worktree derives from, because they share a root commit.
//
// It is a package var for the same reason scanGitleaks is: so a test can
// substitute a table for a set of real git repositories, and so the one
// heavyweight detection pass has a single seam.
//
// It resolves the key directly through gitutil rather than through
// ahoy.Detect: internal/core/ahoy now depends on this package for the store's
// location (Resolve), so reaching back into it from here would be an import
// cycle. gitutil.RootCommit is the same call ahoy's own identity pass makes,
// and it answers the only question asked here.
var resolveRootSHA = func(cwd string) (string, bool) {
	sha := gitutil.RootCommit(cwd)
	if sha == "" {
		return "", false
	}
	return sha, true
}

// Destination is the repository a run of Ingest writes into: its root for the
// scanner configuration, and its root-commit SHA for the store key.
type Destination struct {
	RepoRoot string `json:"repo_root"`
	RootSHA  string `json:"root_sha"`
}

// verify proves the two halves of the destination name ONE repository
// (iss-2609091911060345).
//
// A Destination is a pair, and the whole point of making it an operand was that
// a transcript is redacted under the configuration of the repository it is
// stored in. Shape-checking each half separately does not establish that: a
// mismatched pair builds the scanner from repository A's pii.json and
// gitleaks.json and then files the redacted record into repository B's lane,
// which is a privacy fault rather than a misfiling. No front door can reach it
// today — the one caller derives both halves from a single detection — and that
// is exactly why the check belongs here: the seam's argument is that a
// destination is never inferred, so it cannot rest on its callers inferring
// both halves correctly. A second caller, in core or in a later surface, would
// re-open the fault silently.
//
// It fails CLOSED on a root whose own root commit does not resolve. An
// unresolvable root is not evidence that the pair agrees, and accepting one
// would leave the invariant defended only where it happens to be checkable.
func (d Destination) verify() error {
	sha, ok := resolveRootSHA(d.RepoRoot)
	if !ok || sha == "" {
		return fmt.Errorf("history: ingest cannot resolve the root commit of the destination repository at %s, so it cannot prove that root owns the store key %s; name a git repository with commits as the destination",
			fsutil.RedactHome(d.RepoRoot), d.RootSHA)
	}
	if sha != d.RootSHA {
		return fmt.Errorf("history: ingest destination is inconsistent — the repository at %s has root commit %s, not the store key %s; the pair must name ONE repository, because the scanner is built from the root and the records are filed under the key, and a mismatch redacts under one repository's configuration while filing into another's corpus",
			fsutil.RedactHome(d.RepoRoot), sha, d.RootSHA)
	}
	return nil
}

// IngestOptions carries the policy a run applies.
type IngestOptions struct {
	// Adopt names the harness project directories this run claims. A transcript
	// that would otherwise be an orphan and whose project is named here is
	// ingested under THIS repository's redaction configuration and stamped with
	// the name it was adopted from.
	Adopt []string
	// Lineage is the attribution ladder's first rung, supplied by the front
	// door. Optional.
	Lineage LineageLookup
	// MaxDepth bounds a directory walk; zero means ingestDefaultDepth.
	MaxDepth int
}

// Ingested is one transcript that entered the store.
type Ingested struct {
	Path           string `json:"path"`
	SessionID      string `json:"session_id"`
	AgentID        string `json:"agent_id,omitempty"`
	AdoptedProject string `json:"adopted_project,omitempty"`
	RecordPath     string `json:"record_path"`
	// Wrote is false when the store already held these bytes. An ingest is
	// idempotent through Capture's own key, so a second run of the same
	// material reports every file and writes none of them.
	Wrote bool `json:"wrote"`
	// Via names which step placed the owning repository: session-cwd, store, or
	// file-cwd. It is reported because the three are not equally strong, and an
	// operator reading a surprising placement needs to know which one answered.
	Via string `json:"via"`
}

// IngestSkip is a transcript this run deliberately did not store.
type IngestSkip struct {
	Path      string `json:"path"`
	SessionID string `json:"session_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	Reason    string `json:"reason"`
	// RootSHA names the repository that owns it, when one was resolved. A SHA
	// names a repository without naming it.
	RootSHA string `json:"root_sha,omitempty"`
}

// Orphan is a transcript whose owning repository could not be found on disk.
// Nothing is written for one unless its project is adopted by name.
type Orphan struct {
	Path      string `json:"path"`
	Project   string `json:"project"`
	SessionID string `json:"session_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	// Cwd is the working directory the transcript recorded, home-redacted.
	Cwd string `json:"cwd,omitempty"`
}

// IngestFailure is a transcript this run could not read or could not store.
type IngestFailure struct {
	Path string `json:"path"`
	Err  string `json:"error"`
}

// IngestResult reports four populations. Nothing is silent.
type IngestResult struct {
	Captured []Ingested      `json:"captured"`
	Skipped  []IngestSkip    `json:"skipped"`
	Orphans  []Orphan        `json:"orphans"`
	Failed   []IngestFailure `json:"failed"`
}

// Ingest reads the named sources and stores every transcript the destination
// repository owns.
//
// sources are explicit file or directory paths. There is no implicit "scan the
// harness's store" mode: a vendor path baked in here would be the on-disk
// dependency this design rules out, so a repository that ingests regularly
// declares its roots in its own configuration and the front door passes them in.
func Ingest(dest Destination, sources []string, opts IngestOptions) (IngestResult, error) {
	if dest.RepoRoot == "" {
		return IngestResult{}, errors.New("history: ingest needs an explicit destination repository root; it is never derived from the working directory, because a transcript must be redacted under the configuration of the repository it is stored in")
	}
	if !rootSHARe.MatchString(dest.RootSHA) {
		return IngestResult{}, errors.New(rootSHAErrMsg)
	}
	if err := dest.verify(); err != nil {
		return IngestResult{}, err
	}
	if len(sources) == 0 {
		return IngestResult{}, errors.New("history: ingest needs at least one source path; declare them in " + ConfigRelPath + " or name them on the command line")
	}
	// Resolving is what creates the destination store when it is absent, and
	// what migrates a corpus left at the legacy location into it (iss-95). It
	// is done here, before any source is read, so a destination that cannot be
	// opened refuses the whole run rather than failing per transcript.
	if _, err := Resolve(dest.RepoRoot, dest.RootSHA); err != nil {
		return IngestResult{}, err
	}

	candidates, res := discoverTranscripts(sources, opts.MaxDepth)
	probes := make([]transcriptProbe, 0, len(candidates))
	for _, c := range candidates {
		p, err := probeTranscript(c)
		if err != nil {
			res.Failed = append(res.Failed, IngestFailure{Path: c.path, Err: err.Error()})
			continue
		}
		probes = append(probes, p)
	}

	// The spine before the branches, so a truncated or partly failing run leaves
	// the part that makes the rest legible — the same ordering the drain uses.
	sort.SliceStable(probes, func(i, j int) bool {
		if (probes[i].agentID == "") != (probes[j].agentID == "") {
			return probes[i].agentID == ""
		}
		return probes[i].path < probes[j].path
	})

	owners := placeSessions(probes)
	adopt := map[string]struct{}{}
	for _, name := range opts.Adopt {
		adopt[name] = struct{}{}
	}
	for _, p := range probes {
		ingestOne(dest, opts, p, owners[p.sessionID], adopt, &res)
	}
	return res, nil
}

// sessionPlacement is what the session-level pass concluded about one session.
type sessionPlacement struct {
	rootSHA   string
	via       string
	ambiguous bool
}

// placeSessions resolves the owning repository once per SESSION, before any
// file is considered on its own.
//
// A session's main-thread transcript is the authority: it is the one that ran
// in the repository, and its cwd survives when a sub-agent's worktree does not.
// Failing that, a store that already holds a record or a session note for the
// id knows which repository owns it — that is a placement this machine made
// earlier, not a guess. Only a session neither answers for falls through to its
// files' own recorded directories.
func placeSessions(probes []transcriptProbe) map[string]sessionPlacement {
	// The store index is built at most ONCE per run, and only if a session
	// needs it. Asking the store per session would re-read every record in
	// every store for every session that fell through, which on a populated
	// machine is the difference between a verb and a coffee break.
	var index ownerIndex
	storeOwner := func(sessionID string) string {
		if index == nil {
			index = storeSessionIndex()
		}
		// A refusal (no store, or several) is no placement; the session falls
		// through to its files' own directories.
		sha, err := index.owner(sessionID)
		if err != nil {
			return ""
		}
		return sha
	}
	mainCwds := map[string][]string{}
	for _, p := range probes {
		if p.sessionID == "" || p.agentID != "" {
			continue
		}
		mainCwds[p.sessionID] = append(mainCwds[p.sessionID], p.cwds...)
	}
	out := map[string]sessionPlacement{}
	cache := map[string]string{}
	for _, p := range probes {
		if p.sessionID == "" {
			continue
		}
		if _, done := out[p.sessionID]; done {
			continue
		}
		shas := resolveAll(mainCwds[p.sessionID], cache)
		switch len(shas) {
		case 1:
			out[p.sessionID] = sessionPlacement{rootSHA: shas[0], via: "session-cwd"}
			continue
		case 0:
		default:
			out[p.sessionID] = sessionPlacement{ambiguous: true, via: "session-cwd"}
			continue
		}
		if sha := storeOwner(p.sessionID); sha != "" {
			out[p.sessionID] = sessionPlacement{rootSHA: sha, via: "store"}
			continue
		}
		out[p.sessionID] = sessionPlacement{}
	}
	return out
}

// resolveAll maps a set of recorded working directories to the distinct
// repositories they belong to, in first-seen order. Directories that no longer
// exist simply do not answer; several directories inside one repository (a
// worktree, a subdirectory) collapse to one SHA, which is why the count is over
// repositories and not over paths.
func resolveAll(cwds []string, cache map[string]string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, cwd := range cwds {
		sha, cached := cache[cwd]
		if !cached {
			sha, _ = resolveRootSHA(cwd)
			cache[cwd] = sha
		}
		if sha == "" {
			continue
		}
		if _, dup := seen[sha]; dup {
			continue
		}
		seen[sha] = struct{}{}
		out = append(out, sha)
	}
	return out
}

// ingestOne places and, where the destination owns it, stores one transcript.
func ingestOne(dest Destination, opts IngestOptions, p transcriptProbe, placed sessionPlacement, adopt map[string]struct{}, res *IngestResult) {
	if p.sessionID == "" {
		res.Skipped = append(res.Skipped, IngestSkip{Path: p.path, Reason: p.skipReason()})
		return
	}
	sha, via := placed.rootSHA, placed.via
	ambiguous := placed.ambiguous
	if sha == "" && !ambiguous {
		// Fallback: the file's own recorded directory.
		shas := resolveAll(p.cwds, map[string]string{})
		switch len(shas) {
		case 1:
			sha, via = shas[0], "file-cwd"
		case 0:
		default:
			ambiguous, via = true, "file-cwd"
		}
	}
	switch {
	case ambiguous:
		res.Skipped = append(res.Skipped, IngestSkip{Path: p.path, SessionID: p.sessionID, AgentID: p.agentID, Reason: SkipAmbiguousOwner})
		return
	case sha == "":
		if _, claimed := adopt[p.project]; !claimed {
			res.Orphans = append(res.Orphans, Orphan{
				Path: p.path, Project: p.project, SessionID: p.sessionID,
				AgentID: p.agentID, Cwd: fsutil.RedactHome(p.firstCwd()),
			})
			return
		}
		store(dest, opts, p, "adopted", p.project, res)
		return
	case sha != dest.RootSHA:
		res.Skipped = append(res.Skipped, IngestSkip{
			Path: p.path, SessionID: p.sessionID, AgentID: p.agentID,
			Reason: SkipOwnedElsewhere, RootSHA: sha,
		})
		return
	}
	store(dest, opts, p, via, "", res)
}

// store runs one probed transcript through the unchanged, fail-closed Capture.
func store(dest Destination, opts IngestOptions, p transcriptProbe, via, adopted string, res *IngestResult) {
	meta := CaptureMeta{
		SessionID:      p.sessionID,
		Kind:           "native",
		AgentID:        p.agentID,
		LineageSource:  "ingest",
		AdoptedProject: adopted,
	}
	if p.agentID != "" {
		meta.SpawnAttribution = "unattributed"
		if opts.Lineage != nil {
			ref := LineageRef{SessionID: p.sessionID, AgentID: p.agentID, SourcePath: p.path}
			if h, ok := opts.Lineage(ref); ok && h.SpawnDepth > 0 {
				meta.SpawnAttribution = "sidecar"
				meta.AgentType = h.AgentType
				meta.SpawnDepth = h.SpawnDepth
				meta.SpawnToolUseID = h.SpawnToolUseID
				if agentIDRe.MatchString(h.ParentAgentID) {
					meta.ParentAgentID = h.ParentAgentID
				}
			}
		}
	}
	// Re-read: the probe kept the transcript's identity, not its bytes.
	raw, err := fsutil.ReadGuarded(p.path, maxTranscriptBytes)
	if err != nil {
		res.Failed = append(res.Failed, IngestFailure{Path: p.path, Err: err.Error()})
		return
	}
	out, err := Capture(dest.RepoRoot, dest.RootSHA, raw, meta)
	if err != nil {
		res.Failed = append(res.Failed, IngestFailure{Path: p.path, Err: err.Error()})
		return
	}
	res.Captured = append(res.Captured, Ingested{
		Path: p.path, SessionID: p.sessionID, AgentID: p.agentID,
		AdoptedProject: adopted, RecordPath: out.Record.Path, Wrote: out.Wrote, Via: via,
	})
}

// candidate is one file a walk turned up, with the project directory it was
// found under.
type candidate struct {
	path    string
	project string
}

// discoverTranscripts expands the sources into candidate files.
//
// The project name is the first path segment below a directory source — the
// name AS GIVEN, never decoded into a path. It is what `adopt_projects` matches
// and what an orphan is reported under, and it is a label rather than a
// location for exactly the reason the name is not reversible.
func discoverTranscripts(sources []string, maxDepth int) ([]candidate, IngestResult) {
	if maxDepth <= 0 {
		maxDepth = ingestDefaultDepth
	}
	var out []candidate
	var res IngestResult
	for _, src := range sources {
		fi, err := os.Lstat(src)
		if err != nil {
			res.Failed = append(res.Failed, IngestFailure{Path: src, Err: err.Error()})
			continue
		}
		switch {
		case fi.Mode().IsRegular():
			out = append(out, candidate{path: src, project: filepath.Base(filepath.Dir(src))})
		case fi.IsDir():
			out = append(out, walkSource(src, maxDepth, &res)...)
		default:
			res.Failed = append(res.Failed, IngestFailure{Path: src, Err: "not a regular file or directory (a symlinked source is refused)"})
		}
	}
	return out, res
}

// walkSource walks one directory source to a bounded depth. WalkDir stats with
// Lstat, so a symlinked directory is reported as a non-directory entry and is
// never descended into; a symlinked FILE is skipped here and would be refused
// by the guarded read anyway.
func walkSource(src string, maxDepth int, res *IngestResult) []candidate {
	var out []candidate
	base := filepath.Base(src)
	_ = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			res.Failed = append(res.Failed, IngestFailure{Path: path, Err: err.Error()})
			return nil
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return nil
		}
		segments := strings.Split(filepath.ToSlash(rel), "/")
		if d.IsDir() {
			if rel != "." && len(segments) >= maxDepth {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !strings.HasSuffix(path, transcriptSuffix) {
			return nil
		}
		project := base
		if len(segments) > 1 {
			project = segments[0]
		}
		out = append(out, candidate{path: path, project: project})
		return nil
	})
	return out
}

// transcriptProbe is what one transcript file says about itself.
type transcriptProbe struct {
	path    string
	project string

	sessionID  string
	agentID    string
	cwds       []string
	sessionIDs int
	agentIDs   int
}

func (p transcriptProbe) firstCwd() string {
	if len(p.cwds) == 0 {
		return ""
	}
	return p.cwds[0]
}

// skipReason names why a probe carries no usable session.
func (p transcriptProbe) skipReason() string {
	if p.agentIDs > 1 {
		return SkipAmbiguousAgent
	}
	return SkipNoSession
}

// probeTranscript reads one transcript and pulls out the identity its lines
// carry: which session it belongs to, which agent produced it (empty on a main
// thread), and every distinct working directory it recorded.
//
// The whole file is read, and the bytes are DISCARDED: the probe keeps only the
// few scalars it extracted, and a transcript that turns out to be this
// repository's is read a second time when it is stored. The spec asks for a
// bounded prefix plus the final line, which would make the first pass cheaper
// still; what is not negotiable is that a run holds one transcript in memory at
// a time and not all of them, because the source of a real run is a directory
// measured in hundreds of megabytes and every file in it would otherwise be
// resident at once. The second read is paid only for the files actually stored.
func probeTranscript(c candidate) (transcriptProbe, error) {
	raw, err := fsutil.ReadGuarded(c.path, maxTranscriptBytes)
	if err != nil {
		return transcriptProbe{}, err
	}
	p := transcriptProbe{path: c.path, project: c.project}
	sessions := map[string]struct{}{}
	agents := map[string]struct{}{}
	cwds := map[string]struct{}{}
	forEachJSONLine(string(raw), func(line transcriptLineIdentity) {
		if line.SessionID != "" && sessionIDRe.MatchString(line.SessionID) {
			if _, dup := sessions[line.SessionID]; !dup {
				sessions[line.SessionID] = struct{}{}
				p.sessionID = line.SessionID
			}
		}
		if line.AgentID != "" && agentIDRe.MatchString(line.AgentID) {
			if _, dup := agents[line.AgentID]; !dup {
				agents[line.AgentID] = struct{}{}
				p.agentID = line.AgentID
			}
		}
		if line.Cwd != "" {
			if _, dup := cwds[line.Cwd]; !dup {
				cwds[line.Cwd] = struct{}{}
				p.cwds = append(p.cwds, line.Cwd)
			}
		}
	})
	p.sessionIDs, p.agentIDs = len(sessions), len(agents)
	// A file that names two sessions or two agents is not one transcript, and a
	// transcript is never split between records.
	if p.sessionIDs != 1 || p.agentIDs > 1 {
		p.sessionID, p.agentID = "", ""
	}
	return p, nil
}

// ownerIndex maps a session id to the root-commit SHAs of every store lane
// that already knows it (storeSessionIndex builds it once per run).
type ownerIndex map[string][]string

// owner is THE session-ownership rule, and it has this one definition
// (iss-2609091911066372): the root-commit SHA of the store that already holds
// a session — through a session note, or through a stored record naming it.
//
// It is the placement this machine made earlier, recovered rather than
// recomputed, and it is what lets a session whose directories are all gone
// still reach its own store. A session two stores claim is refused rather than
// guessed, for the same reason SessionRepo refuses one: a transcript filed
// against the wrong repository is redacted by the wrong repository's scanner
// configuration.
func (idx ownerIndex) owner(sessionID string) (string, error) {
	if !safeIDSegment(sessionID) {
		return "", fmt.Errorf("history: sessionID must be non-empty, match [A-Za-z0-9._-]+ and not be a directory reference")
	}
	switch found := idx[sessionID]; len(found) {
	case 1:
		return found[0], nil
	case 0:
		return "", fmt.Errorf("history: no store has seen session %s", sessionID)
	default:
		return "", fmt.Errorf("history: session %s is claimed by %d stores; refusing to guess which repository owns its transcripts", sessionID, len(found))
	}
}

// storeSessionIndex maps every session id the user-level store's lanes know
// about to the lanes that know it — through a session note, or through a
// stored record's session_id.
//
// A store that cannot be listed is skipped rather than fatal: one unreadable
// store is not a reason to refuse a placement every other store can make.
func storeSessionIndex() ownerIndex {
	index := ownerIndex{}
	root, err := userStoreBase()
	if err != nil {
		return index
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return index
	}
	for _, e := range entries {
		if !e.IsDir() || !rootSHARe.MatchString(e.Name()) {
			continue
		}
		seen := map[string]struct{}{}
		notes, err := os.ReadDir(filepath.Join(root, e.Name(), "sessions"))
		if err == nil {
			for _, n := range notes {
				if !n.IsDir() {
					seen[n.Name()] = struct{}{}
				}
			}
		}
		records, err := listRecords(filepath.Join(root, e.Name(), recordsDirName))
		if err == nil {
			for _, r := range records {
				seen[r.SessionID] = struct{}{}
			}
		}
		for id := range seen {
			index[id] = append(index[id], e.Name())
		}
	}
	return index
}
