package summary

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAggregateArgs_Kind(t *testing.T) {
	args := AggregateArgs{}
	assert.Equal(t, "daily_aggregate", args.Kind())
}
