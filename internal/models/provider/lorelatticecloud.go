package provider

import (
	"fmt"
	"os"
	"strings"

	"github.com/Pototoooo/lorelattice/internal/types"
)

const (
	ProviderLoreLatticeCloud ProviderName = "lorelatticecloud"
)

// LoreLatticeCloudBaseURL is intentionally deployment-defined. LoreLattice does
// not assume ownership of, or silently route requests to, an external cloud
// endpoint.
var LoreLatticeCloudBaseURL = strings.TrimRight(
	strings.TrimSpace(os.Getenv("LORELATTICE_CLOUD_BASE_URL")),
	"/",
)

type LoreLatticeCloudProvider struct{}

func init() {
	Register(&LoreLatticeCloudProvider{})
}

func (p *LoreLatticeCloudProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderLoreLatticeCloud,
		DisplayName: "LoreLatticeCloud",
		Description: "LoreLattice云服务，模型：chat, embedding, rerank, vlm",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: LoreLatticeCloudBaseURL,
			types.ModelTypeEmbedding:   LoreLatticeCloudBaseURL,
			types.ModelTypeRerank:      LoreLatticeCloudBaseURL,
			types.ModelTypeVLLM:        LoreLatticeCloudBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
			types.ModelTypeEmbedding,
			types.ModelTypeRerank,
			types.ModelTypeVLLM,
		},
		RequiresAuth: true,
	}
}

func (p *LoreLatticeCloudProvider) ValidateConfig(config *Config) error {
	if LoreLatticeCloudBaseURL == "" {
		return fmt.Errorf("LORELATTICE_CLOUD_BASE_URL is not configured")
	}
	// AppID/AppSecret 通过专用初始化接口写入，此处仅做结构校验。
	// 其中 AppSecret 字段当前实际承载上游 API Key。
	return nil
}
