package helper

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

func ResolveMappedModelName(originModelName, modelMapping string, relayMode int) (string, bool, error) {
	isResponsesCompact := relayMode == constant.RelayModeResponsesCompact
	mappingModelName := originModelName
	if isResponsesCompact && strings.HasSuffix(originModelName, ratio_setting.CompactModelSuffix) {
		mappingModelName = strings.TrimSuffix(originModelName, ratio_setting.CompactModelSuffix)
	}

	mapped := false
	upstreamModelName := mappingModelName
	if modelMapping != "" && modelMapping != "{}" {
		modelMap := make(map[string]string)
		err := common.UnmarshalJsonStr(modelMapping, &modelMap)
		if err != nil {
			return "", false, fmt.Errorf("unmarshal_model_mapping_failed")
		}

		currentModel := mappingModelName
		visitedModels := map[string]bool{
			currentModel: true,
		}
		for {
			mappedModel, exists := modelMap[currentModel]
			if !exists || mappedModel == "" {
				break
			}
			if visitedModels[mappedModel] {
				if mappedModel == currentModel {
					if currentModel == originModelName {
						mapped = false
						upstreamModelName = currentModel
						break
					}
					mapped = true
					upstreamModelName = currentModel
					break
				}
				return "", false, errors.New("model_mapping_contains_cycle")
			}
			visitedModels[mappedModel] = true
			currentModel = mappedModel
			mapped = true
			upstreamModelName = currentModel
		}
	}

	return upstreamModelName, mapped, nil
}

func PrepareBillingModel(c *gin.Context, info *relaycommon.RelayInfo) error {
	if info == nil {
		return nil
	}
	billingModelName, _, err := ResolveMappedModelName(info.OriginModelName, c.GetString("model_mapping"), info.RelayMode)
	if err != nil {
		return err
	}
	if billingModelName != "" {
		if info.RelayMode == constant.RelayModeResponsesCompact {
			billingModelName = ratio_setting.WithCompactModelSuffix(billingModelName)
		}
		info.BillingModelName = billingModelName
	}
	return nil
}

func ModelMappedHelper(c *gin.Context, info *relaycommon.RelayInfo, request dto.Request) error {
	if info.ChannelMeta == nil {
		info.ChannelMeta = &relaycommon.ChannelMeta{}
	}

	upstreamModelName, mapped, err := ResolveMappedModelName(info.OriginModelName, c.GetString("model_mapping"), info.RelayMode)
	if err != nil {
		return err
	}
	if upstreamModelName != "" {
		info.UpstreamModelName = upstreamModelName
		if info.BillingModelName == "" {
			info.BillingModelName = upstreamModelName
			if info.RelayMode == constant.RelayModeResponsesCompact {
				info.BillingModelName = ratio_setting.WithCompactModelSuffix(upstreamModelName)
			}
		}
	}
	info.IsModelMapped = mapped
	if info.RelayMode == constant.RelayModeResponsesCompact {
		info.OriginModelName = ratio_setting.WithCompactModelSuffix(upstreamModelName)
	}
	if request != nil {
		request.SetModelName(info.UpstreamModelName)
	}
	return nil
}
