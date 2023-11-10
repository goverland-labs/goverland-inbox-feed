package feed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	coresdk "github.com/goverland-labs/core-web-sdk"
	"github.com/goverland-labs/core-web-sdk/feed"
	"github.com/goverland-labs/inbox-api/protobuf/inboxapi"
	events "github.com/goverland-labs/platform-events/events/inbox"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	"github.com/goverland-labs/inbox-feed/pkg/helpers"
)

const (
	maxPrefillElements = 200
	minPrefillElements = 5
)

type Publisher interface {
	PublishJSON(ctx context.Context, subject string, obj any) error
}

type SubscriptionsFinder interface {
	FindSubscribers(ctx context.Context, in *inboxapi.FindSubscribersRequest, opts ...grpc.CallOption) (*inboxapi.UserList, error)
	ListSubscriptions(ctx context.Context, in *inboxapi.ListSubscriptionRequest, opts ...grpc.CallOption) (*inboxapi.ListSubscriptionResponse, error)
}

type Service struct {
	repo          *Repo
	subscriptions SubscriptionsFinder
	sdk           *coresdk.Client
	publisher     Publisher
}

func NewService(repo *Repo, subscriptions SubscriptionsFinder, sdk *coresdk.Client, p Publisher) *Service {
	return &Service{
		repo:          repo,
		subscriptions: subscriptions,
		sdk:           sdk,
		publisher:     p,
	}
}

func (s *Service) Process(ctx context.Context, item Item) error {
	// skip dao objects from feed
	if item.DAO() {
		return nil
	}

	resp, err := s.subscriptions.FindSubscribers(ctx, &inboxapi.FindSubscribersRequest{
		DaoId: item.DaoID.String(),
	})
	if err != nil {
		return err
	}

	var pushTitle, pushBody string
	for _, sub := range resp.Users {
		subscriberID, err := uuid.Parse(sub.GetUserId())
		if err != nil {
			return fmt.Errorf("unable to parse subscriber id '%s': %w", sub.GetUserId(), err)
		}

		personalized := item
		personalized.SubscriberID = subscriberID

		if personalized.CreatedAt.IsZero() {
			personalized.CreatedAt = time.Now()
		}

		if err := s.repo.CreateOrUpdate(&personalized); err != nil {
			return fmt.Errorf("unable to save feed item '%s' for subscriber '%s': %w", personalized.ID, sub.GetUserId(), err)
		}

		if !item.AllowSending() {
			continue
		}

		if pushTitle == "" {
			di, err := s.sdk.GetDao(ctx, item.DaoID)
			if err != nil {
				log.Error().Err(err).Msgf("core client: get dao: %s: %s", item.DaoID.String(), err.Error())
				continue
			}
			pushTitle = fmt.Sprintf("%s: %s", di.Name, convertAction(item.Action))
		}

		if pushBody == "" {
			pr, err := s.sdk.GetProposal(ctx, item.ProposalID)
			if err != nil {
				log.Error().Err(err).Msgf("core client: get proposal: %s: %s", item.ProposalID, err.Error())
				continue
			}
			pushBody = pr.Title
		}

		// todo: send image url here
		if err = s.publisher.PublishJSON(ctx, events.SubjectPushCreated, events.PushPayload{
			Title:  pushTitle,
			Body:   pushBody,
			UserID: subscriberID,
		}); err != nil {
			log.Error().
				Err(err).
				Fields(map[string]string{"user_id": subscriberID.String()}).
				Msg("publish push notification")
		}
	}

	return nil
}

func convertAction(action Action) string {
	switch action {
	case ProposalCreated:
		return "New proposal created"
	case ProposalVotingQuorumReached:
		return "Quorum reached"
	case ProposalVotingEndsSoon:
		return "Vote finishes soon"
	}

	return ""
}

func (s *Service) MarkAsReadByID(ctx context.Context, subscriberID uuid.UUID, id ...uuid.UUID) error {
	return s.repo.MarkAsReadByID(ctx, subscriberID, id...)
}

func (s *Service) MarkAsUnreadByID(ctx context.Context, subscriberID uuid.UUID, id ...uuid.UUID) error {
	return s.repo.MarkAsUnreadByID(ctx, subscriberID, id...)
}

func (s *Service) MarkAsReadByTime(ctx context.Context, subscriberID uuid.UUID, t time.Time) error {
	// dirty fix to mark as read without counting nanoseconds
	t = t.Add(time.Second)

	return s.repo.MarkAsReadByTime(ctx, subscriberID, t)
}

func (s *Service) MarkAsArchivedByID(ctx context.Context, subscriberID uuid.UUID, id ...uuid.UUID) error {
	return s.repo.MarkAsArchivedByID(ctx, subscriberID, id...)
}

func (s *Service) MarkAsUnarchivedByID(ctx context.Context, subscriberID uuid.UUID, id ...uuid.UUID) error {
	return s.repo.MarkAsUnarchivedByID(ctx, subscriberID, id...)
}

func (s *Service) MarkAsArchivedByTime(ctx context.Context, subscriberID uuid.UUID, t time.Time) error {
	return s.repo.MarkAsArchivedByTime(ctx, subscriberID, t)
}

func (s *Service) FindByFilters(ctx context.Context, subscriberID uuid.UUID, filters []Filter) ([]Item, error) {
	filters = append(filters, FilterBySubscriberID(subscriberID))

	list, err := s.repo.FindByFilters(ctx, filters)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (s *Service) HasFeed(ctx context.Context, subscriberID uuid.UUID) (has bool, err error) {
	filters := []Filter{
		FilterBySubscriberID(subscriberID),
		WithLimit(1, 0),
	}

	result, err := s.repo.FindByFilters(ctx, filters)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return len(result) > 0, nil
}

func (s *Service) CountByFilters(ctx context.Context, subscriberID uuid.UUID, filters []Filter) (count int64, err error) {
	filters = append(filters, FilterBySubscriberID(subscriberID))

	found, err := s.repo.CountByFilters(ctx, filters)
	if err != nil {
		return 0, err
	}

	return found, nil
}

func (s *Service) Subscribe(ctx context.Context, subscriberID, daoID uuid.UUID) error {
	df, err := s.getDaoFeed(ctx, daoID)
	if err != nil {
		return fmt.Errorf("getDaoFeed: %w", err)
	}
	if df == nil {
		return nil
	}

	subscriberFeed := append([]feed.Item{}, df.Items...)
	sort.Slice(subscriberFeed, func(i, j int) bool {
		return subscriberFeed[i].CreatedAt.After(subscriberFeed[j].CreatedAt)
	})

	for _, item := range subscriberFeed {
		if err := s.repo.CreateOrUpdate(convertCoreFeedItemToInternal(subscriberID, item)); err != nil {
			log.Error().Err(err).Str("feed_id", item.ID.String()).Msg("unable to save feed")

			continue
		}
	}

	return nil
}

func (s *Service) getDaoFeed(ctx context.Context, daoID uuid.UUID) (*feed.Feed, error) {
	daoFeed, err := s.sdk.GetFeedByFilters(ctx, coresdk.FeedByFiltersRequest{
		IsActive: helpers.Ptr(true),
		DaoList:  []string{daoID.String()},
		Types:    []string{"proposal"},
		Limit:    maxPrefillElements,
	})
	if err != nil {
		return nil, fmt.Errorf("get feed by filters: %w", err)
	}

	if daoFeed != nil && len(daoFeed.Items) > 0 {
		return daoFeed, nil
	}

	daoFeed, err = s.sdk.GetFeedByFilters(ctx, coresdk.FeedByFiltersRequest{
		IsActive: helpers.Ptr(false),
		DaoList:  []string{daoID.String()},
		Types:    []string{"proposal"},
		Limit:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("get feed by filters: %w", err)
	}

	return daoFeed, nil
}

func convertCoreFeedItemToInternal(subscriberID uuid.UUID, item feed.Item) *Item {
	var timeline Timeline
	err := json.Unmarshal(item.Timeline, &timeline)
	if err != nil {
		log.Warn().Err(err).Str("feed_id", item.ID.String()).Msg("unable to unmarshal feed timeline")
	}

	return &Item{
		ID:           item.ID,
		SubscriberID: subscriberID,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    time.Now(),
		DaoID:        item.DaoID,
		ProposalID:   item.ProposalID,
		DiscussionID: item.DiscussionID,
		Type:         Type(item.Type),
		Action:       Action(item.Action),
		Snapshot:     item.Snapshot,
		Timeline:     timeline,
	}
}
