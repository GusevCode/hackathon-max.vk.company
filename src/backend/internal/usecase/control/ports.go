package control

import (
	"context"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type BotGateway interface {
	GetInfo(context.Context) (domain.BotInfo, error)
	Events(context.Context) <-chan domain.Event
	Send(context.Context, domain.OutgoingMessage) (string, error)
	DeleteMessage(context.Context, string) error
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
	SaveEvidence(domain.Evidence)
	Evidences(string) []domain.Evidence
	SaveAnalysis(domain.EvidenceAnalysis)
	Analysis(string) (domain.EvidenceAnalysis, bool)
	ClearTasks(string) []string
}

type PhotoStore interface {
	UploadURL(context.Context, string, string, string) error
	Read(context.Context, string, int64) ([]byte, string, error)
	DeleteObject(context.Context, string) error
	DeleteAllTaskPhotos(context.Context) error
}

type NotificationPublisher interface {
	Publish(context.Context, domain.Notification) error
}

type InspectionPublisher interface {
	PublishInspection(context.Context, domain.InspectionRequested) error
}

type Storage interface {
	SaveUser(context.Context, domain.User) error
	SaveTask(context.Context, domain.Task) error
	SaveEvidence(context.Context, domain.Evidence) error
	DeleteEvidence(context.Context, string, string) error
	SaveReview(context.Context, domain.Review) error
	SaveAnalysis(context.Context, domain.EvidenceAnalysis) error
	SaveObject(context.Context, domain.Object) error
	SaveWorkType(context.Context, domain.WorkType) error
	SaveInvite(context.Context, domain.Invite) error
	ClearTasks(context.Context, string) error
}
