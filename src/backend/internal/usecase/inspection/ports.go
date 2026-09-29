package inspection

import (
	"context"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type EvidenceImage struct {
	Kind        string
	ContentType string
	Data        []byte
}

type EvidenceInput struct {
	Title       string
	Description string
	WorkType    string
	Comment     string
	Images      []EvidenceImage
}

type EvidenceAnalyzer interface {
	Analyze(context.Context, EvidenceInput) (domain.EvidenceAnalysis, error)
}

type Delivery struct {
	Request domain.InspectionRequested
	Ack     func() error
}

type Subscriber interface {
	SubscribeInspections(context.Context) (<-chan Delivery, error)
}

type EvidenceReader interface {
	Read(context.Context, string, int64) ([]byte, string, error)
}

type AnalysisCache interface {
	SaveAnalysis(domain.EvidenceAnalysis)
}

type AnalysisPersistence interface {
	SaveAnalysis(context.Context, domain.EvidenceAnalysis) error
}
