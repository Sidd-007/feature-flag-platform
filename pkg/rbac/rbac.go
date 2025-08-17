package rbac

import (
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

// RBAC handles role-based access control using Casbin
type RBAC struct {
	enforcer *casbin.Enforcer
}

// Subject represents an entity that can perform actions (user, API token, service)
type Subject struct {
	ID   string
	Type string // "user", "api_token", "service"
}

// Object represents a resource that can be acted upon
type Object struct {
	Type      string // "org", "project", "env", "flag", "experiment", "segment"
	ID        string
	OrgID     string
	ProjectID string
	EnvID     string
}

// Action represents an action that can be performed
type Action string

const (
	ActionCreate  Action = "create"
	ActionRead    Action = "read"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionPublish Action = "publish"
	ActionStart   Action = "start"
	ActionStop    Action = "stop"
	ActionManage  Action = "manage"
)

// Role represents a role in the system
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// NewRBAC creates a new RBAC instance with default policies
func NewRBAC() (*RBAC, error) {
	// Casbin model configuration
	modelText := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && regexMatch(r.act, p.act)
`

	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	rbac := &RBAC{
		enforcer: enforcer,
	}

	// Add default policies
	if err := rbac.loadDefaultPolicies(); err != nil {
		return nil, fmt.Errorf("failed to load default policies: %w", err)
	}

	return rbac, nil
}

// loadDefaultPolicies loads the default RBAC policies
func (r *RBAC) loadDefaultPolicies() error {
	// Owner permissions - can do everything in their organization
	policies := [][]string{
		// Organization level permissions for owners
		{"role:owner", "org:*", "create|read|update|delete|manage"},
		{"role:owner", "project:*", "create|read|update|delete|manage"},
		{"role:owner", "env:*", "create|read|update|delete|manage"},
		{"role:owner", "flag:*", "create|read|update|delete|publish|manage"},
		{"role:owner", "segment:*", "create|read|update|delete|manage"},
		{"role:owner", "experiment:*", "create|read|update|delete|start|stop|manage"},
		{"role:owner", "user:*", "create|read|update|delete|manage"},
		{"role:owner", "token:*", "create|read|update|delete|manage"},
		{"role:owner", "audit:*", "read"},
		{"role:owner", "analytics:*", "read"},

		// Admin permissions - similar to owner but cannot delete org
		{"role:admin", "org:*", "read|update|manage"},
		{"role:admin", "project:*", "create|read|update|delete|manage"},
		{"role:admin", "env:*", "create|read|update|delete|manage"},
		{"role:admin", "flag:*", "create|read|update|delete|publish|manage"},
		{"role:admin", "segment:*", "create|read|update|delete|manage"},
		{"role:admin", "experiment:*", "create|read|update|delete|start|stop|manage"},
		{"role:admin", "user:*", "create|read|update|manage"},
		{"role:admin", "token:*", "create|read|update|delete|manage"},
		{"role:admin", "audit:*", "read"},
		{"role:admin", "analytics:*", "read"},

		// Editor permissions - can modify flags and experiments
		{"role:editor", "org:*", "read"},
		{"role:editor", "project:*", "read|update"},
		{"role:editor", "env:*", "read|update"},
		{"role:editor", "flag:*", "create|read|update|publish"},
		{"role:editor", "segment:*", "create|read|update"},
		{"role:editor", "experiment:*", "create|read|update|start|stop"},
		{"role:editor", "analytics:*", "read"},

		// Viewer permissions - read-only access
		{"role:viewer", "org:*", "read"},
		{"role:viewer", "project:*", "read"},
		{"role:viewer", "env:*", "read"},
		{"role:viewer", "flag:*", "read"},
		{"role:viewer", "segment:*", "read"},
		{"role:viewer", "experiment:*", "read"},
		{"role:viewer", "analytics:*", "read"},

		// API token scopes
		{"scope:read", "org:*", "read"},
		{"scope:read", "project:*", "read"},
		{"scope:read", "env:*", "read"},
		{"scope:read", "flag:*", "read"},
		{"scope:read", "segment:*", "read"},
		{"scope:read", "experiment:*", "read"},
		{"scope:read", "analytics:*", "read"},

		{"scope:write", "org:*", "read"},
		{"scope:write", "project:*", "read"},
		{"scope:write", "env:*", "read"},
		{"scope:write", "flag:*", "read|update|publish"},
		{"scope:write", "segment:*", "read|update"},
		{"scope:write", "experiment:*", "read|update"},
		{"scope:write", "analytics:*", "read"},

		{"scope:admin", "org:*", "read|update"},
		{"scope:admin", "project:*", "create|read|update|delete"},
		{"scope:admin", "env:*", "create|read|update|delete"},
		{"scope:admin", "flag:*", "create|read|update|delete|publish"},
		{"scope:admin", "segment:*", "create|read|update|delete"},
		{"scope:admin", "experiment:*", "create|read|update|delete|start|stop"},
		{"scope:admin", "user:*", "read|manage"},
		{"scope:admin", "token:*", "create|read|update|delete"},
		{"scope:admin", "analytics:*", "read"},

		// Service permissions - full access for internal services
		{"service:*", "*:*", "create|read|update|delete|publish|start|stop|manage"},
	}

	for _, policy := range policies {
		if _, err := r.enforcer.AddPolicy(policy); err != nil {
			return fmt.Errorf("failed to add policy %v: %w", policy, err)
		}
	}

	return nil
}

// Enforce checks if a subject can perform an action on an object
func (r *RBAC) Enforce(subject Subject, object Object, action Action) (bool, error) {
	subjectStr := r.formatSubject(subject)
	objectStr := r.formatObject(object)
	actionStr := string(action)

	allowed, err := r.enforcer.Enforce(subjectStr, objectStr, actionStr)
	if err != nil {
		return false, fmt.Errorf("enforcement error: %w", err)
	}

	return allowed, nil
}

// AssignRole assigns a role to a subject for a specific organization
func (r *RBAC) AssignRole(subject Subject, role Role, orgID string) error {
	subjectStr := r.formatSubject(subject)
	roleStr := r.formatRole(role, orgID)

	if _, err := r.enforcer.AddRoleForUser(subjectStr, roleStr); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

// RemoveRole removes a role from a subject for a specific organization
func (r *RBAC) RemoveRole(subject Subject, role Role, orgID string) error {
	subjectStr := r.formatSubject(subject)
	roleStr := r.formatRole(role, orgID)

	if _, err := r.enforcer.DeleteRoleForUser(subjectStr, roleStr); err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	return nil
}

// GetRolesForUser gets all roles for a subject
func (r *RBAC) GetRolesForUser(subject Subject) ([]string, error) {
	subjectStr := r.formatSubject(subject)
	roles, err := r.enforcer.GetRolesForUser(subjectStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}

	return roles, nil
}

// GetUsersForRole gets all users with a specific role
func (r *RBAC) GetUsersForRole(role Role, orgID string) ([]string, error) {
	roleStr := r.formatRole(role, orgID)
	users, err := r.enforcer.GetUsersForRole(roleStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get users for role: %w", err)
	}

	return users, nil
}

// HasRole checks if a subject has a specific role in an organization
func (r *RBAC) HasRole(subject Subject, role Role, orgID string) (bool, error) {
	subjectStr := r.formatSubject(subject)
	roleStr := r.formatRole(role, orgID)

	hasRole, err := r.enforcer.HasRoleForUser(subjectStr, roleStr)
	if err != nil {
		return false, fmt.Errorf("failed to check role: %w", err)
	}

	return hasRole, nil
}

// AddPolicy adds a custom policy
func (r *RBAC) AddPolicy(subject, object, action string) error {
	if _, err := r.enforcer.AddPolicy(subject, object, action); err != nil {
		return fmt.Errorf("failed to add policy: %w", err)
	}
	return nil
}

// RemovePolicy removes a custom policy
func (r *RBAC) RemovePolicy(subject, object, action string) error {
	if _, err := r.enforcer.RemovePolicy(subject, object, action); err != nil {
		return fmt.Errorf("failed to remove policy: %w", err)
	}
	return nil
}

// GetPolicies returns all policies
func (r *RBAC) GetPolicies() [][]string {
	return r.enforcer.GetPolicy()
}

// SavePolicy saves the current policy to persistent storage
func (r *RBAC) SavePolicy() error {
	return r.enforcer.SavePolicy()
}

// LoadPolicy loads policy from persistent storage
func (r *RBAC) LoadPolicy() error {
	return r.enforcer.LoadPolicy()
}

// formatSubject formats a subject for use in Casbin
func (r *RBAC) formatSubject(subject Subject) string {
	return fmt.Sprintf("%s:%s", subject.Type, subject.ID)
}

// formatObject formats an object for use in Casbin
func (r *RBAC) formatObject(object Object) string {
	parts := []string{object.Type}

	// Build hierarchical object identifier
	if object.OrgID != "" {
		parts = append(parts, object.OrgID)
		if object.ProjectID != "" {
			parts = append(parts, object.ProjectID)
			if object.EnvID != "" {
				parts = append(parts, object.EnvID)
			}
		}
	}

	if object.ID != "" {
		parts = append(parts, object.ID)
	}

	return strings.Join(parts, ":")
}

// formatRole formats a role for use in Casbin
func (r *RBAC) formatRole(role Role, orgID string) string {
	return fmt.Sprintf("role:%s:%s", string(role), orgID)
}

// CheckPermission is a convenience method that combines common checks
func (r *RBAC) CheckPermission(subjectType, subjectID, objectType, objectID, orgID, projectID, envID string, action Action) (bool, error) {
	subject := Subject{
		ID:   subjectID,
		Type: subjectType,
	}

	object := Object{
		Type:      objectType,
		ID:        objectID,
		OrgID:     orgID,
		ProjectID: projectID,
		EnvID:     envID,
	}

	return r.Enforce(subject, object, action)
}

// CanUserAccessFlag checks if a user can access a specific flag
func (r *RBAC) CanUserAccessFlag(userID, flagID, envID, projectID, orgID string, action Action) (bool, error) {
	return r.CheckPermission("user", userID, "flag", flagID, orgID, projectID, envID, action)
}

// CanAPITokenAccessFlag checks if an API token can access a specific flag
func (r *RBAC) CanAPITokenAccessFlag(tokenID, flagID, envID, projectID, orgID string, action Action) (bool, error) {
	return r.CheckPermission("api_token", tokenID, "flag", flagID, orgID, projectID, envID, action)
}

// CanUserManageOrg checks if a user can manage an organization
func (r *RBAC) CanUserManageOrg(userID, orgID string) (bool, error) {
	return r.CheckPermission("user", userID, "org", orgID, orgID, "", "", ActionManage)
}

// GetUserOrgRole gets a user's role in a specific organization
func (r *RBAC) GetUserOrgRole(userID, orgID string) (Role, error) {
	subject := Subject{
		ID:   userID,
		Type: "user",
	}

	roles, err := r.GetRolesForUser(subject)
	if err != nil {
		return "", err
	}

	// Look for role in this organization
	rolePrefix := fmt.Sprintf("role:")
	orgSuffix := fmt.Sprintf(":%s", orgID)

	for _, role := range roles {
		if strings.HasPrefix(role, rolePrefix) && strings.HasSuffix(role, orgSuffix) {
			// Extract role name: role:admin:org123 -> admin
			parts := strings.Split(role, ":")
			if len(parts) >= 3 {
				return Role(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("no role found for user %s in org %s", userID, orgID)
}

// ValidateRole validates if a role string is valid
func (r *RBAC) ValidateRole(role string) bool {
	switch Role(role) {
	case RoleOwner, RoleAdmin, RoleEditor, RoleViewer:
		return true
	default:
		return false
	}
}

// ValidateAction validates if an action string is valid
func (r *RBAC) ValidateAction(action string) bool {
	switch Action(action) {
	case ActionCreate, ActionRead, ActionUpdate, ActionDelete,
		ActionPublish, ActionStart, ActionStop, ActionManage:
		return true
	default:
		return false
	}
}

// GetPermissionsForRole returns all permissions for a given role
func (r *RBAC) GetPermissionsForRole(role Role) map[string][]Action {
	permissions := make(map[string][]Action)

	switch role {
	case RoleOwner:
		permissions["org"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionManage}
		permissions["project"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionManage}
		permissions["env"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionManage}
		permissions["flag"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionPublish, ActionManage}
		permissions["experiment"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionStart, ActionStop, ActionManage}
		permissions["segment"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionManage}
		permissions["user"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionManage}
		permissions["token"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionManage}
		permissions["audit"] = []Action{ActionRead}
		permissions["analytics"] = []Action{ActionRead}

	case RoleAdmin:
		permissions["org"] = []Action{ActionRead, ActionUpdate, ActionManage}
		permissions["project"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionManage}
		permissions["env"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionManage}
		permissions["flag"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionPublish, ActionManage}
		permissions["experiment"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionStart, ActionStop, ActionManage}
		permissions["segment"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionManage}
		permissions["user"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionManage}
		permissions["token"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionManage}
		permissions["audit"] = []Action{ActionRead}
		permissions["analytics"] = []Action{ActionRead}

	case RoleEditor:
		permissions["org"] = []Action{ActionRead}
		permissions["project"] = []Action{ActionRead, ActionUpdate}
		permissions["env"] = []Action{ActionRead, ActionUpdate}
		permissions["flag"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionPublish}
		permissions["experiment"] = []Action{ActionCreate, ActionRead, ActionUpdate, ActionStart, ActionStop}
		permissions["segment"] = []Action{ActionCreate, ActionRead, ActionUpdate}
		permissions["analytics"] = []Action{ActionRead}

	case RoleViewer:
		permissions["org"] = []Action{ActionRead}
		permissions["project"] = []Action{ActionRead}
		permissions["env"] = []Action{ActionRead}
		permissions["flag"] = []Action{ActionRead}
		permissions["experiment"] = []Action{ActionRead}
		permissions["segment"] = []Action{ActionRead}
		permissions["analytics"] = []Action{ActionRead}
	}

	return permissions
}
