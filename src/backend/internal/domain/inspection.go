package domain

import "time"

type AnalysisStatus string

const (
	AnalysisPending   AnalysisStatus = "pending"
	AnalysisSucceeded AnalysisStatus = "succeeded"
	AnalysisFailed    AnalysisStatus = "failed"
)

type EvidenceQuality string

const (
	EvidenceQualityUsable   EvidenceQuality = "usable"
	EvidenceQualityPoor     EvidenceQuality = "poor"
	EvidenceQualityUnusable EvidenceQuality = "unusable"
)

type AnalysisRecommendation string

const (
	RecommendationApprove AnalysisRecommendation = "approve"
	RecommendationRework  AnalysisRecommendation = "rework"
	RecommendationUnknown AnalysisRecommendation = "unknown"
)

type InspectionImage struct {
	Kind      string
	ObjectKey string
}

type InspectionRequested struct {
	ID           string
	TaskID       string
	SubmissionID string
	Title        string
	Description  string
	WorkType     string
	Comment      string
	Images       []InspectionImage
	RequestedAt  time.Time
}

type EvidenceAnalysis struct {
	ID                  string
	TaskID              string
	SubmissionID        string
	Status              AnalysisStatus
	Relevant            bool
	Quality             EvidenceQuality
	Observations        []string
	MissingRequirements []string
	CommentSummary      string
	Recommendation      AnalysisRecommendation
	Confidence          float64
	Questions           []string
	Model               string
	Provider            string
	PromptVersion       string
	InputHash           string
	PromptTokens        int
	CompletionTokens    int
	TotalTokens         int
	CostRUB             float64
	ErrorCode           string
	RequestedAt         time.Time
	CompletedAt         time.Time
}
