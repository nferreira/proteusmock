package match

import "testing"

func steps(spec ...[2]int) []CompiledStep {
	out := make([]CompiledStep, 0, len(spec))
	for _, s := range spec {
		out = append(out, CompiledStep{Times: s[0], Response: CompiledResponse{Status: s[1]}})
	}
	return out
}

func statuses(cs *CompiledScenario, n int) []int {
	got := make([]int, 0, n)
	for i := 0; i < n; i++ {
		got = append(got, cs.NextResponse(nil).Status)
	}
	return got
}

func TestNextResponseWithoutSequenceIsTheResponse(t *testing.T) {
	cs := &CompiledScenario{Response: CompiledResponse{Status: 201}}
	for _, s := range statuses(cs, 3) {
		if s != 201 {
			t.Fatalf("got %d, want 201", s)
		}
	}
}

func TestNextResponseWalksTheStepsThenRepeatsTheLast(t *testing.T) {
	cs := &CompiledScenario{Steps: steps([2]int{2, 200}, [2]int{1, 201}, [2]int{1, 202})}
	want := []int{200, 200, 201, 202, 202, 202}
	got := statuses(cs, len(want))
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d: got %v, want %v", i+1, got, want)
		}
	}
}

func TestResetSequenceStartsOver(t *testing.T) {
	cs := &CompiledScenario{Steps: steps([2]int{1, 200}, [2]int{1, 201})}
	statuses(cs, 2)
	cs.ResetSequence()
	if got := statuses(cs, 2); got[0] != 200 || got[1] != 201 {
		t.Fatalf("got %v after reset", got)
	}
}

func staged(counter string, spec ...[2]int) *CompiledScenario {
	stages := make([]CompiledStage, 0, len(spec))
	for _, s := range spec {
		stages = append(stages, CompiledStage{From: s[0], Response: CompiledResponse{Status: s[1]}})
	}
	return &CompiledScenario{Counter: counter, Stages: stages}
}

func TestStagedResponseFollowsTheCounterNotTheCallCount(t *testing.T) {
	counters := &Counters{}
	cs := staged("leases", [2]int{0, 200}, [2]int{2, 201}, [2]int{3, 202})
	var got []int
	for i := 0; i < 3; i++ { // reading does not move the counter
		got = append(got, cs.NextResponse(counters).Status)
	}
	counters.Inc("leases")
	got = append(got, cs.NextResponse(counters).Status) // 1: still the first stage
	counters.Inc("leases")
	got = append(got, cs.NextResponse(counters).Status) // 2
	counters.Inc("leases")
	counters.Inc("leases")
	got = append(got, cs.NextResponse(counters).Status) // 4: the last stage keeps answering
	want := []int{200, 200, 200, 200, 201, 202}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestCountersAreIndependentAndReset(t *testing.T) {
	counters := &Counters{}
	counters.Inc("a")
	counters.Inc("a")
	counters.Inc("b")
	if counters.Get("a") != 2 || counters.Get("b") != 1 || counters.Get("c") != 0 {
		t.Fatalf("unexpected counters: a=%d b=%d c=%d", counters.Get("a"), counters.Get("b"), counters.Get("c"))
	}
	counters.Reset()
	if counters.Get("a") != 0 {
		t.Fatal("reset must zero every counter")
	}
}
