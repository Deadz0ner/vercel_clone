package builder

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vercel-clone/internal/config"
)

func RunBuildContainer(projectPath string) (string, error) {
	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project path: %w", err)
	}

	projectInfo, err := os.Stat(absProjectPath)
	if err != nil {
		return "", fmt.Errorf("failed to access project path: %w", err)
	}
	if !projectInfo.IsDir() {
		return "", fmt.Errorf("project path must be a directory: %s", absProjectPath)
	}

	cfg := config.Load()
	image := cfg.BuilderImage
	buildScript := cfg.BuilderCommand

	containerName := fmt.Sprintf("build-%s-%d", sanitizeName(filepath.Base(absProjectPath)), time.Now().Unix())
	args := []string{
		"run",
		"--name", containerName,
		"--network", "bridge",
		"-v", absProjectPath + ":/workspace/project",
		"-w", "/workspace/project",
		image,
		"sh",
		"-lc",
		buildScript,
	}

	log.Printf("[BUILDER] starting docker build container=%s image=%s project=%s", containerName, image, absProjectPath)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(newTaggedWriter(os.Stdout, "[CONTAINER][STDOUT] "), &stdout)
	cmd.Stderr = io.MultiWriter(newTaggedWriter(os.Stderr, "[CONTAINER][STDERR] "), &stderr)
	printContainerBanner("start", containerName, absProjectPath, image, buildScript)
	log.Printf("[CONTAINER] attaching to live docker output container=%s", containerName)

	runErr := cmd.Run()
	state := inspectContainerState(containerName)
	cleanupForced := state.Known && state.Running
	cleanupMessage := cleanupContainer(containerName)

	if runErr != nil {
		log.Printf("[CONTAINER] docker process returned with error container=%s err=%v", containerName, runErr)
		printContainerFooter("failed", containerName, state, cleanupMessage, cleanupForced)
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[BUILDER] build timed out after=2m0s container=%s project=%s", containerName, absProjectPath)
			return "", fmt.Errorf(
				"docker build timed out after 2 minutes; container=%s was killed\nstdout:\n%s\nstderr:\n%s",
				containerName,
				stdout.String(),
				stderr.String(),
			)
		}
		return "", fmt.Errorf(
			"docker build container failed: %w\nstdout:\n%s\nstderr:\n%s",
			runErr,
			stdout.String(),
			stderr.String(),
		)
	}
	log.Printf("[CONTAINER] docker process finished cleanly container=%s", containerName)
	printContainerFooter("success", containerName, state, cleanupMessage, cleanupForced)

	artifactPath, err := detectArtifactPath(absProjectPath)
	if err != nil {
		return "", fmt.Errorf(
			"build completed but artifact directory was not found: %w\nstdout:\n%s\nstderr:\n%s",
			err,
			stdout.String(),
			stderr.String(),
		)
	}

	log.Printf("[BUILDER] build successful artifact_path=%s", artifactPath)
	return artifactPath, nil
}

func sanitizeName(name string) string {
	name = strings.ToLower(name)
	replacer := strings.NewReplacer(" ", "-", "_", "-", ".", "-")
	name = replacer.Replace(name)

	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}

	clean := strings.Trim(b.String(), "-")
	if clean == "" {
		return "job"
	}
	return clean
}

func printContainerBanner(stage, containerName, projectPath, image, buildScript string) {
	fmt.Fprintf(
		os.Stdout,
		"\n%s[CONTAINER] ==================================================\n[CONTAINER] LIVE LOGS %s\n[CONTAINER] container=%s\n[CONTAINER] project=%s\n[CONTAINER] image=%s\n[CONTAINER] command=%s\n[CONTAINER] ==================================================%s\n\n",
		colorCyan,
		strings.ToUpper(stage),
		containerName,
		projectPath,
		image,
		buildScript,
		colorReset,
	)
}

func printContainerFooter(status, containerName string, state containerState, cleanupMessage string, cleanupForced bool) {
	color := colorGreen
	if status != "success" {
		color = colorRed
	}

	killed := state.wasKilled() || cleanupForced

	fmt.Fprintf(
		os.Stdout,
		"\n%s[CONTAINER] ==================================================\n[CONTAINER] LIVE LOGS END\n[CONTAINER] container=%s\n[CONTAINER] status=%s\n[CONTAINER] runtime_state=%s\n[CONTAINER] exit_code=%d\n[CONTAINER] killed=%s\n[CONTAINER] oom_killed=%t\n[CONTAINER] cleanup=%s\n[CONTAINER] note=%s\n[CONTAINER] ==================================================%s\n\n",
		color,
		containerName,
		strings.ToUpper(status),
		state.statusLabel(),
		state.ExitCode,
		boolLabel(killed),
		state.OOMKilled,
		cleanupMessage,
		killNote(killed, cleanupForced),
		colorReset,
	)
}

type containerState struct {
	Status    string
	ExitCode  int
	OOMKilled bool
	Running   bool
	Dead      bool
	Err       string
	Known     bool
}

func inspectContainerState(containerName string) containerState {
	args := []string{
		"inspect",
		"--format",
		"{{.State.Status}}|{{.State.ExitCode}}|{{.State.OOMKilled}}|{{.State.Running}}|{{.State.Dead}}|{{.State.Error}}",
		containerName,
	}

	output, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		log.Printf("[CONTAINER] unable to inspect container=%s err=%v output=%s", containerName, err, strings.TrimSpace(string(output)))
		return containerState{Known: false, Err: strings.TrimSpace(string(output))}
	}

	parts := strings.SplitN(strings.TrimSpace(string(output)), "|", 6)
	if len(parts) != 6 {
		return containerState{Known: false, Err: "unexpected docker inspect output"}
	}

	state := containerState{Known: true}
	state.Status = parts[0]
	fmt.Sscanf(parts[1], "%d", &state.ExitCode)
	state.OOMKilled = parts[2] == "true"
	state.Running = parts[3] == "true"
	state.Dead = parts[4] == "true"
	state.Err = parts[5]
	return state
}

func (s containerState) wasKilled() bool {
	if !s.Known {
		return false
	}
	if s.OOMKilled || s.Dead {
		return true
	}
	return s.Status == "dead" || s.Status == "removing"
}

func (s containerState) statusLabel() string {
	if !s.Known {
		if s.Err == "" {
			return "UNKNOWN"
		}
		return "UNKNOWN (" + s.Err + ")"
	}
	if s.Err != "" {
		return strings.ToUpper(s.Status) + " (" + s.Err + ")"
	}
	return strings.ToUpper(s.Status)
}

func cleanupContainer(containerName string) string {
	cmd := exec.Command("docker", "rm", "-f", containerName)
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		if strings.Contains(trimmed, "No such container") {
			return "already removed"
		}
		log.Printf("[CONTAINER] cleanup failed container=%s err=%v output=%s", containerName, err, trimmed)
		if trimmed == "" {
			return "cleanup failed"
		}
		return "cleanup failed (" + trimmed + ")"
	}
	if trimmed == "" {
		return "removed"
	}
	return "removed (" + trimmed + ")"
}

func boolLabel(v bool) string {
	if v {
		return "YES"
	}
	return "NO"
}

const (
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"
)

func killNote(killed, cleanupForced bool) string {
	if cleanupForced {
		return "container was still present, so cleanup used docker rm -f"
	}
	if killed {
		return "container did not exit normally"
	}
	return "container completed normally and was removed during cleanup"
}

type taggedWriter struct {
	dst         io.Writer
	tag         string
	atLineStart bool
}

func newTaggedWriter(dst io.Writer, tag string) *taggedWriter {
	return &taggedWriter{
		dst:         dst,
		tag:         tag,
		atLineStart: true,
	}
}

func (w *taggedWriter) Write(p []byte) (int, error) {
	for i, b := range p {
		if w.atLineStart {
			if _, err := io.WriteString(w.dst, w.tag); err != nil {
				return i, err
			}
			w.atLineStart = false
		}
		if _, err := w.dst.Write([]byte{b}); err != nil {
			return i, err
		}
		if b == '\n' {
			w.atLineStart = true
		}
	}
	return len(p), nil
}

func detectArtifactPath(projectPath string) (string, error) {
	candidates := []string{
		filepath.Join(projectPath, "dist"),
		filepath.Join(projectPath, "build"),
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no dist/ or build/ directory present under %s", projectPath)
}
