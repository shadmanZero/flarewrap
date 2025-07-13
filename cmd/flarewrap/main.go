package main

import (
	"context"
	"fmt"
	"os"
	"github.com/shadmanZero/flarewrap/internal/util"
)

func main() {
	ctx := context.Background()
	
	firecrackerPath, err := util.CheckFirecracker(ctx)
	if err != nil {
		fmt.Println("Firecracker not found")
		os.Exit(1)
	}

	fmt.Println("Firecracker found at", firecrackerPath)
}