package main

import "github.com/thecodearcher/limen"

// cliConfig is the JSON format of .limen/schemas.json (written by the library when CLI export is enabled).
type cliConfig struct {
	Schemas            limen.SchemaDefinitionMap `json:"schemas"`
	UseAutoIncrementID bool                      `json:"useAutoIncrementID"`
}
