package control

import (
	"sync"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
)

type MemoryRepository struct {
	mu        sync.RWMutex
	users     map[int64]domain.User
	invites   map[string]domain.Invite
	orgs      map[string]domain.Organization
	objects   map[string]domain.Object
	workTypes map[string]domain.WorkType
	tasks     map[string]domain.Task
	reviews   []domain.Review
	evidence  map[string][]domain.Evidence
	analyses  map[string]domain.EvidenceAnalysis
}

func NewMemoryRepository(initialAdmin int64) *MemoryRepository {
	r := &MemoryRepository{users: map[int64]domain.User{}, invites: map[string]domain.Invite{}, orgs: map[string]domain.Organization{}, objects: map[string]domain.Object{}, workTypes: map[string]domain.WorkType{}, tasks: map[string]domain.Task{}, evidence: map[string][]domain.Evidence{}, analyses: map[string]domain.EvidenceAnalysis{}}
	r.orgs["system"] = domain.Organization{ID: "system", Name: "Основная организация"}
	r.users[initialAdmin] = domain.User{ID: "initial-admin", OrganizationID: "system", MaxUserID: initialAdmin, DisplayName: "Первый администратор", Roles: []domain.Role{domain.RoleAdmin}, Status: domain.UserActive}
	return r
}

func (r *MemoryRepository) UserByMaxID(id int64) (domain.User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	return u, ok
}
func (r *MemoryRepository) SaveUser(u domain.User) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.MaxUserID] = u
}
func (r *MemoryRepository) Users(org string) []domain.User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.User{}
	for _, u := range r.users {
		if u.OrganizationID == org {
			out = append(out, u)
		}
	}
	return out
}
func (r *MemoryRepository) Invite(code string) (domain.Invite, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.invites[code]
	return i, ok
}
func (r *MemoryRepository) SaveInvite(i domain.Invite) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invites[i.Code] = i
}
func (r *MemoryRepository) Invites() []domain.Invite {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Invite, 0, len(r.invites))
	for _, invite := range r.invites {
		out = append(out, invite)
	}
	return out
}
func (r *MemoryRepository) Organization(id string) (domain.Organization, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orgs[id]
	return o, ok
}
func (r *MemoryRepository) SaveOrganization(o domain.Organization) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orgs[o.ID] = o
}
func (r *MemoryRepository) Objects(org string) []domain.Object {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.Object{}
	for _, o := range r.objects {
		if o.OrganizationID == org {
			out = append(out, o)
		}
	}
	return out
}
func (r *MemoryRepository) SaveObject(o domain.Object) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.objects[o.ID] = o
}
func (r *MemoryRepository) WorkTypes(org string) []domain.WorkType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.WorkType{}
	for _, w := range r.workTypes {
		if w.OrganizationID == org {
			out = append(out, w)
		}
	}
	return out
}
func (r *MemoryRepository) SaveWorkType(w domain.WorkType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workTypes[w.ID] = w
}
func (r *MemoryRepository) Task(id string) (domain.Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tasks[id]
	return t, ok
}
func (r *MemoryRepository) SaveTask(t domain.Task) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[t.ID] = t
}
func (r *MemoryRepository) Tasks(org string) []domain.Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []domain.Task{}
	for _, t := range r.tasks {
		if t.OrganizationID == org {
			out = append(out, t)
		}
	}
	return out
}
func (r *MemoryRepository) SaveReview(v domain.Review) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reviews = append(r.reviews, v)
}

func (r *MemoryRepository) SaveEvidence(evidence domain.Evidence) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.evidence[evidence.TaskID] = append(r.evidence[evidence.TaskID], evidence)
}

func (r *MemoryRepository) Evidences(taskID string) []domain.Evidence {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]domain.Evidence(nil), r.evidence[taskID]...)
}

func (r *MemoryRepository) SaveAnalysis(analysis domain.EvidenceAnalysis) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, exists := r.analyses[analysis.TaskID]
	if !exists || current.ID == analysis.ID || !analysis.RequestedAt.Before(current.RequestedAt) {
		r.analyses[analysis.TaskID] = analysis
	}
}

func (r *MemoryRepository) Analysis(taskID string) (domain.EvidenceAnalysis, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	analysis, ok := r.analyses[taskID]
	return analysis, ok
}

func (r *MemoryRepository) ClearTasks(organizationID string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0)
	for id, task := range r.tasks {
		if task.OrganizationID != organizationID {
			continue
		}
		ids = append(ids, id)
		delete(r.tasks, id)
		delete(r.evidence, id)
		delete(r.analyses, id)
	}
	keptReviews := r.reviews[:0]
	for _, review := range r.reviews {
		remove := false
		for _, id := range ids {
			if review.TaskID == id {
				remove = true
				break
			}
		}
		if !remove {
			keptReviews = append(keptReviews, review)
		}
	}
	r.reviews = keptReviews
	return ids
}
