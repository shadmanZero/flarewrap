package flarewrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/namespaces"
	"github.com/shadmanZero/flarewrap/internal/image"
	"github.com/shadmanZero/flarewrap/internal/storage"
	"github.com/shadmanZero/flarewrap/internal/util"
)

type Flarewrap struct {
	WorkingDir      string
	FirecrackerPath string
	KernelPath      string
}

// NewFlarewrap creates a new Flarewrap instance
func NewFlarewrap(workingDir string, firecrackerPath string, kernelPath string) *Flarewrap {
	return &Flarewrap{
		WorkingDir:      workingDir,
		FirecrackerPath: firecrackerPath,
		KernelPath:      kernelPath,
	}
}


func (a *Flarewrap) NewImage(imageRef string, imageName string) *image.Image {
	// Create new image instance
	img := image.NewImage(imageRef, imageName)
	sm := storage.NewStorageManager(a.WorkingDir)
	sm.CreateDirectoryStructure()

	return img
}


// NewMachine creates a new machine instance
func (f *Flarewrap) NewMachine(cpuCores, memoryMB, storageMB int, name string, image *image.Image) *Machine {
	return &Machine{
		CPUCores:    cpuCores,
		MemoryMB:    memoryMB,
		StorageMB:   storageMB,
		StorageType: "default",
		Name:        name,
		Image:       image.ImageRef,
	}
}

// Start creates rootfs snapshot and mounts it
func (f *Flarewrap) Start(ctx context.Context, machine *Machine) error {
	// Add namespace to context
	ctx = namespaces.WithNamespace(ctx, "default")
	
	// Initialize containerd client
	socketPath, err := util.InitContainerdClient(ctx)
	if err != nil {
		return fmt.Errorf("containerd initialization failed: %w", err)
	}

	client, err := containerd.New(socketPath)
	if err != nil {
		return fmt.Errorf("failed to create containerd client: %w", err)
	}
	defer client.Close()

	// Pull image
	fmt.Printf("📥 Pulling image %s...\n", machine.Image)
	img, err := client.Pull(ctx, machine.Image, containerd.WithPullUnpack)
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Create snapshot service
	svc := client.SnapshotService(containerd.DefaultSnapshotter)

	// Create unique snapshot key
	snapKey := fmt.Sprintf("%s-%s", machine.Name, machine.Image)

	// Get the image's unpacked snapshot key (this is the parent)
	imageSnapKey := img.Target().Digest.String()

	// Create snapshot from the image's unpacked layers
	fmt.Printf("📦 Creating snapshot %q...\n", snapKey)
	if _, err := svc.Prepare(ctx, snapKey, imageSnapKey); err != nil {
		return fmt.Errorf("snapshot prepare failed: %w", err)
	}

	// Get mounts for the snapshot
	mounts, err := svc.Mounts(ctx, snapKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve mounts: %w", err)
	}

	// Create target directory in working directory
	target := filepath.Join(f.WorkingDir, "rootfs", snapKey)
	if err := os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("failed to create mount point %s: %w", target, err)
	}

	// Mount snapshot
	fmt.Printf("🔗 Mounting snapshot to %s...\n", target)
	if err := mount.All(mounts, target); err != nil {
		return fmt.Errorf("mount failed: %w", err)
	}

	fmt.Println("✅ Rootfs is ready at:", target)
	
	// TODO: Here you would start the Firecracker VM with the rootfs
	// For now, just return success
	return nil
}