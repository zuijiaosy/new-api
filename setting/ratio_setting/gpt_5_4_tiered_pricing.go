package ratio_setting

import "github.com/QuantumNous/new-api/types"

const (
	GPT54TieredPricingModelName      = "gpt-5.4"
	GPT54TieredPricingThreshold      = 272000
	gpt54ShortContextModelRatio      = 1.25
	gpt54LongContextModelRatio       = 2.5
	gpt54TieredPricingCacheRatio     = 0.1
	gpt54ShortContextCompletionRatio = 6.0
	gpt54LongContextCompletionRatio  = 4.5
)

func IsGPT54TieredPricingModel(modelName string) bool {
	return modelName == GPT54TieredPricingModelName
}

func ApplyPromptTokenPricingOverrides(modelName string, inputTokens int, priceData *types.PriceData) bool {
	return applyGPT54TieredPricing(modelName, inputTokens, priceData)
}

func applyGPT54TieredPricing(modelName string, inputTokens int, priceData *types.PriceData) bool {
	if priceData == nil || !IsGPT54TieredPricingModel(modelName) {
		return false
	}

	priceData.TieredPricingApplied = true
	priceData.TieredPricingInputTokens = inputTokens
	priceData.TieredPricingThreshold = GPT54TieredPricingThreshold
	priceData.CacheRatio = gpt54TieredPricingCacheRatio

	if inputTokens > GPT54TieredPricingThreshold {
		priceData.ModelRatio = gpt54LongContextModelRatio
		priceData.CompletionRatio = gpt54LongContextCompletionRatio
		priceData.TieredPricingTier = "long"
		return true
	}

	priceData.ModelRatio = gpt54ShortContextModelRatio
	priceData.CompletionRatio = gpt54ShortContextCompletionRatio
	priceData.TieredPricingTier = "short"
	return true
}
