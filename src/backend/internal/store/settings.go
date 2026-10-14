package store

import (
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Sentinel errors for the setup/settings boundary. Handlers map these to
// stable errorCode values; nothing here leaks internal error text.
var (
	// ErrSetupCompleted is returned when bootstrap is attempted after an
	// administrator already exists or the setup marker is present.
	ErrSetupCompleted = errors.New("setup already completed")
	// ErrInvalidSetting covers malformed setting keys/values.
	ErrInvalidSetting = errors.New("invalid setting")
)

// Canonical system_settings keys. The setup marker is reserved: only the
// bootstrap transaction may write it.
const (
	SettingSetupCompletedAt = "setup.completed_at"
	SettingSiteName         = "site.name"
	SettingSiteDescription  = "site.description"
	SettingSMTPHost         = "smtp.host"
	SettingSMTPPort         = "smtp.port"
	SettingSMTPUser         = "smtp.user"
	SettingSMTPPassword     = "smtp.password"
	SettingSMTPFrom         = "smtp.from"
	SettingRegistrationOn   = "registration.enabled"
)

const (
	DefaultSiteName = "PromptOS"
	defaultSMTPPort = "587"

	maxSiteNameRunes        = 60
	maxSiteDescriptionRunes = 200
)

// SettingsManager persists runtime site configuration. Implementations must be
// safe for concurrent use.
type SettingsManager interface {
	Get(key string) (string, bool, error)
	GetAll() (map[string]string, error)
	// SetMany upserts the given keys in one round trip. An empty value deletes
	// the key so callers can clear a setting; the reserved setup marker is
	// always ignored.
	SetMany(values map[string]string) error
}

// SetupManager owns the one-shot bootstrap: it reports whether an
// administrator exists and creates the first administrator plus initial
// settings atomically.
type SetupManager interface {
	// HasAdmin reports whether any active administrator exists.
	HasAdmin() (bool, error)
	// Completed reports whether the setup wizard marker has been written.
	Completed() (bool, error)
	// CompleteSetup atomically writes the setup marker, creates the first
	// administrator account, grants the admin role, and stores the given
	// settings. It returns ErrSetupCompleted when bootstrap already happened
	// and ErrUserExists when the email is already registered.
	CompleteSetup(input SetupInput) (int, error)
}

// SetupInput carries the wizard payload into the store boundary.
type SetupInput struct {
	Username  string
	Email     string
	Password  string
	Settings  map[string]string
	RequestID string
	Now       time.Time
}

// SiteSettings is the typed projection of the wizard-owned setting keys.
// Zero values mean "not configured"; callers apply their own defaults.
type SiteSettings struct {
	SiteName            string
	SiteDescription     string
	SMTPHost            string
	SMTPPort            string
	SMTPUser            string
	SMTPPassword        string
	SMTPFrom            string
	RegistrationEnabled bool
	// RegistrationSet distinguishes "absent (default on)" from an explicit
	// opt-out stored by the wizard.
	RegistrationSet bool
}

// ParseSiteSettings projects raw rows into typed settings with defaults.
func ParseSiteSettings(all map[string]string) SiteSettings {
	enabled, set := all[SettingRegistrationOn]
	parsed := SiteSettings{
		SiteName:        strings.TrimSpace(all[SettingSiteName]),
		SiteDescription: strings.TrimSpace(all[SettingSiteDescription]),
		SMTPHost:        strings.TrimSpace(all[SettingSMTPHost]),
		SMTPPort:        strings.TrimSpace(all[SettingSMTPPort]),
		SMTPUser:        strings.TrimSpace(all[SettingSMTPUser]),
		SMTPPassword:    all[SettingSMTPPassword],
		SMTPFrom:        strings.TrimSpace(all[SettingSMTPFrom]),
		RegistrationSet: set,
	}
	if !set {
		parsed.RegistrationEnabled = true
	} else {
		parsed.RegistrationEnabled = enabled == "true" || enabled == "1" || strings.EqualFold(enabled, "yes")
	}
	if parsed.SMTPPort == "" {
		parsed.SMTPPort = defaultSMTPPort
	}
	return parsed
}

// SMTPConfigured reports whether the typed settings carry a usable SMTP
// sender (host and from address are the two required fields everywhere else).
func (s SiteSettings) SMTPConfigured() bool {
	return s.SMTPHost != "" && s.SMTPFrom != ""
}

// ValidateSiteSettings enforces boundary rules shared by the setup wizard and
// the admin settings console. SMTP fields are all-or-nothing beyond the port
// default, mirroring the env-based configuration contract.
func ValidateSiteSettings(s SiteSettings) error {
	if n := len([]rune(s.SiteName)); n > maxSiteNameRunes {
		return ErrInvalidSetting
	}
	if n := len([]rune(s.SiteDescription)); n > maxSiteDescriptionRunes {
		return ErrInvalidSetting
	}
	if !utf8.ValidString(s.SiteName) || !utf8.ValidString(s.SiteDescription) {
		return ErrInvalidSetting
	}
	if containsDisallowedControl(s.SiteName) || containsDisallowedControl(s.SiteDescription) {
		return ErrInvalidSetting
	}
	if s.SMTPHost != "" || s.SMTPUser != "" || s.SMTPPassword != "" || s.SMTPFrom != "" {
		if s.SMTPHost == "" || s.SMTPFrom == "" {
			return ErrInvalidSetting
		}
		if port, err := strconv.Atoi(s.SMTPPort); err != nil || port < 1 || port > 65535 {
			return ErrInvalidSetting
		}
		if (s.SMTPUser == "") != (s.SMTPPassword == "") {
			return ErrInvalidSetting
		}
	}
	return nil
}

// SettingKeys returns the writable (non-reserved) canonical keys.
func SettingKeys() []string {
	return []string{
		SettingSiteName, SettingSiteDescription,
		SettingSMTPHost, SettingSMTPPort, SettingSMTPUser, SettingSMTPPassword, SettingSMTPFrom,
		SettingRegistrationOn,
	}
}

// NormalizeSettingsMap keeps only known keys, trims values, and drops empty
// entries so updates are sparse. The reserved setup marker is never accepted.
func NormalizeSettingsMap(values map[string]string) map[string]string {
	known := SettingKeys()
	normalized := make(map[string]string, len(values))
	for _, key := range known {
		value, ok := values[key]
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if key == SettingSMTPPassword {
			// Passwords keep interior spacing; only outer whitespace is noise.
			value = strings.Trim(value, " \t\r\n")
		}
		if value != "" {
			normalized[key] = value
		}
	}
	return normalized
}
