package feed

import (
	"context"
	"fmt"
	"time"

	"github.com/goverland-labs/platform-events/events/inbox"
	client "github.com/goverland-labs/platform-events/pkg/natsclient"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/goverland-labs/inbox-feed/internal/config"
)

const (
	maxPendingElements = 100
	rateLimit          = 500 * client.KiB
	executionTtl       = time.Minute
)

type Consumer struct {
	conn     *nats.Conn
	consumer *client.Consumer[inbox.FeedPayload]
	service  *Service
}

func NewConsumer(conn *nats.Conn, service *Service) *Consumer {
	return &Consumer{
		conn:    conn,
		service: service,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	group := config.GenerateGroupName("inbox_feed")

	opts := []client.ConsumerOpt{
		client.WithRateLimit(rateLimit),
		client.WithMaxAckPending(maxPendingElements),
		client.WithAckWait(executionTtl),
	}

	consumer, err := client.NewConsumer(ctx, c.conn, group, inbox.SubjectFeedUpdated, c.handler(), opts...)
	if err != nil {
		return fmt.Errorf("consume for %s/%s: %w", group, inbox.SubjectFeedUpdated, err)
	}
	c.consumer = consumer

	log.Info().Msg("feed consumer is started")

	// todo: handle correct stopping the consumer by context

	<-ctx.Done()
	return c.consumer.Close()
}

func (c *Consumer) handler() inbox.FeedHandler {
	return func(payload inbox.FeedPayload) error {
		converted := convertPayloadToInternal(payload)

		if err := c.service.Process(context.TODO(), converted); err != nil {
			return err
		}

		return nil
	}
}

func convertPayloadToInternal(payload inbox.FeedPayload) Item {
	createdAt := time.Now()
	if len(payload.Timeline) > 0 {
		createdAt = payload.Timeline[len(payload.Timeline)-1].CreatedAt
	}

	return Item{
		ID:           payload.ID,
		CreatedAt:    createdAt,
		UpdatedAt:    time.Now(),
		DaoID:        payload.DaoID,
		ProposalID:   payload.ProposalID,
		DiscussionID: payload.DiscussionID,
		Type:         convertPayloadTypeToInternal(payload.Type),
		Action:       convertPayloadActionToInternal(payload.Action),
		Snapshot:     payload.Snapshot,
		Timeline:     convertPayloadTimelineToInternal(payload.Timeline),
	}
}

func convertPayloadTimelineToInternal(timeline []inbox.TimelineItem) Timeline {
	converted := make(Timeline, 0, len(timeline))

	for _, t := range timeline {
		converted = append(converted, TimelineInfo{
			CreatedAt: t.CreatedAt,
			Action:    convertPayloadActionToInternal(t.Action),
		})
	}

	return converted
}

var payloadActionMap = map[inbox.TimelineAction]Action{
	inbox.DaoCreated:                  DaoCreated,
	inbox.DaoUpdated:                  DaoUpdated,
	inbox.ProposalCreated:             ProposalCreated,
	inbox.ProposalUpdated:             ProposalUpdated,
	inbox.ProposalVotingStartsSoon:    ProposalVotingStartsSoon,
	inbox.ProposalVotingEndsSoon:      ProposalVotingEndsSoon,
	inbox.ProposalVotingStarted:       ProposalVotingStarted,
	inbox.ProposalVotingQuorumReached: ProposalVotingQuorumReached,
	inbox.ProposalVotingEnded:         ProposalVotingEnded,
}

func convertPayloadActionToInternal(action inbox.TimelineAction) Action {
	converted, ok := payloadActionMap[action]

	if !ok {
		log.Warn().Any("action", action).Msg("unknown payload timeline action")
	}

	return converted
}

func convertPayloadTypeToInternal(t inbox.Type) Type {
	switch t {
	case inbox.TypeDao:
		return Dao
	case inbox.TypeProposal:
		return Proposal
	default:
	}

	log.Warn().Any("type", t).Msg("unknown payload type")
	return ""
}
