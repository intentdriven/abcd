---
schema_version: 1
id: "iss-2611040554460352"
slug: "the-session-chapter-says-the-page-s"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-session.md"
remedy: "State that the policy also admits inline styles, or drop 'unsafe-inline' from style-src if the page can do without it."
decision_needed: "Should the session page's content-security policy keep admitting inline styles ('unsafe-inline' in style-src), with the chapter amended to say so, or be tightened to 'self' only, which changes what the page is allowed to load and may change how it renders?"
---

The session chapter says the page's content-security policy loads scripts and styles from the page's own origin only (.abcd/development/brief/04-surfaces/37-session.md:145-147). internal/surface/session/web/index.html:6 declares style-src 'self' 'unsafe-inline', so inline styles are admitted as well; only script-src is 'self' alone.
