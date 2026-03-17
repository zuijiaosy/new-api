package helper

import (
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func TestModelPriceHelperAppliesGPT54SnapshotTieredPricing(t *testing.T) {
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
	if priceData.ModelRatio != 2.5 {
		t.Fatalf("expected model ratio 2.5, got %v", priceData.ModelRatio)
	}
	if priceData.CompletionRatio != 4.5 {
		t.Fatalf("expected completion ratio 4.5, got %v", priceData.CompletionRatio)
	}
	if priceData.CacheRatio != 0.1 {
		t.Fatalf("expected cache ratio 0.1, got %v", priceData.CacheRatio)
	}
	if priceData.QuotaToPreConsume != 750000 {
		t.Fatalf("expected pre-consume quota 750000, got %d", priceData.QuotaToPreConsume)
	}
	if !priceData.TieredPricingApplied || priceData.TieredPricingTier != "long" {
		t.Fatalf("expected long tiered pricing metadata, got applied=%v tier=%q", priceData.TieredPricingApplied, priceData.TieredPricingTier)
	}
}
