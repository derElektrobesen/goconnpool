package goconnpool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeck(t *testing.T) {
	t.Parallel()
	ass := require.New(t)

	d := deck{}

	ass.Nil(d.pop())

	d.push(1)
	ass.Equal(1, d.pop())
	ass.Nil(d.pop())

	d.push(1)
	d.push(2)
	d.push(3)

	ass.Equal(3, d.size())

	ass.Equal(1, d.pop())
	ass.Equal(2, d.pop())
	ass.Equal(3, d.pop())
}

func TestRoundRobin(t *testing.T) {
	t.Parallel()
	ass := require.New(t)

	rr := roundRobin{}
	ass.Panics(func() { rr.next() }, "empty container")

	rr.push(1)
	ass.Equal(1, rr.next())
	ass.Equal(1, rr.next())

	rr.push(2)
	ass.Equal(1, rr.next())

	rr.push(3)
	ass.Equal(2, rr.next())
	ass.Equal(3, rr.next())
	ass.Equal(1, rr.next())

	ass.Equal(3, rr.size())
}
