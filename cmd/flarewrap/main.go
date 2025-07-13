package main

import (
	"context"
	"fmt"
	"os"

	"github.com/shadmanZero/flarewrap/internal/util"
	"github.com/shadmanZero/flarewrap/pkg/flarewrap"
)

var (
	DEFAULT_VMLINUX = "/opt/kata/share/kata-containers/vmlinux.container"
)

func main() {
	ctx := context.Background()
	path, err := util.CheckFirecracker(ctx)
	if err != nil {
		fmt.Println("Firecracker not found")
		os.Exit(1)
	}

	fw := flarewrap.NewFlarewrap("/tmp/flarewrap", path, DEFAULT_VMLINUX)
	image := fw.NewImage("alpine:latest","alpine")
	machine := fw.NewMachine(1, 1024, 1024*5, "default",image)
	fw.Start(ctx, machine)
	fmt.Println(machine)


}