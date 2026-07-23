// Command genschema generates the JSON Schema for the recipe format, at
// recipe/recipe.schema.json.
package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/invopop/jsonschema"

	"github.com/fornellas/resonance/recipe"
)

func main() {
	reflector := &jsonschema.Reflector{}

	schema := reflector.Reflect(&recipe.Recipe{})

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal schema: %v", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile("recipe.schema.json", data, 0o644); err != nil {
		log.Fatalf("failed to write schema: %v", err)
	}
}
