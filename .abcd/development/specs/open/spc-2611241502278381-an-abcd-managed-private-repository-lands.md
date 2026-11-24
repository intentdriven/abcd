---
id: spc-2611241502278381
slug: an-abcd-managed-private-repository-lands
intent: itd-2610261025516272
origin: researcher-authored
production_mode: hand-written
---
# A project with no merge queue still lands only what was tested

## Summary

Delivers itd-2610261025516272. Where no merge queue gates a project's default branch, `abcd land` becomes the queue. It builds a candidate (the change joined to the default branch's tip) in the fixed check directory, runs the landing check the default branch declares on that candidate, and moves the default branch only by an act the shared remote refuses if the tip has moved, so the tree that lands is the tree that was checked. A reservation on the shared remote gives landings from several machines a turn. Every landing's report says what stood behind it: an independent verdict on that exact commit, or the lander's own machine alone. Where a queue gates the default branch none of this switches on. Nothing here posts a status or turns a hosted run off. Six steps: four pull requests of code, then a proof on a practice project, then a proof on one real private project, after which the spec closes.

## Approach

### What exists today

`abcd land` runs `make preflight` by name (`internal/core/landing/land.go`), pushes the branch and arms the forge's merge. The receipt that admits a push is minted by `scripts/preflight-receipt.sh` and read by `.githooks/pre-push`, both in abcd's own tree; `ahoy` installs neither in a managed project. `landing.ReadMergeRule` already reads from the ruleset mirror whether a merge queue gates the default branch. The release hold (`internal/core/landing/releasehold.go`) is read before a landing. Reservations are branches under one prefix on the shared remote, judged by `internal/core/lease`, with subjects `iss-`, `itd-` and `rel-`. Where no queue exists the forge merges a tree nobody tested, and nothing orders two machines.

### The declared landing check, run by the binary (step 1)

The landing check is the set of checks the declaration of itd-2611241456272313 gives a landing point, read from the default branch's tip. `land` runs those commands in order in place of the fixed `make preflight`; abcd's own repository declares `make preflight`, so its behaviour does not change. The receipt's mint and check move into the binary (`abcd land receipt mint|check`), and abcd's own script and hook become callers, so a managed project gets the same receipt and `ahoy` can install the pre-push hook wherever a landing check is declared. A receipt records which declared checks ran, each as run or reused. A landing is refused when the receipt for its tree covers only part of the declared landing check, naming the parts left out. A project with no declaration gets no local landing and is told so.

### The candidate and the pinned landing (step 2)

The route is on only when `ReadMergeRule` finds no queue on the default branch. Then:

1. `land` fetches the default branch and builds the candidate in the fixed check directory itd-2611011325421304 settled (`~/.abcd.noindex/tests/<root-sha>/`), never in a live worktree: the branch merged onto the tip, or rebased where the project's merge method is rebase. The joining commit is composed by abcd and carries the `abcd:<version>` trailer.
2. It runs the landing check on the candidate, reusing results where the project's check can.
3. It lands the candidate by one of two acts, both refused by the remote when the tip moved. Where the default branch accepts a push, a plain fast-forward push of the candidate. Where a pull request is required, the branch is pushed with the candidate as its head and the forge's merge is called pinned to that head commit, with the up-to-date rule on.
4. A refusal for a moved tip starts again from 1, and the recheck reruns only what the new commits reach where results are reused.
5. After the landing it reads the default branch's tree id and compares it with the candidate's. A difference is reported as a failure of the landing, in the result and at exit, never silently.

Because the candidate contains the default tip, a merge or squash of it has the candidate's tree; the comparison in 5 is the proof and not an assumption. Landings are not batched: one candidate, one change. An intermittent failure fails the landing and `land` does not retry on its own.

### The landing turn (step 3)

A new reservation subject names the default branch's landing turn, beside `iss-`, `itd-` and `rel-`, under the existing prefix and with the existing empty-tree shape. `land` takes it before building the candidate and releases it after the tree comparison. Its lease is the check's last measured duration plus a margin, renewed while the check runs and capped by the lease package's maximum, so a machine that stops loses the turn at its stated end and the next landing proceeds with no one clearing it. A `land` that finds the turn taken stops with the holder and the end time and resumes when run again; it does not hold a session. There is no fairness order beyond first to take the turn. The release hold is read first, as today. The turn is order only: step 2's compare and swap is what keeps an untested merge out, so a lapsed or stolen turn costs a wasted check and never an untested landing.

### What stood behind the landing (step 4)

`land`'s result gains a `behind` field with one of three values: the forge's check of the merged result, an independent verdict that judged the exact commit landed (named, with its source), or the lander's own machine. The independent value is read through a small interface that returns "none" until itd-2611011325430933 supplies a source. Where nothing on the forge enforces any check, the result says that the local route is the whole fence. A request to land a change whose head branch does not live in the project's own repository is refused before anything is fetched or run, naming the forge's own route. `abcd landings` shows the same field per row.

### The two proofs (steps 5 and 6)

Step 5 runs the route on a practice project made for the purpose: a repository with no queue, two clones on two computers, twenty landings, some started together and some with a machine stopped mid-check. Step 6 repeats it on one real private project that the product thinker names when the time comes, against that project's own figures from before. Each proof is a research note giving, per landing, the check time, the wait behind other landings, the reruns after a moved default branch, and the result of the tree comparison. The note for step 6 names neither the project nor its owner. The spec closes with step 6 and not before.

## How the Acceptance Criteria are met

1. The check as the default branch declares it, for any toolchain: step 1.
2. A partial check refused, naming the parts left out: the receipt's per-check record (step 1).
3. The candidate checked and the landed tree identical to it, compared afterwards: step 2.
4. A stale landing refused and the new candidate checked: step 2, parts 3 and 4.
5. Two computers land one at a time and the waiter is told who and until when: step 3, with step 2 for "neither lands a merge it did not test".
6. A stopped machine's hold lapses at its stated end: the lease (step 3).
7. A release hold is waited for: read before the turn (step 3).
8. A queue-gated project uses the queue and takes no hold: the switch on `ReadMergeRule` (step 2), tested in step 3 for the hold.
9. An outside change is refused before fetching: step 4.
10. What stood behind each landing, and the whole-fence statement: step 4.
11. Twenty landings on a practice project: step 5.
12. Twenty landings on one real private project, and only then finished: step 6.

## Open Questions

1. resolved: the route is chosen by whether a queue gates the default branch, read from the ruleset mirror, not by visibility and not by a probe of the forge.
2. resolved: the landing check is the declaration of itd-2611241456272313; this spec adds no second list of gates.
3. resolved: the turn is a reservation subject, not a new protocol and not a service on the local network.
4. resolved: no status is posted and no hosted run is turned off by this spec.
5. resolved: the practice project comes first and one real private project second (the product thinker, 2026-11-24).

## Footprint

- packages: internal/core/landing/landingcheck.go, internal/core/landing/receiptmint.go, internal/core/landing/candidate.go, internal/core/landing/pinned.go, internal/core/landing/treecheck.go, internal/core/landing/turn.go, internal/core/landing/behind.go, internal/core/landing/outsider.go, internal/core/landing/land.go, internal/core/landing/ruleset.go, internal/core/landing/releasehold.go, internal/core/claim/namespace.go, internal/core/claim/landturn.go, internal/core/landings/landings.go, internal/core/ahoy/commit_checks.go, internal/abcdhome/abcdhome.go, internal/ghcontract/rest-fields.txt, internal/surface/cli/land.go, internal/surface/cli/landings.go, internal/surface/cli/peers.go, scripts/preflight-receipt.sh, .githooks/pre-push, Makefile, commands/land.md, commands/landings.md, commands/peers.md, docs/reference/cli/commands.md, docs/reference/cli/commands/land.json, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/brief/04-surfaces/42-landings.md, .abcd/development/brief/04-surfaces/37-reserve.md, .abcd/development/brief/04-surfaces/01-ahoy.md, .abcd/development/research/notes/landing-without-a-queue-practice-project.md, .abcd/development/research/notes/landing-without-a-queue-real-project.md
- tests: internal/core/landing/landingcheck_test.go, internal/core/landing/receiptmint_test.go, internal/core/landing/candidate_test.go, internal/core/landing/pinned_test.go, internal/core/landing/treecheck_test.go, internal/core/landing/turn_test.go, internal/core/landing/behind_test.go, internal/core/landing/outsider_test.go, internal/core/claim/landturn_test.go, internal/core/landings/landings_behind_test.go, internal/surface/cli/land_noqueue_surface_test.go, internal/surface/cli/githook_prepush_test.go, internal/surface/cli/peers_surface_test.go

## Steps

1. The declared landing check, run by the binary
   - criteria: 1, 2
   - needs: none
   - packages: internal/core/landing/landingcheck.go, internal/core/landing/receiptmint.go, internal/core/landing/land.go, internal/core/ahoy/commit_checks.go, internal/surface/cli/land.go, scripts/preflight-receipt.sh, .githooks/pre-push, Makefile, commands/land.md, docs/reference/cli/commands.md, docs/reference/cli/commands/land.json, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/brief/04-surfaces/01-ahoy.md
   - tests: internal/core/landing/landingcheck_test.go: in a fixture project whose default branch declares two commands and whose branch declares a third, `land` runs the default branch's two; a project in a toolchain abcd does not know lands the same way; no declaration gives the refusal that names it. internal/core/landing/receiptmint_test.go: a receipt is minted only for a clean tree and names each declared check as run or reused; a receipt covering one of two declared checks is refused at landing, naming the other; the receipts abcd's own script minted before still read. internal/surface/cli/githook_prepush_test.go: the hook admits and refuses exactly as before through the binary.
2. The candidate and the pinned landing
   - criteria: 3, 4, 8
   - needs: 1
   - packages: internal/core/landing/candidate.go, internal/core/landing/pinned.go, internal/core/landing/treecheck.go, internal/core/landing/land.go, internal/core/landing/ruleset.go, internal/abcdhome/abcdhome.go, internal/ghcontract/rest-fields.txt, internal/surface/cli/land.go, commands/land.md, .abcd/development/brief/04-surfaces/38-land.md
   - tests: internal/core/landing/candidate_test.go: on a bare fixture remote, the candidate is built in the check directory and never in the worktree, contains the default tip, and is a rebase where the method is rebase; its joining commit carries the composed trailer. internal/core/landing/pinned_test.go: a fast-forward push lands the candidate; with the default branch moved after the check, the push is refused by the remote, nothing lands, and a new candidate is checked; the forge's pinned merge is called with the candidate's head and a mismatch reply lands nothing; with a queue in the ruleset mirror none of this runs and the queue is used. internal/core/landing/treecheck_test.go: equal tree ids pass; a forced difference is reported as a failed landing at exit, watched fail first.
3. The landing turn
   - criteria: 5, 6, 7, 8
   - needs: 2
   - packages: internal/core/claim/namespace.go, internal/core/claim/landturn.go, internal/core/landing/turn.go, internal/core/landing/releasehold.go, internal/core/landing/land.go, internal/surface/cli/land.go, internal/surface/cli/peers.go, commands/land.md, commands/peers.md, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/brief/04-surfaces/37-reserve.md
   - tests: internal/core/claim/landturn_test.go: the subject is taken, renewed and released on a fixture remote and refused to a second holder; it is listed by `peers`. internal/core/landing/turn_test.go: with two clones of one bare remote landing at once, one lands and the other stops naming the holder and the end time, then resumes and lands against the first one's result, and the default branch never holds a tree neither checked; a holder that stops loses the turn at its stated end and the next landing proceeds with no manual step; a release hold is waited for before the turn is asked; a queue-gated project takes no turn.
4. What stood behind the landing
   - criteria: 9, 10
   - needs: 2
   - packages: internal/core/landing/behind.go, internal/core/landing/outsider.go, internal/core/landing/land.go, internal/core/landings/landings.go, internal/surface/cli/land.go, internal/surface/cli/landings.go, commands/land.md, commands/landings.md, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/brief/04-surfaces/42-landings.md
   - tests: internal/core/landing/behind_test.go: a queue landing reads as the forge's check of the merged result; a queue-less landing with an independent verdict on the exact commit names it, and one on another commit does not count; with none it reads as the lander's own machine; a project with no enforced check carries the whole-fence statement. internal/core/landing/outsider_test.go: a head branch outside the project's repository is refused with no fetch and no command run, naming the forge's route. internal/core/landings/landings_behind_test.go and internal/surface/cli/land_noqueue_surface_test.go: the field is in the row, the result and `--json`.
5. Twenty landings on a practice project
   - criteria: 11
   - needs: 1, 2, 3, 4
   - packages: .abcd/development/research/notes/landing-without-a-queue-practice-project.md
   - tests: the note's twenty landings from two computers on a repository with no queue, at least three started together and at least one with a machine stopped mid-check, each with its check time, wait, reruns and tree comparison, and each of the Mechanism's four falsifiers answered from them
6. Twenty landings on one real private project
   - criteria: 12
   - needs: 5
   - packages: .abcd/development/research/notes/landing-without-a-queue-real-project.md
   - tests: the note's twenty landings on the project the product thinker names, against that project's own figures from before, naming neither the project nor its owner

## Size

Four pull requests and two proofs. Step 1 moves the receipt that gates every push of abcd's own repository into the binary, which is the second risk of the spec: the hook's behaviour must be identical, and its existing tests are the guard. Step 2 is the riskiest part: a mistake there lands an untested tree with no error, across merge methods and forge rules abcd's own repository never exercises, because that repository has a queue and so cannot dogfood the main path. The tree comparison after every landing and the bare-remote fixtures are the guard in code, and step 5 is the guard in use. Step 3 reuses the reservation and lease machinery whole; the new code is one subject and `land`'s stop and resume. Step 4 is a field and a refusal. Steps 5 and 6 are elapsed time, not code: step 6 cannot start until the product thinker names the project, and the spec stays open until it is done. No step runs a preflight in this planning lane; each step's builder runs the repository's definition of done.
