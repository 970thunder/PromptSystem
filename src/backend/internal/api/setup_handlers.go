package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"promptos-backend/internal/store"
)

// Setup wizard + public site info endpoints.
//
// Security model:
//   - GET /setup/status is public and leaks nothing beyond bootstrap state.
//   - POST /setup/initialize works only once per deployment: the settings
//     store writes the setup marker atomically and refuses when an
//     administrator already exists (including out-of-band provisioning).
//   - POST /setup/test-smtp is only reachable BEFORE initialization completes;
//     afterwards the same capability requires the admin role. This prevents
//     anonymous SMTP relaying/probing on an initialized site.

type setupStatusResponse struct {
	Initialized      bool   `json:"initialized"`
	StorageMode      string `json:"storageMode"`
	Environment      string `json:"environment"`
	SMTPConfigured   bool   `json:"smtpConfigured"`
	EmailAuthEnabled bool   `json:"emailAuthEnabled"`
	RegistrationOpen bool   `json:"registrationOpen"`
	SiteName         string `json:"siteName"`
	SiteDescription  string `json:"siteDescription"`
}

type setupSMTPPayload struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	From     string `json:"from"`
}

type setupInitializeRequest struct {
	Username            string            `json:"username"`
	Email               string            `json:"email"`
	Password            string            `json:"password"`
	SiteName            string            `json:"siteName"`
	SiteDescription     string            `json:"siteDescription"`
	SMTP                *setupSMTPPayload `json:"smtp"`
	RegistrationEnabled *bool             `json:"registrationEnabled"`
}

type setupInitializeResponse struct {
	AdminID int `json:"adminId"`
}

type setupTestSMTPRequest struct {
	SMTP      setupSMTPPayload `json:"smtp"`
	Recipient string           `json:"recipient"`
}

// siteSettings resolves typed settings from the settings store; a nil store
// (directly constructed test servers) or a read error yields defaults.
// siteSettings resolves typed settings from the settings store. The snapshot
// is cached until a write path calls invalidateSettingsCaches; a failed read
// yields defaults WITHOUT caching so a transient store error cannot pin
// wrong registration/SMTP state.
func (s *server) siteSettings() store.SiteSettings {
	if p := s.cachedSettings.Load(); p != nil {
		return *p
	}
	var all map[string]string
	if s.settingsStore != nil {
		if read, err := s.settingsStore.GetAll(); err == nil {
			all = read
			settings := store.ParseSiteSettings(all)
			s.cachedSettings.Store(&settings)
			return settings
		}
	}
	return store.ParseSiteSettings(all)
}

// invalidateSettingsCaches drops the server settings snapshot and the email
// sender's cached resolution. Every settings write path must call this so
// "save takes effect immediately" holds.
func (s *server) invalidateSettingsCaches() {
	s.cachedSettings.Store(nil)
	if inv, ok := s.emailSender.(interface{ Invalidate() }); ok {
		inv.Invalidate()
	}
}

// emailConfigured reports whether any sender source (settings or environment)
// can send mail. Test fakes that implement Configured() are honored.
func (s *server) emailConfigured() bool {
	if s.emailSender == nil {
		return false
	}
	if c, ok := s.emailSender.(interface{ Configured() bool }); ok {
		return c.Configured()
	}
	return true
}

func (s *server) setupInitialized() bool {
	if s.setupStore == nil {
		return false
	}
	if completed, err := s.setupStore.Completed(); err == nil && completed {
		return true
	}
	if has, err := s.setupStore.HasAdmin(); err == nil && has {
		return true
	}
	return false
}

func (s *server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	settings := s.siteSettings()
	name := settings.SiteName
	if name == "" {
		name = store.DefaultSiteName
	}
	writeJSON(w, http.StatusOK, apiResponse[setupStatusResponse]{
		Code:    200,
		Message: "Success",
		Data: setupStatusResponse{
			Initialized:      s.setupInitialized(),
			StorageMode:      s.storageMode,
			Environment:      s.config.AppEnv,
			SMTPConfigured:   s.emailConfigured(),
			EmailAuthEnabled: s.config.EmailAuthEnabled,
			RegistrationOpen: !s.registrationDisabled(),
			SiteName:         name,
			SiteDescription:  settings.SiteDescription,
		},
	})
}

// registrationDisabled centralizes the registration toggle so the register
// handler and the public site info agree.
func (s *server) registrationDisabled() bool {
	return !s.siteSettings().RegistrationEnabled
}

func (s *server) handleSetupInitialize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	if s.setupStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiResponse[any]{Code: 503, ErrorCode: "SETUP_UNAVAILABLE", Message: "Setup is unavailable"})
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "setup_initialize",
		rateLimitRule{bucket: rateLimitIP(r), limit: 5, window: 10 * time.Minute}) {
		return
	}

	var payload setupInitializeRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}
	if len(payload.Password) > maxPasswordBytes {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, Message: "Password is too long", ErrorCode: "INVALID_PASSWORD"})
		return
	}
	if strings.TrimSpace(payload.Username) == "" || strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.Password) == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, ErrorCode: "INVALID_REQUEST", Message: "Username, email, and password are required"})
		return
	}
	if !store.IsValidEmail(payload.Email) {
		writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, ErrorCode: "INVALID_EMAIL", Message: "Invalid email address"})
		return
	}

	settings := map[string]string{}
	if name := strings.TrimSpace(payload.SiteName); name != "" {
		settings[store.SettingSiteName] = name
	}
	if desc := strings.TrimSpace(payload.SiteDescription); desc != "" {
		settings[store.SettingSiteDescription] = desc
	}
	if payload.SMTP != nil {
		smtp := store.SiteSettings{
			SMTPHost:     strings.TrimSpace(payload.SMTP.Host),
			SMTPPort:     strings.TrimSpace(payload.SMTP.Port),
			SMTPUser:     strings.TrimSpace(payload.SMTP.User),
			SMTPPassword: payload.SMTP.Password,
			SMTPFrom:     strings.TrimSpace(payload.SMTP.From),
		}
		if err := store.ValidateSiteSettings(smtp); err != nil {
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, ErrorCode: "INVALID_SMTP", Message: "Invalid SMTP configuration"})
			return
		}
		if smtp.SMTPConfigured() {
			settings[store.SettingSMTPHost] = smtp.SMTPHost
			settings[store.SettingSMTPPort] = smtp.SMTPPort
			settings[store.SettingSMTPUser] = smtp.SMTPUser
			if smtp.SMTPPassword != "" {
				settings[store.SettingSMTPPassword] = smtp.SMTPPassword
			}
			settings[store.SettingSMTPFrom] = smtp.SMTPFrom
		}
	}
	if payload.RegistrationEnabled != nil {
		if *payload.RegistrationEnabled {
			settings[store.SettingRegistrationOn] = "true"
		} else {
			settings[store.SettingRegistrationOn] = "false"
		}
	}

	adminID, err := s.setupStore.CompleteSetup(store.SetupInput{
		Username:  payload.Username,
		Email:     payload.Email,
		Password:  payload.Password,
		Settings:  settings,
		RequestID: requestIDFrom(r),
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrSetupCompleted):
			writeJSON(w, http.StatusConflict, apiResponse[any]{Code: 409, ErrorCode: "SETUP_COMPLETED", Message: "Setup has already been completed"})
		case errors.Is(err, store.ErrUserExists):
			writeJSON(w, http.StatusConflict, apiResponse[any]{Code: 409, ErrorCode: "USER_EXISTS", Message: "Email already registered"})
		case errors.Is(err, store.ErrWeakPassword):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, ErrorCode: "WEAK_PASSWORD", Message: "Password must be at least 8 characters"})
		case errors.Is(err, store.ErrPasswordTooLong):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, ErrorCode: "INVALID_PASSWORD", Message: "Password must be 72 bytes or fewer"})
		case errors.Is(err, store.ErrInvalidEmail):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, ErrorCode: "INVALID_EMAIL", Message: "Invalid email address"})
		case errors.Is(err, store.ErrInvalidUser), errors.Is(err, store.ErrInvalidContent), errors.Is(err, store.ErrUnsafeContent), errors.Is(err, store.ErrContentTooLong):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, ErrorCode: "INVALID_USER", Message: "Username is invalid"})
		case errors.Is(err, store.ErrInvalidSetting):
			writeJSON(w, http.StatusBadRequest, apiResponse[any]{Code: 400, ErrorCode: "INVALID_SETTING", Message: "Invalid settings payload"})
		default:
			writeJSON(w, http.StatusInternalServerError, apiResponse[any]{Code: 500, ErrorCode: "INTERNAL_ERROR", Message: "Setup failed"})
		}
		return
	}

	writeJSON(w, http.StatusOK, apiResponse[setupInitializeResponse]{
		Code:    200,
		Message: "Success",
		Data:    setupInitializeResponse{AdminID: adminID},
	})
	s.invalidateSettingsCaches()
}

func (s *server) handleSetupTestSMTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	if s.setupInitialized() {
		writeJSON(w, http.StatusForbidden, apiResponse[any]{Code: 403, ErrorCode: "SETUP_COMPLETED", Message: "Use the admin settings console after initialization"})
		return
	}
	if !s.enforceRateLimits(r.Context(), w, "setup_test_smtp",
		rateLimitRule{bucket: rateLimitIP(r), limit: 3, window: 10 * time.Minute}) {
		return
	}
	var payload setupTestSMTPRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, err)
		return
	}
	if err := s.sendTestEmail(r, payload.SMTP, payload.Recipient); err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apiResponse[map[string]bool]{Code: 200, Message: "Success", Data: map[string]bool{"sent": true}})
}

// sendTestEmail validates the payload and delivers a fixed test message via a
// one-off sender, so a wizard configuration can be verified before it is
// persisted.
func (s *server) sendTestEmail(r *http.Request, smtp setupSMTPPayload, recipient string) error {
	recipient = strings.TrimSpace(recipient)
	if !store.IsValidEmail(recipient) {
		return &apiError{Status: http.StatusBadRequest, Code: "INVALID_EMAIL", Message: "Invalid recipient email address"}
	}
	sender := newSMTPSenderFromSettings(store.SiteSettings{
		SMTPHost:     strings.TrimSpace(smtp.Host),
		SMTPPort:     strings.TrimSpace(smtp.Port),
		SMTPUser:     strings.TrimSpace(smtp.User),
		SMTPPassword: smtp.Password,
		SMTPFrom:     strings.TrimSpace(smtp.From),
	})
	if sender == nil {
		return &apiError{Status: http.StatusBadRequest, Code: "INVALID_SMTP", Message: "SMTP host and from address are required"}
	}
	if err := sender.Send(r.Context(), recipient, "PromptOS SMTP 测试邮件", "这是一封来自 PromptOS 初始化向导的测试邮件，收到即表示邮件配置有效。"); err != nil {
		// The recipient is an administrator (or the pre-init wizard), so the
		// underlying transport error is the actionable signal.
		reason := strings.TrimSpace(err.Error())
		if len(reason) > 200 {
			reason = reason[:200]
		}
		return &apiError{Status: http.StatusBadGateway, Code: "EMAIL_SEND_FAILED", Message: "Failed to send test email: " + reason}
	}
	return nil
}

func (s *server) handleSiteInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	settings := s.siteSettings()
	name := settings.SiteName
	if name == "" {
		name = store.DefaultSiteName
	}
	writeJSON(w, http.StatusOK, apiResponse[siteInfoResponse]{
		Code:    200,
		Message: "Success",
		Data: siteInfoResponse{
			Name:               name,
			Description:        settings.SiteDescription,
			RegistrationOpen:   !s.registrationDisabled(),
			EmailAuthEnabled:   s.config.EmailAuthEnabled,
			GitHubOAuthEnabled: s.config.GitHubOAuthEnabled,
		},
	})
}

type siteInfoResponse struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	RegistrationOpen   bool   `json:"registrationOpen"`
	EmailAuthEnabled   bool   `json:"emailAuthEnabled"`
	GitHubOAuthEnabled bool   `json:"githubOAuthEnabled"`
}
