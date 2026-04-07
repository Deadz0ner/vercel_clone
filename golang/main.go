package main

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

type service struct {
	name      string
	dir       string
	endpoints []string
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("[bootstrap] multi-service runner starting")

	wd, err := os.Getwd()
	if err != nil {
		log.Printf("[warn] unable to determine working directory: %v", err)
	} else {
		log.Printf("[bootstrap] working directory: %s", wd)
	}

	services := []service{
		{
			name:      "entry",
			dir:       "cmd/1.entry",
			endpoints: []string{"GET /health", "POST /deploy", "WS /ws/{id}"},
		},
		{
			name:      "build",
			dir:       "cmd/2.build",
			endpoints: []string{"(worker — no HTTP endpoints)"},
		},
		{
			name:      "serve",
			dir:       "cmd/3.serve",
			endpoints: []string{"GET /{projectId}/{filepath}"},
		},
	}

	log.Println("[routes] project endpoint inventory:")
	for _, svc := range services {
		if len(svc.endpoints) == 0 {
			log.Printf("[routes] service=%s endpoints=none-defined", svc.name)
			continue
		}
		for _, ep := range svc.endpoints {
			log.Printf("[routes] service=%s endpoint=%s", svc.name, ep)
		}
	}

	type runningService struct {
		name string
		cmd  *exec.Cmd
	}

	var running []runningService

	for _, svc := range services {
		mainFile := filepath.Join(svc.dir, "main.go")
		if _, err := os.Stat(mainFile); err != nil {
			log.Printf("[skip] %s (missing %s): %v", svc.name, mainFile, err)
			continue
		}

		cmd := exec.Command("go", "run", "./"+svc.dir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		start := time.Now()
		log.Printf("[start] launching service=%s cmd=%q", svc.name, cmd.String())
		if err := cmd.Start(); err != nil {
			log.Printf("[error] service=%s failed to start: %v", svc.name, err)
			continue
		}

		log.Printf("[start] service=%s pid=%d startup_time=%s", svc.name, cmd.Process.Pid, time.Since(start))
		running = append(running, runningService{name: svc.name, cmd: cmd})
	}

	if len(running) == 0 {
		log.Println("[fatal] no services started. Add main.go inside cmd/* service folders.")
		return
	}

	log.Printf("[ready] services running=%d", len(running))
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("[signal] received=%s. shutting down services", sig)

	for _, item := range running {
		if item.cmd.Process != nil {
			log.Printf("[stop] sending interrupt service=%s pid=%d", item.name, item.cmd.Process.Pid)
			if err := item.cmd.Process.Signal(os.Interrupt); err != nil {
				log.Printf("[warn] failed to signal service=%s: %v", item.name, err)
			}
		}
	}

	for _, item := range running {
		if err := item.cmd.Wait(); err != nil {
			log.Printf("[exit] service=%s exited with error: %v", item.name, err)
			continue
		}
		log.Printf("[exit] service=%s exited cleanly", item.name)
	}

	log.Println("[shutdown] all services stopped")
}
