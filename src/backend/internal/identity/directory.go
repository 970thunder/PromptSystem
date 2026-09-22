// Package identity 提供统一资料中心（id.isoumao.cn/profile）的只读同步客户端。
// 昵称与头像以身份中心为权威来源，本站只保存一份用于展示的副本。
package identity

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const cacheLifetime = 5 * time.Minute

type cacheEntry struct {
	displayName string
	avatarURL   string
	fetchedAt   time.Time
}

// Directory 按 OIDC subject 读取统一资料，结果在进程内缓存，资料中心不可用时保持站点现有资料。
type Directory struct {
	baseURL string
	client  *http.Client

	mu    sync.Mutex
	cache map[string]cacheEntry
}

// NewDirectory 构造同步客户端；baseURL 为空表示关闭同步（本地开发与自动化测试）。
func NewDirectory(baseURL string) *Directory {
	return &Directory{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client:  &http.Client{Timeout: 4 * time.Second},
		cache:   make(map[string]cacheEntry),
	}
}

// Enabled 表示是否配置了资料中心地址。
func (d *Directory) Enabled() bool { return d != nil && d.baseURL != "" }

// Lookup 返回统一资料中的昵称与头像；查询失败或无资料时返回空字符串。
func (d *Directory) Lookup(ctx context.Context, subject string) (string, string) {
	if !d.Enabled() || strings.TrimSpace(subject) == "" {
		return "", ""
	}

	d.mu.Lock()
	if entry, ok := d.cache[subject]; ok && time.Since(entry.fetchedAt) < cacheLifetime {
		d.mu.Unlock()
		return entry.displayName, entry.avatarURL
	}
	d.mu.Unlock()

	displayName, avatarURL := d.fetch(ctx, subject)
	d.mu.Lock()
	d.cache[subject] = cacheEntry{displayName: displayName, avatarURL: avatarURL, fetchedAt: time.Now()}
	d.mu.Unlock()
	return displayName, avatarURL
}

func (d *Directory) fetch(ctx context.Context, subject string) (string, string) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, d.baseURL+"/api/public/"+subject, nil)
	if err != nil {
		return "", ""
	}
	response, err := d.client.Do(request)
	if err != nil {
		log.Printf("unified profile lookup failed: %v", err)
		return "", ""
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", ""
	}

	var payload struct {
		DisplayName string `json:"displayName"`
		AvatarURL   string `json:"avatarUrl"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		log.Printf("unified profile decode failed: %v", err)
		return "", ""
	}
	return strings.TrimSpace(payload.DisplayName), strings.TrimSpace(payload.AvatarURL)
}
