package guard

import "testing"

// TestApproveReadsTheWidenedWriters: the write model reads rsync's
// destination, the command a find runs with -exec or -execdir, go build -o,
// sort -o, ditto's destination, split's prefix and uniq's output file, so a
// write of each into the git directory is refused, and the same command
// writing inside the working tree is approved.
func TestApproveReadsTheWidenedWriters(t *testing.T) {
	r, _ := gitRig(t)
	r.run(t, []approveCase{
		{`rsync -a tools/hooks/ .git/hooks/`, protectedOrRegistry},
		{`rsync -av --exclude x hook.sh .git/hooks/pre-commit`, protectedOrRegistry},
		{`find . -name hook.sh -exec cp {} .git/hooks/pre-commit \;`, protectedOrRegistry},
		{`find tools -name '*.sh' -exec install -m 755 -t .git/hooks/ {} +`, protectedOrRegistry},
		{`find . -name hook.sh -execdir cp {} ` + r.dir + `/.git/hooks/pre-commit \;`, protectedOrRegistry},
		{`go build -o .git/hooks/pre-commit ./cmd/hook`, protectedOrRegistry},
		{`go build -o=.git/hooks/pre-push ./cmd/hook`, protectedOrRegistry},
		{`sort -o .git/config config.txt`, protectedOrRegistry},
		{`sort --output=.git/config config.txt`, protectedOrRegistry},
		{`ditto tools/hooks .git/hooks`, protectedOrRegistry},
		{`split -l 10 hook.sh .git/hooks/pre-`, protectedOrRegistry},
		{`uniq config.txt .git/config`, protectedOrRegistry},
		{`uniq -c config.txt .git/config`, protectedOrRegistry},
		// The same writers inside the working tree.
		{`rsync -a src/ build/`, nil},
		{`find . -name '*.o' -exec cp {} build/ \;`, nil},
		{`find . -name '*.o' -exec ls -l {} +`, nil},
		{`go build -o bin/abcd ./cmd/abcd`, nil},
		{`go build ./...`, nil},
		{`sort -o sorted.txt words.txt`, nil},
		{`ditto src build`, nil},
		{`split -l 10 words.txt parts-`, nil},
		{`uniq words.txt unique.txt`, nil},
		{`uniq words.txt`, nil},
	})
}
