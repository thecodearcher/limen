package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thecodearcher/aegis"
)

func loadSchemaFromConfig(filePath string) (aegis.SchemaDefinitionMap, error) {
	loaded, err := loadConfig(filePath)
	if err != nil {
		return nil, err
	}
	return loaded.Schemas, nil
}

func loadConfig(filePath string) (*aegis.CliConfig, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("no schema file found; please ensure your app is run at least once with aegis initialized to generate the schema needed for this operation: %w", err)
	}

	return parseConfigFile(absPath)
}

func parseConfigFile(filePath string) (*aegis.CliConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config aegis.CliConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	return &config, nil
}
