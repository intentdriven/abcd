---
schema_version: 1
id: "iss-2609252305404421"
slug: "the-shell-guard-reads-a-here-document-handed-to-sh-c-bash-c"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
---

The shell guard reads a here-document handed to sh -c, bash -c or eval as the payload only one level deep. When the document's own text is a command-position $(cat <<'F' ... F) whose document is literal, the payload re-read sees a bare substitution in command position and warns, and a warn runs: an outer sh -c document whose text is an inner command-position document holding a forced git push reaches the push in bash 3.2 and 5.3 (review6-guard finding 1). literalHeredocOutput runs only for a substitution inside double quotes (tokenize.go), and an unquoted one's fixed output, word-split as bash splits it, is never read. The reference page, the plugin page and the help say the document is read as the command it runs, which overclaims.
