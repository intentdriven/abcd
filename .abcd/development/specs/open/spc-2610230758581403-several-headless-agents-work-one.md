---
id: spc-2610230758581403
slug: several-headless-agents-work-one
intent: itd-2610170634394993
origin: researcher-authored
production_mode: hand-written
---
# several-headless-agents-work-one

## Summary

The design record for itd-2610170634394993, planned on the product thinker's sign-off of 2026-10-23 (dcn-2610230757556283). It builds on reservations (itd-2609091034175565), announcements (itd-2609151838312703), the load gate, and the stall slice (#1101). It is bound by [adr-2610222026262980](../../decisions/adrs/2610222026262980-coordination-between-headless-agents-is.md), accepted at that sign-off and stated as invariant 25. It was drafted on 2026-10-22 and revised after the Opus and Sonnet review round 1. Every step is test-driven: each test is watched fail first.

Seven parts, each landing as its own pull request in this order:

1. **The pass.** `land`, `build`, `drain` and `status` compare what the records on the remote say should happen with what is happening, and act on the difference before they act. No process stays alive. Every deadline is derived when the records are read (dcn-2610180720366282).
2. **Dependencies.** A `Depends-On: #N` line in the pull request body, which `land` writes from the agent's declaration (dcn-2610180720362692). A dependent branch is not queued while #N is open. Once #N merges, the branch is rebuilt by merging the default branch in, then preflighted and queued (dcn-2610180720369860). A dependency that was closed unmerged, cannot be resolved, or forms a cycle is refused with its reason. The re-submitting agent is given the pull requests that merged since its merge base and touch its footprint (dcn-2610180720407459).
3. **Fencing and version.** A push of the work branch and a tag, made by abcd under a claim, travel in one `git push --atomic` together with a renewal of the claim branch. If the claim's tip moved, the renewal is not a fast-forward and the forge refuses the whole push. A queue entry cannot share a push: abcd re-reads the claim tip immediately before it enqueues and passes the pull request's head as `expectedHeadOid` (GitHub's enqueue mutation), which narrows the window to the one call. That window is stated in the ADR, not claimed closed. A take-over waits for expiry plus a fixed delay. Coordination records gain `abcd-coord-version: 1`, and a reader refuses a higher version, naming both (dcn-2610180720415626, ADR rule 3). The atomic push is probed first (dcn-2610180720397970 left it to planning): step 3 opens with a live probe on a scratch repository.
4. **Budgets.** The machine budget is the load gate's slots, weighted per job (dcn-2610180720376153). The model budget is one pool per computer, which every session on that computer draws on, sized by the repository's setting and itd-2610100704270254's class weights (dcn-2610222025116937, its reach ruled per computer in dcn-2610230757556283). It is kept on the computer, as the load gate's slots are, with no pushes; a session on another computer or account draws on that computer's own pool. Every wait on either budget has a deadline and ends in a named refusal.
5. **The central view in `abcd status`** (dcn-2610180723342596). One line per working copy (dcn-2610180720397774), keyed on the reservation's copy key. The line is fixed-form: a stage, the ids it waits on, and a state of working, waiting (with its next look) or stopped (flagged). It is carried on the person's existing `abcd-status` branch (dcn-2610180720402007), pushed only when it changes and only after the checklist is written, and carries its own expiry (dcn-2610180720402361). It reads live, then not seen for a stated time, then takeable, then gone. Its state follows the renewal, the process and the last commit, with the agent's own words as a hint.
6. **Own credential** (C3, dcn-2610222025116835). abcd's GitHub App issues each build or drain lane abcd launches a token of its own: a user access token from the device flow (dcn-2610222048069284). The person authorises the App once in the browser by entering a code; the device flow needs only the client id, and its token refreshes without the client secret (docs.github.com, "Generating a user access token for a GitHub App" and "Refreshing user access tokens"). The refresh token (six months, replaced at each refresh) is the only issuing secret. It lives on the person's computer in the system's credential store, outside every session and on the fence's read-deny list. Each launch is handed an eight-hour access token through abcd's credential helper, the expiry the product thinker kept at sign-off; a container or a separate account on the same computer gets a fresh one before the old lapses, and a cloud session, which cannot be reached after launch, stops pushing when its token lapses and queues what it still has to push as owed work. The lane uses the token only through abcd's own git credential helper and forge calls, never `GH_TOKEN` or `GITHUB_TOKEN`. Commits stay authored by the person. The App is installed on the one repository, which bounds the token's reach. Branch deletion is refused by rulesets on the protected prefixes only (dcn-2610180720415625). The installation token stays the fallback behind the `forgecred` seam.
7. **The login check before abcd launches a build or drain lane** (C1, dcn-2610222030477927, narrowed by dcn-2610230710469123). It covers only the lanes abcd launches on its own; the person's own sessions, away mode included, keep their login, and a drain the person starts and watches may count as attended. It runs before the lane's token is issued and reads only signals that carry no secret:
   - `gh auth status --hostname <host>`, by exit code, with its output discarded;
   - whether `GH_TOKEN`, `GITHUB_TOKEN`, `GH_ENTERPRISE_TOKEN` or `GITHUB_ENTERPRISE_TOKEN` is set, never its value;
   - whether gh's hosts file or `~/.git-credentials` has a line for the host, matched on the host alone.

   It never asks a credential helper, so a login kept only in a keychain, or an SSH key, is out of its sight. The refusal names each place it found a login and says what it cannot see. It has a deadline of 10 seconds, for `gh` only.

**The person's web step** (C3). The person registers abcd's GitHub App once in the browser, through the manifest flow, which abcd prepares and links (dcn-2610180720389628), installs it on the repository, and authorises it by entering the device code. Only the person can take this step. Following the principle instructions-to-a-person-are-verified-and-linked, the exact URL is written when the step is put to the person, checked against the docs on that day (docs.github.com, "Registering a GitHub App from a manifest").

## Footprint

- packages: internal/core/coord, internal/core/forgecred, internal/core/landing, internal/core/claim, internal/core/implement, internal/core/statusline, internal/surface/cli, commands/land.md, commands/build.md, commands/drain.md, commands/status.md, .abcd/development/brief/04-surfaces, .abcd/development/brief/05-internals, .abcd/development/brief/02-constraints/03-invariants.md, .abcd/work/rulesets
- tests: internal/core/coord, internal/core/forgecred, internal/core/landing, internal/core/claim, internal/core/implement, internal/core/statusline, internal/surface/cli

## Steps

1. **The `coord` package, as a pure function**
   It reads the records and returns the decisions. Table tests: the same records give the same decisions, and a pass stopped mid-way and re-run from the records reaches the same decisions (technical criterion 2).
   - packages: internal/core/coord
   - tests: internal/core/coord
2. **`Depends-On`**
   Parse it, and refuse a closed-unmerged, unresolvable or cyclic dependency with its reason (story 2). `land` does not queue while the dependency is open, then merges it in, preflights and queues (story 1); this is a live test against a scratch repository on the forge, per real-tests-over-fakes. The re-submit summary (technical criterion 5).
   - packages: internal/core/coord, internal/core/landing, internal/surface/cli
   - tests: internal/core/coord, internal/core/landing
3. **Fencing and version**
   Opens with the live probe of `git push --atomic` with a claim renewal on a scratch repository. A queue or tag under a moved tip is refused; a take-over waits for expiry plus the delay (story 3); the version field, and the refusal of a newer one (technical criterion 3).
   - packages: internal/core/claim, internal/core/landing, internal/core/coord
   - tests: internal/core/claim, internal/core/landing
4. **Budgets**
   Weighted load slots, and the per-computer model pool kept beside them; every wait has its deadline and named refusal (technical criterion 1 and story 10).
   - packages: internal/core/implement
   - tests: internal/core/implement
5. **Central view**
   The line format and when it is pushed (technical criterion 4), the expiry ladder, and that the state follows the machine (stories 4 and 5).
   - packages: internal/core/statusline, internal/surface/cli
   - tests: internal/core/statusline, internal/surface/cli
6. **Forge credential**
   The device-flow token is issued with the narrow permissions and its eight-hour expiry and reaches git and the forge only through abcd's helper (story 9). A settings change, a deletion on a protected prefix, and a request to a repository outside the token's reach are each refused by the forge (stories 6 and 7); this is a live test with a test App on a scratch repository.
   - packages: internal/core/forgecred
   - tests: internal/core/forgecred
7. **Login check**
   Each signal is detected and no value is ever read, printed or logged: a test plants a token in each place and greps every output and log for it, and a test helper that records being called shows no credential helper starts. The refusal names each place, and a clean environment passes (story 8); this is a live test in a container. The person's own session is not checked.
   - packages: internal/core/forgecred, internal/surface/cli
   - tests: internal/core/forgecred
8. **Brief and docs**
   Invariant 25 is rewritten from the rule the build must meet to the present tense, naming the tests that hold each part; the command pages and generated docs follow.
   - packages: .abcd/development/brief/02-constraints/03-invariants.md, commands/land.md, commands/build.md, commands/drain.md, commands/status.md
   - tests: internal/surface/cli

## Size

About 12 to 15 days of agent work. Steps 2 and 6 carry the live forge tests and are the long poles.
