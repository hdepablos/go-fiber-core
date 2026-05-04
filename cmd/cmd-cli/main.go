/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/joho/godotenv"

	"go-fiber-core/cmd/cmd-cli/cmd"
	_ "go-fiber-core/internal/services/examplesregistry"
)

func main() {
	_ = godotenv.Load()
	cmd.Execute()
}
