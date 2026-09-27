package windsurf_test

import (
	"testing"

	"github.com/raillen/prumo/internal/connectors/testkit"
	"github.com/raillen/prumo/internal/connectors/windsurf"
)

func TestWindsurfConnector(t *testing.T) {
	c := windsurf.NewConnector()
	testkit.RunAll(t, c)
}
