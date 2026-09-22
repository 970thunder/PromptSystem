package api

import (
	"context"
	"log"
	"time"

	"promptos-backend/internal/store"
)

// unifiedProfileSyncInterval 控制同一用户重复回源资料中心的频率。
const unifiedProfileSyncInterval = 15 * time.Second

// syncUnifiedProfile 把身份中心的昵称与头像同步到本站展示副本。
// 站点不再维护自己的资料编辑：昵称/头像只在这里被写入，业务数据不受影响。
func (s *server) syncUnifiedProfile(ctx context.Context, user store.AuthUser) store.AuthUser {
	if !s.identityDirectory.Enabled() || user.OIDCSubject == "" {
		return user
	}
	if !user.ProfileSyncedAt.IsZero() && time.Since(user.ProfileSyncedAt) < unifiedProfileSyncInterval {
		return user
	}

	displayName, avatarURL := s.identityDirectory.Lookup(ctx, user.OIDCSubject)
	if displayName == "" && avatarURL == "" {
		return user
	}
	if err := s.userStore.ApplyUnifiedProfile(user.ID, displayName, avatarURL); err != nil {
		log.Printf("unified profile apply failed: %v", err)
		return user
	}

	updated, found := s.getAuthService().FindByID(user.ID)
	if !found {
		return user
	}
	return updated
}
