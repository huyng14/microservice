package authorization

import (
	"errors"
	"testing"
)

func TestGrantForSeparatesActionAndScope(t *testing.T) {
	tests := []struct {
		name   string
		role   Role
		action string
		scope  Scope
		ok     bool
	}{
		{"admin wildcard", RoleAdmin, "job:*", ScopeAny, true},
		{"admin wildcard", RoleAdmin, JobUpdate, ScopeAny, true},
		{"recruiter views any resume", RoleRecruiter, ResumeView, ScopeAny, true},
		{"recruiter owns jobs", RoleRecruiter, JobUpdate, ScopeOwn, true},
		{"consultant owns resume", RoleConsultant, ResumeUpdate, ScopeOwn, true},
		{"consultant views any job", RoleConsultant, JobView, ScopeAny, true},
		{"consultant cannot create job", RoleConsultant, JobCreate, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grant, ok := GrantFor(tt.role, tt.action)
			if ok != tt.ok {
				t.Fatalf("GrantFor() ok = %v, want %v", ok, tt.ok)
			}
			if ok && grant.Scope != tt.scope {
				t.Fatalf("GrantFor() scope = %q, want %q", grant.Scope, tt.scope)
			}
		})
	}
}

func TestCanAccessEnforcesOwnership(t *testing.T) {
	consultant := UserContext{UserID: "consultant-1", Role: RoleConsultant}
	if err := CanAccess(consultant, ResumeView, "consultant-1"); err != nil {
		t.Fatalf("owner was denied: %v", err)
	}
	if err := CanAccess(consultant, ResumeView, "consultant-2"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-owner error = %v, want ErrForbidden", err)
	}

	admin := UserContext{UserID: "admin-1", Role: RoleAdmin}
	if err := CanAccess(admin, ResumeView, "consultant-2"); err != nil {
		t.Fatalf("admin was denied: %v", err)
	}
}
