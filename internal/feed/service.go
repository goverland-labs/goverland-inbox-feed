package feed

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/goverland-labs/inbox-api/protobuf/inboxapi"
	"google.golang.org/grpc"
)

type SubscriptionsFinder interface {
	FindSubscribers(ctx context.Context, in *inboxapi.FindSubscribersRequest, opts ...grpc.CallOption) (*inboxapi.UserList, error)
}

type Service struct {
	repo          *Repo
	subscriptions SubscriptionsFinder
}

func NewService(repo *Repo, subscriptions SubscriptionsFinder) *Service {
	return &Service{
		repo:          repo,
		subscriptions: subscriptions,
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

	return int64(found), nil
}
