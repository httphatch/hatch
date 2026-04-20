package session

import "testing"

func TestPortAllocator_Allocate(t *testing.T) {
	a := NewPortAllocator(50000, 50100)
	id := SessionID{Project: "myapp", Name: "s1"}

	ports, err := a.Allocate(id, 3)
	if err != nil {
		t.Fatalf("Allocate() error: %v", err)
	}
	if len(ports) != 3 {
		t.Fatalf("got %d ports, want 3", len(ports))
	}

	for _, p := range ports {
		if p < 50000 || p > 50100 {
			t.Errorf("port %d out of range [50000, 50100]", p)
		}
	}

	// Verify ports are unique.
	seen := make(map[int]bool)
	for _, p := range ports {
		if seen[p] {
			t.Errorf("duplicate port %d", p)
		}
		seen[p] = true
	}

	// Verify ports appear in reserved map.
	reserved := a.Reserved()
	for _, p := range ports {
		if owner, ok := reserved[p]; !ok {
			t.Errorf("port %d not in reserved map", p)
		} else if owner != id {
			t.Errorf("port %d reserved by %v, want %v", p, owner, id)
		}
	}
}

func TestPortAllocator_Release(t *testing.T) {
	a := NewPortAllocator(50000, 50010)
	id := SessionID{Project: "myapp", Name: "s1"}

	ports, err := a.Allocate(id, 2)
	if err != nil {
		t.Fatalf("Allocate() error: %v", err)
	}

	a.Release(id)

	reserved := a.Reserved()
	for _, p := range ports {
		if _, ok := reserved[p]; ok {
			t.Errorf("port %d still reserved after Release", p)
		}
	}
}

func TestPortAllocator_ExcludePorts(t *testing.T) {
	a := NewPortAllocator(50000, 50100)
	excluded := []int{50010, 50011, 50012}
	a.ExcludePorts(excluded)
	id := SessionID{Project: "myapp", Name: "s1"}

	ports, err := a.Allocate(id, 10)
	if err != nil {
		t.Fatalf("Allocate() error: %v", err)
	}

	excludedSet := make(map[int]bool)
	for _, p := range excluded {
		excludedSet[p] = true
	}
	for _, p := range ports {
		if excludedSet[p] {
			t.Errorf("excluded port %d was allocated", p)
		}
	}
}

func TestPortAllocator_AllocateSpecific(t *testing.T) {
	a := NewPortAllocator(50000, 50100)
	id := SessionID{Project: "myapp", Name: "s1"}

	err := a.AllocateSpecific(id, []int{50050, 50051})
	if err != nil {
		t.Fatalf("AllocateSpecific() error: %v", err)
	}

	reserved := a.Reserved()
	if reserved[50050] != id {
		t.Errorf("port 50050 not reserved for %v", id)
	}
	if reserved[50051] != id {
		t.Errorf("port 50051 not reserved for %v", id)
	}

	// Second allocation for same ports should fail.
	id2 := SessionID{Project: "myapp", Name: "s2"}
	err = a.AllocateSpecific(id2, []int{50050})
	if err == nil {
		t.Error("expected error for already-reserved port, got nil")
	}
}

func TestPortAllocator_RangeExhausted(t *testing.T) {
	// Range of 2 ports, try to allocate 3.
	a := NewPortAllocator(50000, 50001)
	id := SessionID{Project: "myapp", Name: "s1"}

	_, err := a.Allocate(id, 3)
	if err == nil {
		t.Error("expected error for range exhaustion, got nil")
	}
}
