package models

import (
	"context"
)

type Inviter interface {
	InvitePerson(ctx context.Context, i Invite, phase Phase, p Person) (*Invitee, error)
}

type Strategy interface {
	Kind() StrategyKind
	Execute(ctx context.Context, invite Invite, phase Phase) error
	Resume(ctx context.Context, invite Invite, phase Phase, state *PhaseState) error
	HandleEvent(ctx context.Context, invite Invite, phase Phase, state *PhaseState, event Event) error
	Progress(state *PhaseState) string
}
