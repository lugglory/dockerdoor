package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ContainerMount struct {
	Type        string `json:"Type"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
}

type Container struct {
	Id      string `json:"Id"`
	Name    string `json:"Name"`
	Created time.Time `json:"Created"`
	State   struct {
		Running bool `json:"Running"`
	} `json:"State"`
	Mounts []ContainerMount `json:"Mounts"`
}

// normPath normalizes a path for case-insensitive prefix comparison.
// Converts backslashes to forward slashes and lowercases.
func normPath(p string) string {
	p = filepath.ToSlash(p)
	return strings.ToLower(p)
}

type match struct {
	container   Container
	source      string
	destination string
	prefixLen   int
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fatal("cannot get current directory: %v", err)
	}

	// Get all container IDs (running and stopped)
	out, err := exec.Command("docker", "ps", "-aq").Output()
	if err != nil {
		fatal("docker ps failed: %v", err)
	}

	ids := strings.Fields(string(out))
	if len(ids) == 0 {
		fatal("no containers found")
	}

	// Inspect containers in batches to avoid command-line length limits
	var containers []Container
	const batchSize = 100
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		inspectArgs := append([]string{"inspect"}, ids[i:end]...)
		out, err = exec.Command("docker", inspectArgs...).Output()
		if err != nil {
			fatal("docker inspect failed: %v", err)
		}
		var batch []Container
		if err := json.Unmarshal(out, &batch); err != nil {
			fatal("cannot parse docker inspect output: %v", err)
		}
		containers = append(containers, batch...)
	}

	normCwd := normPath(cwd)

	// Find all bind mounts whose source is a prefix of cwd
	var matches []match
	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type != "bind" {
				continue
			}
			normSrc := normPath(m.Source)
			rest := strings.TrimPrefix(normCwd, normSrc)
			// Valid prefix: rest is empty (exact match) or starts with /
			if rest == normCwd {
				continue // not a prefix
			}
			if rest != "" && rest[0] != '/' {
				continue // partial directory name match, not a real prefix
			}
			matches = append(matches, match{
				container:   c,
				source:      m.Source,
				destination: m.Destination,
				prefixLen:   len(normSrc),
			})
		}
	}

	if len(matches) == 0 {
		fatal("no container found with a volume mount covering '%s'", cwd)
	}

	// Keep only the longest prefix matches
	maxLen := 0
	for _, m := range matches {
		if m.prefixLen > maxLen {
			maxLen = m.prefixLen
		}
	}
	var best []match
	for _, m := range matches {
		if m.prefixLen == maxLen {
			best = append(best, m)
		}
	}

	// Among ties, pick the most recently created container
	sort.Slice(best, func(i, j int) bool {
		return best[i].container.Created.After(best[j].container.Created)
	})
	chosen := best[0]

	// Compute the path inside the container (use original cwd to preserve case)
	// Use filepath.ToSlash on source (no ToLower) to get correct byte length for slicing.
	// ToLower can change byte length for non-ASCII chars, making the index wrong.
	cwdSlash := filepath.ToSlash(cwd)
	srcSlash := filepath.ToSlash(chosen.source)
	relPath := cwdSlash[len(srcSlash):] // e.g. "/src/components" or ""
	containerPath := chosen.destination + relPath

	containerID := chosen.container.Id[:12]
	containerName := strings.TrimPrefix(chosen.container.Name, "/")

	// Start the container if it's not running
	if !chosen.container.State.Running {
		fmt.Fprintf(os.Stderr, "Starting container '%s'...\n", containerName)
		startCmd := exec.Command("docker", "start", containerID)
		startCmd.Stdout = os.Stdout
		startCmd.Stderr = os.Stderr
		if err := startCmd.Run(); err != nil {
			fatal("failed to start container '%s': %v", containerName, err)
		}
		// Wait for the container to be ready
		for i := 0; i < 20; i++ {
			if exec.Command("docker", "exec", containerID, "true").Run() == nil {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
	}

	// Exec into the container at the mapped path
	fmt.Fprintf(os.Stderr, "Entering '%s' at %s\n", containerName, containerPath)

	// Try bash first, fall back to sh
	shell := "bash"
	if err := exec.Command("docker", "exec", containerID, "bash", "-c", "true").Run(); err != nil {
		shell = "sh"
	}

	execCmd := exec.Command("docker", "exec", "-it", "-w", containerPath, containerID, shell, "-l")
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	if err := execCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fatal("exec failed: %v", err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dockerdoor: "+format+"\n", args...)
	os.Exit(1)
}
