package ratio_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
)

func TestApplyTieredPricingForGPT54ShortContext(t *testing.T) {
	priceData := &types.PriceData{
		ModelRatio:      1.25,
		CompletionRatio: 6,
		CacheRatio:      0.1,
	}

	applied := ApplyPromptTokenPricingOverrides("gpt-5.4", 272000, priceData)
	if !applied {
		t.Fatal("expected tiered pricing to be applied")
	}
	if priceData.ModelRatio != 1.25 {
		t.Fatalf("expected short-context model ratio 1.25, got %v", priceData.ModelRatio)
	}
	if priceData.CompletionRatio != 6 {
		t.Fatalf("expected short-context completion ratio 6, got %v", priceData.CompletionRatio)
	}
	if priceData.CacheRatio != 0.1 {
		t.Fatalf("expected short-context cache ratio 0.1, got %v", priceData.CacheRatio)
	}
	if !priceData.TieredPricingApplied {
		t.Fatal("expected tiered pricing metadata to be marked as applied")
	}
	if priceData.TieredPricingTier != "short" {
		t.Fatalf("expected short tier, got %q", priceData.TieredPricingTier)
	}
	if priceData.TieredPricingInputTokens != 272000 {
		t.Fatalf("expected input tokens 272000, got %d", priceData.TieredPricingInputTokens)
	}
	if priceData.TieredPricingThreshold != 272000 {
		t.Fatalf("expected threshold 272000, got %d", priceData.TieredPricingThreshold)
	}
}

func TestApplyTieredPricingForGPT54LongContext(t *testing.T) {
	priceData := &types.PriceData{
		ModelRatio:      1.25,
		CompletionRatio: 6,
		CacheRatio:      0.1,
	}

	applied := ApplyPromptTokenPricingOverrides("gpt-5.4", 272001, priceData)
	if !applied {
		t.Fatal("expected tiered pricing to be applied")
	}
	if priceData.ModelRatio != 2.5 {
		t.Fatalf("expected long-context model ratio 2.5, got %v", priceData.ModelRatio)
	}
	if priceData.CompletionRatio != 4.5 {
		t.Fatalf("expected long-context completion ratio 4.5, got %v", priceData.CompletionRatio)
	}
	if priceData.CacheRatio != 0.1 {
		t.Fatalf("expected long-context cache ratio 0.1, got %v", priceData.CacheRatio)
	}
	if priceData.TieredPricingTier != "long" {
		t.Fatalf("expected long tier, got %q", priceData.TieredPricingTier)
	}
}

func TestApplyTieredPricingForGPT54SnapshotModel(t *testing.T) {
	priceData := &types.PriceData{
		ModelRatio:      1.25,
		CompletionRatio: 6,
		CacheRatio:      0.1,
	}

	applied := ApplyPromptTokenPricingOverrides("gpt-5.4-2026-03-05", 500000, priceData)
	if !applied {
		t.Fatal("expected tiered pricing to be applied for snapshot model")
	}
	if priceData.ModelRatio != 2.5 {
		t.Fatalf("expected snapshot model long-context ratio 2.5, got %v", priceData.ModelRatio)
	}
	if priceData.CompletionRatio != 4.5 {
		t.Fatalf("expected snapshot model long-context completion ratio 4.5, got %v", priceData.CompletionRatio)
	}
	if priceData.TieredPricingTier != "long" {
		t.Fatalf("expected snapshot model long tier, got %q", priceData.TieredPricingTier)
	}
}

func TestApplyTieredPricingForGPT54ProModel(t *testing.T) {
	priceData := &types.PriceData{
		ModelRatio:      3,
		CompletionRatio: 9,
		CacheRatio:      0.2,
	}

	applied := ApplyPromptTokenPricingOverrides("gpt-5.4-pro-2026-03-05", 500000, priceData)
	if applied {
		t.Fatal("expected tiered pricing not to be applied for pro model")
	}
	if priceData.ModelRatio != 3 {
		t.Fatalf("expected model ratio unchanged, got %v", priceData.ModelRatio)
	}
	if priceData.CompletionRatio != 9 {
		t.Fatalf("expected completion ratio unchanged, got %v", priceData.CompletionRatio)
	}
	if priceData.CacheRatio != 0.2 {
		t.Fatalf("expected cache ratio unchanged, got %v", priceData.CacheRatio)
	}
	if priceData.TieredPricingApplied {
		t.Fatal("expected tiered pricing metadata to remain disabled")
	}
}
