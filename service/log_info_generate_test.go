package service

import (
	"net/http/httptest"
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func TestGenerateTextOtherInfoIncludesTieredPricingMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)

	relayInfo := &relaycommon.RelayInfo{
		StartTime:         time.Unix(100, 0),
		FirstResponseTime: time.Unix(100, 0),
		ChannelMeta:       &relaycommon.ChannelMeta{},
		PriceData: types.PriceData{
			TieredPricingApplied:     true,
			TieredPricingTier:        "long",
			TieredPricingInputTokens: 300000,
			TieredPricingThreshold:   272000,
		},
	}

	other := GenerateTextOtherInfo(ctx, relayInfo, 2.5, 1, 4.5, 100, 0.1, 0, -1)
	if other["tiered_pricing_applied"] != true {
		t.Fatalf("expected tiered_pricing_applied=true, got %#v", other["tiered_pricing_applied"])
	}
	if other["tiered_pricing_tier"] != "long" {
		t.Fatalf("expected tiered_pricing_tier=long, got %#v", other["tiered_pricing_tier"])
	}
	if other["tiered_pricing_input_tokens"] != 300000 {
		t.Fatalf("expected tiered_pricing_input_tokens=300000, got %#v", other["tiered_pricing_input_tokens"])
	}
	if other["tiered_pricing_threshold"] != 272000 {
		t.Fatalf("expected tiered_pricing_threshold=272000, got %#v", other["tiered_pricing_threshold"])
	}
}
