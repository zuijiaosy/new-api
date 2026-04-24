package ratio_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/types"
)

func TestApplyTieredPricingForGPT55ShortContext(t *testing.T) {
	priceData := &types.PriceData{
		ModelRatio:      1.25,
		CompletionRatio: 6,
		CacheRatio:      0.1,
	}

	applied := ApplyPromptTokenPricingOverrides("gpt-5.5", 272000, priceData)
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

func TestApplyTieredPricingForGPT55LongContext(t *testing.T) {
	priceData := &types.PriceData{
		ModelRatio:      1.25,
		CompletionRatio: 6,
		CacheRatio:      0.1,
	}

	applied := ApplyPromptTokenPricingOverrides("gpt-5.5", 272001, priceData)
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

func TestApplyTieredPricingDoesNotApplyForGPT55SnapshotModel(t *testing.T) {
	priceData := &types.PriceData{
		ModelRatio:      1.25,
		CompletionRatio: 6,
		CacheRatio:      0.1,
	}

	applied := ApplyPromptTokenPricingOverrides("gpt-5.5-2026-03-05", 500000, priceData)
	if applied {
		t.Fatal("expected tiered pricing not to be applied for snapshot model")
	}
	if priceData.ModelRatio != 1.25 {
		t.Fatalf("expected snapshot model ratio unchanged, got %v", priceData.ModelRatio)
	}
	if priceData.CompletionRatio != 6 {
		t.Fatalf("expected snapshot model completion ratio unchanged, got %v", priceData.CompletionRatio)
	}
	if priceData.CacheRatio != 0.1 {
		t.Fatalf("expected snapshot model cache ratio unchanged, got %v", priceData.CacheRatio)
	}
	if priceData.TieredPricingApplied {
		t.Fatal("expected snapshot model tiered pricing metadata to remain disabled")
	}
}

func TestApplyTieredPricingDoesNotApplyForGPT55MiniModel(t *testing.T) {
	priceData := &types.PriceData{
		ModelRatio:      0.2,
		CompletionRatio: 6,
		CacheRatio:      0.1,
	}

	applied := ApplyPromptTokenPricingOverrides("gpt-5.5-mini", 500000, priceData)
	if applied {
		t.Fatal("expected tiered pricing not to be applied for mini model")
	}
	if priceData.ModelRatio != 0.2 {
		t.Fatalf("expected mini model ratio unchanged, got %v", priceData.ModelRatio)
	}
	if priceData.CompletionRatio != 6 {
		t.Fatalf("expected mini model completion ratio unchanged, got %v", priceData.CompletionRatio)
	}
	if priceData.CacheRatio != 0.1 {
		t.Fatalf("expected mini model cache ratio unchanged, got %v", priceData.CacheRatio)
	}
	if priceData.TieredPricingApplied {
		t.Fatal("expected mini model tiered pricing metadata to remain disabled")
	}
}

func TestApplyTieredPricingForGPT55ProModel(t *testing.T) {
	priceData := &types.PriceData{
		ModelRatio:      3,
		CompletionRatio: 9,
		CacheRatio:      0.2,
	}

	applied := ApplyPromptTokenPricingOverrides("gpt-5.5-pro-2026-03-05", 500000, priceData)
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
