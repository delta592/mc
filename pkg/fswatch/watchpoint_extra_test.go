package fswatch

import (
	"testing"
)

func TestEventDiffEvent(t *testing.T) {
	tests := []struct {
		diff eventDiff
		want Event
	}{
		{eventDiff{0, 0}, 0},
		{eventDiff{Create, Create | Remove}, Remove},
		{eventDiff{0, Create}, Create},
		{eventDiff{Create | Remove, Create}, 0},
	}
	for _, tc := range tests {
		if got := tc.diff.Event(); got != tc.want {
			t.Errorf("eventDiff%v.Event() = %v, want %v", tc.diff, got, tc.want)
		}
	}
}

func TestWatchpointAddWidensTotal(t *testing.T) {
	wp := make(watchpoint)
	first := make(chan EventInfo, 1)
	second := make(chan EventInfo, 1)

	if diff := wp.Add(first, Create); diff != (eventDiff{0, Create}) {
		t.Fatalf("Add(first, Create) = %v, want [0 Create]", diff)
	}
	if got := wp.Total(); got != Create {
		t.Fatalf("Total() = %v, want Create", got)
	}

	// Adding an event already covered reports no change.
	if diff := wp.Add(first, Create); diff != none {
		t.Fatalf("Add(first, Create) again = %v, want none", diff)
	}

	if diff := wp.Add(second, Remove); diff != (eventDiff{Create, Create | Remove}) {
		t.Fatalf("Add(second, Remove) = %v, want [Create Create|Remove]", diff)
	}
	if got := wp.Total(); got != Create|Remove {
		t.Fatalf("Total() = %v, want Create|Remove", got)
	}
}

func TestWatchpointDelNarrowsTotal(t *testing.T) {
	wp := make(watchpoint)
	first := make(chan EventInfo, 1)
	second := make(chan EventInfo, 1)
	wp.Add(first, Create)
	wp.Add(second, Remove)

	if diff := wp.Del(second, Remove); diff != (eventDiff{Create | Remove, Create}) {
		t.Fatalf("Del(second, Remove) = %v, want [Create|Remove Create]", diff)
	}
	if got := wp.Total(); got != Create {
		t.Fatalf("Total() = %v, want Create", got)
	}

	if diff := wp.Del(first, Create); diff != (eventDiff{Create, 0}) {
		t.Fatalf("Del(first, Create) = %v, want [Create 0]", diff)
	}
	if got := wp.Total(); got != 0 {
		t.Fatalf("Total() = %v, want 0", got)
	}
	if len(wp) != 0 {
		t.Fatalf("watchpoint = %v, want empty after removing every channel", wp)
	}
}

func TestWatchpointDelUnknownChannel(t *testing.T) {
	wp := make(watchpoint)
	c := make(chan EventInfo, 1)
	wp.Add(c, Create)

	if diff := wp.Del(make(chan EventInfo, 1), Create); diff != none {
		t.Fatalf("Del(unknown) = %v, want none", diff)
	}
	if got := wp.Total(); got != Create {
		t.Fatalf("Total() = %v, want Create", got)
	}
}

func TestWatchpointDryAdd(t *testing.T) {
	wp := make(watchpoint)
	c := make(chan EventInfo, 1)

	if diff := wp.dryAdd(c, Create); diff != (eventDiff{0, Create}) {
		t.Fatalf("dryAdd() = %v, want [0 Create]", diff)
	}
	// dryAdd must not mutate the watchpoint.
	if len(wp) != 0 {
		t.Fatalf("dryAdd mutated the watchpoint: %v", wp)
	}

	wp.Add(c, Create|Remove)
	if diff := wp.dryAdd(c, Create); diff != none {
		t.Fatalf("dryAdd(covered) = %v, want none", diff)
	}
	// Internal events are ignored by dryAdd.
	if diff := wp.dryAdd(c, Create|recursive); diff != none {
		t.Fatalf("dryAdd(internal) = %v, want none", diff)
	}
}

func TestWatchpointIsRecursive(t *testing.T) {
	wp := make(watchpoint)
	c := make(chan EventInfo, 1)

	wp.Add(c, Create)
	if wp.IsRecursive() {
		t.Fatal("IsRecursive() = true for a plain watchpoint")
	}

	wp.Add(c, Create|recursive)
	if !wp.IsRecursive() {
		t.Fatal("IsRecursive() = false after adding a recursive event")
	}
}

func TestWatchpointTotalStripsInternal(t *testing.T) {
	wp := make(watchpoint)
	c := make(chan EventInfo, 1)
	wp.Add(c, Create|recursive|omit)
	if got := wp.Total(); got&internal != 0 {
		t.Fatalf("Total() = %v, want internal events stripped", got)
	}
	if got := wp.Total(); got != Create {
		t.Fatalf("Total() = %v, want Create", got)
	}
}

func TestWatchpointDispatchMatchesEventSet(t *testing.T) {
	wp := make(watchpoint)
	create := make(chan EventInfo, 1)
	remove := make(chan EventInfo, 1)
	wp.Add(create, Create)
	wp.Add(remove, Remove)

	wp.Dispatch(&testEvent{path: "/x", ev: Create}, 0)

	select {
	case ei := <-create:
		if ei.Event() != Create {
			t.Fatalf("create channel got %v", ei.Event())
		}
	default:
		t.Fatal("create channel received nothing")
	}
	select {
	case ei := <-remove:
		t.Fatalf("remove channel got %v, want nothing", ei.Event())
	default:
	}
}

func TestWatchpointDispatchIgnoresUnmatchedTotal(t *testing.T) {
	wp := make(watchpoint)
	c := make(chan EventInfo, 1)
	wp.Add(c, Create)

	wp.Dispatch(&testEvent{path: "/x", ev: Write}, 0)
	select {
	case ei := <-c:
		t.Fatalf("channel got %v, want nothing", ei.Event())
	default:
	}
}

// TestWatchpointDispatchDropsOnFullChannel checks Dispatch never blocks when a
// receiver is not keeping up.
func TestWatchpointDispatchDropsOnFullChannel(t *testing.T) {
	wp := make(watchpoint)
	c := make(chan EventInfo, 1)
	wp.Add(c, Create)

	// Fill the buffer, then dispatch again; the second event is dropped.
	wp.Dispatch(&testEvent{path: "/x", ev: Create}, 0)
	wp.Dispatch(&testEvent{path: "/y", ev: Create}, 0)

	if len(c) != 1 {
		t.Fatalf("channel length = %d, want 1 (second event dropped)", len(c))
	}
	if ei := <-c; ei.Path() != "/x" {
		t.Fatalf("channel holds %q, want the first event", ei.Path())
	}
}

// TestWatchpointDispatchExtraEvent covers the extra-event argument used to
// deliver events to recursive watchpoints.
func TestWatchpointDispatchExtraEvent(t *testing.T) {
	wp := make(watchpoint)
	c := make(chan EventInfo, 1)
	wp.Add(c, Create|recursive)

	wp.Dispatch(&testEvent{path: "/x", ev: Create}, recursive)
	select {
	case ei := <-c:
		if ei.Path() != "/x" {
			t.Fatalf("got %q", ei.Path())
		}
	default:
		t.Fatal("recursive watchpoint received nothing")
	}
}

func TestEventStringIncludesNames(t *testing.T) {
	if got := (Create | Remove).String(); got == "" {
		t.Fatal("Event.String() is empty")
	}
	if got := Event(0).String(); got != "" {
		t.Fatalf("Event(0).String() = %q, want empty", got)
	}
}

func TestWatcherStubReturnsItsError(t *testing.T) {
	stub := watcherStub{error: errWatcherBoom}
	if err := stub.Watch("/x", Create); err == nil {
		t.Fatal("Watch() = nil, want the stub error")
	}
	if err := stub.Rewatch("/x", Create, Remove); err == nil {
		t.Fatal("Rewatch() = nil, want the stub error")
	}
	if err := stub.Unwatch("/x"); err == nil {
		t.Fatal("Unwatch() = nil, want the stub error")
	}
	if err := stub.Close(); err == nil {
		t.Fatal("Close() = nil, want the stub error")
	}
}
