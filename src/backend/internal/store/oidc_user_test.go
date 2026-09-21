package store

import "testing"

func TestUpsertOIDCUserBindsBySubjectAndEmail(t *testing.T) {
	s := NewUserStore()
	first, err := s.UpsertOIDCUser("subject-1", "New Name", "new@example.com", "https://example.com/a.png")
	if err != nil {
		t.Fatalf("create OIDC user: %v", err)
	}
	second, err := s.UpsertOIDCUser("subject-1", "Other Name", "new@example.com", "")
	if err != nil {
		t.Fatalf("repeat OIDC user: %v", err)
	}
	if first.ID != second.ID || second.OIDCSubject != "subject-1" {
		t.Fatalf("subject did not remain bound: %+v %+v", first, second)
	}
	legacy, err := s.Register("Legacy", "legacy@example.com", "password123")
	if err != nil {
		t.Fatalf("create legacy user: %v", err)
	}
	bound, err := s.UpsertOIDCUser("subject-2", "Legacy", "legacy@example.com", "")
	if err != nil {
		t.Fatalf("bind legacy user: %v", err)
	}
	if bound.ID != legacy.ID || bound.OIDCSubject != "subject-2" {
		t.Fatalf("legacy account was not preserved: %+v", bound)
	}
}
