package window

import (
	. "gopkg.in/check.v1"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type WindowTestSuite struct{}

var _ = Suite(&WindowTestSuite{})

func (*WindowTestSuite) TestMean(c *C) {
	data := []int{}
	c.Assert(Mean(data), Equals, 0)

	data = []int{10, 20, 30, 40}
	c.Assert(Mean(data), Equals, 25)

	data = []int{8, 6, 5, 1000}
	c.Assert(Mean(data), Equals, 254)

	data = []int{0, 7, 10, 9, 1000000}
	c.Assert(Mean(data), Equals, 200005)
}

func (*WindowTestSuite) TestCalculateChangeIndicator(c *C) {
	data := []int{0, 7, 10, 9}
	c.Assert(CalculateChangeIndicator(data, 1000000), Equals, "+++")
	c.Assert(CalculateChangeIndicator(data, 1000), Equals, "++")
	c.Assert(CalculateChangeIndicator(data, 100), Equals, "+")
	c.Assert(CalculateChangeIndicator(data, 10), Equals, "")
	c.Assert(CalculateChangeIndicator(data, 0), Equals, "-")

	data = []int{1000000, 1000000, 1000000, 1000000}
	c.Assert(CalculateChangeIndicator(data, 1000000), Equals, "")
	c.Assert(CalculateChangeIndicator(data, 100000), Equals, "-")
	c.Assert(CalculateChangeIndicator(data, 10000), Equals, "--")
	c.Assert(CalculateChangeIndicator(data, 1000), Equals, "---")

	data = []int{0, 0, 0, 0, 0}
	c.Assert(CalculateChangeIndicator(data, 0), Equals, "")
}
