package aegis

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CliConfig represents the JSON file format containing schemas
type CliConfig struct {
	Schemas            SchemaDefinitionMap `json:"schemas"`
	UseAutoIncrementID bool                `json:"useAutoIncrementID"`
}

// calculateHash computes MD5 hash of the given bytes and returns hex string
func calculateHash(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func (c *Config) serializeSchemasToJSON(schemas SchemaDefinitionMap) ([]byte, error) {
	file := CliConfig{
		Schemas:            schemas,
		UseAutoIncrementID: c.Schema.IDGenerator == nil,
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(file); err != nil {
		return nil, fmt.Errorf("failed to encode schemas: %w", err)
	}
	return buf.Bytes(), nil
}

func (c *Config) prepareCLIConfig(schemas SchemaDefinitionMap) error {
	if c.CLI == nil || !c.CLI.Enabled {
		return nil
	}

	outputPath := filepath.Join(".", ".aegis", "schemas.json")

	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	currentJSON, err := c.serializeSchemasToJSON(schemas)
	if err != nil {
		return fmt.Errorf("failed to serialize schemas: %w", err)
	}

	if _, err := os.Stat(outputPath); err == nil {
		existingData, err := os.ReadFile(outputPath)
		if err != nil {
			return writeToFile(currentJSON, outputPath)
		}

		if calculateHash(existingData) == calculateHash(currentJSON) {
			// schemas haven't changed, skip write
			return nil
		}
	}

	return writeToFile(currentJSON, outputPath)
}
