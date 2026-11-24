package services

import (
	"context"
	"testing"
	"time"
)

func TestStartEventCompletionScheduler_GracefulStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	StartEventCompletionScheduler(ctx, nil, 10*time.Millisecond)

	time.Sleep(25 * time.Millisecond)
	cancel()
	time.Sleep(15 * time.Millisecond)

	affected, err := MarkCompletedEventsOnce(context.Background(), nil)
	if err != nil || affected != 0 {
		t.Fatalf("Ekspektasi nil db mengembalikan 0, nil; dapat %d, %v", affected, err)
	}
}
