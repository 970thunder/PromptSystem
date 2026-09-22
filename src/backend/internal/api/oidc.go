package api

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"promptos-backend/internal/store"
)

const oidcStateKey = "promptos:oidc:state:"
const oidcStateTTL = 10 * time.Minute

type oidcLoginState struct {
	Verifier, Nonce string
	ReturnTo        string
	// Popup 表示本次授权来自站点弹窗登录，回调需要返回 postMessage 收尾页。
	Popup bool
	// ClientNonce 由站点弹窗生成，回调原样回传，供弹窗校验消息归属。
	ClientNonce string
	ExpiresAt   time.Time
}

var oidcMemoryStates sync.Map

type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
	EndSessionEndpoint    string `json:"end_session_endpoint"`
	Issuer                string `json:"issuer"`
}
type oidcTokenResponse struct {
	IDToken string `json:"id_token"`
}
type oidcClaims struct {
	Issuer            string `json:"iss"`
	Subject           string `json:"sub"`
	Audience          any    `json:"aud"`
	Nonce             string `json:"nonce"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Picture           string `json:"picture"`
	Expiry            int64  `json:"exp"`
	IssuedAt          int64  `json:"iat"`
}
type oidcJWKSet struct {
	Keys []struct {
		Kty string   `json:"kty"`
		Kid string   `json:"kid"`
		Alg string   `json:"alg"`
		N   string   `json:"n"`
		E   string   `json:"e"`
		X5C []string `json:"x5c"`
	} `json:"keys"`
}

func (s *server) oidcConfigured() bool {
	return s.config.OIDCEnabled && s.config.OIDCIssuer != "" && s.config.OIDCClientID != "" && s.config.OIDCClientSecret != "" && s.config.OIDCRedirectURI != ""
}
func (s *server) oidcDiscoveryURL() string {
	return strings.TrimRight(s.config.OIDCIssuer, "/") + "/.well-known/openid-configuration"
}

func (s *server) handleOIDCStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	if !s.oidcConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, apiResponse[any]{Code: 503, Message: "OIDC is not configured"})
		return
	}
	disc, err := s.loadOIDCDiscovery(r.Context())
	if err != nil {
		log.Printf("oidc discovery failed: %v", err)
		writeJSON(w, 503, apiResponse[any]{Code: 503, Message: "OIDC provider unavailable"})
		return
	}
	verifier, err := randomURLToken(32)
	if err != nil {
		writeJSON(w, 500, apiResponse[any]{Code: 500, Message: "Failed to start OIDC"})
		return
	}
	nonce, err := randomURLToken(24)
	if err != nil {
		writeJSON(w, 500, apiResponse[any]{Code: 500, Message: "Failed to start OIDC"})
		return
	}
	state, err := randomURLToken(24)
	if err != nil {
		writeJSON(w, 500, apiResponse[any]{Code: 500, Message: "Failed to start OIDC"})
		return
	}
	entry := oidcLoginState{
		Verifier:    verifier,
		Nonce:       nonce,
		ReturnTo:    safeOIDCReturnTo(r.URL.Query().Get("returnTo")),
		Popup:       isPopupRequest(r.URL.Query().Get("popup")),
		ClientNonce: sanitizeClientNonce(r.URL.Query().Get("nonce")),
		ExpiresAt:   time.Now().Add(oidcStateTTL),
	}
	if s.cache != nil {
		raw, _ := json.Marshal(entry)
		if err := s.cache.Set(r.Context(), oidcStateKey+state, string(raw), oidcStateTTL); err != nil {
			log.Printf("oidc state redis store failed: %v", err)
		} else {
			entry = oidcLoginState{}
		}
	}
	if entry.Verifier != "" {
		oidcMemoryStates.Store(state, entry)
	}
	http.SetCookie(w, &http.Cookie{Name: "promptos_oidc_state", Value: state, Path: "/", MaxAge: int(oidcStateTTL / time.Second), HttpOnly: true, Secure: s.config.IsProduction(), SameSite: http.SameSiteLaxMode})
	// Compute the PKCE S256 challenge without exposing the verifier.
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	params := url.Values{}
	params.Set("client_id", s.config.OIDCClientID)
	params.Set("redirect_uri", s.config.OIDCRedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", state)
	params.Set("nonce", nonce)
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	http.Redirect(w, r, disc.AuthorizationEndpoint+"?"+params.Encode(), http.StatusFound)
}

func (s *server) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	if !s.oidcConfigured() {
		s.redirectOAuthError(w, r, "OIDC is not configured")
		return
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	cookie, err := r.Cookie("promptos_oidc_state")
	if state == "" || err != nil || cookie.Value != state {
		s.redirectOAuthError(w, r, "Invalid OIDC state")
		return
	}
	entry, ok := s.consumeOIDCState(r.Context(), state)
	if !ok {
		s.redirectOAuthError(w, r, "OIDC state expired")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "promptos_oidc_state", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.config.IsProduction(), SameSite: http.SameSiteLaxMode})
	// 弹窗流程的失败也要回到收尾页，让弹窗自己提示并关闭，而不是跳转到前端页面。
	fail := func(message string) {
		if entry.Popup {
			writeOIDCPopupResult(w, s.config.FrontendURL, entry.ClientNonce, false, message)
			return
		}
		s.redirectOAuthError(w, r, message)
	}
	if providerErr := strings.TrimSpace(r.URL.Query().Get("error")); providerErr != "" {
		fail(providerErr)
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		fail("Missing authorization code")
		return
	}
	disc, err := s.loadOIDCDiscovery(r.Context())
	if err != nil {
		fail("OIDC provider unavailable")
		return
	}
	tokens, err := s.exchangeOIDCCode(r.Context(), disc.TokenEndpoint, code, entry.Verifier)
	if err != nil {
		log.Printf("oidc code exchange failed: %v", err)
		fail("OIDC code exchange failed")
		return
	}
	claims, err := s.verifyOIDCIDToken(r.Context(), disc, tokens.IDToken, entry.Nonce)
	if err != nil {
		log.Printf("oidc id token rejected: %v", err)
		fail("OIDC identity verification failed")
		return
	}
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if !claims.EmailVerified || !store.IsValidEmail(email) {
		fail("OIDC account must have a verified email")
		return
	}
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Name
	}
	if username == "" {
		username = strings.Split(email, "@")[0]
	}
	user, err := s.userStore.UpsertOIDCUser(claims.Subject, username, email, claims.Picture)
	if err != nil {
		log.Printf("oidc user upsert failed: %v", err)
		s.redirectOAuthError(w, r, "Failed to create user session")
		return
	}
	// 首次登录与每次登录都以资料中心为准：令牌没有昵称时也能拿到用户设置的统一昵称与头像。
	user = s.syncUnifiedProfile(r.Context(), user)
	token, err := s.tokenManager.Generate(user.ID, user.Email, user.SessionVer)
	if err != nil {
		s.redirectOAuthError(w, r, "Token generation failed")
		return
	}
	s.setAuthCookie(w, token)
	// 仅用于单点登出的 id_token_hint（HttpOnly，不参与业务请求）。
	http.SetCookie(w, &http.Cookie{
		Name:     "promptos_id_token_hint",
		Value:    tokens.IDToken,
		Path:     "/",
		MaxAge:   s.config.JWTExpireHours * 60 * 60,
		HttpOnly: true,
		Secure:   s.config.IsProduction(),
		SameSite: http.SameSiteLaxMode,
	})
	if entry.Popup {
		writeOIDCPopupResult(w, s.config.FrontendURL, entry.ClientNonce, true, "")
		return
	}
	frontend := strings.TrimRight(s.config.FrontendURL, "/")
	callback := frontend + "/auth/callback?oidc=1&redirect=" + url.QueryEscape(entry.ReturnTo)
	http.Redirect(w, r, callback, http.StatusFound)
}

func safeOIDCReturnTo(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "\r\n") {
		return "/"
	}
	return value
}

func (s *server) consumeOIDCState(ctx context.Context, state string) (oidcLoginState, bool) {
	if s.cache != nil {
		if raw, err := s.cache.GetAndDelete(ctx, oidcStateKey+state); err == nil && raw != "" {
			var e oidcLoginState
			if json.Unmarshal([]byte(raw), &e) == nil && time.Now().Before(e.ExpiresAt) {
				return e, true
			}
		}
	}
	if value, ok := oidcMemoryStates.LoadAndDelete(state); ok {
		e, valid := value.(oidcLoginState)
		return e, valid && time.Now().Before(e.ExpiresAt)
	}
	return oidcLoginState{}, false
}

func (s *server) loadOIDCDiscovery(ctx context.Context) (oidcDiscovery, error) {
	var d oidcDiscovery
	if err := s.getJSON(ctx, s.oidcDiscoveryURL(), &d); err != nil {
		return d, err
	}
	if d.AuthorizationEndpoint == "" || d.TokenEndpoint == "" || d.JWKSURI == "" || d.Issuer == "" {
		return d, errors.New("incomplete oidc discovery")
	}
	if strings.TrimRight(d.Issuer, "/") != strings.TrimRight(s.config.OIDCIssuer, "/") {
		return d, errors.New("issuer mismatch")
	}
	return d, nil
}
func (s *server) getJSON(ctx context.Context, endpoint string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(target)
}

func (s *server) exchangeOIDCCode(ctx context.Context, endpoint, code, verifier string) (oidcTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", s.config.OIDCRedirectURI)
	form.Set("client_id", s.config.OIDCClientID)
	form.Set("client_secret", s.config.OIDCClientSecret)
	form.Set("code_verifier", verifier)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return oidcTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return oidcTokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oidcTokenResponse{}, fmt.Errorf("token endpoint status %d", resp.StatusCode)
	}
	var out oidcTokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&out); err != nil {
		return out, err
	}
	if out.IDToken == "" {
		return out, errors.New("missing id_token")
	}
	return out, nil
}

func (s *server) verifyOIDCIDToken(ctx context.Context, discovery oidcDiscovery, raw, nonce string) (oidcClaims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return oidcClaims{}, errors.New("invalid jwt")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return oidcClaims{}, err
	}
	var header struct{ Alg, Kid string }
	if json.Unmarshal(headerBytes, &header) != nil || header.Alg != "RS256" || header.Kid == "" {
		return oidcClaims{}, errors.New("unsupported jwt header")
	}
	var claims oidcClaims
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, err
	}
	if json.Unmarshal(payload, &claims) != nil {
		return claims, errors.New("invalid claims")
	}
	if strings.TrimRight(claims.Issuer, "/") != strings.TrimRight(discovery.Issuer, "/") || claims.Subject == "" || claims.Nonce != nonce {
		return claims, errors.New("issuer, subject or nonce mismatch")
	}
	now := time.Now().Unix()
	if claims.Expiry == 0 || claims.Expiry <= now || (claims.IssuedAt != 0 && claims.IssuedAt > now+60) {
		return claims, errors.New("token time claims invalid")
	}
	if !audienceContains(claims.Audience, s.config.OIDCClientID) {
		return claims, errors.New("audience mismatch")
	}
	var keys oidcJWKSet
	if err := s.getJSON(ctx, discovery.JWKSURI, &keys); err != nil {
		return claims, err
	}
	var key *rsa.PublicKey
	for _, item := range keys.Keys {
		if item.Kid == header.Kid && item.Kty == "RSA" {
			key, err = jwkRSA(item.N, item.E)
			break
		}
	}
	if err != nil || key == nil {
		return claims, errors.New("signing key not found")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return claims, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sig) != nil {
		return claims, errors.New("invalid signature")
	}
	return claims, nil
}
func audienceContains(value any, expected string) bool {
	switch v := value.(type) {
	case string:
		return v == expected
	case []any:
		for _, item := range v {
			if item == expected {
				return true
			}
		}
	}
	return false
}
func jwkRSA(n, e string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, err
	}
	ev := 0
	for _, b := range eb {
		ev = ev*256 + int(b)
	}
	if ev == 0 {
		return nil, errors.New("invalid exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: ev}, nil
}
func randomURLToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// isPopupRequest 判断本次授权是否来自站点弹窗登录（popup=1 或 popup=true）。
func isPopupRequest(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true":
		return true
	default:
		return false
	}
}

// sanitizeClientNonce 只保留字母数字，长度上限 64，避免把外部输入直接写进收尾页。
func sanitizeClientNonce(value string) string {
	var builder strings.Builder
	for _, char := range strings.TrimSpace(value) {
		if builder.Len() >= 64 {
			break
		}
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

// writeOIDCPopupResult 输出弹窗收尾页：只向站点自身 Origin 发送一次结果，然后自动关窗。
func writeOIDCPopupResult(w http.ResponseWriter, frontendURL, nonce string, ok bool, message string) {
	status := "error"
	if ok {
		status = "ok"
	}
	payload, err := json.Marshal(map[string]any{
		"source":  "isoumao-login",
		"status":  status,
		"nonce":   nonce,
		"message": message,
	})
	if err != nil {
		http.Error(w, "popup payload failed", http.StatusInternalServerError)
		return
	}
	escaped := strings.ReplaceAll(string(payload), "<", "\\u003C")
	origin, err := json.Marshal(strings.TrimRight(frontendURL, "/"))
	if err != nil {
		http.Error(w, "popup origin failed", http.StatusInternalServerError)
		return
	}
	html := strings.ReplaceAll(oidcPopupPage, "__PAYLOAD__", escaped)
	html = strings.ReplaceAll(html, "__ORIGIN__", string(origin))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, html)
}

const oidcPopupPage = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="robots" content="noindex, nofollow">
<title>isoumao 登录</title>
</head>
<body style="margin:0;display:flex;align-items:center;justify-content:center;height:100vh;font-family:system-ui,'Microsoft YaHei',sans-serif;color:#2a2a3c">
<p>正在完成登录…</p>
<script>
(function () {
  var payload = __PAYLOAD__;
  // 页面内嵌登录：优先把结果回传给承载本站页面的 iframe 父窗口；
  // 独立窗口场景保留 window.opener，保持两种入口都能收到结果。
  var target = window.parent && window.parent !== window ? window.parent : (window.opener && !window.opener.closed ? window.opener : null);
  try { if (target) { target.postMessage(payload, __ORIGIN__); } } catch (error) {}
  if (window.parent === window && payload.status === 'ok') { window.close(); }
})();
</script>
</body>
</html>
`
