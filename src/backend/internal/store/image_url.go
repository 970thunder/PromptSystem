package store

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"unicode"
)

// uploadPathPrefix is the URL path prefix under which locally uploaded images
// are served. Only paths under this prefix, or explicit HTTPS hostnames that the
// deployment has allowlisted, may be persisted as prompt cover/image references.
const uploadPathPrefix = "/uploads/"

// ErrInvalidImageURL is returned whenever a cover or image reference cannot be
// safely persisted (javascript:, data:, file:, a private/intranet host, an
// unallowlisted remote host, or a malformed value).
var ErrInvalidImageURL = errors.New("invalid image url")

// ValidateImageURL reports whether image may be persisted on a prompt. A value
// is accepted only when it is a local upload path under uploadPathPrefix or an
// HTTPS URL whose hostname is present in allowedHTTPSDomains. javascript:,
// data:, file:, and any private/intranet (RFC1918, loopback, link-local, etc.)
// URL is always rejected, regardless of the allowlist, because such values can
// only originate from untrusted input and must never reach persisted fields.
func ValidateImageURL(image string, allowedHTTPSDomains []string) error {
	raw := strings.TrimSpace(image)
	if raw == "" || len([]rune(raw)) > 1024 || strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return ErrInvalidImageURL
	}
	if strings.HasPrefix(raw, uploadPathPrefix) {
		return nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ErrInvalidImageURL
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ErrInvalidImageURL
	}
	// Only https is acceptable for remote references. This also rejects
	// javascript:, data:, file:, ftp:, and other non-http schemes.
	if !strings.EqualFold(parsed.Scheme, "https") {
		return ErrInvalidImageURL
	}

	host := strings.ToLower(parsed.Hostname())
	if isPrivateHost(host) {
		return ErrInvalidImageURL
	}
	if !containsHost(allowedHTTPSDomains, host) {
		return ErrInvalidImageURL
	}

	return nil
}

// ValidateImageURLs applies ValidateImageURL to every element of images. It
// returns the first offending value wrapped with context, or nil when all pass.
func ValidateImageURLs(images []string, allowedHTTPSDomains []string) error {
	normalized := make(map[string]struct{}, len(images))
	for _, raw := range images {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, seen := normalized[value]; seen {
			continue
		}
		normalized[value] = struct{}{}
		if err := ValidateImageURL(value, allowedHTTPSDomains); err != nil {
			return err
		}
	}
	return nil
}

func isPrivateHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		// A bare hostname is not an IP literal; remote references are gated by
		// the allowlist, so there is nothing to reject here.
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	return ip.IsPrivate()
}

func containsHost(allowlist []string, host string) bool {
	for _, allowed := range allowlist {
		if hostAllowed(allowed, host) {
			return true
		}
	}
	return false
}

// hostAllowed reports whether hostname matches a single allowlist entry. Entries
// may be written as a bare hostname ("img.cdn.com"), with an explicit port
// ("img.cdn.com:443"), or with a scheme ("https://img.cdn.com"). The match is an
// exact hostname comparison so a crafted sibling domain can never satisfy the
// allowlist.
func hostAllowed(entry, host string) bool {
	entry = strings.ToLower(strings.TrimSpace(entry))
	if entry == "" {
		return false
	}
	if i := strings.Index(entry, "://"); i >= 0 {
		entry = entry[i+3:]
	}
	if strings.HasPrefix(entry, "*.") {
		suffix := entry[1:]
		return strings.HasSuffix(host, suffix) && host != strings.TrimPrefix(suffix, ".")
	}
	if h, _, err := net.SplitHostPort(entry); err == nil {
		entry = h
	}
	return strings.EqualFold(entry, host)
}
