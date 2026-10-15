package platformapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tsundeer/isoumao-shared/contract"

	"promptos-backend/internal/platformapi"
)

// 这些用例不需要 MySQL/Redis：它们验证平台面的装配与否决路径，
// 让 PromptSystem 与另外三个服务一样有平台面的回归保护。

func newServer(t *testing.T, ping func(context.Context) error) *httptest.Server {
	t.Helper()
	h, err := platformapi.New(platformapi.Deps{
		Version:     "test",
		Environment: "development",
		InstanceID:  "i-1",
		Token:       "platform-tok",
		Ping:        ping,
		StorageNote: "s3:bucket-name",
	})
	if err != nil {
		t.Fatalf("挂载平台面失败: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle(contract.BasePath+"/", http.StripPrefix(contract.BasePath, h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestPlatformFaceIdentifiesService(t *testing.T) {
	srv := newServer(t, func(context.Context) error { return nil })
	c := contract.NewClient(srv.URL, "platform-tok", "probe")
	info, err := c.Info(context.Background())
	if err != nil {
		t.Fatalf("Info 失败: %v", err)
	}
	if info.Service != "promptos-backend" {
		t.Fatalf("服务名不符: %q", info.Service)
	}
	health, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health 失败: %v", err)
	}
	if health.Status != contract.StatusOK || len(health.Checks) != 2 {
		t.Fatalf("健康结果不符: %+v", health)
	}
}

// 站点的就绪探测必须能被平台面复用：全绿不报错、任一不健康即报错（评审 P2-3）。
// 原有用例只覆盖「未接线 → degraded」那一半，接线这一半没人覆盖，
// 于是 main.go 少传 Ping 这件事一直没被发现。
func TestPingFromReadiness(t *testing.T) {
	if err := platformapi.PingFromReadiness(func(context.Context) map[string]bool {
		return map[string]bool{"mysql": true, "redis": true}
	})(context.Background()); err != nil {
		t.Fatalf("全部就绪时不应报错: %v", err)
	}
	err := platformapi.PingFromReadiness(func(context.Context) map[string]bool {
		return map[string]bool{"mysql": true, "redis": false}
	})(context.Background())
	if err == nil {
		t.Fatal("redis 不健康时必须报错（否则平台面会说一切正常）")
	}
}

// 未接线探测必须记为 degraded，而不是让 /health 看起来一切正常。
func TestPlatformFaceReportsUnwiredProbe(t *testing.T) {
	srv := newServer(t, nil)
	c := contract.NewClient(srv.URL, "platform-tok", "probe")
	health, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health 失败: %v", err)
	}
	if health.Status != contract.StatusDegraded {
		t.Fatalf("未接线探测应为 degraded: %+v", health)
	}
}

func TestPlatformFaceReportsDatabaseDown(t *testing.T) {
	srv := newServer(t, func(context.Context) error { return errors.New("boom") })
	c := contract.NewClient(srv.URL, "platform-tok", "probe")
	health, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health 失败: %v", err)
	}
	if health.Status != contract.StatusDown {
		t.Fatalf("库不可用应为 down: %+v", health)
	}
}

func TestPlatformFaceRejectsWrongToken(t *testing.T) {
	srv := newServer(t, nil)
	c := contract.NewClient(srv.URL, "wrong", "probe")
	if _, err := c.Info(context.Background()); err == nil {
		t.Fatal("错误令牌应被拒绝")
	} else if ce, ok := err.(*contract.Error); !ok || ce.Code != contract.CodeUnauthorized {
		t.Fatalf("错误码应为 unauthorized，实际 %v", err)
	}
}
