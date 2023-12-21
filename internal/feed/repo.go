package feed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repo struct {
	conn *gorm.DB
}

func NewRepo(conn *gorm.DB) *Repo {
	return &Repo{conn: conn}
}

func (r *Repo) CreateOrUpdate(item *Item) error {
	var (
		_ = item.DaoID
		_ = item.ProposalID
		_ = item.Snapshot
		_ = item.Action
		_ = item.Timeline
		_ = item.CreatedAt
		_ = item.UpdatedAt
	)

	// nolint:godox
	// FIXME: Reset readAt if item was updated
	// FIXME: Don't react if archivedAt is not null

	tx := r.conn.Begin()

	var found Item

	query := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("dao_id = @dao_id and proposal_id = @proposal_id", sql.Named("dao_id", item.DaoID), sql.Named("proposal_id", item.ProposalID)).
		First(&found)

	if query.Error != nil && !errors.Is(query.Error, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return query.Error
	}

	timeline, err := json.Marshal(item.Timeline)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("unalbe to marshal timeline: %w", err)
	}

	cl := clause.OnConflict{
		Columns: []clause.Column{{Name: "subscriber_id"}, {Name: "dao_id"}, {Name: "proposal_id"}},
		DoUpdates: clause.Set{
			{Column: clause.Column{Name: "snapshot"}, Value: item.Snapshot},
			{Column: clause.Column{Name: "timeline"}, Value: timeline},
			{Column: clause.Column{Name: "action"}, Value: item.Action},
			{Column: clause.Column{Name: "created_at"}, Value: item.CreatedAt},
			{Column: clause.Column{Name: "updated_at"}, Value: time.Now()},
		},
	}

	query = tx.Debug().Clauses(cl).Create(item)

	if query.Error != nil {
		tx.Rollback()
		return query.Error
	}

	return tx.Commit().Error
}

func (r *Repo) MarkAsReadByID(_ context.Context, subscriberID uuid.UUID, id ...uuid.UUID) error {
	var (
		dummy Item
		_     = dummy.SubscriberID
		_     = dummy.ReadAt
	)

	now := time.Now()
	err := r.conn.
		Model(&Item{}).
		Where("subscriber_id = @subscriber_id", sql.Named("subscriber_id", subscriberID)).
		Where("id in (@ids)", sql.Named("ids", id)).
		Update("read_at", now).
		Error

	return err
}

func (r *Repo) MarkAsUnreadByID(_ context.Context, subscriberID uuid.UUID, id ...uuid.UUID) error {
	var (
		dummy Item
		_     = dummy.SubscriberID
		_     = dummy.ReadAt
	)

	err := r.conn.
		Model(&Item{}).
		Where("subscriber_id = @subscriber_id", sql.Named("subscriber_id", subscriberID)).
		Where("id in (@ids)", sql.Named("ids", id)).
		Update("read_at", gorm.Expr("NULL")).
		Error

	return err
}

func (r *Repo) MarkAsReadByTime(_ context.Context, subscriberID uuid.UUID, t time.Time) error {
	var (
		dummy Item
		_     = dummy.SubscriberID
		_     = dummy.ReadAt
		_     = dummy.CreatedAt
	)

	now := time.Now()
	err := r.conn.
		Model(&Item{}).
		Where("subscriber_id = @subscriber_id", sql.Named("subscriber_id", subscriberID)).
		Where("updated_at <= @before", sql.Named("before", t)).
		Update("read_at", now).
		Error

	return err
}

func (r *Repo) MarkAsArchivedByID(_ context.Context, subscriberID uuid.UUID, id ...uuid.UUID) error {
	var (
		dummy Item
		_     = dummy.SubscriberID
		_     = dummy.ArchivedAt
	)

	now := time.Now()
	err := r.conn.
		Model(&Item{}).
		Where("subscriber_id = @subscriber_id", sql.Named("subscriber_id", subscriberID)).
		Where("id in (@ids)", sql.Named("ids", id)).
		Update("archived_at", now).
		Error

	return err
}

func (r *Repo) MarkAsUnarchivedByID(_ context.Context, subscriberID uuid.UUID, id ...uuid.UUID) error {
	var (
		dummy Item
		_     = dummy.SubscriberID
		_     = dummy.ArchivedAt
	)

	err := r.conn.
		Model(&Item{}).
		Where("subscriber_id = @subscriber_id", sql.Named("subscriber_id", subscriberID)).
		Where("id in (@ids)", sql.Named("ids", id)).
		Update("archived_at", gorm.Expr("NULL")).
		Error

	return err
}

func (r *Repo) MarkAsArchivedByTime(_ context.Context, subscriberID uuid.UUID, t time.Time) error {
	var (
		dummy Item
		_     = dummy.SubscriberID
		_     = dummy.CreatedAt
		_     = dummy.ArchivedAt
	)

	now := time.Now()
	err := r.conn.
		Model(&Item{}).
		Where("subscriber_id = @subscriber_id", sql.Named("subscriber_id", subscriberID)).
		Where("created_at <= @before", sql.Named("before", t)).
		Update("archived_at", now).
		Error

	return err
}

func (r *Repo) CountByFilters(_ context.Context, filters []Filter) (int64, error) {
	query := r.conn.Model(&Item{})
	for _, f := range filters {
		f(query)
	}

	var count int64
	err := query.Count(&count).Error

	return count, err
}

func (r *Repo) FindByFilters(_ context.Context, filters []Filter) ([]Item, error) {
	query := r.conn.Model(&Item{})
	for _, f := range filters {
		f(query)
	}

	var list []Item
	err := query.Find(&list).Error

	return list, err
}

func (r *Repo) AutoArchive(_ context.Context) error {
	var (
		dummy Item
		_     = dummy.ArchivedAt
		_     = dummy.Snapshot
	)

	return r.conn.Exec(`
		UPDATE items
		SET archived_at = now()
		WHERE archived_at IS NULL
		  AND created_at < now() - INTERVAL '7 day'
		  AND to_timestamp((snapshot -> 'end')::double precision) < now() - INTERVAL '7 day'
`).Error
}
