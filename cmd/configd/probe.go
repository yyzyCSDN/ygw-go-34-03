package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func Probe(cfg Config) error {
	cfg.Addr = "127.0.0.1:0"
	srv := NewServer(cfg)
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}
	httpSrv := &http.Server{Handler: srv.Handler()}
	go func() {
		_ = httpSrv.Serve(ln)
	}()
	base := "http://" + ln.Addr().String()
	client := &http.Client{Timeout: 3 * time.Second}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx)
	}()

	resp, err := client.Get(base + "/healthz")
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned %d", resp.StatusCode)
	}

	body, _ := json.Marshal(map[string]any{
		"app":     "checkout",
		"group":   "default",
		"entries": map[string]string{"timeout.ms": "1500", "retry.max": "3"},
	})
	resp, err = client.Post(base+"/api/publish", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("publish returned %d", resp.StatusCode)
	}

	resp, err = client.Get(base + "/api/pull?app=checkout&group=default")
	if err != nil {
		return err
	}
	pullBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(pullBody), "timeout.ms") {
		return fmt.Errorf("pull did not return published key")
	}

	resp, err = client.Get(base + "/api/status")
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status returned %d", resp.StatusCode)
	}

	resp, err = client.Get(base + "/api/history")
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("history returned %d", resp.StatusCode)
	}

	resp, err = client.Get(base + "/api/export?app=checkout&group=default")
	if err != nil {
		return err
	}
	exportBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(exportBody), "retry.max") {
		return fmt.Errorf("export did not return published keys")
	}

	resp, err = client.Get(base + "/")
	if err != nil {
		return err
	}
	page, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(page), "ConfigCenter") {
		return fmt.Errorf("console marker missing")
	}
	return nil
}
