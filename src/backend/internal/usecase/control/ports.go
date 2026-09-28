package control

import (
	"context"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type BotGateway interface {
	GetInfo(context.Context) (domain.BotInfo, error)
	Events(context.Context) <-chan domain.Event
	Send(context.Context, domain.OutgoingMessage) error
	AnswerCallback(context.Context, string, string) error
}

type Repository interface {
	UserByMaxID(int64) (domain.User, bool)
	SaveUser(domain.User)
	Users(string) []domain.User
	Invite(string) (domain.Invite, bool)
	SaveInvite(domain.Invite)
	Organization(string) (domain.Organization, bool)
	SaveOrganization(domain.Organization)
	Objects(string) []domain.Object
	SaveObject(domain.Object)
	WorkTypes(string) []domain.WorkType
	SaveWorkType(domain.WorkType)
	Task(string) (domain.Task, bool)
	SaveTask(domain.Task)
	Tasks(string) []domain.Task
	SaveReview(domain.Review)
}

type PhotoStore interface {
	UploadURL(context.Context, string, string, string) error
}

type NotificationPublisher interface {
	Publish(context.Context, domain.Notification) error
}

type Storage interface {
	SaveUser(context.Context, domain.User) error
	SaveTask(context.Context, domain.Task) error
	SaveEvidence(context.Context, domain.Evidence) error
	SaveReview(context.Context, domain.Review) error
	SaveObject(context.Context, domain.Object) error
	SaveWorkType(context.Context, domain.WorkType) error
	SaveInvite(context.Context, domain.Invite) error
}
