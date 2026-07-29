package authorization

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Role string

const (
	RoleAdmin      Role = "ADMIN"
	RoleRecruiter  Role = "RECRUITER"
	RoleConsultant Role = "CONSULTANT"
)

type Scope string

const (
	ScopeOwn Scope = "own"
	ScopeAny Scope = "any"
)

const (
	ResumeCreate = "resume:create"
	ResumeView   = "resume:view"
	ResumeUpdate = "resume:update"
	ResumeDelete = "resume:delete"
	JobCreate    = "job:create"
	JobView      = "job:view"
	JobUpdate    = "job:update"
	JobDelete    = "job:delete"
)

type UserContext struct {
	UserID string
	Role   Role
}

type Grant struct {
	Action string
	Scope  Scope
}

var ErrForbidden = errors.New("forbidden")

var RolePermissions = map[Role][]Grant{
	RoleAdmin: {
		{Action: "resume:*", Scope: ScopeAny},
		{Action: "job:*", Scope: ScopeAny},
	},
	RoleRecruiter: {
		{Action: ResumeView, Scope: ScopeAny},
		{Action: JobCreate, Scope: ScopeOwn},
		{Action: JobView, Scope: ScopeOwn},
		{Action: JobUpdate, Scope: ScopeOwn},
		{Action: JobDelete, Scope: ScopeOwn},
	},
	RoleConsultant: {
		{Action: ResumeCreate, Scope: ScopeOwn},
		{Action: ResumeView, Scope: ScopeOwn},
		{Action: ResumeUpdate, Scope: ScopeOwn},
		{Action: ResumeDelete, Scope: ScopeOwn},
		{Action: JobView, Scope: ScopeAny},
	},
}

func GrantFor(role Role, action string) (Grant, bool) {
	for _, grant := range RolePermissions[role] {
		if grant.Action == action || (strings.HasSuffix(grant.Action, ":*") &&
			strings.HasPrefix(action, strings.TrimSuffix(grant.Action, "*"))) {
			return grant, true
		}
	}
	return Grant{}, false
}

func User(c *gin.Context) (UserContext, bool) {
	value, ok := c.Get("user")
	if !ok {
		return UserContext{}, false
	}
	user, ok := value.(UserContext)
	return user, ok
}

func RequirePermission(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := User(c)
		if !ok {
			log.Println("RequirePermission() error: ", http.StatusUnauthorized, "missing authenticated user")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
			return
		}
		grant, ok := GrantFor(user.Role, action)
		if !ok {
			log.Println("RequirePermission() error: ", http.StatusForbidden, "forbidden for user", user.UserID, "with role", user.Role, "for action", action)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Set("authorizationGrant", grant)
		c.Next()
	}
}

func CanAccess(user UserContext, action, ownerID string) error {
	grant, ok := GrantFor(user.Role, action)
	if !ok {
		return ErrForbidden
	}
	if grant.Scope == ScopeOwn && (ownerID == "" || ownerID != user.UserID) {
		return ErrForbidden
	}
	return nil
}
