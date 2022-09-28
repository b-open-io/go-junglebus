package transports

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithDebugging will test the method WithDebugging()
func TestWithDebugging(t *testing.T) {

	t.Run("get opts", func(t *testing.T) {
		opt := WithDebugging(false)
		assert.IsType(t, *new(ClientOps), opt)
	})

	t.Run("debug false", func(t *testing.T) {
		opts := []ClientOps{
			WithDebugging(false),
			WithHTTP(""),
		}
		c, err := NewTransport(opts...)
		require.NoError(t, err)
		require.NotNil(t, c)

		assert.Equal(t, false, c.IsDebug())
	})

	t.Run("debug true", func(t *testing.T) {
		opts := []ClientOps{
			WithDebugging(true),
			WithHTTP(""),
		}
		c, err := NewTransport(opts...)
		require.NoError(t, err)
		require.NotNil(t, c)

		assert.Equal(t, true, c.IsDebug())
	})
}
