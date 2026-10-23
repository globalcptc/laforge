package builder

import (
	"context"
	"testing"
	"time"
)

func TestPhaseTimerAccumulatesInOrder(t *testing.T) {
	ctx, timer := WithPhaseTimer(context.Background())
	RecordPhase(ctx, "create", 2*time.Second)
	RecordPhase(ctx, "start", time.Second)
	RecordPhase(ctx, "create", 3*time.Second) // a retried phase adds up
	done := StartPhase(ctx, "queued")
	done()

	got := timer.Phases()
	if len(got) != 3 || got[0].Name != "create" || got[1].Name != "start" || got[2].Name != "queued" {
		t.Fatalf("phases = %v", got)
	}
	if got[0].Duration != 5*time.Second {
		t.Fatalf("create = %v, want 5s", got[0].Duration)
	}
}

func TestRecordPhaseWithoutTimerIsNoop(t *testing.T) {
	RecordPhase(context.Background(), "create", time.Second)
	StartPhase(context.Background(), "start")()
	var nilTimer *PhaseTimer
	if nilTimer.Phases() != nil {
		t.Fatal("nil timer should report no phases")
	}
}
