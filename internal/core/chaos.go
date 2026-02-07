package core

import "sync"

type ChaosFlags struct {
	mu               sync.Mutex
	NextTokenExpired bool
	InvalidSignature bool
	Simulate500      bool
}

func NewChaosFlags() *ChaosFlags {
	return &ChaosFlags{
		NextTokenExpired: false,
		InvalidSignature: false,
		Simulate500:      false,
	}
}

func (c *ChaosFlags) ToggleNextTokenExpired() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.NextTokenExpired = !c.NextTokenExpired
	return c.NextTokenExpired
}

func (c *ChaosFlags) ConsumeNextTokenExpired() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.NextTokenExpired {
		c.NextTokenExpired = false
		return true
	}
	return false
}

func (c *ChaosFlags) ToggleInvalidSignature() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.InvalidSignature = !c.InvalidSignature
	return c.InvalidSignature
}

func (c *ChaosFlags) IsInvalidSignature() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.InvalidSignature
}

func (c *ChaosFlags) ToggleSimulate500() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Simulate500 = !c.Simulate500
	return c.Simulate500
}

func (c *ChaosFlags) IsSimulate500() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Simulate500
}
