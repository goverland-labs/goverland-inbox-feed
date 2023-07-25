package feed

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	coresdk "github.com/goverland-labs/core-web-sdk"
	"github.com/goverland-labs/core-web-sdk/dao"
	"github.com/goverland-labs/inbox-api/protobuf/inboxapi"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

const maxPrefillElements = 10

type SubscriptionsFinder interface {
	FindSubscribers(ctx context.Context, in *inboxapi.FindSubscribersRequest, opts ...grpc.CallOption) (*inboxapi.UserList, error)
	ListSubscriptions(ctx context.Context, in *inboxapi.ListSubscriptionRequest, opts ...grpc.CallOption) (*inboxapi.ListSubscriptionResponse, error)
}

type Service struct {
	repo          *Repo
	subscriptions SubscriptionsFinder
	sdk           *coresdk.Client
}

func NewService(repo *Repo, subscriptions SubscriptionsFinder, sdk *coresdk.Client) *Service {
	return &Service{
		repo:          repo,
		subscriptions: subscriptions,
		sdk:           sdk,
	}
}

func (s *Service) Process(ctx context.Context, item Item) error {
	resp, err := s.subscriptions.FindSubscribers(ctx, &inboxapi.FindSubscribersRequest{
		DaoId: item.DaoID.String(),
	})
	if err != nil {
		return err
	}

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

		// TODO: Send push notification
	}

	return nil
}

func (s *Service) MarkAsReadByID(ctx context.Context, subscriberID uuid.UUID, id ...uuid.UUID) error {
	return s.repo.MarkAsReadByID(ctx, subscriberID, id...)
}

func (s *Service) MarkAsUnreadByID(ctx context.Context, subscriberID uuid.UUID, id ...uuid.UUID) error {
	return s.repo.MarkAsUnreadByID(ctx, subscriberID, id...)
}

func (s *Service) MarkAsReadByTime(ctx context.Context, subscriberID uuid.UUID, t time.Time) error {
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

func (s *Service) CountByFilters(ctx context.Context, subscriberID uuid.UUID, filters []Filter) (count int64, err error) {
	filters = append(filters, FilterBySubscriberID(subscriberID))

	found, err := s.repo.CountByFilters(ctx, filters)
	if err != nil {
		return 0, err
	}

	return found, nil
}

func (s *Service) PrefillFeed(ctx context.Context, subscriberID uuid.UUID) error {
	subscriptions, err := s.subscriptions.ListSubscriptions(ctx, &inboxapi.ListSubscriptionRequest{
		SubscriberId: subscriberID.String(),
	})
	if err != nil {
		return err
	}

	var subscriberFeed []dao.FeedItem

	// TODO: Get feed by multiple DAOs
	for _, sub := range subscriptions.GetItems() {
		daoId, err := uuid.Parse(sub.GetDaoId())
		if err != nil {
			log.Warn().Err(err).Str("dao_id", sub.GetDaoId()).Msg("unable to parse dao id")
			continue
		}

		daoFeed, err := s.sdk.GetDaoFeed(ctx, daoId, coresdk.GetDaoFeedRequest{Limit: maxPrefillElements})
		if err != nil {
			log.Warn().Err(err).Str("dao_id", sub.GetDaoId()).Msg("unable to fetch dao feed")
			continue
		}
		subscriberFeed = append(subscriberFeed, daoFeed.Items...)
	}

	sort.Slice(subscriberFeed, func(i, j int) bool {
		return subscriberFeed[i].CreatedAt.After(subscriberFeed[j].CreatedAt)
	})

	if len(subscriberFeed) > maxPrefillElements {
		subscriberFeed = subscriberFeed[:maxPrefillElements]
	}

	for _, item := range subscriberFeed {
		err := s.repo.CreateOrUpdate(convertCoreFeedItemToInternal(subscriberID, item))
		if err != nil {
			log.Error().Err(err).Str("feed_id", item.ID.String()).Msg("unable to save feed")
			continue
		}
	}

	return nil
}

func convertCoreFeedItemToInternal(subscriberID uuid.UUID, item dao.FeedItem) *Item {
	var timeline Timeline
	err := json.Unmarshal(item.Timeline, &timeline)
	if err != nil {
		log.Warn().Err(err).Str("feed_id", item.ID.String()).Msg("unable to unmarshal feed timeline")
	}

	return &Item{
		ID:           item.ID,
		SubscriberID: subscriberID,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
		DaoID:        item.DaoID,
		ProposalID:   item.ProposalID,
		DiscussionID: item.DiscussionID,
		Type:         Type(item.Type),
		Action:       Action(item.Action),
		Snapshot:     item.Snapshot,
		Timeline:     timeline,
	}
}
