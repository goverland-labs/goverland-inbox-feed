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
	"github.com/rs/zerolog/log"
	"go.openly.dev/pointy"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	"github.com/goverland-labs/inbox-feed/pkg/helpers"
)

const (
	maxPrefillElements = 200
)

type SubscriptionsFinder interface {
	FindSubscribers(ctx context.Context, in *inboxapi.FindSubscribersRequest, opts ...grpc.CallOption) (*inboxapi.UserList, error)
	ListSubscriptions(ctx context.Context, in *inboxapi.ListSubscriptionRequest, opts ...grpc.CallOption) (*inboxapi.ListSubscriptionResponse, error)
}

type SettingsProvider interface {
	GetFeedSettings(ctx context.Context, in *inboxapi.GetFeedSettingsRequest, opts ...grpc.CallOption) (*inboxapi.GetFeedSettingsResponse, error)
}

type Service struct {
	repo          *Repo
	subscriptions SubscriptionsFinder
	settings      SettingsProvider
	sdk           *coresdk.Client
}

func NewService(repo *Repo, subscriptions SubscriptionsFinder, sp SettingsProvider, sdk *coresdk.Client) *Service {
	return &Service{
		repo:          repo,
		subscriptions: subscriptions,
		settings:      sp,
		sdk:           sdk,
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

	for _, sub := range resp.Users {
		subscriberID, err := uuid.Parse(sub.GetUserId())
		if err != nil {
			return fmt.Errorf("unable to parse subscriber id '%s': %w", sub.GetUserId(), err)
		}

		var prInfo ShortProposalInfo
		err = json.Unmarshal(item.Snapshot, &prInfo)
		if err != nil {
			return fmt.Errorf("unmarshal snapshot: %w", err)
		}

		list, err := s.repo.FindByFilters(context.Background(), []Filter{
			FilterBySubscriberID(subscriberID),
			FilterByProposalID(item.ProposalID),
		})

		if err != nil {
			return fmt.Errorf("find by filters: %w", err)
		}

		// do not add item if it already closed
		if len(list) == 0 && !prInfo.Active() {
			return nil
		}

		personalized := item
		personalized.SubscriberID = subscriberID

		if personalized.CreatedAt.IsZero() {
			personalized.CreatedAt = time.Now()
		}

		if err := s.repo.CreateOrUpdate(&personalized); err != nil {
			return fmt.Errorf("unable to save feed item '%s' for subscriber '%s': %w", personalized.ID, sub.GetUserId(), err)
		}
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
	// dirty fix to mark as read without counting nanoseconds
	t = t.Add(time.Second)

	return s.repo.MarkAsReadByTime(ctx, subscriberID, t)
}

func (s *Service) MarkAsUnreadByTime(ctx context.Context, subscriberID uuid.UUID, t time.Time) error {
	// dirty fix to mark as read without counting nanoseconds
	t = t.Add(-time.Second)

	return s.repo.MarkAsUnreadByTime(ctx, subscriberID, t)
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

	return daoFeed, nil
}

func (s *Service) markExpiredAsAutoArchived(ctx context.Context) error {
	err := s.repo.AutoArchive(ctx)
	if err != nil {
		return fmt.Errorf("s.repo.AutoArchive: %w", err)
	}

	return nil
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

func (s *Service) TryAutoarchive(ctx context.Context, userID uuid.UUID, proposalID string) error {
	set, err := s.settings.GetFeedSettings(ctx, &inboxapi.GetFeedSettingsRequest{
		UserId: userID.String(),
	})
	if err != nil {
		return fmt.Errorf("get feed settings: %w", err)
	}

	// skip if it's disabled by user config
	if !set.GetFeedSettings().GetArchiveProposalAfterVote() {
		return nil
	}

	items, err := s.repo.FindByFilters(ctx, []Filter{
		FilterBySubscriberID(userID),
		FilterByArchivedStatus(pointy.Bool(false)),
		FilterByProposalID(proposalID),
	})
	if err != nil {
		return fmt.Errorf("find by filters: %w", err)
	}

	if len(items) == 0 {
		log.Warn().Msgf("can't find [%s] feed item by proposal id %s", userID, proposalID)
		return nil
	}

	if len(items) > 1 {
		log.Warn().Msgf("find [%s] few feed item by proposal id %s", userID, proposalID)
	}

	if err = s.repo.MarkAsReadByID(ctx, userID, items[0].ID); err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}

	if err = s.repo.MarkAsArchivedByID(ctx, userID, items[0].ID); err != nil {
		return fmt.Errorf("mark as archived: %w", err)
	}

	return nil
}

func (s *Service) SaveSettings(ctx context.Context, subscriber uuid.UUID, autoarchiveAfterDays int) error {
	set, err := s.repo.GetFeedSettings(ctx, subscriber)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("get feed settings: %w", err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		set = &Settings{
			SubscriberID: subscriber,
		}
	}

	set.AutoarchiveAfterDays = autoarchiveAfterDays
	if err = s.repo.StoreSettings(ctx, set); err != nil {
		return fmt.Errorf("store settings: %w", err)
	}

	return nil
}
