---
schema_version: 1
id: "iss-2610312056566760"
slug: "when-a-command-s-program-is-a-shell"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "recording review remedies under away mode"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
remedy: "Waits on a security review of the exemption's bounds before it is built. The shape to review: keep the guessed-eval reading behind an unknown program; stop reading as a script only the command word of a segment opened by a '(' at which neither bash nor zsh opens a subshell, decided by both shells' grammar and never by a list of the words before it, and keep reading it wherever either shell runs a subshell. Tests that stay refused when x.sh holds a refused command: E=eval && $E \"rm -rf ~\"; time (./x.sh), time -p (./x.sh), time -- (./x.sh), time -p -- (./x.sh), coproc (./x.sh) and coproc NAME (./x.sh); and the zsh forms repeat 3 (./x.sh), for i (a b) (./x.sh) and while false (./x.sh). The reproduced line is allowed."
---

The guard refused 'A=/path/to/scripts/abcd.sh && $A capture remedy iss-N "each a payload (internal/core/guard/payload.go). Either flatten."' as script-runs-hazard on an 'rm -rf' text in that Go file, though no shell runs the file. The cause is the tokenizer's split at a '(' that follows a command's words (parenWord, internal/core/guard/tokenize.go): the text after the '(' is read as a new command, so targetsOf (internal/core/guard/script.go) resolves the repository path there and reads it as a script the command runs, and '. Either' becomes a source of a missing file. After ordinary command words bash refuses such a '(' as a syntax error, so the path is never run; the same split is also made after words such as time and coproc, where the subshell does run, so the split alone does not mark a line the shell refuses. The variable reaches it only because guessedEvalPayload (internal/core/guard/payload.go) reads quoted arguments behind an unknown program name as possible eval text, which is the correct fail-closed reading; the literal 'eval ... "(path)"' form and the unquoted spelled-program form block the same way, while the same path quoted without a '(' is allowed (reproduced with abcd guard check on 2026-10-31). It bites a remedy or message that writes a guard source path after a '('.
