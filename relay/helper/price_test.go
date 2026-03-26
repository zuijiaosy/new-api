package helper

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func TestModelPriceHelperDoesNotApplyGPT54TieredPricingToSnapshotModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ratio_setting.InitRatioSettings()

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-5.4-2026-03-05",
		UsingGroup:      "default",
	}

	priceData, err := ModelPriceHelper(ctx, info, 300000, &types.TokenCountMeta{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if priceData.ModelRatio != 1.25 {
		t.Fatalf("expected model ratio 1.25, got %v", priceData.ModelRatio)
	}
	if priceData.CompletionRatio != 6 {
		t.Fatalf("expected completion ratio 6, got %v", priceData.CompletionRatio)
	}
	if priceData.CacheRatio != 1 {
		t.Fatalf("expected cache ratio 1, got %v", priceData.CacheRatio)
	}
	if priceData.QuotaToPreConsume != 375000 {
		t.Fatalf("expected pre-consume quota 375000, got %d", priceData.QuotaToPreConsume)
	}
	if priceData.TieredPricingApplied {
		t.Fatalf("expected tiered pricing metadata to remain disabled, got applied=%v tier=%q", priceData.TieredPricingApplied, priceData.TieredPricingTier)
	}
}
