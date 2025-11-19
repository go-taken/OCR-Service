package pkg

import (
	"encoding/json"
	"fmt"
	"log"
)

func Print(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal json: %v", err)
	}
	fmt.Println(string(data))
}
