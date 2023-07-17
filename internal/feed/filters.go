package feed

import (
	"database/sql"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Filter func(query *gorm.DB) *gorm.DB

func FilterBySubscriberID(id uuid.UUID) Filter {
	var (
		dummy Item
		_     = dummy.SubscriberID
	)

	return func(query *gorm.DB) *gorm.DB {
		return query.Where("subscriber_id = @subscriber_id", sql.Named("subscriber_id", id))
	}
}

func FilterByArchivedStatus(status *bool) Filter {
	var (
		dummy Item
		_     = dummy.ArchivedAt
	)

	return func(query *gorm.DB) *gorm.DB {
		if status == nil {
			return query
		}

		if *status {
			return query.Where("archived_at is not null")
		}

		return query.Where("archived_at is null")
	}
}

func FilterByReadStatus(status *bool) Filter {
	var (
		dummy Item
		_     = dummy.ReadAt
	)

	return func(query *gorm.DB) *gorm.DB {
		if status == nil {
			return query
		}

		if *status {
			return query.Where("read_at is not null")
		}

		return query.Where("read_at is null")
	}
}

func WithLimit(limit, offset int) Filter {
	return func(query *gorm.DB) *gorm.DB {
		return query.Offset(offset).Limit(limit)
	}
}

func SortedByCreatedAtDesc() Filter {
	var (
		dummy Item
		_     = dummy.CreatedAt
	)

	return func(query *gorm.DB) *gorm.DB {
		return query.Order("created_at desc")
	}
}

func SortedByUpdatedAtDesc() Filter {
	var (
		dummy Item
		_     = dummy.UpdatedAt
	)

	return func(query *gorm.DB) *gorm.DB {
		return query.Order("updated_at desc")
	}
}
