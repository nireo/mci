package mci

import (
	"io"

	"github.com/goccy/go-yaml"
)

// Stage represents a stage in the pipeline i.e. testing stage, linting stage, pre-build etc
type Stage struct {
	Name     string
	Commands []string
}

// Pipeline represents the different stages that a build should go through with the related
// commands.
type Pipeline struct {
	Stages []Stage
	Image  string
}

// PipelineFromReader parses YAML format from a given reader and outputs the pipeline.
func PipelineFromReader(r io.Reader) (*Pipeline, error) {
	var pipeline Pipeline
	err := yaml.NewDecoder(r).Decode(&pipeline)
	if err != nil {
		return nil, err
	}

	return &pipeline, nil
}
