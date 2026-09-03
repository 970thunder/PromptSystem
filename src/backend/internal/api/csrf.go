package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

const (
	authCookieName = "promptos_session"
	csrfCookieName = "promptos_csrf"
)

func (s *server) authCookieEnabled() bool {
	return s.config.AuthCookieEnabled || s.config.IsProduction()
}

func (s *server) authCookieSecure() bool {
	return s.config.IsProduction()
}

func (s *server) setAuthCookie(w http.ResponseWriter, token string) {
	if !s.authCookieEnabled() || token == "" {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   s.config.JWTExpireHours * 60 * 60,
		HttpOnly: true,
		Secure:   s.authCookieSecure(),
		SameSite: http.SameSiteLaxMode,
	})
	s.setCSRFCookie(w)
}

func (s *server) setCSRFCookie(w http.ResponseWriter) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    hex.EncodeToString(value),
		Path:     "/",
		MaxAge:   s.config.JWTExpireHours * 60 * 60,
		HttpOnly: false,
		Secure:   s.authCookieSecure(),
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookies(w http.ResponseWriter, secure bool) {
	for _, name := range []string{authCookieName, csrfCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: name == authCookieName,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func csrfTokenFromRequest(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
}

func (s *server) withCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authCookieEnabled() || isSafeHTTPMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		if _, hasBearer := bearerTokenFromRequest(r); hasBearer {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(authCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			next.ServeHTTP(w, r)
			return
		}
		csrfCookie, err := r.Cookie(csrfCookieName)
		provided := csrfTokenFromRequest(r)
		if err != nil || provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(csrfCookie.Value)) != 1 {
			writeJSON(w, http.StatusForbidden, apiResponse[any]{
				Code:      http.StatusForbidden,
				Message:   "CSRF validation failed",
				ErrorCode: "CSRF_INVALID",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isSafeHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func bearerTokenFromRequest(r *http.Request) (string, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	return token, token != ""
}

func sessionTokenFromRequest(r *http.Request) (string, bool) {
	if token, ok := bearerTokenFromRequest(r); ok {
		return token, true
	}
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return "", false
	}
	token := strings.TrimSpace(cookie.Value)
	return token, token != ""
}
