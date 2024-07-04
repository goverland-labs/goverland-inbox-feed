package feed

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/goverland-labs/goverland-platform-events/events/inbox"
	client "github.com/goverland-labs/goverland-platform-events/pkg/natsclient"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/goverland-labs/inbox-feed/internal/config"
)

const (
	maxPendingElements = 100
	rateLimit          = 500 * client.KiB
	executionTtl       = time.Minute
)

type closable interface {
	Close() error
}

type Consumer struct {
	conn      *nats.Conn
	consumers []closable
	service   *Service
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

	cfu, err := client.NewConsumer(ctx, c.conn, group, inbox.SubjectFeedUpdated, c.handler(), opts...)
	if err != nil {
		return fmt.Errorf("consume for %s/%s: %w", group, inbox.SubjectFeedUpdated, err)
	}
	cvc, err := client.NewConsumer(ctx, c.conn, group, inbox.SubjectVoteCreated, c.handlerVoteCreated(), opts...)
	if err != nil {
		return fmt.Errorf("consume for %s/%s: %w", group, inbox.SubjectFeedUpdated, err)
	}
	fcc, err := client.NewConsumer(ctx, c.conn, group, inbox.SubjectFeedSettingsUpdated, c.handlerSettingsUpdated(), opts...)
	if err != nil {
		return fmt.Errorf("consume for %s/%s: %w", group, inbox.SubjectFeedSettingsUpdated, err)
	}
	c.consumers = append(c.consumers, cfu, cvc, fcc)

	log.Info().Msg("feed consumer is started")

	// todo: handle correct stopping the cfu by context

	<-ctx.Done()
	return c.stop()
}

func (c *Consumer) stop() error {
	for _, cs := range c.consumers {
		if err := cs.Close(); err != nil {
			log.Error().Err(err).Msg("close feed consumer")
		}
	}

	return nil
}

func (c *Consumer) handler() inbox.FeedHandler {
	return func(payload inbox.FeedPayload) error {
		converted := convertPayloadToInternal(payload)

		if err := c.service.Process(context.TODO(), converted); err != nil {
			log.Error().Err(err).Msgf("process item: %s", converted.ID)
			return err
		}

		return nil
	}
}

func (c *Consumer) handlerVoteCreated() inbox.VoteHandler {
	return func(payload inbox.VotePayload) error {
		if err := c.service.TryAutoarchive(context.TODO(), payload.UserID, payload.ProposalID); err != nil {
			log.Error().Err(err).Msgf("process voting: %s", payload.UserID)
			return err
		}

		return nil
	}
}

func (c *Consumer) handlerSettingsUpdated() inbox.FeedSettingsHandler {
	return func(payload inbox.FeedSettingsPayload) error {
		if err := c.service.SaveSettings(context.TODO(), payload.SubscriberID, payload.AutoarchiveAfterDays); err != nil {
			log.Error().Err(err).Msgf("process settings: %s", payload.SubscriberID)
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

	var minCreatedAt, maxCreatedAt, createdAt, finishedAt, quorumReachedAt time.Time
	var createdIdx, quorumReachedIdx int
	for idx, t := range timeline {
		action := convertPayloadActionToInternal(t.Action)

		if action == ProposalCreated {
			createdAt = t.CreatedAt
			createdIdx = idx
		}

		if action == ProposalVotingQuorumReached {
			quorumReachedAt = t.CreatedAt
			quorumReachedIdx = idx
		}

		if action == ProposalVotingEnded {
			finishedAt = t.CreatedAt
		}

		if minCreatedAt.IsZero() || minCreatedAt.After(t.CreatedAt) {
			minCreatedAt = t.CreatedAt
		}

		if maxCreatedAt.Before(t.CreatedAt) {
			maxCreatedAt = t.CreatedAt
		}

		converted = append(converted, TimelineInfo{
			CreatedAt: t.CreatedAt,
			Action:    action,
		})
	}

	if !createdAt.IsZero() && !createdAt.Equal(minCreatedAt) {
		converted[createdIdx].CreatedAt = minCreatedAt
	}

	if !quorumReachedAt.IsZero() && !finishedAt.IsZero() && quorumReachedAt.After(finishedAt) {
		converted[quorumReachedIdx].CreatedAt = finishedAt
	}

	sort.Slice(converted, func(i, j int) bool {
		if converted[i].CreatedAt.Equal(converted[j].CreatedAt) {
			return actionWeight(converted[i].Action) < actionWeight(converted[j].Action)
		}

		return converted[i].CreatedAt.Before(converted[j].CreatedAt)
	})

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

func actionWeight(a Action) int {
	switch a {
	case DaoCreated, ProposalCreated:
		return 1
	case ProposalVotingQuorumReached:
		return 3
	case ProposalVotingEnded:
		return 4
	default:
		return 2
	}
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
