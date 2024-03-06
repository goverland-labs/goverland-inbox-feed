package feed

import (
	"testing"
	"time"

	"github.com/goverland-labs/goverland-platform-events/events/inbox"
	"github.com/stretchr/testify/assert"
)

func TestConvertPayloadTimelineToInternal(t *testing.T) {
	now := time.Now()

	for name, tc := range map[string]struct {
		input  []inbox.TimelineItem
		output Timeline
	}{
		"empty": {
			input:  []inbox.TimelineItem{},
			output: Timeline{},
		},
		"correct sequence": {
			input: []inbox.TimelineItem{
				{
					CreatedAt: now,
					Action:    inbox.ProposalCreated,
				},
				{
					CreatedAt: now.Add(time.Minute),
					Action:    inbox.ProposalUpdated,
				},
				{
					CreatedAt: now.Add(time.Minute * 2),
					Action:    inbox.ProposalVotingQuorumReached,
				},
				{
					CreatedAt: now.Add(time.Minute * 3),
					Action:    inbox.ProposalVotingEnded,
				},
				{
					CreatedAt: now.Add(time.Minute * 4),
					Action:    inbox.ProposalUpdated,
				},
			},
			output: Timeline{
				{
					CreatedAt: now,
					Action:    ProposalCreated,
				},
				{
					CreatedAt: now.Add(time.Minute),
					Action:    ProposalUpdated,
				},
				{
					CreatedAt: now.Add(time.Minute * 2),
					Action:    ProposalVotingQuorumReached,
				},
				{
					CreatedAt: now.Add(time.Minute * 3),
					Action:    ProposalVotingEnded,
				},
				{
					CreatedAt: now.Add(time.Minute * 4),
					Action:    ProposalUpdated,
				},
			},
		},
		"updated early than created": {
			input: []inbox.TimelineItem{
				{
					CreatedAt: now.Add(time.Hour),
					Action:    inbox.ProposalCreated,
				},
				{
					CreatedAt: now.Add(time.Minute),
					Action:    inbox.ProposalUpdated,
				},
			},
			output: Timeline{
				{
					CreatedAt: now.Add(time.Minute),
					Action:    ProposalCreated,
				},
				{
					CreatedAt: now.Add(time.Minute),
					Action:    ProposalUpdated,
				},
			},
		},
		"quorum reached after votes finished": {
			input: []inbox.TimelineItem{
				{
					CreatedAt: now,
					Action:    inbox.ProposalVotingEnded,
				},
				{
					CreatedAt: now.Add(time.Minute),
					Action:    inbox.ProposalVotingQuorumReached,
				},
			},
			output: Timeline{
				{
					CreatedAt: now,
					Action:    ProposalVotingQuorumReached,
				},
				{
					CreatedAt: now,
					Action:    ProposalVotingEnded,
				},
			},
		},
		"chaotic timeline order": {
			input: []inbox.TimelineItem{
				{
					CreatedAt: now,
					Action:    inbox.ProposalVotingEnded,
				},
				{
					CreatedAt: now.Add(time.Minute),
					Action:    inbox.ProposalVotingQuorumReached,
				},
				{
					CreatedAt: now.Add(-time.Minute * 5),
					Action:    inbox.ProposalCreated,
				},
				{
					CreatedAt: now.Add(-time.Minute * 3),
					Action:    inbox.ProposalVotingStarted,
				},
				{
					CreatedAt: now.Add(-time.Minute * 2),
					Action:    inbox.ProposalUpdated,
				},
			},
			output: Timeline{
				{
					CreatedAt: now.Add(-time.Minute * 5),
					Action:    ProposalCreated,
				},
				{
					CreatedAt: now.Add(-time.Minute * 3),
					Action:    ProposalVotingStarted,
				},
				{
					CreatedAt: now.Add(-time.Minute * 2),
					Action:    ProposalUpdated,
				},
				{
					CreatedAt: now,
					Action:    ProposalVotingQuorumReached,
				},
				{
					CreatedAt: now,
					Action:    ProposalVotingEnded,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			actual := convertPayloadTimelineToInternal(tc.input)
			assert.Equal(t, tc.output, actual)
		})
	}
}
