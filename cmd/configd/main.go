package main

import (
	"context"
	"flag"
	"log"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	probe := flag.Bool("probe", false, "run the self probe and exit")
	ackMs := flag.Int("ack-timeout-ms", 1500, "watch ack timeout in milliseconds")
	retries := flag.Int("max-retries", 3, "delivery retry count")
	flag.Parse()

	cfg := DefaultConfig()
	cfg.Addr = *addr
	cfg.AckTimeout = time.Duration(*ackMs) * time.Millisecond
	cfg.MaxRetries = *retries
	if *probe {
		if err := Probe(cfg); err != nil {
			log.Fatalf("probe failed: %v", err)
		}
		log.Println("probe passed")
		return
	}

	srv := NewServer(cfg)
	log.Printf("config center listening on %s", cfg.Addr)
	if err := srv.Run(context.Background()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
