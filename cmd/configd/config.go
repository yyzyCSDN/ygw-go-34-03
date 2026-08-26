package main

import "time"

type Config struct {
	Addr       string
	AckTimeout time.Duration
	MaxRetries int
	EvictEvery time.Duration
	IdleAfter  time.Duration
}

func DefaultConfig() Config {
	return Config{
		Addr:       "127.0.0.1:8080",
		AckTimeout: 1500 * time.Millisecond,
		MaxRetries: 3,
		EvictEvery: 2 * time.Second,
		IdleAfter:  10 * time.Second,
	}
}
