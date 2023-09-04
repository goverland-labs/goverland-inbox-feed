package feed

import (
	"encoding/json"
	"reflect"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	Dao      Type = "dao"
	Proposal Type = "proposal"

	DaoCreated                  Action = "dao.created"
	DaoUpdated                  Action = "dao.updated"
	ProposalCreated             Action = "proposal.created"
	ProposalUpdated             Action = "proposal.updated"
	ProposalVotingStartsSoon    Action = "proposal.voting.starts_soon"
	ProposalVotingEndsSoon      Action = "proposal.voting.ends_soon"
	ProposalVotingStarted       Action = "proposal.voting.started"
	ProposalVotingQuorumReached Action = "proposal.voting.quorum_reached"
	ProposalVotingEnded         Action = "proposal.voting.ended"
)

type Type string

type Action string

type TimelineInfo struct {
	CreatedAt time.Time `json:"created_at"`
	Action    Action    `json:"action"`
}

type Timeline []TimelineInfo

func (t *Timeline) Sort() {
	if t == nil || len(*t) == 0 {
		return
	}

	sort.SliceStable(*t, func(i, j int) bool {
		return (*t)[i].CreatedAt.Before((*t)[j].CreatedAt)
	})
}

func (t Timeline) Equal(updated Timeline) bool {
	if len(t) != len(updated) {
		return false
	}

	t.Sort()
	updated.Sort()

	return reflect.DeepEqual(t, updated)
}

type Item struct {
	ID           uuid.UUID `gorm:"primary_key" json:"id"`
	SubscriberID uuid.UUID `gorm:"primary_key;uniqueIndex:feed_item_dao_proposal_uidx"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt  `gorm:"index"`
	ReadAt       *time.Time      `json:"read_at" gorm:"index"`
	ArchivedAt   *time.Time      `json:"archived_at" gorm:"index"`
	DaoID        uuid.UUID       `json:"dao_id" gorm:"uniqueIndex:feed_item_dao_proposal_uidx"`
	ProposalID   string          `json:"proposal_id" gorm:"uniqueIndex:feed_item_dao_proposal_uidx"`
	DiscussionID string          `json:"discussion_id"`
	Type         Type            `json:"type"`
	Action       Action          `json:"action"`
	Snapshot     json.RawMessage `gorm:"type:jsonb;serializer:json" json:"dao,omitempty"`
	Timeline     Timeline        `gorm:"type:jsonb;serializer:json" json:"timeline"`
}

func (i Item) DAO() bool {
	return i.ProposalID == "" && i.DiscussionID == ""
}

func (i Item) AllowSending() bool {
	switch i.Action {
	case ProposalCreated,
		ProposalVotingQuorumReached,
		ProposalVotingEndsSoon:
		return true
	}

	return false
}
