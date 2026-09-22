//go:build smtp_live

// 文件作用：真实 SMTP 发送冒烟测试（不进入常规 CI）。用于接入新邮件凭据或
// 密钥轮换后的端到端验证：使用与生产一致的 smtpEmailSender 发送一封测试邮件。
//
// 运行方式（凭据只通过环境变量注入，不得写入仓库）：
//
//	SMOKE_SMTP_TO=ops@example.com \
//	SMOKE_SMTP_FROM=hello@mail.example.com \
//	SMOKE_SMTP_USER=hello@mail.example.com \
//	SMOKE_SMTP_PASSWORD=secret \
//	go test -tags=smtp_live -run TestLiveSMTPSend ./internal/api/ -v
package api

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"promptos-backend/internal/config"
)

func TestLiveSMTPSend(t *testing.T) {
	cfg := config.Config{
		SMTPHost:     os.Getenv("SMOKE_SMTP_HOST"),
		SMTPPort:     os.Getenv("SMOKE_SMTP_PORT"),
		SMTPUser:     os.Getenv("SMOKE_SMTP_USER"),
		SMTPPassword: os.Getenv("SMOKE_SMTP_PASSWORD"),
		SMTPFrom:     os.Getenv("SMOKE_SMTP_FROM"),
	}
	recipient := strings.TrimSpace(os.Getenv("SMOKE_SMTP_TO"))
	if cfg.SMTPHost == "" || cfg.SMTPFrom == "" || recipient == "" {
		t.Skip("SMOKE_SMTP_HOST/SMOKE_SMTP_FROM/SMOKE_SMTP_TO not set; skipping live SMTP check")
	}
	if cfg.SMTPPort == "" {
		cfg.SMTPPort = "587"
	}

	sender := newSMTPEmailSender(cfg)
	if sender == nil {
		t.Fatal("sender not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := sender.Send(ctx, recipient, "PromptOS SMTP 冒烟测试", "这是一封来自 PromptOS smtpcheck 的测试邮件，验证生产邮件凭据。"); err != nil {
		t.Fatalf("live SMTP send failed: %v", err)
	}
}
