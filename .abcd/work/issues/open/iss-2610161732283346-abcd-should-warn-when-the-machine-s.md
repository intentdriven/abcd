---
schema_version: 1
id: "iss-2610161732283346"
slug: "abcd-should-warn-when-the-machine-s"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "abcd land lab cleanup, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "abcd ahoy (install and its health check)"
remedy: "Have abcd's install check read the forge login's granted scopes and the organisation's member settings read-only, and warn in plain words when the machine's login could change or destroy more than abcd needs: an org-admin scope (admin:org), the delete_repo scope, or members allowed to delete repositories. Name each one, what an agent on the machine could do with it, and the web page where the person removes it. Never change a scope or a setting itself."
---

abcd should warn when the machine's forge login, or the team's settings, hand agents more power than abcd needs. On 2026-10-16 the product thinker found, only by asking, that this machine's gh login carried admin:org (change the [redacted-user] organisation's settings, members and roles) and that the organisation let members delete repositories. abcd never calls the organisation API, so it needs neither. The product thinker turned member deletion off in the web UI, chose to remove admin:org, and ruled that team settings change only through the website, never through the command line. abcd's install and health checks should surface these risks unprompted, read-only, and leave the change to the person in the browser. Related: itd-2610161647248332 (abcd finds out what the forge can do).
