package autonomy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestTheOwedReadCountsNoPreflightReceiptInADriveLane: a receipt for a head
// found only in a drive lane's worktree, which the lane's stages can write,
// does not clear the preflight the head is owed; the same receipt in the main
// checkout does.
func TestTheOwedReadCountsNoPreflightReceiptInADriveLane(t *testing.T) {
	f := newFx(t)
	head := f.head("main")
	lane := filepath.Join(t.TempDir(), "run-0000000000000001-lane-1")
	f.git("worktree", "add", "-q", "--detach", lane, "main")
	dir := filepath.Join(lane, ".abcd", ".work.local", "preflight-receipts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, head), []byte("ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := &owedReader{ctx: context.Background(), root: f.root}
	if r.hasReceipt(head) {
		t.Fatal("a receipt in a drive lane's worktree cleared the head's preflight")
	}
	f.receipt(head)
	if !r.hasReceipt(head) {
		t.Fatal("a receipt in the main checkout did not clear the head's preflight")
	}
}
