package ratio_setting

import "github.com/QuantumNous/new-api/types"

const (
	GPT55TieredPricingModelName      = "gpt-5.5"
	GPT55TieredPricingThreshold      = 272000
	gpt55ShortContextModelRatio      = 1.25
	gpt55LongContextModelRatio       = 2.5
	gpt55TieredPricingCacheRatio     = 0.1
	gpt55ShortContextCompletionRatio = 6.0
	gpt55LongContextCompletionRatio  = 4.5
)

func IsGPT55TieredPricingModel(modelName string) bool {
	return modelName == GPT55TieredPricingModelName
}

func applyGPT55TieredPricing(modelName string, inputTokens int, priceData *types.PriceData) bool {
	if priceData == nil || !IsGPT55TieredPricingModel(modelName) {
		return false
	}

	priceData.TieredPricingApplied = true
	priceData.TieredPricingInputTokens = inputTokens
	priceData.TieredPricingThreshold = GPT55TieredPricingThreshold
	priceData.CacheRatio = gpt55TieredPricingCacheRatio

	if inputTokens > GPT55TieredPricingThreshold {
		priceData.ModelRatio = gpt55LongContextModelRatio
		priceData.CompletionRatio = gpt55LongContextCompletionRatio
		priceData.TieredPricingTier = "long"
		return true
	}

	priceData.ModelRatio = gpt55ShortContextModelRatio
	priceData.CompletionRatio = gpt55ShortContextCompletionRatio
	priceData.TieredPricingTier = "short"
	return true
}
