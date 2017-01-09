package ring

import (
	. "gopkg.in/check.v1"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type RingTestSuite struct{}

var _ = Suite(&RingTestSuite{})

func (*RingTestSuite) TestRing(c *C) {
	r := New(5)
	c.Assert(len(r.Items), Equals, 5)

	for i := 1; i <= 10; i++ {
		r.Push(i)
	}

	c.Assert(r.Items, DeepEquals, []int{6, 7, 8, 9, 10})

	// Make a ring of 6 items
	r = New(6)
	// Push 7 items
	r.Push(1)
	r.Push(10)
	r.Push(99)
	r.Push(50)
	r.Push(77)
	r.Push(83)
	r.Push(2)
	// The oldest item should be gone
	c.Assert(r.Items, DeepEquals, []int{2, 10, 99, 50, 77, 83})
}
