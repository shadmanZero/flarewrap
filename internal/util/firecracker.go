package util

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

var (
	ErrFirecrackerNotFound = errors.New("firecracker not found in PATH")
	defaultFirecrackerTimeout = 3 * time.Second
	ErrFirecrackerExecution = errors.New("firecracker execution failed")
)

func CheckFirecracker(ctx context.Context) (string,error) {
	path, err := exec.LookPath("firecracker")
	if err != nil {
		return "", fmt.Errorf("firecracker not found in PATH: %w", err)
	}

	if err := verifyFirecracker(ctx, path); err != nil {
		return "", fmt.Errorf("firecracker verification failed: %w", err)
	}

	return path, nil
}

func verifyFirecracker(ctx context.Context, path string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultFirecrackerTimeout)
	defer cancel()

	
	cmd := exec.CommandContext(ctx, path, "--version")
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFirecrackerExecution, err)
	}
	

	return nil
}
func IsFirecrackerInstalled(ctx context.Context) (bool, error) {
	path, err := CheckFirecracker(ctx)
	if err != nil {
		return false, err
	}

	return path != "", nil
}