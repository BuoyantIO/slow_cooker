package ring

// Provides a ring buffer of integers with a lightweight interface.
type IntRing struct {
	Items      []int
	CurrentIdx int
}

// Returns an IntRing with a given size.
func New(size int) IntRing {
	return IntRing{
		Items:      make([]int, size),
		CurrentIdx: 0,
	}
}

// Push(item) adds the given item as the most recent item
// overwriting the oldest item.
func (r *IntRing) Push(item int) {
	r.Items[r.CurrentIdx] = item
	r.CurrentIdx = (r.CurrentIdx + 1) % len(r.Items)
}
