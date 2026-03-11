package main

import (
	"context"
	"fmt"
	"os"

	"github.com/openhost/cli/internal/config"
	"github.com/openhost/cli/internal/providers"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: openhost <command> <config.yaml>")
		fmt.Println("Commands: create, status, delete")
		os.Exit(1)
	}
	command := os.Args[1]
	configPath := os.Args[2]
	cfg, err := config.ParseYAML(configPath)
	if err != nil {
		fmt.Printf("Config error: %v\n", err)
		os.Exit(1)
	}

	vpsSpec := &providers.VPSSpec{
		Name:   cfg.Name,
		Region: cfg.Provider.Region,
		Plan:   cfg.Provider.Plan,
	}

	provider := providers.NewProvider(cfg.Provider.Name)
	if provider == nil {
		fmt.Printf("Provider not found: %s\n", cfg.Provider.Name)
		os.Exit(1)
	}

	ctx := context.Background()

	switch command {
	case "create":
		instance, err := provider.CreateVPS(ctx, vpsSpec)
		if err != nil {
			fmt.Printf("Failed to create VPS: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("VPS created: %+v\n", instance)
	case "status":
		if len(os.Args) < 4 {
			fmt.Println("Usage: openhost status <config.yaml> <instance_id>")
			os.Exit(1)
		}
		instanceID := os.Args[3]
		status, err := provider.GetInstanceStatus(ctx, instanceID)
		if err != nil {
			fmt.Printf("Failed to get instance status: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Instance status: %s\n", status)
	case "delete":
		if len(os.Args) < 4 {
			fmt.Println("Usage: openhost delete <config.yaml> <instance_id>")
			os.Exit(1)
		}
		instanceID := os.Args[3]
		err := provider.DeleteVPS(ctx, instanceID)
		if err != nil {
			fmt.Printf("Failed to delete VPS: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("VPS deleted")
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Commands: create, status, delete")
		os.Exit(1)
	}
}
