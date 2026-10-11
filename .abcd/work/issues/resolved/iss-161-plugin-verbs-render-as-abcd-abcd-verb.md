---
schema_version: 1
id: "iss-161"
slug: "plugin-verbs-render-as-abcd-abcd-verb"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "marketplace-install smoke test"
found_at: "commands/abcd"
resolution: "Verb files move flat from commands/abcd/ into commands/, so the harness registers each as /abcd:<verb> rather than /abcd:abcd:<verb> — the subdirectory named after the plugin was itself an extra namespace segment. No collision to resolve: commands/abcd.md (the bare status board) already sat at the top level and no verb file was named abcd.md. surface_coverage's commands_dir follows, gaining a bare_command field so the top-level board file is excluded the way README already is; index_drift's commands entry follows too."
impact: fix
---

Plugin verbs render as /abcd:abcd:<verb> instead of /abcd:<verb>: the verb files live in commands/abcd/, and the harness maps a commands/ subdirectory to a namespace segment, so the directory name duplicates the plugin prefix. Verbs need to live flat under commands/ (or the subdirectory renamed away from the plugin name) for the documented /abcd:<verb> surface.