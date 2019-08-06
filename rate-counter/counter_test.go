package counter

import (
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	c := NewCounter(1)

	if c.Rate() != 0 {
		t.Errorf("must be 0, got %d", c.Rate())
	}

	c.Incr()
	if c.Rate() != 1 {
		t.Errorf("must be 1, got %d", c.Rate())
	}
	c.Incr()
	c.Incr()
	if c.Rate() != 3 {
		t.Errorf("must be 3, got %d", c.Rate())
	}
	time.Sleep(time.Second * 2)
	if c.Rate() != 0 {
		t.Errorf("must be 0, got %d", c.Rate())
	}
}
