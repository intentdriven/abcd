package runner

import (
	"context"
	"errors"
	"testing"
	"time"
)

// stuckQuota is a runner that reports its quota only after release closes,
// whatever its context says: a reporter that ignores its deadline.
type stuckQuota struct{ release chan struct{} }

func (stuckQuota) Name() string { return "stuck" }
func (stuckQuota) Run(context.Context, Request) (Answer, []byte, error) {
	return Answer{}, nil, errors.New("not run")
}
func (s stuckQuota) Quota(context.Context) (Quota, error) {
	<-s.release
	return Quota{Remaining: 1}, nil
}

// TestAQuotaReporterThatIgnoresItsDeadlineStillReturns: the run-start quota
// query passed its deadline to the reporter and then waited on it, so a
// reporter that ignored the context held the start of a build or a drain for
// as long as it chose. quotaOf now returns the deadline's error when the
// context ends, whether or not the reporter does.
func TestAQuotaReporterThatIgnoresItsDeadlineStillReturns(t *testing.T) {
	r := stuckQuota{release: make(chan struct{})}
	defer close(r.release)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, _, err := quotaOf(ctx, r)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("a reporter past its deadline returns the deadline's error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("quotaOf waited on a reporter that ignored its deadline")
	}
}
