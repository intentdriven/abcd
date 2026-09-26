---
schema_version: 1
id: "iss-148"
slug: "guard-registry-coverage-gaps-found-while-wiring-itd-103-regi"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "The tokenizer suspends the enclosing command when a substitution opens and resumes it when it closes, so the argv after a substitution stays the enclosing command's: cd s && rm $(true) -rf * blocks. The three reverted-design regressions are pinned (chain across a newline, leading-position substitution, nested bare paren) and an unterminated substitution still emits every suspended command."
impact: fix
resolved_by:
  commit: "2733f2538576334f3053995f4f85e96e1bd8bd8b"
---

guard registry coverage gaps found while wiring itd-103 (registry content, not matching semantics): a push whose refspec carries a leading plus is a force in disguise and no entry describes it; xargs, timeout and exec are absent from the matcher's wrappers set, so a hazard launched through one of them is not seen; a backtick command substitution is not followed, while the dollar-paren form is; and a wrapper that IS in the set defangs an entry the moment it carries its own flags, because only the wrapper name is stepped over — `sudo <hazard>` is seen, `sudo -u bob <hazard>` is not, and the same holds for `env -i` and `time -p`. That last one is the sharpest: it turns an entry the registry does describe into an allow with one extra token, and it is the only item here that a facilitator would reasonably assume was covered. Candidates for the admission gate as the registry grows from reality; the wrapper-flag item is matcher-side (`commandOf` in internal/core/guard/match.go), not registry content.

---

**Re-scoped in place (2026-08-01, the v0.5.0 guard round).** Three of the four
sub-parts shipped on `v050/iss-159`; the fourth was attempted, failed review
three times, and was reverted. The entry stays OPEN on that one item rather than
being resolved, following the iss-96 precedent — a `capture resolve` would have
to state a resolution for scope that is not settled.

**Shipped.**

- The `+refspec` force in disguise — a new bundled `git-push-force-refspec`
  entry, carried by a new optional `arg_prefixes` pattern field, so
  `git push origin +main:main` blocks while `origin main:main` does not
  (`3672c5f`).
- `xargs`, `timeout` and `exec` joined the matcher's `wrappers` set, with
  `timeout`'s mandatory `DURATION` operand stepped over too — `timeout 30 rm -rf /`
  had read as a command called `30` (`cd869e4`).
- A wrapper's OWN flags no longer defang the entry behind it: the walk steps over
  the value-taking flags each wrapper documents, off an explicit per-wrapper
  table, so `sudo -u bob <hazard>` is seen where it previously read `-u` as the
  command name (`cd869e4`). The complementary miss — a value flag the table does
  not name, such as the bundled short form `sudo -Hu bob` — is a stated limit on
  all four coverage surfaces, and is a non-match rather than a false block.

**Still open: the backtick sub-part.** "A backtick command substitution is not
followed, while the dollar-paren form is" is unchanged. It was implemented, then
extended with a frame/stack mechanism meant to let a substitution's content be
checked as its own segment while the ENCLOSING command's tokens survived the
boundary; three rounds of adversarial review (correctness and security, fresh
independent reviewer each round) each returned a genuine guard-bypass regression
traceable to that redesign. In order: a substitution written before trailing
flags truncated the enclosing segment, turning `rm` + substitution + `-rf *` from
block into allow; the frame fix for that renumbered the enclosing command's chain
when a newline fell inside a substitution, defeating `after_cd` entries; and,
worst, a substitution in LEADING command position left a bare `$` as the
enclosing command's argv[0], so `commandOf` returned `"$"` and NO registry entry
could match — a defeat of the whole registry, not of one entry. A nested bare `(`
inside `$( … )` mis-popped the frame and re-opened the first bug in a narrower
shape, and an unterminated backtick failed OPEN. Patching a fourth time inside
the same round was refused; both forms are back to `main`'s behaviour exactly, so
the LITERAL ask here — parity with `$( … )` as it actually behaves — is met by
neither form having moved, and the gap is disclosed as a v1 limit on all four
coverage surfaces again.

**Update (v0.6.7 — gh-312 / iss-2608270500192596).** Backtick following was
re-implemented and this time it holds: `$(…)` and backticks are both followed
into command position, and the three revert-era regressions are gone —
empirically, a leading-position substitution no longer defeats the registry
(`$(true) gh repo delete` blocks), the newline-in-substitution chain blocks, an
unterminated backtick blocks rather than failing OPEN, and a nested paren blocks.
The four coverage surfaces now state backticks ARE followed. What stays OPEN here
is therefore narrowed to the deep sub-part below ONLY: a substitution written
before an enclosing command's trailing flags still truncates them for BOTH forms
(`cd s && rm $(true) -rf *` allows), the pre-existing `$(…)` gap this record
already names — the "neither form followed" framing above is superseded by this
update and retained only as history.

Carried forward as its own work, not as a patch to this one: a design that
follows the payload AND preserves the enclosing command's tokens, chain index and
command position across the substitution boundary. Note for whoever takes it that
`$( … )` on `main` already loses the flags written after a substitution
(`rm $(true) -rf *` does not read as a recursive force delete) — a pre-existing
gap this round surfaced and did not introduce, and the reason the two problems
are one problem.

## Grounds

- pursued: a substitution written before trailing flags keeps them in the enclosing command; shown wrong by any substitution shape in TestSubstitutionKeepsTheEnclosingCommandWhole answering allow
