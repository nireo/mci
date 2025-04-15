package mci

import (
	"bytes"
	"reflect"
	"testing"
)

func TestPipelineFromReader(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    *Pipeline
		wantErr bool
	}{
		{
			name: "valid pipeline",
			input: `
image: testimage
stages:
  - name: test
    commands:
      - go test ./...
  - name: lint
    commands:
      - golangci-lint run
`,
			want: &Pipeline{
				Image: "testimage",
				Stages: []Stage{
					{
						Name:     "test",
						Commands: []string{"go test ./..."},
					},
					{
						Name:     "lint",
						Commands: []string{"golangci-lint run"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty pipeline",
			input: `
image: testimage
stages: []
`,
			want: &Pipeline{
				Image:  "testimage",
				Stages: []Stage{},
			},
			wantErr: false,
		},
		{
			name: "empty commands",
			input: `
image: testimage
stages:
  - name: test
    commands: []
`,
			want: &Pipeline{
				Image: "testimage",
				Stages: []Stage{
					{
						Name:     "test",
						Commands: []string{},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader([]byte(tc.input))
			got, err := PipelineFromReader(r)

			if (err != nil) != tc.wantErr {
				t.Errorf("PipelineFromReader() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("PipelineFromReader() = %v, want %v", got, tc.want)
			}
		})
	}
}
