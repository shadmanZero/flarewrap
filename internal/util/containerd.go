package util

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"
)

var (
	// ErrContainerdNotFound is returned when containerd binary is not found
	ErrContainerdNotFound = errors.New("containerd not found in PATH")
	// ErrContainerdSocketNotFound is returned when containerd socket is not accessible
	ErrContainerdSocketNotFound = errors.New("containerd socket not accessible")
)

// ContainerdInfo holds information about containerd installation
type ContainerdInfo struct {
	BinaryPath string
	SocketPath string
	Version    string
}

// GetDefaultContainerdSocket returns the default containerd socket path for Linux
func GetDefaultContainerdSocket() string {
	return "/run/containerd/containerd.sock"
}

// CheckContainerdSocket verifies if the containerd socket is accessible
func CheckContainerdSocket(ctx context.Context, socketPath string) error {
	if socketPath == "" {
		socketPath = GetDefaultContainerdSocket()
	}

	// Check if socket file exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: socket file does not exist at %s", ErrContainerdSocketNotFound, socketPath)
	}

	// Try to connect to the socket with timeout
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return fmt.Errorf("%w: failed to connect to socket %s: %v", ErrContainerdSocketNotFound, socketPath, err)
	}
	defer conn.Close()

	return nil
}

// CheckContainerd checks if containerd is installed and returns its information
func CheckContainerd(ctx context.Context) (*ContainerdInfo, error) {
	// Check if containerd binary exists in PATH
	binaryPath, err := exec.LookPath("containerd")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrContainerdNotFound, err)
	}

	// Get default socket path
	socketPath := GetDefaultContainerdSocket()

	// Check if socket is accessible
	if err := CheckContainerdSocket(ctx, socketPath); err != nil {
		return nil, err
	}

	// Get containerd version
	version, err := getContainerdVersion(ctx, binaryPath)
	if err != nil {
		// Don't fail if we can't get version, just use empty string
		version = "unknown"
	}

	return &ContainerdInfo{
		BinaryPath: binaryPath,
		SocketPath: socketPath,
		Version:    version,
	}, nil
}

// IsContainerdInstalled is a simple boolean check for containerd installation
func IsContainerdInstalled(ctx context.Context) bool {
	_, err := CheckContainerd(ctx)
	return err == nil
}

// GetContainerdSocket returns the containerd socket path, checking multiple locations
func GetContainerdSocket(ctx context.Context) (string, error) {
	// Try common Linux socket locations in order of preference
	socketPaths := []string{
		"/run/containerd/containerd.sock",
		"/var/run/containerd/containerd.sock",
		"/tmp/containerd.sock",
	}

	// Also check CONTAINERD_ADDRESS environment variable
	if envSocket := os.Getenv("CONTAINERD_ADDRESS"); envSocket != "" {
		socketPaths = append([]string{envSocket}, socketPaths...)
	}

	for _, socketPath := range socketPaths {
		if err := CheckContainerdSocket(ctx, socketPath); err == nil {
			return socketPath, nil
		}
	}

	return "", fmt.Errorf("%w: tried paths: %v", ErrContainerdSocketNotFound, socketPaths)
}

// getContainerdVersion gets the containerd version
func getContainerdVersion(ctx context.Context, binaryPath string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get containerd version: %w", err)
	}

	return string(output), nil
}

// InitContainerdClient returns the socket path ready for client initialization
func InitContainerdClient(ctx context.Context) (string, error) {
	info, err := CheckContainerd(ctx)
	if err != nil {
		return "", fmt.Errorf("containerd initialization failed: %w", err)
	}

	return info.SocketPath, nil
}

