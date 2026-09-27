package cline_test

import (
	"testing"

	"github.com/raillen/prumo/internal/connectors/cline"
	"github.com/raillen/prumo/internal/connectors/testkit"
)

func TestClineConnector(t *testing.T) {
	c := cline.NewConnector()
	testkit.RunAll(t, c)
}
