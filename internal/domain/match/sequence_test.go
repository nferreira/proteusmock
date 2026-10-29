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
		got = append(got, cs.NextResponse().Status)
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
