package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// TFVars represents the structure of variables expected by Terraform
type TFVars struct {
	Environment string            `json:"environment"`
	ProjectName string            `json:"project_name"`
	AppEnvVars  map[string]string `json:"app_env_vars"`
	DeployMode  string            `json:"deploy_mode"`
}

func main() {
	mode := flag.String("mode", "local", "Mode: local, lambda, or eks")
	envFile := flag.String("env", ".env", "Path to .env file")
	output := flag.String("output", "terraform/generated.tfvars.json", "Output file path")
	environment := flag.String("environment", "local", "Target environment: local, staging, prod")
	flag.Parse()

	// 1. Load .env file
	envMap, err := godotenv.Read(*envFile)
	if err != nil {
		fmt.Printf("Error reading .env file: %v\n", err)
		os.Exit(1)
	}

	// 2. Prepare TFVars
	tfVars := TFVars{
		Environment: *environment,
		ProjectName: "GoFiberCore",
		AppEnvVars:  make(map[string]string),
		DeployMode:  "lambda", // Default
	}

	// Set deploy_mode based on flag
	if *mode == "eks" {
		tfVars.DeployMode = "eks"
	} else {
		tfVars.DeployMode = "lambda"
	}

	// 3. Populate AppEnvVars from .env
	for k, v := range envMap {
		tfVars.AppEnvVars[k] = v
	}

	// 4. Apply intelligent overrides based on mode
	// Both EKS (Pods) and Lambda (Containers) run in isolated Docker environments.
	// To access services running on the host (like Postgres/Redis in Docker Compose) or LocalStack on the host,
	// we must use 'host.docker.internal'.
	if *mode == "eks" || *mode == "lambda" {
		overrides := map[string]string{
			"REDIS_HOST":       "host.docker.internal",
			"GORM_WRITE_HOST":  "host.docker.internal",
			"GORM_READ_HOST":   "host.docker.internal",
			"PGX_WRITE_HOST":   "host.docker.internal",
			"PGX_READ_HOST":    "host.docker.internal",
			"AWS_ENDPOINT_URL": "http://host.docker.internal:4566", // Access LocalStack via host
		}
		for k, v := range overrides {
			tfVars.AppEnvVars[k] = v
			fmt.Printf("Override [%s]: %s = %s\n", *mode, k, v)
		}
	}

	// 5. Write to JSON file
	jsonData, err := json.MarshalIndent(tfVars, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		os.Exit(1)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*output, jsonData, 0644); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully generated %s for mode %s\n", *output, *mode)
}
