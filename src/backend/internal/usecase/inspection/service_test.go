package inspection

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type fakeAnalyzer struct {
	result domain.EvidenceAnalysis
	calls  int
}

func (f *fakeAnalyzer) Analyze(context.Context, EvidenceInput) (domain.EvidenceAnalysis, error) {
	f.calls++
	return f.result, nil
}

type fakeEvidenceReader struct{}

func (fakeEvidenceReader) Read(context.Context, string, int64) ([]byte, string, error) {
	return []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}, "image/jpeg", nil
}

type fakeAnalysisCache struct {
	analysis domain.EvidenceAnalysis
}

func (f *fakeAnalysisCache) SaveAnalysis(analysis domain.EvidenceAnalysis) {
	f.analysis = analysis
}

func TestProcessStoresSuccessfulAnalysis(t *testing.T) {
	analyzer := &fakeAnalyzer{result: domain.EvidenceAnalysis{
		Relevant: true, Quality: domain.EvidenceQualityUsable,
		Recommendation: domain.RecommendationApprove, Confidence: 0.8, Model: "openai/gpt-4o",
	}}
	cache := &fakeAnalysisCache{}
	service := NewService(nil, analyzer, fakeEvidenceReader{}, cache, slog.Default(), Config{
		Timeout: time.Second, MaxImages: 4, MaxImageBytes: 1024, PromptVersion: "v1",
	})
	requestedAt := time.Now()
	shouldAck, err := service.process(context.Background(), domain.InspectionRequested{
		ID: "inspection-1", TaskID: "task-1", SubmissionID: "submission-1",
		Title: "Уборка лифта", Images: []domain.InspectionImage{{Kind: "after", ObjectKey: "tasks/task-1/after.jpg"}},
		RequestedAt: requestedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !shouldAck {
		t.Fatal("process() should acknowledge a persisted analysis")
	}
	if analyzer.calls != 1 || cache.analysis.Status != domain.AnalysisSucceeded {
		t.Fatalf("analysis was not completed: %#v", cache.analysis)
	}
	if cache.analysis.TaskID != "task-1" || cache.analysis.SubmissionID != "submission-1" || cache.analysis.InputHash == "" {
		t.Fatalf("analysis metadata was not populated: %#v", cache.analysis)
	}
}

func TestProcessRejectsTooManyImagesBeforeAnalyzerCall(t *testing.T) {
	analyzer := &fakeAnalyzer{}
	cache := &fakeAnalysisCache{}
	service := NewService(nil, analyzer, fakeEvidenceReader{}, cache, slog.Default(), Config{
		Timeout: time.Second, MaxImages: 1, MaxImageBytes: 1024, PromptVersion: "v1",
	})
	shouldAck, err := service.process(context.Background(), domain.InspectionRequested{
		ID: "inspection-1", TaskID: "task-1", SubmissionID: "submission-1",
		Images:      []domain.InspectionImage{{ObjectKey: "one"}, {ObjectKey: "two"}},
		RequestedAt: time.Now(),
	})
	if err == nil {
		t.Fatal("process() error = nil, want image limit error")
	}
	if !shouldAck {
		t.Fatal("process() should acknowledge a persisted terminal error")
	}
	if analyzer.calls != 0 || cache.analysis.ErrorCode != "too_many_images" {
		t.Fatalf("unexpected failed analysis: %#v", cache.analysis)
	}
}

type failingAnalysisPersistence struct{}

func (failingAnalysisPersistence) SaveAnalysis(context.Context, domain.EvidenceAnalysis) error {
	return errors.New("storage unavailable")
}

func TestProcessDoesNotAcknowledgeWhenPersistenceFails(t *testing.T) {
	analyzer := &fakeAnalyzer{result: domain.EvidenceAnalysis{
		Quality: domain.EvidenceQualityUsable, Recommendation: domain.RecommendationUnknown,
	}}
	service := NewService(nil, analyzer, fakeEvidenceReader{}, &fakeAnalysisCache{}, slog.Default(), Config{
		Timeout: time.Second, MaxImages: 4, MaxImageBytes: 1024, PromptVersion: "v1",
	}, failingAnalysisPersistence{})
	shouldAck, err := service.process(context.Background(), domain.InspectionRequested{
		ID: "inspection-1", TaskID: "task-1", Images: []domain.InspectionImage{{Kind: "after", ObjectKey: "photo"}}, RequestedAt: time.Now(),
	})
	if err == nil || shouldAck {
		t.Fatalf("process() = (%t, %v), want persistence error without acknowledgement", shouldAck, err)
	}
}
