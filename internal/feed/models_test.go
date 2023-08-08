package feed

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestModel_Equal(t *testing.T) {
	now := time.Now()
	for name, tc := range map[string]struct {
		original Timeline
		updated  Timeline
		equal    bool
	}{
		"empty": {
			equal: true,
		},
		"one element": {
			original: Timeline{
				{
					CreatedAt: now,
					Action:    "action",
				},
			},
			updated: Timeline{
				{
					CreatedAt: now,
					Action:    "action",
				},
			},
			equal: true,
		},
		"few elements in different order": {
			original: Timeline{
				{
					CreatedAt: now,
					Action:    "action",
				},
				{
					CreatedAt: now.Add(time.Second),
					Action:    "action2",
				},
			},
			updated: Timeline{
				{
					CreatedAt: now.Add(time.Second),
					Action:    "action2",
				},
				{
					CreatedAt: now,
					Action:    "action",
				},
			},
			equal: true,
		},
		"different date": {
			original: Timeline{
				{
					CreatedAt: now.Add(-time.Second),
					Action:    "action",
				},
			},
			updated: Timeline{
				{
					CreatedAt: now,
					Action:    "action",
				},
			},
			equal: false,
		},
		"different length": {
			original: Timeline{},
			updated: Timeline{
				{
					CreatedAt: now,
					Action:    "action",
				},
			},
			equal: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			actual := tc.original.Equal(tc.updated)
			require.Equal(t, tc.equal, actual)
		})
	}
}
