package control

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

func (s *Service) persistUser(ctx context.Context, user domain.User) {
	if s.storage != nil {
		if err := s.storage.SaveUser(ctx, user); err != nil {
			s.logger.Warn("persist user", "error", err, "user_id", user.MaxUserID)
		}
	}
}

func (s *Service) persistTask(ctx context.Context, task domain.Task) {
	if s.storage != nil {
		if err := s.storage.SaveTask(ctx, task); err != nil {
			s.logger.Warn("persist task", "error", err, "task_id", task.ID)
		}
	}
}

func (s *Service) persistPhotos(ctx context.Context, taskID, kind string, photos []domain.Photo) error {
	for index, photo := range photos {
		key := fmt.Sprintf("tasks/%s/%s/%d-%d.jpg", taskID, kind, time.Now().UnixNano(), index)
		if err := s.photos.UploadURL(ctx, key, photo.URL); err != nil {
			return err
		}
		if s.storage != nil {
			evidence := domain.Evidence{ID: newCode(), TaskID: taskID, Kind: kind, ObjectKey: key, CreatedAt: time.Now()}
			if err := s.storage.SaveEvidence(ctx, evidence); err != nil {
				return err
			}
		}
	}
	return nil
}

func newCode() string {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "LOCAL1"
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, 8)
	for i := range out {
		out[i] = alphabet[int(b[i%len(b)])%len(alphabet)]
	}
	return string(out)
}

func shortID(id string) string {
	if len(id) > 6 {
		return id[:6]
	}
	return id
}
