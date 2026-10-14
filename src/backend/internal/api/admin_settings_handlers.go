package api

import (
	"net/http"
	"strings"
	"time"

	"promptos-backend/internal/store"
)

// Admin site-settings console: reads the wizard-owned configuration, updates
// it with partial semantics, and offers the post-init SMTP probe. Secrets are
// masked on read and an empty smtp password means "keep the stored value".

type adminSettingsResponse struct {
	SiteName            string `json:"siteName"`
	SiteDescription     string `json:"siteDescription"`
	SMTPHost            string `json:"smtpHost"`
	SMTPPort            string `json:"smtpPort"`
	SMTPUser            string `json:"smtpUser"`
	SMTPPasswordSet     bool   `json:"smtpPasswordSet"`
	SMTPFrom            string `json:"smtpFrom"`
	RegistrationEnabled bool   `json:"registrationEnabled"`
	SMTPFromEnv         bool   `json:"smtpFromEnv"`
}

type adminSettingsRequest struct {
	SiteName            string `json:"siteName"`
	SiteDescription     string `json:"siteDescription"`
	SMTPHost            string `json:"smtpHost"`
	SMTPPort            string `json:"smtpPort"`
	SMTPUser            string `json:"smtpUser"`
	SMTPPassword        string `json:"smtpPassword"`
	SMTPFrom            string `json:"smtpFrom"`
	RegistrationEnabled *bool  `json:"registrationEnabled"`
}

type adminTestSMTPRequest struct {
	Recipient string `json:"recipient"`
}

// handleAdminSettings dispatches GET/PUT for the settings console.
func (s *server) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAdminSettingsGet(w, r)
	case http.MethodPut:
		s.handleAdminSettingsUpdate(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (s *server) handleAdminSettingsGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	if s.settingsStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiResponse[any]{Code: 503, ErrorCode: "SETTINGS_UNAVAILABLE", Message: "Settings storage is unavailable"})
		return
	}
	all, err := s.settingsStore.GetAll()
	if err != nil {
		writeStoreError(w, err)
		return
	}
	settings := store.ParseSiteSettings(all)
	writeJSON(w, http.StatusOK, apiResponse[adminSettingsResponse]{
		Code:    200,
		Message: "Success",
		Data: adminSettingsResponse{
			SiteName:            settings.SiteName,
			SiteDescription:     settings.SiteDescription,
			SMTPHost:            settings.SMTPHost,
			SMTPPort:            settings.SMTPPort,
			SMTPUser:            settings.SMTPUser,
			SMTPPasswordSet:     settings.SMTPPassword != "",
			SMTPFrom:            settings.SMTPFrom,
			RegistrationEnabled: settings.RegistrationEnabled,
			SMTPFromEnv:         s.emailConfigured(),
		},
	})
}

func (s *server) handleAdminSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeMethodNotAllowed(w)
		return
	}
	if s.settingsStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiResponse[any]{Code: 503, ErrorCode: "SETTINGS_UNAVAILABLE", Message: "Settings storage is unavailable"})
		return
	}
	var payload adminSettingsRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}

	all, err := s.settingsStore.GetAll()
	if err != nil {
		writeStoreError(w, err)
		return
	}
	current := store.ParseSiteSettings(all)

	next := store.SiteSettings{
		SiteName:            strings.TrimSpace(payload.SiteName),
		SiteDescription:     strings.TrimSpace(payload.SiteDescription),
		SMTPHost:            strings.TrimSpace(payload.SMTPHost),
		SMTPPort:            strings.TrimSpace(payload.SMTPPort),
		SMTPUser:            strings.TrimSpace(payload.SMTPUser),
		SMTPPassword:        current.SMTPPassword, // empty payload password keeps stored secret
		SMTPFrom:            strings.TrimSpace(payload.SMTPFrom),
		RegistrationEnabled: current.RegistrationEnabled,
		RegistrationSet:     current.RegistrationSet,
	}
	if payload.SMTPPassword != "" {
		next.SMTPPassword = payload.SMTPPassword
	}
	if payload.RegistrationEnabled != nil {
		next.RegistrationEnabled = *payload.RegistrationEnabled
		next.RegistrationSet = true
	}
	if next.SMTPPort == "" {
		next.SMTPPort = current.SMTPPort
	}
	// Clearing the whole SMTP block must also drop the stored secret so the
	// env fallback takes over again.
	if next.SMTPHost == "" && next.SMTPFrom == "" {
		next.SMTPPassword = ""
	}
	if err := store.ValidateSiteSettings(next); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, ErrorCode: "INVALID_SETTING", Message: "Invalid settings payload"})
		return
	}

	values := map[string]string{
		store.SettingSiteName:        next.SiteName,
		store.SettingSiteDescription: next.SiteDescription,
		store.SettingSMTPHost:        next.SMTPHost,
		store.SettingSMTPPort:        next.SMTPPort,
		store.SettingSMTPUser:        next.SMTPUser,
		store.SettingSMTPFrom:        next.SMTPFrom,
	}
	if next.SMTPPassword != "" {
		values[store.SettingSMTPPassword] = next.SMTPPassword
	}
	if next.RegistrationSet {
		if next.RegistrationEnabled {
			values[store.SettingRegistrationOn] = "true"
		} else {
			values[store.SettingRegistrationOn] = "false"
		}
	}
	if err := s.settingsStore.SetMany(values); err != nil {
		writeStoreError(w, err)
		return
	}
	// 保存即生效：丢掉服务端快照与发件器缓存。
	s.invalidateSettingsCaches()

	// Audit records which keys changed, never the values (secrets!).
	changed := make([]string, 0, len(values))
	for key, value := range values {
		if stored, ok := all[key]; !ok || stored != value {
			changed = append(changed, key)
		}
	}
	if payload.SMTPPassword != "" && !containsString(changed, store.SettingSMTPPassword) {
		changed = append(changed, store.SettingSMTPPassword)
	}
	if len(changed) > 0 {
		s.auditSettingsChange(r, changed)
	}

	updated, err := s.settingsStore.GetAll()
	if err != nil {
		writeStoreError(w, err)
		return
	}
	settings := store.ParseSiteSettings(updated)
	writeJSON(w, http.StatusOK, apiResponse[adminSettingsResponse]{
		Code:    200,
		Message: "Success",
		Data: adminSettingsResponse{
			SiteName:            settings.SiteName,
			SiteDescription:     settings.SiteDescription,
			SMTPHost:            settings.SMTPHost,
			SMTPPort:            settings.SMTPPort,
			SMTPUser:            settings.SMTPUser,
			SMTPPasswordSet:     settings.SMTPPassword != "",
			SMTPFrom:            settings.SMTPFrom,
			RegistrationEnabled: settings.RegistrationEnabled,
			SMTPFromEnv:         s.emailConfigured(),
		},
	})
}

func (s *server) handleAdminTestSMTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "admin_test_smtp",
		rateLimitRule{bucket: rateLimitIP(r), limit: 3, window: 10 * time.Minute}) {
		return
	}
	var payload adminTestSMTPRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}
	if s.emailSender == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiResponse[any]{Code: 503, ErrorCode: "EMAIL_NOT_CONFIGURED", Message: "Email service is not configured"})
		return
	}
	if err := s.emailSender.Send(r.Context(), strings.TrimSpace(payload.Recipient), "PromptOS SMTP 测试邮件", "这是一封来自 PromptOS 管理控制台的测试邮件，收到即表示当前邮件配置有效。"); err != nil {
		reason := strings.TrimSpace(err.Error())
		if len(reason) > 200 {
			reason = reason[:200]
		}
		writeJSON(w, http.StatusBadGateway, apiResponse[any]{Code: 502, ErrorCode: "EMAIL_SEND_FAILED", Message: "Failed to send test email: " + reason})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse[map[string]bool]{Code: 200, Message: "Success", Data: map[string]bool{"sent": true}})
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

// auditSettingsChange appends a settings.update audit event when a moderation
// store is wired; settings changes still succeed without it (memory mode).
func (s *server) auditSettingsChange(r *http.Request, changedKeys []string) {
	auditor, ok := s.settingsStore.(interface {
		AppendSettingsAudit(actorID int, changedKeys []string, requestID string) error
	})
	if !ok {
		return
	}
	actorID, _ := userIDFromContext(r.Context())
	if actorID <= 0 {
		return
	}
	_ = auditor.AppendSettingsAudit(actorID, changedKeys, requestIDFrom(r))
}
