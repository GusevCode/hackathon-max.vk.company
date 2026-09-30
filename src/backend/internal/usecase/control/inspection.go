package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/usecase/access"
)

func (s *Service) requestInspection(ctx context.Context, task domain.Task, after []domain.Evidence) {
	if s.inspections == nil || len(after) == 0 {
		return
	}
	request := domain.InspectionRequested{
		ID: newCode(), TaskID: task.ID, SubmissionID: task.SubmissionID,
		Title: task.Title, Description: task.Description, Comment: task.Comment, RequestedAt: time.Now(),
	}
	for _, workType := range s.repo.WorkTypes(task.OrganizationID) {
		if workType.ID == task.WorkTypeID {
			request.WorkType = workType.Name
			break
		}
	}
	if evidence, ok := latestEvidence(s.repo.Evidences(task.ID), "before"); ok {
		request.Images = append(request.Images, domain.InspectionImage{Kind: evidence.Kind, ObjectKey: evidence.ObjectKey})
	}
	if evidence, ok := latestEvidence(after, "after"); ok {
		request.Images = append(request.Images, domain.InspectionImage{Kind: evidence.Kind, ObjectKey: evidence.ObjectKey})
	}
	pending := domain.EvidenceAnalysis{
		ID: request.ID, TaskID: task.ID, SubmissionID: task.SubmissionID,
		Status: domain.AnalysisPending, Recommendation: domain.RecommendationUnknown,
		PromptVersion: s.promptVersion, RequestedAt: request.RequestedAt,
	}
	s.repo.SaveAnalysis(pending)
	if s.storage != nil {
		if err := s.storage.SaveAnalysis(ctx, pending); err != nil {
			s.logger.Warn("persist pending evidence analysis", "error", err, "inspection_id", request.ID, "task_id", task.ID)
		}
	}
	if err := s.inspections.PublishInspection(ctx, request); err != nil {
		pending.Status = domain.AnalysisFailed
		pending.ErrorCode = "publish_failed"
		pending.CompletedAt = time.Now()
		s.repo.SaveAnalysis(pending)
		if s.storage != nil {
			if persistErr := s.storage.SaveAnalysis(ctx, pending); persistErr != nil {
				s.logger.Warn("persist failed evidence analysis", "error", persistErr, "inspection_id", request.ID, "task_id", task.ID)
			}
		}
		s.logger.Warn("publish inspection request", "error", err, "inspection_id", request.ID, "task_id", task.ID)
	}
}

// latestEvidence keeps the inspection request small and deterministic: when
// several photos of the same kind are attached, only the most recently saved
// one is sent to the AI analyzer. If timestamps are equal, the later item in
// the repository result wins, which preserves upload order in the in-memory
// repository and for photos from the same upload batch.
func latestEvidence(evidences []domain.Evidence, kind string) (domain.Evidence, bool) {
	var latest domain.Evidence
	found := false
	for _, evidence := range evidences {
		if evidence.Kind != kind {
			continue
		}
		if !found || !evidence.CreatedAt.Before(latest.CreatedAt) {
			latest = evidence
			found = true
		}
	}
	return latest, found
}

func (s *Service) reanalyzeTask(ctx context.Context, event domain.Event, user domain.User, task domain.Task) error {
	if s.inspections == nil || task.Status != domain.TaskSubmitted || !access.CanReviewTask(user, task) {
		return s.sendHome(ctx, event.ChatID, user, "Повторный анализ доступен только для задания на проверке.")
	}
	after := make([]domain.Evidence, 0)
	for _, evidence := range s.repo.Evidences(task.ID) {
		if evidence.Kind == "after" && evidence.SubmissionID == task.SubmissionID {
			after = append(after, evidence)
		}
	}
	if len(after) == 0 {
		return s.sendHome(ctx, event.ChatID, user, "Для повторного анализа не найдены фотографии после выполнения.")
	}
	s.requestInspection(ctx, task, after)
	return s.taskCard(ctx, event, user, task)
}

func (s *Service) analysisText(task domain.Task) string {
	analysis, ok := s.repo.Analysis(task.ID)
	if !ok {
		return ""
	}
	if analysis.SubmissionID != task.SubmissionID {
		return "\n\n🤖 ИИ-анализ: результат относится к другой отправке фотографий."
	}
	switch analysis.Status {
	case domain.AnalysisPending:
		return "\n\n🤖 ИИ-анализ: выполняется."
	case domain.AnalysisFailed:
		return "\n\n🤖 ИИ-анализ: выполнить анализ не удалось. Проверьте фотографии вручную."
	case domain.AnalysisSucceeded:
		lines := []string{
			"\n\n🤖 Предварительный ИИ-анализ",
			"Рекомендация: " + recommendationName(analysis.Recommendation),
			fmt.Sprintf("Уверенность: %.0f%%", analysis.Confidence*100),
		}
		appendItems := func(label string, values []string) {
			if len(values) == 0 {
				return
			}
			lines = append(lines, label)
			for index, value := range values {
				if index == 5 {
					lines = append(lines, "… и другие наблюдения")
					break
				}
				lines = append(lines, "• "+shortAnalysisText(value))
			}
		}
		appendItems("Видимые признаки:", analysis.Observations)
		appendItems("Недостающие требования:", analysis.MissingRequirements)
		appendItems("Проверьте вручную:", analysis.Questions)
		lines = append(lines, "ИИ не принимает решение. Итог подтверждает руководитель.")
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}

func recommendationName(recommendation domain.AnalysisRecommendation) string {
	switch recommendation {
	case domain.RecommendationApprove:
		return "можно принять"
	case domain.RecommendationRework:
		return "нужна переделка"
	default:
		return "нужна ручная проверка"
	}
}

func shortAnalysisText(value string) string {
	const maxRunes = 240
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "…"
}
