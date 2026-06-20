package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveMappedModelNameUsesChainTail(t *testing.T) {
	model, mapped, err := ResolveMappedModelName(
		"claude-sonnet-4.6",
		`{"claude-sonnet-4.6":"deepseek-chat","deepseek-chat":"deepseek-v4-pro"}`,
		relayconstant.RelayModeChatCompletions,
	)

	require.NoError(t, err)
	require.True(t, mapped)
	require.Equal(t, "deepseek-v4-pro", model)
}

func TestResolveMappedModelNameSelfMappingIsNotMapped(t *testing.T) {
	model, mapped, err := ResolveMappedModelName(
		"deepseek-v4-pro",
		`{"deepseek-v4-pro":"deepseek-v4-pro"}`,
		relayconstant.RelayModeChatCompletions,
	)

	require.NoError(t, err)
	require.False(t, mapped)
	require.Equal(t, "deepseek-v4-pro", model)
}

func TestResolveMappedModelNameDetectsCycle(t *testing.T) {
	_, _, err := ResolveMappedModelName(
		"claude-sonnet-4.6",
		`{"claude-sonnet-4.6":"deepseek-v4-pro","deepseek-v4-pro":"claude-sonnet-4.6"}`,
		relayconstant.RelayModeChatCompletions,
	)

	require.Error(t, err)
	require.EqualError(t, err, "model_mapping_contains_cycle")
}

func TestPrepareBillingModelUsesMappedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("model_mapping", `{"claude-sonnet-4.6":"deepseek-v4-pro"}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "claude-sonnet-4.6",
		RelayMode:       relayconstant.RelayModeChatCompletions,
	}

	require.NoError(t, PrepareBillingModel(ctx, info))

	require.Equal(t, "claude-sonnet-4.6", info.OriginModelName)
	require.Equal(t, "deepseek-v4-pro", info.BillingModelName)
}

func TestPrepareBillingModelKeepsCompactBillingSuffix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses/compact", nil)
	ctx.Set("model_mapping", `{"gpt-5":"deepseek-v4-pro"}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: ratio_setting.WithCompactModelSuffix("gpt-5"),
		RelayMode:       relayconstant.RelayModeResponsesCompact,
	}

	require.NoError(t, PrepareBillingModel(ctx, info))

	require.Equal(t, ratio_setting.WithCompactModelSuffix("deepseek-v4-pro"), info.BillingModelName)
}
