package store

import "testing"

// 昵称与头像统一由 isoumao 身份中心维护：同步值优先，本站字段仅作兜底。
func TestToPublicUserPrefersUnifiedProfile(t *testing.T) {
	unified := ToPublicUser(AuthUser{
		ID:          7,
		Username:    "site-handle",
		DisplayName: "统一昵称",
		Avatar:      "/uploads/local.png",
		AvatarURL:   "https://id.isoumao.cn/profile/avatar/subject-7",
	})
	if unified.Username != "统一昵称" {
		t.Fatalf("unified username = %q, want 统一昵称", unified.Username)
	}
	if unified.Avatar != "https://id.isoumao.cn/profile/avatar/subject-7" {
		t.Fatalf("unified avatar = %q", unified.Avatar)
	}

	fallback := ToPublicUser(AuthUser{ID: 8, Username: "site-handle", Avatar: "/uploads/local.png"})
	if fallback.Username != "site-handle" || fallback.Avatar != "/uploads/local.png" {
		t.Fatalf("fallback = %q / %q", fallback.Username, fallback.Avatar)
	}
}

func TestApplyUnifiedProfileKeepsSiteUsername(t *testing.T) {
	users := NewUserStore()
	created, err := users.Register("site-handle", "member@example.com", "Member-password-2026")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := users.ApplyUnifiedProfile(created.ID, "统一昵称", "https://id.isoumao.cn/profile/avatar/x"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, found := users.FindByID(created.ID)
	if !found {
		t.Fatal("user not found after sync")
	}
	if updated.Username != "site-handle" || updated.DisplayName != "统一昵称" {
		t.Fatalf("username/displayName = %q / %q", updated.Username, updated.DisplayName)
	}
	if updated.AvatarURL != "https://id.isoumao.cn/profile/avatar/x" {
		t.Fatalf("avatarURL = %q", updated.AvatarURL)
	}
}
