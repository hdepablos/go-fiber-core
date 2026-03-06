/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/joho/godotenv"

	"go-fiber-core/cmd/cmd-cli/cmd"
	// Importar servicios para que se registren al ejecutar comandos CLI
	_ "go-fiber-core/internal/services/test/imputation"
	_ "go-fiber-core/internal/services/test/steps_concurrent"
)

func main() {
	_ = godotenv.Load()
	cmd.Execute()
}
