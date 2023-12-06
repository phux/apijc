package app

import (
	"encoding/json"
	"fmt"
	"os"
)

type URLs struct {
	Targets           []Target            `json:"targets"`
	SequentialTargets map[string][]Target `json:"sequentialTargets"`
}

func NewURLs(targets []Target, sequentialTargets map[string][]Target) *URLs {
	return &URLs{
		Targets:           targets,
		SequentialTargets: sequentialTargets,
	}
}

func LoadURLsFromFile(path string) (*URLs, error) {
	file, _ := os.ReadFile(path)

	var urls *URLs

	err := json.Unmarshal(file, &urls)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal %s file: %w", path, err)
	}

	return urls, nil
}
