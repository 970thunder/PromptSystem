// Package platformapi 把 PromptSystem 后端接入「平台统一契约」（/api/platform/v1）。
//
// 读面（能力清单、健康探测、状态汇总）已下沉到 contract.NewStandardHandler：
// 这里只保留站点差异项，避免复制一份实现。
package platformapi

import (
	"context"
	"net/http"

	"github.com/Tsundeer/isoumao-shared/contract"
)

// Deps 是接入所需的最小依赖。
type Deps struct {
	Version      string
	Environment  string
	InstanceID   string
	Token        string
	TrustedAddrs []string
	AllowedPeers []string
	Ping         func(context.Context) error
	StorageNote  string
}

// New 构造标准平台面（info / health）。
func New(deps Deps) (http.Handler, error) {
	return contract.NewStandardHandler(contract.StandardDeps{
		Service:           "promptos-backend",
		Version:           deps.Version,
		Environment:       deps.Environment,
		InstanceID:        deps.InstanceID,
		Token:             deps.Token,
		TrustedProxyCIDRs: deps.TrustedAddrs,
		AllowedPeers:      deps.AllowedPeers,
		Ping:              deps.Ping,
		ExtraChecks:       []contract.Check{contract.StorageCheck(deps.StorageNote)},
	})
}
