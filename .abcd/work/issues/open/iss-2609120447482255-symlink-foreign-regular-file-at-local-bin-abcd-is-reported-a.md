---
schema_version: 1
id: "iss-2609120447482255"
slug: "symlink-foreign-regular-file-at-local-bin-abcd-is-reported-a"
severity: "nitpick"
category: "observation"
source: "user-observation"
found_during: "ahoy-install-onboarding-2026-09-12"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.11.1"
deferral_reason: "first half fixed on main, merged from fix/drain-ahoy (internal/core/ahoy/foreign_entry.go; the symlink.foreign gap and describeEntry now give the file's size, modification time and abcd build identity). Ruling owed to the product thinker (away; run A 2026-09-28, lane drainAhoy; rulings-owed AX) on the second: whether install pauses for the person to clear a foreign PATH entry and re-detects within the same run, an interactive step no install question takes today."
remedy: "Waits on ruling AX (pause for the person to clear a foreign file): if paused: in an interactive install only, after reporting symlink.foreign, ask through the Prompter whether the file has been removed and re-detect that one entry, never under --yes or a non-TTY stdin, proven by an ahoy test whose scripted prompter removes the file between answers; if not: resolve the record on the first half already on main (internal/core/ahoy/foreign_entry.go)."
---

symlink.foreign (regular file at ~/.local/bin/abcd) is reported as unresolvable with 'resolve manually' but says nothing about what the file is. Reporting size, mtime, and whether it identifies as an abcd binary (and which version) would let the user decide safely instead of inspecting by hand. The removal then needs a second full install run; a re-detect after the user clears it in the same run would save the round-trip.

---

_Relocated from another repository's ledger on 2026-09-15. It was captured by an
`ahoy install` onboarding session whose working directory was a teaching-materials
repository, so the finding landed where nothing could resolve or detect it: that
tree has no installer, no plugin root and no `~/.local/bin` surface. The id,
the `found_during` stamp and the body are unchanged; only the ledger it sits in
has moved. The store resolved correctly — it wrote to the repository it was
standing in — and the reason nothing refused the write is recorded as
iss-2609120511058115._

## Remedy grounds (2026-09-29)

- The first half is fixed at the base (foreign_entry.go reports size, modification time and abcd build identity), so the remedy covers only the re-detect.
- Rejected: re-detecting in a non-interactive run, which would wait on a person who is not there.
