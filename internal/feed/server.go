package feed

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/goverland-labs/inbox-api/protobuf/inboxapi"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/goverland-labs/inbox-feed/pkg/helpers"
)

const defaultPageLimit = 100

type Server struct {
	inboxapi.UnimplementedFeedServer

	service *Service
}

func NewServer(service *Service) *Server {
	return &Server{
		service: service,
	}
}

func (s *Server) GetUserFeed(ctx context.Context, req *inboxapi.GetUserFeedRequest) (*inboxapi.FeedList, error) {
	subscriberID, err := uuid.Parse(req.GetSubscriberId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid subscriber id")
	}

	filters := []Filter{
		SortedByActuality(),
	}

	var unreadStateFilters []Filter

	switch req.GetReadState() {
	case inboxapi.GetUserFeedRequest_Exclude:
		unreadStateFilters = append(unreadStateFilters, FilterByReadStatus(helpers.Ptr(false)))
	case inboxapi.GetUserFeedRequest_ExcludeOther:
		unreadStateFilters = append(unreadStateFilters, FilterByReadStatus(helpers.Ptr(true)))
	default:
		// GetUserFeedRequest_Include is default behaviour
	}

	switch req.GetArchivedState() {
	case inboxapi.GetUserFeedRequest_Exclude:
		filters = append(filters, FilterByArchivedStatus(helpers.Ptr(false)))
	case inboxapi.GetUserFeedRequest_ExcludeOther:
		filters = append(filters, FilterByArchivedStatus(helpers.Ptr(true)))
	default:
		// GetUserFeedRequest_Include is default behaviour
	}

	var pageLimit = defaultPageLimit
	if req.GetLimit() > 0 {
		// TODO: Limit max value
		pageLimit = int(req.GetLimit())
	}

	totalCount, err := s.service.CountByFilters(ctx, subscriberID, append(filters, unreadStateFilters...))
	if err != nil {
		log.Error().Err(err).Msg("unable to get total count of feed events")
		return nil, status.Error(codes.Internal, "something went wrong")
	}

	var pageOffset int
	if req.GetOffset() > 0 {
		pageOffset = int(req.GetOffset())
	}

	filters = append(filters, WithLimit(pageLimit, pageOffset))

	unreadCount, err := s.service.CountByFilters(ctx, subscriberID, append(filters, FilterByReadStatus(helpers.Ptr(false))))
	if err != nil {
		log.Error().Err(err).Msg("unable to get unread count of feed events")
		return nil, status.Error(codes.Internal, "something went wrong")
	}

	list, err := s.service.FindByFilters(ctx, subscriberID, append(filters, unreadStateFilters...))
	if err != nil {
		log.Error().Err(err).Msg("unable to get user feed")
		return nil, status.Error(codes.Internal, "something went wrong")
	}

	resp := &inboxapi.FeedList{
		List:        convertToProto(list),
		TotalCount:  uint32(totalCount),
		UnreadCount: uint32(unreadCount),
	}

	return resp, nil
}

func (s *Server) MarkAsRead(ctx context.Context, req *inboxapi.MarkAsReadRequest) (*emptypb.Empty, error) {
	subscriberID, err := uuid.Parse(req.GetSubscriberId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid subscriber id")
	}

	if len(req.GetIds()) != 0 {
		ids, err := helpers.ConvertStringsToUUIDs(req.GetIds())
		if err != nil {
			log.Warn().Err(err).Strs("ids", req.GetIds()).Msg("unable convert strings to UUIDs")
			return nil, status.Error(codes.InvalidArgument, "invalid id format")
		}

		if err := s.service.MarkAsReadByID(ctx, subscriberID, ids...); err != nil {
			log.Warn().Err(err).Strs("ids", req.GetIds()).Msg("unable to mark as paid")
			return nil, status.Error(codes.Internal, "something went wrong")
		}

		return &emptypb.Empty{}, nil
	}

	if req.GetBefore() != nil {
		if err := s.service.MarkAsReadByTime(ctx, subscriberID, req.GetBefore().AsTime()); err != nil {
			log.Warn().Err(err).Any("before", req.GetBefore().AsTime()).Msg("unable to mark as paid")
			return nil, status.Error(codes.Internal, "something went wrong")
		}

		return &emptypb.Empty{}, nil
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) MarkAsUnread(ctx context.Context, req *inboxapi.MarkAsUnreadRequest) (*emptypb.Empty, error) {
	subscriberID, err := uuid.Parse(req.GetSubscriberId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid subscriber id")
	}

	if len(req.GetIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty id list")
	}

	ids, err := helpers.ConvertStringsToUUIDs(req.GetIds())
	if err != nil {
		log.Warn().Err(err).Strs("ids", req.GetIds()).Msg("unable convert strings to UUIDs")
		return nil, status.Error(codes.InvalidArgument, "invalid id format")
	}

	if err := s.service.MarkAsUnreadByID(ctx, subscriberID, ids...); err != nil {
		log.Warn().Err(err).Strs("ids", req.GetIds()).Msg("unable to mark as unread")
		return nil, status.Error(codes.Internal, "something went wrong")
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) MarkAsArchived(ctx context.Context, req *inboxapi.MarkAsArchivedRequest) (*emptypb.Empty, error) {
	subscriberID, err := uuid.Parse(req.GetSubscriberId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid subscriber id")
	}

	if len(req.GetIds()) != 0 {
		ids, err := helpers.ConvertStringsToUUIDs(req.GetIds())
		if err != nil {
			log.Warn().Err(err).Strs("ids", req.GetIds()).Msg("unable convert strings to UUIDs")
			return nil, status.Error(codes.InvalidArgument, "invalid id format")
		}

		if err := s.service.MarkAsArchivedByID(ctx, subscriberID, ids...); err != nil {
			log.Warn().Err(err).Strs("ids", req.GetIds()).Msg("unable to mark as arhived")
			return nil, status.Error(codes.Internal, "something went wrong")
		}

		return &emptypb.Empty{}, nil
	}

	if req.GetBefore() != nil {
		if err := s.service.MarkAsArchivedByTime(ctx, subscriberID, req.GetBefore().AsTime()); err != nil {
			log.Warn().Err(err).Any("before", req.GetBefore().AsTime()).Msg("unable to mark as archived")
			return nil, status.Error(codes.Internal, "something went wrong")

		}
		return &emptypb.Empty{}, nil
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) UserSubscribe(ctx context.Context, req *inboxapi.UserSubscribeRequest) (*emptypb.Empty, error) {
	subscriberID, err := uuid.Parse(req.GetSubscriberId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid subscriber id")
	}

	daoID, err := uuid.Parse(req.GetDaoId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid dao id")
	}

	if err = s.service.Subscribe(ctx, subscriberID, daoID); err != nil {
		log.Error().Err(err).Msgf("subscribe %s to %s", subscriberID.String(), daoID.String())

		return nil, status.Error(codes.Internal, "internal err")
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) MarkAsUnarchived(ctx context.Context, req *inboxapi.MarkAsUnarchivedRequest) (*emptypb.Empty, error) {
	subscriberID, err := uuid.Parse(req.GetSubscriberId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid subscriber id")
	}

	if len(req.GetIds()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty id list")
	}

	ids, err := helpers.ConvertStringsToUUIDs(req.GetIds())
	if err != nil {
		log.Warn().Err(err).Strs("ids", req.GetIds()).Msg("unable convert strings to UUIDs")
		return nil, status.Error(codes.InvalidArgument, "invalid id format")
	}

	if err := s.service.MarkAsUnarchivedByID(ctx, subscriberID, ids...); err != nil {
		log.Warn().Err(err).Strs("ids", req.GetIds()).Msg("unable to mark as unarchived")
		return nil, status.Error(codes.Internal, "something went wrong")
	}

	return &emptypb.Empty{}, nil
}

func convertToProto(list []Item) []*inboxapi.FeedItem {
	converted := make([]*inboxapi.FeedItem, 0, len(list))

	for _, item := range list {
		var proposalID *string
		if item.ProposalID != "" {
			proposalID = helpers.Ptr(item.ProposalID)
		}

		var discussionID *string
		if item.DiscussionID != "" {
			discussionID = helpers.Ptr(item.DiscussionID)
		}

		timeline, err := json.Marshal(item.Timeline)
		if err != nil {
			log.Warn().Err(err).Str("feed_id", item.ID.String()).Msg("unable to marshal timeline")
		}

		var readAt *timestamppb.Timestamp
		if item.ReadAt != nil {
			readAt = timestamppb.New(*item.ReadAt)
		}

		var archivedAt *timestamppb.Timestamp
		if item.ArchivedAt != nil {
			archivedAt = timestamppb.New(*item.ArchivedAt)
		}

		converted = append(converted, &inboxapi.FeedItem{
			Id:           item.ID.String(),
			CreatedAt:    timestamppb.New(item.CreatedAt),
			UpdatedAt:    timestamppb.New(item.UpdatedAt),
			ReadAt:       readAt,
			ArchivedAt:   archivedAt,
			DaoId:        item.DaoID.String(),
			ProposalId:   proposalID,
			DiscussionId: discussionID,
			Type:         string(item.Type),
			Action:       string(item.Action),
			Snapshot:     item.Snapshot,
			Timeline:     timeline,
		})
	}

	return converted
}
