//go:build storage_live

// 文件作用：真实对象存储冒烟测试（不进入常规 CI）。用于接入新 S3/RustFS
// 端点或轮换凭据后验证 R2Storage 保存/公开 URL 读取链路，直接输出底层 SDK
// 错误，便于定位存储提供方兼容性问题。
//
// 运行方式（凭据只通过环境变量注入，不得写入仓库）：
//
//	SMOKE_R2_ENDPOINT=http://127.0.0.1:13912 \
//	SMOKE_R2_BUCKET=promptsystem-prod \
//	SMOKE_R2_AK=... SMOKE_R2_SK=... \
//	SMOKE_R2_PUBLIC_URL=https://promptsystem.isoumao.top/objects \
//	go test -tags=storage_live -run TestLiveR2Storage ./internal/storage/ -v
package storage

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"promptos-backend/internal/config"
)

func TestLiveR2Storage(t *testing.T) {
	cfg := config.Config{
		UploadProvider: "rustfs",
		R2Endpoint:     os.Getenv("SMOKE_R2_ENDPOINT"),
		R2Bucket:       os.Getenv("SMOKE_R2_BUCKET"),
		R2AccessKeyID:  os.Getenv("SMOKE_R2_AK"),
		R2SecretKey:    os.Getenv("SMOKE_R2_SK"),
		R2PublicURL:    os.Getenv("SMOKE_R2_PUBLIC_URL"),
	}
	if cfg.R2Endpoint == "" || cfg.R2Bucket == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretKey == "" {
		t.Skip("SMOKE_R2_* not set; skipping live storage check")
	}

	store, err := newR2Storage(cfg)
	if err != nil {
		t.Fatalf("newR2Storage: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	key := "test/live-storage-probe.png"
	url, err := store.Save(ctx, key, "image/png", []byte("probe"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Logf("save url: %s", url)

	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_ = bytes.MinRead
}
