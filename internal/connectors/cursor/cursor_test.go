package cursor_test

import (
	"testing"

	"github.com/raillen/prumo/internal/connectors/cursor"
	"github.com/raillen/prumo/internal/connectors/testkit"
)

func TestCursorConnector(t *testing.T) {
	c := cursor.NewConnector()
	testkit.RunAll(t, c)
}
