---
schema_version: 1
id: "iss-2609290055092630"
slug: "fsutil-redactroot-and-the-scanner-s-home"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/paths.go"
remedy: "Waits on ruling BB: if (a), make fsutil.RedactRoot redact a multi-segment home under a longer root as SweepCallerHome does, accepting the '/private~/...' rendering; if (b), write the difference and its reason beside both functions. Either way add one table test that runs both redactors over the same inputs (a home under a longer root, a trailing '.' or '.git' on the repo root, a case-variant home on a case-folding host) and asserts the ruled agreement, so the two cannot drift apart again."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the technical facilitator (rulings-owed BB; lane reanchor, run A 2026-09-29): which leading-boundary rule both redactors follow for a home of two or more segments under a longer root, the scanner sweep's (redact it, iss-324) or fsutil.RedactRoot's (leave it whole, iss-2609230641546141, so '/private~/wt/a' is never produced); the '.' polarity's rendering and the case-folding split in the 2026-09-29 evidence ride on the same ruling"
---

fsutil.RedactRoot and the scanner's home sweep still disagree on the LEADING boundary for a home of two or more segments under a longer root: SweepCallerHome sweeps '/Volumes/Backup/srv/qzhome/x' to '/Volumes/Backup~/x' (iss-324: a longer root does not make the name someone else's), while RedactRoot, and so RedactHome, the CLI error scrub and the install receipt, leave it whole because the byte before the root is a path-segment byte (iss-2609230641546141 chose that to stop '/private~/wt/a' mangles). The identity survives in CLI output wherever a message names a home under a backup volume or mount. Closing it needs a choice between the two rules for the '~' polarity (the '.' repo polarity has no counterpart); found while unifying the trailing half under iss-2608292037564347.

## Evidence, 2026-09-29

The review of the trailing-boundary unification (iss-2608292037564347) found two further places where the redactors read a path differently. Both belong to the question this record already asks: should RedactRoot and the scanner's home sweep agree.

1. The '.' repo polarity renders a suffixed repo root as a dot pair. Since 5d8bdc084 '.' ends a name, so a message naming the repo root with a trailing dot or a `.git` suffix is redacted with the replacement's own '.' left in front of the suffix: `fsutil.RedactRoot("see /Users/alice/proj.", "/Users/alice/proj", ".")` returns `see ..`, and `fsutil.RedactRoot("clone /Users/alice/proj.git now", "/Users/alice/proj", ".")` returns `clone ..git now`. Both read like a parent-directory reference. The redaction is the safe direction: the same probe at 5d8bdc084's parent returns both strings whole, leaking the full path. A fix is a choice of rendering for the '.' polarity, not of boundary.
2. On a case-folding host (macOS/APFS) the two redactors disagree on a case-variant home: `fsutil.RedactRoot("open /USERS/ALICE/a", "/Users/alice", "~")` returns `open ~/a`, while `scanner.SweepCallerHome("open /USERS/ALICE/a", "/Users/alice")` returns the string whole. RedactRoot folds case by design (iss-2608270908341622); the sweep does not, and whether the store's backstop covers the variant is unverified. This predates 5d8bdc084.

Probes: an external `fsutil_test` probe on a `git archive` copy of fix/drain-drift-2 at b8d69535c, and the same probe on a copy of 5d8bdc084's parent, both on darwin.

## Evidence, 2026-11-27

iss-2611270714040589 taught fsutil.RedactHome every spelling a path carries the home in (as written, cleaned, symlink-resolved from both, the macOS data-volume firmlink, and both Unicode normal forms). The scanner's literal sweep still reads one (dcn-2611270841518400), so the two redactors now also disagree on the spelling of the home, and a store write keeps three shapes of it that RedactHome takes out. Probed on darwin on a `git archive` copy of redact-home-1127 at 969e72904, through `fsutil.RedactHome`, `scanner.SweepCallerHome(text, scanner.CallerHome())` and `scanner.RedactRefusal`:

1. A non-ASCII home in the other normal form. With HOME a /Users home whose name is `café` written composed (NFC), and the text `open <that home>/notes/a.md: denied` naming it decomposed (NFD, the form macOS hands back): RedactHome gives `open ~/notes/a.md: denied`; the sweep and RedactRefusal both return the text whole.
2. A linked home whose target's last segment is not the username. With HOME `<tmp>/Users/alice` a link to `<tmp>/Volumes/Data/u1042`, the text naming the resolved `/private<tmp>/Volumes/Data/u1042/notes/a.md`: RedactHome gives `~/notes/a.md`; the sweep and RedactRefusal return it whole, and the identity detector has no username in it to find.
3. The layout around a redacted username, for a home outside /Users and /home. With HOME `/srv//people/alice`, the text `open /srv/people/alice/notes/a.md: denied`: RedactHome gives `open ~/notes/a.md: denied`; the sweep returns it whole, and RedactRefusal gives `open /srv/people/[redacted-user]/notes/a.md: denied`, so the layout around the username (`/srv/people/`) survives. With the same HOME and the text written as HOME is, all three give `~/notes/a.md`.

4. The reverse: a clean HOME, with the text spelling the home uncleanly. With HOME `/Users/alice`, the text `open /Users//alice/notes/a.md: denied`: RedactHome and the sweep both return it whole, and RedactRefusal gives `open /Users//[redacted-user]/notes/a.md: denied`; left so on purpose (dcn-2611270959355624), since matching every spelling a text can give would mean normalising the text.

The shared table test this record's remedy asks for should carry these four rows beside the 2026-09-29 ones.

Closed by 5d10bd1f2 on the same branch: shapes 1, 2 and 3. scanner.SweepCallerHome, scanner.SurvivingCallerHome and the home_path_self detector now read every spelling fsutil.HomeSpellings gives RedactHome (dcn-2611271029154024, superseding dcn-2611270841518400), and the same probe at 5d10bd1f2 gives `open ~/notes/a.md: denied` through all three paths for each. Shape 4 is unchanged and stays a decision (dcn-2611270959355624). Still open here: the leading-boundary ruling BB, the '.' polarity's dot pair and the case fold (the 2026-09-29 evidence).

## Remedy grounds (2026-09-29)

- Why: ruling BB's two answers, each with its change; it is unanswered and none is picked. The shared test is common to both, because the 2026-09-29 evidence shows the redactors already disagree on three inputs, not one.
- Rejected: fixing the dot-pair rendering or the case fold alone, which the record ties to the same ruling.
