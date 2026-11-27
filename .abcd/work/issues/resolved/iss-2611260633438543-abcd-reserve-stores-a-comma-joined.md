---
schema_version: 1
id: "iss-2611260633438543"
slug: "abcd-reserve-stores-a-comma-joined"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "reserving a two-fix branch while away, 2026-11-26"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/reserve.md"
remedy: "Refuse a --footprint value containing a comma that names no existing path, naming the repeated-flag form (one --footprint per path); and add --add-footprint (or make a renewal keep held paths unless --replace-footprint is given) so widening a reservation never drops what it held. Test both refusals and the union."
resolution: "abcd reserve refuses a --footprint value that joins two or more paths with commas and names no path in the checkout (exit 2, nothing pushed), naming the repeated-flag form; an existing path named with a comma is reserved as written. The renewal half is closed by dcn-2611260650120281: a renewal's footprint replaces the held one by design, and the warning names each path it drops."
impact: fix
---

abcd reserve stores a comma-joined --footprint value as one path, so --footprint a.go,b.go reserves a single path named with a comma and the overlap check never matches either file; and a renewal with a new --footprint replaces the reservation's paths rather than adding to them, which it warns about only after the paths it dropped are no longer held.

## Grounds

- pursued: we expect a comma-joined footprint never to reach origin again; a reservation tip carrying an Abcd-Footprint path with a comma that names no file would show it wrong
