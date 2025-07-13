package flarewrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

// Start creates rootfs by exporting container filesystem and creating ext4 image
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

	// 1. Pull the image
	fmt.Printf("📥 Pulling image %s...\n", machine.Image)
	img, err := client.Pull(ctx, machine.Image, containerd.WithPullUnpack)
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// 2. Create a container from the image
	fmt.Printf("📦 Creating container from image...\n")
	containerID := fmt.Sprintf("flarewrap-%s-%d", machine.Name, os.Getpid())
	container, err := client.NewContainer(ctx, containerID, containerd.WithImage(img))
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	defer container.Delete(ctx, containerd.WithSnapshotCleanup)

	// 3. Create temporary directory for rootfs export
	tempDir := filepath.Join(f.WorkingDir, "temp", "rootfs-tmp")
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("failed to remove temp directory: %w", err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 4. Export container filesystem
	fmt.Printf("📤 Exporting container filesystem...\n")
	if err := f.exportContainerFilesystem(ctx, *client, container, machine, tempDir); err != nil {
		return fmt.Errorf("failed to export container filesystem: %w", err)
	}

	// 5. Create ext4 rootfs image
	rootfsImagePath := filepath.Join(f.WorkingDir, "rootfs", fmt.Sprintf("%s-rootfs.ext4", machine.Name))
	if err := os.MkdirAll(filepath.Dir(rootfsImagePath), 0755); err != nil {
		return fmt.Errorf("failed to create rootfs directory: %w", err)
	}

	fmt.Printf("💾 Creating ext4 rootfs image (%dMB)...\n", machine.StorageMB)
	if err := f.createExt4Image(rootfsImagePath, machine.StorageMB); err != nil {
		return fmt.Errorf("failed to create ext4 image: %w", err)
	}

	// 6. Mount and copy rootfs
	fmt.Printf("🔗 Mounting and copying rootfs...\n")
	if err := f.mountAndCopyRootfs(rootfsImagePath, tempDir); err != nil {
		return fmt.Errorf("failed to mount and copy rootfs: %w", err)
	}

	fmt.Printf("✅ Rootfs image created: %s\n", rootfsImagePath)
	
	// TODO: Here you would start the Firecracker VM with the rootfs image
	return nil
}

// exportContainerFilesystem exports the container's filesystem to a directory
func (f *Flarewrap) exportContainerFilesystem(ctx context.Context, client containerd.Client, container containerd.Container, machine *Machine, targetDir string) error {
	// Get the image to extract its layers
	img, err := container.Image(ctx)
	if err != nil {
		return fmt.Errorf("failed to get container image: %w", err)
	}
	
	// Get the image's snapshot for extraction
	snapshotter := client.SnapshotService(containerd.DefaultSnapshotter)
	
	// Create a temporary snapshot key for extraction
	tempSnapKey := fmt.Sprintf("extract-%s-%d", container.ID(), os.Getpid())
	
	// Prepare snapshot from the image
	_, err = snapshotter.Prepare(ctx, tempSnapKey, img.Target().Digest.String())
	if err != nil {
		// If prepare fails, try to view an existing snapshot
		fmt.Printf("  ⚠️  Prepare failed, trying view: %v\n", err)
		_, err = snapshotter.View(ctx, tempSnapKey, img.Target().Digest.String())
		if err != nil {
			return fmt.Errorf("failed to create snapshot for extraction: %w", err)
		}
	}
	defer snapshotter.Remove(ctx, tempSnapKey)

	// Get mounts for the snapshot
	mounts, err := snapshotter.Mounts(ctx, tempSnapKey)
	if err != nil {
		return fmt.Errorf("failed to get snapshot mounts: %w", err)
	}

	// Create a temporary mount point
	tempMountDir := filepath.Join(f.WorkingDir, "temp", "extract-mount")
	if err := os.MkdirAll(tempMountDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp mount dir: %w", err)
	}
	defer os.RemoveAll(tempMountDir)

	// Mount the snapshot
	fmt.Printf("  🔗 Mounting image snapshot...\n")
	if err := mount.All(mounts, tempMountDir); err != nil {
		return fmt.Errorf("failed to mount snapshot: %w", err)
	}
	defer mount.UnmountAll(tempMountDir, 0)

	// Copy the mounted filesystem to target directory
	fmt.Printf("  📂 Copying real filesystem from image...\n")
	if err := f.runCommand("cp", "-a", tempMountDir+"/.", targetDir+"/"); err != nil {
		return fmt.Errorf("failed to copy filesystem: %w", err)
	}
	
	return nil
}

// createExt4Image creates a blank ext4 filesystem image
func (f *Flarewrap) createExt4Image(imagePath string, sizeMB int) error {
	// Create blank image file
	fmt.Printf("  📄 Creating blank image file (%dMB)...\n", sizeMB)
	if err := f.runCommand("dd", "if=/dev/zero", fmt.Sprintf("of=%s", imagePath), "bs=1M", fmt.Sprintf("count=%d", sizeMB)); err != nil {
		return fmt.Errorf("failed to create blank image: %w", err)
	}

	// Format as ext4
	fmt.Printf("  💾 Formatting as ext4...\n")
	if err := f.runCommand("mkfs.ext4", "-F", imagePath); err != nil {
		return fmt.Errorf("failed to format ext4: %w", err)
	}

	return nil
}

// mountAndCopyRootfs mounts the ext4 image and copies the rootfs
func (f *Flarewrap) mountAndCopyRootfs(imagePath, sourceDir string) error {
	mountPoint := filepath.Join(f.WorkingDir, "temp", "mnt")
	
	// Create mount point
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}
	defer os.RemoveAll(mountPoint)

	// Mount the image
	fmt.Printf("  🔗 Mounting ext4 image...\n")
	if err := f.runCommand("sudo", "mount", "-o", "loop", imagePath, mountPoint); err != nil {
		return fmt.Errorf("failed to mount image: %w", err)
	}

	// Copy rootfs
	fmt.Printf("  📂 Copying rootfs to image...\n")
	if err := f.runCommand("sudo", "cp", "-a", sourceDir+"/.", mountPoint+"/"); err != nil {
		// Try to unmount before returning error
		f.runCommand("sudo", "umount", mountPoint)
		return fmt.Errorf("failed to copy rootfs: %w", err)
	}

	// Unmount
	fmt.Printf("  🔓 Unmounting image...\n")
	if err := f.runCommand("sudo", "umount", mountPoint); err != nil {
		return fmt.Errorf("failed to unmount image: %w", err)
	}

	return nil
}

// runCommand runs a system command
func (f *Flarewrap) runCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
