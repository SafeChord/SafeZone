package main

import "context"

// Planted go vet finding (lostcancel) for SafeZone#108; never merged.
func vetPlanted() context.Context {
	ctx, _ := context.WithCancel(context.Background())
	return ctx
}
