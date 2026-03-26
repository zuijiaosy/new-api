package ratio_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestFormatMatchingModelNameKeepsGPT54Mini(t *testing.T) {
	got := FormatMatchingModelName("gpt-5.4-mini")
	if got != "gpt-5.4-mini" {
		t.Fatalf("expected gpt-5.4-mini to keep original name, got %q", got)
	}
}

func TestGetModelRatioUsesDedicatedGPT54MiniEntry(t *testing.T) {
	backupRatioJSON := mustMarshalMap(t, GetModelRatioCopy())
	defer func() {
		err := UpdateModelRatioByJSONString(backupRatioJSON)
		if err != nil {
			t.Fatalf("failed to restore model ratio: %v", err)
		}
	}()

	ratioMap := GetModelRatioCopy()
	ratioMap["gpt-5.4"] = 1.25
	ratioMap["gpt-5.4-mini"] = 0.2

	ratioJSON := mustMarshalMap(t, ratioMap)
	err := UpdateModelRatioByJSONString(ratioJSON)
	if err != nil {
		t.Fatalf("failed to update model ratio: %v", err)
	}

	ratio, ok, matchName := GetModelRatio("gpt-5.4-mini")
	if !ok {
		t.Fatal("expected gpt-5.4-mini ratio lookup to succeed")
	}
	if matchName != "gpt-5.4-mini" {
		t.Fatalf("expected matched model name gpt-5.4-mini, got %q", matchName)
	}
	if ratio != 0.2 {
		t.Fatalf("expected gpt-5.4-mini ratio 0.2, got %v", ratio)
	}
}

func TestGetModelPriceUsesDedicatedGPT54MiniEntry(t *testing.T) {
	backupPriceJSON := mustMarshalMap(t, GetModelPriceCopy())
	defer func() {
		err := UpdateModelPriceByJSONString(backupPriceJSON)
		if err != nil {
			t.Fatalf("failed to restore model price: %v", err)
		}
	}()

	priceMap := GetModelPriceCopy()
	priceMap["gpt-5.4-mini"] = 0.123

	priceJSON := mustMarshalMap(t, priceMap)
	err := UpdateModelPriceByJSONString(priceJSON)
	if err != nil {
		t.Fatalf("failed to update model price: %v", err)
	}

	price, ok := GetModelPrice("gpt-5.4-mini", false)
	if !ok {
		t.Fatal("expected gpt-5.4-mini price lookup to succeed")
	}
	if price != 0.123 {
		t.Fatalf("expected gpt-5.4-mini price 0.123, got %v", price)
	}
}

func TestGetCompletionRatioUsesDedicatedGPT54MiniEntry(t *testing.T) {
	backupCompletionJSON := mustMarshalMap(t, GetCompletionRatioCopy())
	defer func() {
		err := UpdateCompletionRatioByJSONString(backupCompletionJSON)
		if err != nil {
			t.Fatalf("failed to restore completion ratio: %v", err)
		}
	}()

	completionMap := GetCompletionRatioCopy()
	completionMap["gpt-5.4-mini"] = 2.5

	completionJSON := mustMarshalMap(t, completionMap)
	err := UpdateCompletionRatioByJSONString(completionJSON)
	if err != nil {
		t.Fatalf("failed to update completion ratio: %v", err)
	}

	ratio := GetCompletionRatio("gpt-5.4-mini")
	if ratio != 2.5 {
		t.Fatalf("expected gpt-5.4-mini completion ratio 2.5, got %v", ratio)
	}
}

func TestGetCompletionRatioInfoDoesNotLockGPT54Mini(t *testing.T) {
	backupCompletionJSON := mustMarshalMap(t, GetCompletionRatioCopy())
	defer func() {
		err := UpdateCompletionRatioByJSONString(backupCompletionJSON)
		if err != nil {
			t.Fatalf("failed to restore completion ratio: %v", err)
		}
	}()

	completionMap := GetCompletionRatioCopy()
	completionMap["gpt-5.4-mini"] = 2.5

	completionJSON := mustMarshalMap(t, completionMap)
	err := UpdateCompletionRatioByJSONString(completionJSON)
	if err != nil {
		t.Fatalf("failed to update completion ratio: %v", err)
	}

	info := GetCompletionRatioInfo("gpt-5.4-mini")
	if info.Locked {
		t.Fatal("expected gpt-5.4-mini completion ratio to be configurable")
	}
	if info.Ratio != 2.5 {
		t.Fatalf("expected gpt-5.4-mini completion ratio info 2.5, got %v", info.Ratio)
	}
}

func mustMarshalMap(t *testing.T, data map[string]float64) string {
	t.Helper()
	jsonBytes, err := common.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal map: %v", err)
	}
	return string(jsonBytes)
}
