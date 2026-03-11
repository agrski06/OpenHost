package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openhost/cli/internal/config"
	"github.com/openhost/cli/internal/providers"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: openhost <config.yaml>")
		os.Exit(1)
	}
	configPath := os.Args[1]
	cfg, err := config.ParseYAML(configPath)
	if err != nil {
		fmt.Printf("Config error: %v\n", err)
		os.Exit(1)
	}

	// Prepare VPS spec from config
	_ = &providers.VPSSpec{
		Name:   cfg.Name,
		Region: cfg.Provider.Region,
		Plan:   cfg.Provider.Plan,
		// CloudInitScript: to be generated from game config
	}

	// Provider initialization stub (to be implemented)
	// provider := providers.NewHetznerProvider() // Example
	_ = context.Background()
	// instance, err := provider.CreateVPS(ctx, vpsSpec)
	// if err != nil {
	//     fmt.Printf("Failed to create VPS: %v\n", err)
	//     os.Exit(1)
	// }
	// fmt.Printf("VPS created: %+v\n", instance)

	fmt.Println("Config loaded and provider spec prepared. Orchestration logic goes here.")
}
