---
schema_version: 1
id: "iss-2610071538064699"
slug: "report-field-check-refuses-a-slash-command-name-as-a"
severity: "nitpick"
category: "bug"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610071228216263 from a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0)"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd report (field path check)"
remedy: "none (filed automatically)"
resolution: "the report path check lets a whole-token plugin slash command (/name:verb) through; real paths still refuse"
impact: fix
---

Report field check refuses a slash-command name as a filesystem path

Filing a report whose title named the report command in its slash-command form (slash, abcd, colon, report) was refused: field title names a filesystem location. A slash command is the natural way to name the route in a report about routing. Rewording to "abcd report" passed (filed as rpt-2610071224565746).

Remedy the reporter proposes: Exempt the plugin command form (slash, plugin name, colon, verb) from the path check, or match only paths with a separator after the first segment.

Reported by a managed repository (root commit c372ff6b8fc387ffdf1acf7c51bffe469b70b2e0) through the abcd inbox as rpt-2610071228216263, a defect against abcd v0.13.1, surface abcd report (field path check).

Evidence:

- rpt-2610071228216263 (the report, kept in the inbox)
- rpt-2610071224565746
