package feed

import (
	"encoding/json"
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
	ProposalVotingStarted       Action = "proposal.voting.started"
	ProposalVotingQuorumReached Action = "proposal.voting.quorum_reached"
	ProposalVotingEnded         Action = "proposal.voting.ended"
)

type Type string

type Action string

type Timeline struct {
	CreatedAt time.Time `json:"created_at"`
	Action    Action    `json:"action"`
}

type Item struct {
	ID           uuid.UUID `gorm:"primary_key" json:"id"`
	SubscriberID uuid.UUID `gorm:"index"`
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
	Timeline     []Timeline      `gorm:"type:jsonb;serializer:json" json:"timeline"`
}
