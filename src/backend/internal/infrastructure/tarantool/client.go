package tarantool

import (
	"context"
	"fmt"
	"time"

	"github.com/GusevCode/hackathon-max.vk.company/src/backend/internal/domain"
	tnt "github.com/tarantool/go-tarantool/v2"
)

type Client struct{ conn *tnt.Connection }

func (c *Client) Users(ctx context.Context) ([]domain.User, error) {
	var rows [][]interface{}
	err := c.conn.Do(tnt.NewSelectRequest("users").Index("primary").Iterator(tnt.IterAll).Context(ctx)).GetTyped(&rows)
	if err != nil {
		return nil, err
	}
	users := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		if len(row) < 7 {
			continue
		}
		roles := make([]domain.Role, 0)
		switch rawRoles := row[4].(type) {
		case []interface{}:
			for _, rawRole := range rawRoles {
				if role, ok := rawRole.(string); ok {
					roles = append(roles, domain.Role(role))
				}
			}
		case []string:
			for _, role := range rawRoles {
				roles = append(roles, domain.Role(role))
			}
		}
		managerID := ""
		if len(row) > 7 && row[7] != nil {
			managerID = stringValue(row[7])
		}
		users = append(users, domain.User{ID: stringValue(row[0]), OrganizationID: stringValue(row[1]), MaxUserID: int64Value(row[2]), DisplayName: stringValue(row[3]), Roles: roles, Status: domain.UserStatus(stringValue(row[5])), CreatedAt: time.Unix(int64Value(row[6]), 0), ManagerID: managerID})
	}
	return users, nil
}

func (c *Client) Tasks(ctx context.Context) ([]domain.Task, error) {
	var rows [][]interface{}
	err := c.conn.Do(tnt.NewSelectRequest("tasks").Index("primary").Iterator(tnt.IterAll).Context(ctx)).GetTyped(&rows)
	if err != nil {
		return nil, err
	}
	tasks := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		if len(row) < 10 {
			continue
		}
		tasks = append(tasks, domain.Task{ID: stringValue(row[0]), OrganizationID: stringValue(row[1]), Title: stringValue(row[2]), Description: stringValue(row[3]), ObjectID: stringValue(row[4]), AssigneeID: stringValue(row[5]), ManagerID: stringValue(row[6]), Status: domain.TaskStatus(stringValue(row[7])), DueAt: time.Unix(int64Value(row[8]), 0), CreatedAt: time.Unix(int64Value(row[9]), 0)})
	}
	return tasks, nil
}

func (c *Client) Invites(ctx context.Context) ([]domain.Invite, error) {
	var rows [][]interface{}
	err := c.conn.Do(tnt.NewSelectRequest("invites").Index("primary").Iterator(tnt.IterAll).Context(ctx)).GetTyped(&rows)
	if err != nil {
		return nil, err
	}
	invites := make([]domain.Invite, 0, len(rows))
	for _, row := range rows {
		if len(row) < 6 {
			continue
		}
		roles := make([]domain.Role, 0)
		switch rawRoles := row[2].(type) {
		case []interface{}:
			for _, rawRole := range rawRoles {
				if role, ok := rawRole.(string); ok {
					roles = append(roles, domain.Role(role))
				}
			}
		case []string:
			for _, role := range rawRoles {
				roles = append(roles, domain.Role(role))
			}
		}
		invites = append(invites, domain.Invite{Code: stringValue(row[0]), OrganizationID: stringValue(row[1]), Roles: roles, ManagerID: stringValue(row[3]), ExpiresAt: time.Unix(int64Value(row[4]), 0), UsedBy: int64Value(row[5])})
	}
	return invites, nil
}

func (c *Client) Objects(ctx context.Context) ([]domain.Object, error) {
	var rows [][]interface{}
	err := c.conn.Do(tnt.NewSelectRequest("objects").Index("primary").Iterator(tnt.IterAll).Context(ctx)).GetTyped(&rows)
	if err != nil {
		return nil, err
	}
	objects := make([]domain.Object, 0, len(rows))
	for _, row := range rows {
		if len(row) >= 5 {
			objects = append(objects, domain.Object{ID: stringValue(row[0]), OrganizationID: stringValue(row[1]), Name: stringValue(row[2]), Address: stringValue(row[3]), Kind: stringValue(row[4])})
		}
	}
	return objects, nil
}

func (c *Client) WorkTypes(ctx context.Context) ([]domain.WorkType, error) {
	var rows [][]interface{}
	err := c.conn.Do(tnt.NewSelectRequest("work_types").Index("primary").Iterator(tnt.IterAll).Context(ctx)).GetTyped(&rows)
	if err != nil {
		return nil, err
	}
	workTypes := make([]domain.WorkType, 0, len(rows))
	for _, row := range rows {
		if len(row) >= 3 {
			workTypes = append(workTypes, domain.WorkType{ID: stringValue(row[0]), OrganizationID: stringValue(row[1]), Name: stringValue(row[2])})
		}
	}
	return workTypes, nil
}

func Connect(ctx context.Context, address, user, password string) (*Client, error) {
	dialer := tnt.NetDialer{Address: address, User: user, Password: password}
	connection, err := tnt.Connect(ctx, dialer, tnt.Opts{Timeout: 3 * time.Second, Reconnect: time.Second, MaxReconnects: 3})
	if err != nil {
		return nil, fmt.Errorf("connect Tarantool: %w", err)
	}
	return &Client{conn: connection}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("tarantool client is not initialized")
	}
	_, err := c.conn.Do(tnt.NewPingRequest().Context(ctx)).Get()
	return err
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) SaveUser(ctx context.Context, user domain.User) error {
	roles := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = string(role)
	}
	if user.MaxUserID < 0 {
		return fmt.Errorf("max user id must be non-negative")
	}
	_, err := c.conn.Do(tnt.NewReplaceRequest("users").Tuple([]interface{}{user.ID, user.OrganizationID, uint64(user.MaxUserID), user.DisplayName, roles, string(user.Status), user.CreatedAt.Unix(), user.ManagerID}).Context(ctx)).Get()
	return err
}

func (c *Client) SaveTask(ctx context.Context, task domain.Task) error {
	_, err := c.conn.Do(tnt.NewReplaceRequest("tasks").Tuple([]interface{}{task.ID, task.OrganizationID, task.Title, task.Description, task.ObjectID, task.AssigneeID, task.ManagerID, string(task.Status), task.DueAt.Unix(), task.CreatedAt.Unix()}).Context(ctx)).Get()
	return err
}

func (c *Client) SaveEvidence(ctx context.Context, evidence domain.Evidence) error {
	_, err := c.conn.Do(tnt.NewReplaceRequest("evidence").Tuple([]interface{}{evidence.ID, evidence.TaskID, evidence.Kind, evidence.ObjectKey, "", evidence.CreatedAt.Unix()}).Context(ctx)).Get()
	return err
}

func (c *Client) SaveReview(ctx context.Context, review domain.Review) error {
	_, err := c.conn.Do(tnt.NewReplaceRequest("task_reviews").Tuple([]interface{}{newCode(), review.TaskID, review.ReviewerID, review.Decision, review.Comment, review.CreatedAt.Unix()}).Context(ctx)).Get()
	return err
}

func (c *Client) SaveObject(ctx context.Context, object domain.Object) error {
	_, err := c.conn.Do(tnt.NewReplaceRequest("objects").Tuple([]interface{}{object.ID, object.OrganizationID, object.Name, object.Address, object.Kind, time.Now().Unix()}).Context(ctx)).Get()
	return err
}

func (c *Client) SaveWorkType(ctx context.Context, workType domain.WorkType) error {
	_, err := c.conn.Do(tnt.NewReplaceRequest("work_types").Tuple([]interface{}{workType.ID, workType.OrganizationID, workType.Name, time.Now().Unix()}).Context(ctx)).Get()
	return err
}

func (c *Client) SaveInvite(ctx context.Context, invite domain.Invite) error {
	roles := make([]string, len(invite.Roles))
	for i, role := range invite.Roles {
		roles[i] = string(role)
	}
	if invite.UsedBy < 0 {
		return fmt.Errorf("invite used-by id must be non-negative")
	}
	_, err := c.conn.Do(tnt.NewReplaceRequest("invites").Tuple([]interface{}{invite.Code, invite.OrganizationID, roles, invite.ManagerID, invite.ExpiresAt.Unix(), uint64(invite.UsedBy)}).Context(ctx)).Get()
	return err
}

func (c *Client) ClearTasks(ctx context.Context, organizationID string) error {
	tasks, err := c.Tasks(ctx)
	if err != nil {
		return fmt.Errorf("load tasks for cleanup: %w", err)
	}
	taskIDs := make(map[string]struct{})
	for _, task := range tasks {
		if task.OrganizationID != organizationID {
			continue
		}
		taskIDs[task.ID] = struct{}{}
		if _, err := c.conn.Do(tnt.NewDeleteRequest("tasks").Index("primary").Key([]interface{}{task.ID}).Context(ctx)).Get(); err != nil {
			return fmt.Errorf("delete task %q: %w", task.ID, err)
		}
	}
	if err := c.deleteTaskRows(ctx, "evidence", taskIDs, 1); err != nil {
		return err
	}
	return c.deleteTaskRows(ctx, "task_reviews", taskIDs, 1)
}

func (c *Client) deleteTaskRows(ctx context.Context, space string, taskIDs map[string]struct{}, taskField int) error {
	if len(taskIDs) == 0 {
		return nil
	}
	var rows [][]interface{}
	if err := c.conn.Do(tnt.NewSelectRequest(space).Index("primary").Iterator(tnt.IterAll).Context(ctx)).GetTyped(&rows); err != nil {
		return fmt.Errorf("load %s for cleanup: %w", space, err)
	}
	for _, row := range rows {
		if len(row) <= taskField {
			continue
		}
		if _, ok := taskIDs[stringValue(row[taskField])]; !ok {
			continue
		}
		if _, err := c.conn.Do(tnt.NewDeleteRequest(space).Index("primary").Key([]interface{}{row[0]}).Context(ctx)).Get(); err != nil {
			return fmt.Errorf("delete %s row: %w", space, err)
		}
	}
	return nil
}

func newCode() string { return fmt.Sprintf("review-%d", time.Now().UnixNano()) }

func stringValue(value interface{}) string {
	if result, ok := value.(string); ok {
		return result
	}
	return fmt.Sprint(value)
}
func int64Value(value interface{}) int64 {
	switch result := value.(type) {
	case int64:
		return result
	case uint64:
		return int64(result)
	case int:
		return int64(result)
	case uint:
		return int64(result)
	case float64:
		return int64(result)
	default:
		return 0
	}
}
