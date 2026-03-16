package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunStart_RequiresSelector(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli := New(bytes.NewBuffer(nil), &stdout, &stderr)

	err := cli.runStart(nil)
	assert.Error(t, err)
}

func TestRunStart_RejectsUnknownFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cli := New(bytes.NewBuffer(nil), &stdout, &stderr)

	err := cli.runStart([]string{"--nope", "alpha"})
	assert.Error(t, err)
}
