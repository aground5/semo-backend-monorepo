package main

import (
	"fmt"
	"semo-server/configs"
)

func main() {
	configPath := "/Users/k2zoo/Documents/growingup/ox-hr/semo-backend/configs/file/development.yaml"
	configs.Init(&configPath)

	if err != nil {
		fmt.Println("failed to parse system prompt: %w", err)
	}
}
