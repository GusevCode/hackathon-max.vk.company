package inspection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type Config struct {
	Timeout       time.Duration
	RetryBackoff  time.Duration
	MaxImages     int
	MaxImageBytes int64
	PromptVersion string
}

type Service struct {
	subscriber Subscriber
	analyzer   EvidenceAnalyzer
	evidence   EvidenceReader
	cache      AnalysisCache
	storage    AnalysisPersistence
	logger     *slog.Logger
	config     Config
}

func NewService(subscriber Subscriber, analyzer EvidenceAnalyzer, evidence EvidenceReader, cache AnalysisCache, logger *slog.Logger, config Config, storages ...AnalysisPersistence) *Service {
	var storage AnalysisPersistence
	if len(storages) > 0 {
		storage = storages[0]
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = 500 * time.Millisecond
	}
	return &Service{subscriber: subscriber, analyzer: analyzer, evidence: evidence, cache: cache, storage: storage, logger: logger, config: config}
}

func (s *Service) Run(ctx context.Context) error {
	deliveries, err := s.subscriber.SubscribeInspections(ctx)
	if err != nil {
		return fmt.Errorf("subscribe inspections: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-deliveries:
			if !ok {
				return ctx.Err()
			}
			shouldAck, processErr := s.process(ctx, delivery.Request)
			if processErr != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				s.logger.Warn("analyze task evidence", "error", processErr, "inspection_id", delivery.Request.ID, "task_id", delivery.Request.TaskID)
			}
			if shouldAck && delivery.Ack != nil {
				if ackErr := delivery.Ack(); ackErr != nil {
					s.logger.Warn("ack inspection request", "error", ackErr, "inspection_id", delivery.Request.ID, "task_id", delivery.Request.TaskID)
				}
			}
		}
	}
}

func (s *Service) process(ctx context.Context, request domain.InspectionRequested) (bool, error) {
	analysis := domain.EvidenceAnalysis{
		ID: request.ID, TaskID: request.TaskID, SubmissionID: request.SubmissionID,
		Status: domain.AnalysisFailed, Recommendation: domain.RecommendationUnknown,
		PromptVersion: s.config.PromptVersion, RequestedAt: request.RequestedAt,
	}
	if len(request.Images) == 0 {
		analysis.ErrorCode = "no_images"
		return s.finish(ctx, analysis, fmt.Errorf("inspection has no images"))
	}
	if len(request.Images) > s.config.MaxImages {
		analysis.ErrorCode = "too_many_images"
		return s.finish(ctx, analysis, fmt.Errorf("inspection has %d images, limit is %d", len(request.Images), s.config.MaxImages))
	}

	hash := sha256.New()
	_, _ = io.WriteString(hash, request.Title)
	_, _ = io.WriteString(hash, request.Description)
	_, _ = io.WriteString(hash, request.WorkType)
	_, _ = io.WriteString(hash, request.Comment)
	input := EvidenceInput{Title: request.Title, Description: request.Description, WorkType: request.WorkType, Comment: request.Comment}
	for _, image := range request.Images {
		data, _, readErr := s.evidence.Read(ctx, image.ObjectKey, s.config.MaxImageBytes)
		if readErr != nil {
			analysis.ErrorCode = "read_evidence"
			return s.finish(ctx, analysis, fmt.Errorf("read evidence %q: %w", image.ObjectKey, readErr))
		}
		contentType := normalizeImageContentType(data)
		if !supportedImageType(contentType) {
			analysis.ErrorCode = "unsupported_image"
			return s.finish(ctx, analysis, fmt.Errorf("unsupported evidence content type %q", contentType))
		}
		_, _ = hash.Write(data)
		input.Images = append(input.Images, EvidenceImage{Kind: image.Kind, ContentType: contentType, Data: data})
	}
	analysis.InputHash = hex.EncodeToString(hash.Sum(nil))

	result, analyzeErr := s.analyzeWithRetry(ctx, input)
	if analyzeErr != nil {
		analysis.ErrorCode = errorCode(analyzeErr)
		analysis.CompletedAt = time.Now()
		return s.finish(ctx, analysis, analyzeErr)
	}
	result.ID = request.ID
	result.TaskID = request.TaskID
	result.SubmissionID = request.SubmissionID
	result.Status = domain.AnalysisSucceeded
	result.PromptVersion = s.config.PromptVersion
	result.InputHash = analysis.InputHash
	result.RequestedAt = request.RequestedAt
	result.CompletedAt = time.Now()
	return s.finish(ctx, result, nil)
}

func (s *Service) analyzeWithRetry(ctx context.Context, input EvidenceInput) (domain.EvidenceAnalysis, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		requestCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		result, err := s.analyzer.Analyze(requestCtx, input)
		cancel()
		if err == nil {
			return result, nil
		}
		lastErr = err
		if attempt == 1 || !isTemporary(err) {
			break
		}
		timer := time.NewTimer(s.config.RetryBackoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return domain.EvidenceAnalysis{}, ctx.Err()
		case <-timer.C:
		}
	}
	return domain.EvidenceAnalysis{}, lastErr
}

func (s *Service) finish(ctx context.Context, analysis domain.EvidenceAnalysis, processErr error) (bool, error) {
	if analysis.Status == domain.AnalysisFailed && analysis.CompletedAt.IsZero() {
		analysis.CompletedAt = time.Now()
	}
	if saveErr := s.save(ctx, analysis); saveErr != nil {
		if processErr != nil {
			return false, errors.Join(processErr, saveErr)
		}
		return false, saveErr
	}
	return true, processErr
}

func (s *Service) save(ctx context.Context, analysis domain.EvidenceAnalysis) error {
	s.cache.SaveAnalysis(analysis)
	if s.storage != nil {
		if err := s.storage.SaveAnalysis(ctx, analysis); err != nil {
			return fmt.Errorf("persist evidence analysis %q: %w", analysis.ID, err)
		}
	}
	return nil
}

func normalizeImageContentType(data []byte) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0]))
}

func supportedImageType(contentType string) bool {
	return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/webp" || contentType == "image/gif"
}

type temporaryError interface {
	Temporary() bool
}

type codedError interface {
	Code() string
}

func isTemporary(err error) bool {
	var temporary temporaryError
	return errors.As(err, &temporary) && temporary.Temporary()
}

func errorCode(err error) string {
	var coded codedError
	if errors.As(err, &coded) {
		return coded.Code()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "analyzer_error"
}
