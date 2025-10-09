package ratio_setting

import (
	"encoding/json"
	"one-api/common"
	"one-api/setting/operation_setting"
	"strings"
	"sync"
)

// from songquanpeng/one-api
const (
	USD2RMB = 7.3 // 暂定 1 USD = 7.3 RMB
	USD     = 500 // $0.002 = 1 -> $1 = 500
	RMB     = USD / USD2RMB
)

// modelRatio
// https://platform.openai.com/docs/models/model-endpoint-compatibility
// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/Blfmc9dlf
// https://openai.com/pricing
// TODO: when a new api is enabled, check the pricing here
// 1 === $0.002 / 1K tokens
// 1 === ￥0.014 / 1k tokens

var defaultModelRatio = map[string]float64{
	//"midjourney":                50,
	"gpt-4-gizmo-*":  15,
	"gpt-4o-gizmo-*": 2.5,
	"gpt-4-all":      15,
	"gpt-4o-all":     15,
	"gpt-4":          15,
	//"gpt-4-0314":                   15, //deprecated
	"gpt-4-0613": 15,
	"gpt-4-32k":  30,
	//"gpt-4-32k-0314":               30, //deprecated
	"gpt-4-32k-0613":                          30,
	"gpt-4-1106-preview":                      5,    // $10 / 1M tokens
	"gpt-4-0125-preview":                      5,    // $10 / 1M tokens
	"gpt-4-turbo-preview":                     5,    // $10 / 1M tokens
	"gpt-4-vision-preview":                    5,    // $10 / 1M tokens
	"gpt-4-1106-vision-preview":               5,    // $10 / 1M tokens
	"chatgpt-4o-latest":                       2.5,  // $5 / 1M tokens
	"gpt-4o":                                  1.25, // $2.5 / 1M tokens
	"gpt-4o-audio-preview":                    1.25, // $2.5 / 1M tokens
	"gpt-4o-audio-preview-2024-10-01":         1.25, // $2.5 / 1M tokens
	"gpt-4o-2024-05-13":                       2.5,  // $5 / 1M tokens
	"gpt-4o-2024-08-06":                       1.25, // $2.5 / 1M tokens
	"gpt-4o-2024-11-20":                       1.25, // $2.5 / 1M tokens
	"gpt-4o-realtime-preview":                 2.5,
	"gpt-4o-realtime-preview-2024-10-01":      2.5,
	"gpt-4o-realtime-preview-2024-12-17":      2.5,
	"gpt-4o-mini-realtime-preview":            0.3,
	"gpt-4o-mini-realtime-preview-2024-12-17": 0.3,
	"gpt-4.1":                          1.0,  // $2 / 1M tokens
	"gpt-4.1-2025-04-14":               1.0,  // $2 / 1M tokens
	"gpt-4.1-mini":                     0.2,  // $0.4 / 1M tokens
	"gpt-4.1-mini-2025-04-14":          0.2,  // $0.4 / 1M tokens
	"gpt-4.1-nano":                     0.05, // $0.1 / 1M tokens
	"gpt-4.1-nano-2025-04-14":          0.05, // $0.1 / 1M tokens
	"gpt-image-1":                      2.5,  // $5 / 1M tokens
	"o1":                               7.5,  // $15 / 1M tokens
	"o1-2024-12-17":                    7.5,  // $15 / 1M tokens
	"o1-preview":                       7.5,  // $15 / 1M tokens
	"o1-preview-2024-09-12":            7.5,  // $15 / 1M tokens
	"o1-mini":                          0.55, // $1.1 / 1M tokens
	"o1-mini-2024-09-12":               0.55, // $1.1 / 1M tokens
	"o1-pro":                           75.0, // $150 / 1M tokens
	"o1-pro-2025-03-19":                75.0, // $150 / 1M tokens
	"o3-mini":                          0.55,
	"o3-mini-2025-01-31":               0.55,
	"o3-mini-high":                     0.55,
	"o3-mini-2025-01-31-high":          0.55,
	"o3-mini-low":                      0.55,
	"o3-mini-2025-01-31-low":           0.55,
	"o3-mini-medium":                   0.55,
	"o3-mini-2025-01-31-medium":        0.55,
	"o3":                               1.0,  // $2 / 1M tokens
	"o3-2025-04-16":                    1.0,  // $2 / 1M tokens
	"o3-pro":                           10.0, // $20 / 1M tokens
	"o3-pro-2025-06-10":                10.0, // $20 / 1M tokens
	"o3-deep-research":                 5.0,  // $10 / 1M tokens
	"o3-deep-research-2025-06-26":      5.0,  // $10 / 1M tokens
	"o4-mini":                          0.55, // $1.1 / 1M tokens
	"o4-mini-2025-04-16":               0.55, // $1.1 / 1M tokens
	"o4-mini-deep-research":            1.0,  // $2 / 1M tokens
	"o4-mini-deep-research-2025-06-26": 1.0,  // $2 / 1M tokens
	"gpt-4o-mini":                      0.075,
	"gpt-4o-mini-2024-07-18":           0.075,
	"gpt-4-turbo":                      5, // $0.01 / 1K tokens
	"gpt-4-turbo-2024-04-09":           5, // $0.01 / 1K tokens
	"gpt-4.5-preview":                  37.5,
	"gpt-4.5-preview-2025-02-27":       37.5,
	"gpt-5":                            0.625,
	"gpt-5-2025-08-07":                 0.625,
	"gpt-5-chat-latest":                0.625,
	"gpt-5-mini":                       0.125,
	"gpt-5-mini-2025-08-07":            0.125,
	"gpt-5-nano":                       0.025,
	"gpt-5-nano-2025-08-07":            0.025,
	//"gpt-3.5-turbo-0301":           0.75, //deprecated
	"gpt-3.5-turbo":          0.25,
	"gpt-3.5-turbo-0613":     0.75,
	"gpt-3.5-turbo-16k":      1.5, // $0.003 / 1K tokens
	"gpt-3.5-turbo-16k-0613": 1.5,
	"gpt-3.5-turbo-instruct": 0.75, // $0.0015 / 1K tokens
	"gpt-3.5-turbo-1106":     0.5,  // $0.001 / 1K tokens
	"gpt-3.5-turbo-0125":     0.25,
	"babbage-002":            0.2, // $0.0004 / 1K tokens
	"davinci-002":            1,   // $0.002 / 1K tokens
	"text-ada-001":           0.2,
	"text-babbage-001":       0.25,
	"text-curie-001":         1,
	//"text-davinci-002":               10,
	//"text-davinci-003":               10,
	"text-davinci-edit-001":                     10,
	"code-davinci-edit-001":                     10,
	"whisper-1":                                 15,  // $0.006 / minute -> $0.006 / 150 words -> $0.006 / 200 tokens -> $0.03 / 1k tokens
	"tts-1":                                     7.5, // 1k characters -> $0.015
	"tts-1-1106":                                7.5, // 1k characters -> $0.015
	"tts-1-hd":                                  15,  // 1k characters -> $0.03
	"tts-1-hd-1106":                             15,  // 1k characters -> $0.03
	"davinci":                                   10,
	"curie":                                     10,
	"babbage":                                   10,
	"ada":                                       10,
	"text-embedding-3-small":                    0.01,
	"text-embedding-3-large":                    0.065,
	"text-embedding-ada-002":                    0.05,
	"text-search-ada-doc-001":                   10,
	"text-moderation-stable":                    0.1,
	"text-moderation-latest":                    0.1,
	"claude-instant-1":                          0.4,   // $0.8 / 1M tokens
	"claude-2.0":                                4,     // $8 / 1M tokens
	"claude-2.1":                                4,     // $8 / 1M tokens
	"claude-3-haiku-20240307":                   0.125, // $0.25 / 1M tokens
	"claude-3-5-haiku-20241022":                 0.5,   // $1 / 1M tokens
	"claude-3-sonnet-20240229":                  1.5,   // $3 / 1M tokens
	"claude-3-5-sonnet-20240620":                1.5,
	"claude-3-5-sonnet-20241022":                1.5,
	"claude-3-7-sonnet-20250219":                1.5,
	"claude-3-7-sonnet-20250219-thinking":       1.5,
	"claude-sonnet-4-20250514":                  1.5,
	"claude-sonnet-4-5-20250929":                1.5,
	"claude-3-opus-20240229":                    7.5, // $15 / 1M tokens
	"claude-opus-4-20250514":                    7.5,
	"claude-opus-4-1-20250805":                  7.5,
	"ERNIE-4.0-8K":                              0.120 * RMB,
	"ERNIE-3.5-8K":                              0.012 * RMB,
	"ERNIE-3.5-8K-0205":                         0.024 * RMB,
	"ERNIE-3.5-8K-1222":                         0.012 * RMB,
	"ERNIE-Bot-8K":                              0.024 * RMB,
	"ERNIE-3.5-4K-0205":                         0.012 * RMB,
	"ERNIE-Speed-8K":                            0.004 * RMB,
	"ERNIE-Speed-128K":                          0.004 * RMB,
	"ERNIE-Lite-8K-0922":                        0.008 * RMB,
	"ERNIE-Lite-8K-0308":                        0.003 * RMB,
	"ERNIE-Tiny-8K":                             0.001 * RMB,
	"BLOOMZ-7B":                                 0.004 * RMB,
	"Embedding-V1":                              0.002 * RMB,
	"bge-large-zh":                              0.002 * RMB,
	"bge-large-en":                              0.002 * RMB,
	"tao-8k":                                    0.002 * RMB,
	"PaLM-2":                                    1,
	"gemini-1.5-pro-latest":                     1.25, // $3.5 / 1M tokens
	"gemini-1.5-flash-latest":                   0.075,
	"gemini-2.0-flash":                          0.05,
	"gemini-2.5-pro-exp-03-25":                  0.625,
	"gemini-2.5-pro-preview-03-25":              0.625,
	"gemini-2.5-pro":                            0.625,
	"gemini-2.5-flash-preview-04-17":            0.075,
	"gemini-2.5-flash-preview-04-17-thinking":   0.075,
	"gemini-2.5-flash-preview-04-17-nothinking": 0.075,
	"gemini-2.5-flash-preview-05-20":            0.075,
	"gemini-2.5-flash-preview-05-20-thinking":   0.075,
	"gemini-2.5-flash-preview-05-20-nothinking": 0.075,
	"gemini-2.5-flash-thinking-*":               0.075, // 用于为后续所有2.5 flash thinking budget 模型设置默认倍率
	"gemini-2.5-pro-thinking-*":                 0.625, // 用于为后续所有2.5 pro thinking budget 模型设置默认倍率
	"gemini-2.5-flash-lite-preview-thinking-*":  0.05,
	"gemini-2.5-flash-lite-preview-06-17":       0.05,
	"gemini-2.5-flash":                          0.15,
	"gemini-robotics-er-1.5-preview":            0.15,
	"gemini-embedding-001":                      0.075,
	"text-embedding-004":                        0.001,
	"chatglm_turbo":                             0.3572,     // ￥0.005 / 1k tokens
	"chatglm_pro":                               0.7143,     // ￥0.01 / 1k tokens
	"chatglm_std":                               0.3572,     // ￥0.005 / 1k tokens
	"chatglm_lite":                              0.1429,     // ￥0.002 / 1k tokens
	"glm-4":                                     7.143,      // ￥0.1 / 1k tokens
	"glm-4v":                                    0.05 * RMB, // ￥0.05 / 1k tokens
	"glm-4-alltools":                            0.1 * RMB,  // ￥0.1 / 1k tokens
	"glm-3-turbo":                               0.3572,
	"glm-4-plus":                                0.05 * RMB,
	"glm-4-0520":                                0.1 * RMB,
	"glm-4-air":                                 0.001 * RMB,
	"glm-4-airx":                                0.01 * RMB,
	"glm-4-long":                                0.001 * RMB,
	"glm-4-flash":                               0,
	"glm-4v-plus":                               0.01 * RMB,
	"qwen-turbo":                                0.8572, // ￥0.012 / 1k tokens
	"qwen-plus":                                 10,     // ￥0.14 / 1k tokens
	"text-embedding-v1":                         0.05,   // ￥0.0007 / 1k tokens
	"SparkDesk-v1.1":                            1.2858, // ￥0.018 / 1k tokens
	"SparkDesk-v2.1":                            1.2858, // ￥0.018 / 1k tokens
	"SparkDesk-v3.1":                            1.2858, // ￥0.018 / 1k tokens
	"SparkDesk-v3.5":                            1.2858, // ￥0.018 / 1k tokens
	"SparkDesk-v4.0":                            1.2858,
	"360GPT_S2_V9":                              0.8572, // ¥0.012 / 1k tokens
	"360gpt-turbo":                              0.0858, // ¥0.0012 / 1k tokens
	"360gpt-turbo-responsibility-8k":            0.8572, // ¥0.012 / 1k tokens
	"360gpt-pro":                                0.8572, // ¥0.012 / 1k tokens
	"360gpt2-pro":                               0.8572, // ¥0.012 / 1k tokens
	"embedding-bert-512-v1":                     0.0715, // ¥0.001 / 1k tokens
	"embedding_s1_v1":                           0.0715, // ¥0.001 / 1k tokens
	"semantic_similarity_s1_v1":                 0.0715, // ¥0.001 / 1k tokens
	"hunyuan":                                   7.143,  // ¥0.1 / 1k tokens  // https://cloud.tencent.com/document/product/1729/97731#e0e6be58-60c8-469f-bdeb-6c264ce3b4d0
	// https://platform.lingyiwanwu.com/docs#-计费单元
	// 已经按照 7.2 来换算美元价格
	"yi-34b-chat-0205":       0.18,
	"yi-34b-chat-200k":       0.864,
	"yi-vl-plus":             0.432,
	"yi-large":               20.0 / 1000 * RMB,
	"yi-medium":              2.5 / 1000 * RMB,
	"yi-vision":              6.0 / 1000 * RMB,
	"yi-medium-200k":         12.0 / 1000 * RMB,
	"yi-spark":               1.0 / 1000 * RMB,
	"yi-large-rag":           25.0 / 1000 * RMB,
	"yi-large-turbo":         12.0 / 1000 * RMB,
	"yi-large-preview":       20.0 / 1000 * RMB,
	"yi-large-rag-preview":   25.0 / 1000 * RMB,
	"command":                0.5,
	"command-nightly":        0.5,
	"command-light":          0.5,
	"command-light-nightly":  0.5,
	"command-r":              0.25,
	"command-r-plus":         1.5,
	"command-r-08-2024":      0.075,
	"command-r-plus-08-2024": 1.25,
	"deepseek-chat":          0.27 / 2,
	"deepseek-coder":         0.27 / 2,
	"deepseek-reasoner":      0.55 / 2, // 0.55 / 1k tokens
	// Perplexity online 模型对搜索额外收费，有需要应自行调整，此处不计入搜索费用
	"llama-3-sonar-small-32k-chat":   0.2 / 1000 * USD,
	"llama-3-sonar-small-32k-online": 0.2 / 1000 * USD,
	"llama-3-sonar-large-32k-chat":   1 / 1000 * USD,
	"llama-3-sonar-large-32k-online": 1 / 1000 * USD,
	// grok
	"grok-3-beta":           1.5,
	"grok-3-mini-beta":      0.15,
	"grok-2":                1,
	"grok-2-vision":         1,
	"grok-beta":             2.5,
	"grok-vision-beta":      2.5,
	"grok-3-fast-beta":      2.5,
	"grok-3-mini-fast-beta": 0.3,
	// submodel
	"NousResearch/Hermes-4-405B-FP8":          0.8,
	"Qwen/Qwen3-235B-A22B-Thinking-2507":      0.6,
	"Qwen/Qwen3-Coder-480B-A35B-Instruct-FP8": 0.8,
	"Qwen/Qwen3-235B-A22B-Instruct-2507":      0.3,
	"zai-org/GLM-4.5-FP8":                     0.8,
	"openai/gpt-oss-120b":                     0.5,
	"deepseek-ai/DeepSeek-R1-0528":            0.8,
	"deepseek-ai/DeepSeek-R1":                 0.8,
	"deepseek-ai/DeepSeek-V3-0324":            0.8,
	"deepseek-ai/DeepSeek-V3.1":               0.8,
}

var defaultModelPrice = map[string]float64{
	"suno_music":              0.1,
	"suno_lyrics":             0.01,
	"dall-e-3":                0.04,
	"imagen-3.0-generate-002": 0.03,
	"gpt-4-gizmo-*":           0.1,
	"mj_video":                0.8,
	"mj_imagine":              0.1,
	"mj_edits":                0.1,
	"mj_variation":            0.1,
	"mj_reroll":               0.1,
	"mj_blend":                0.1,
	"mj_modal":                0.1,
	"mj_zoom":                 0.1,
	"mj_shorten":              0.1,
	"mj_high_variation":       0.1,
	"mj_low_variation":        0.1,
	"mj_pan":                  0.1,
	"mj_inpaint":              0,
	"mj_custom_zoom":          0,
	"mj_describe":             0.05,
	"mj_upscale":              0.05,
	"swap_face":               0.05,
	"mj_upload":               0.05,
}

var defaultAudioRatio = map[string]float64{
	"gpt-4o-audio-preview":         16,
	"gpt-4o-mini-audio-preview":    66.67,
	"gpt-4o-realtime-preview":      8,
	"gpt-4o-mini-realtime-preview": 16.67,
}

var defaultAudioCompletionRatio = map[string]float64{
	"gpt-4o-realtime":      2,
	"gpt-4o-mini-realtime": 2,
}

var (
	modelPriceMap      map[string]float64 = nil
	modelPriceMapMutex                    = sync.RWMutex{}
)
var (
	modelRatioMap      map[string]float64 = nil
	modelRatioMapMutex                    = sync.RWMutex{}
)

var (
	CompletionRatio      map[string]float64 = nil
	CompletionRatioMutex                    = sync.RWMutex{}
)

var defaultCompletionRatio = map[string]float64{
	"gpt-4-gizmo-*":  2,
	"gpt-4o-gizmo-*": 3,
	"gpt-4-all":      2,
	"gpt-image-1":    8,
}

// InitRatioSettings initializes all model related settings maps
func InitRatioSettings() {
	// Initialize modelPriceMap
	modelPriceMapMutex.Lock()
	modelPriceMap = defaultModelPrice
	modelPriceMapMutex.Unlock()

	// Initialize modelRatioMap
	modelRatioMapMutex.Lock()
	modelRatioMap = defaultModelRatio
	modelRatioMapMutex.Unlock()

	// Initialize CompletionRatio
	CompletionRatioMutex.Lock()
	CompletionRatio = defaultCompletionRatio
	CompletionRatioMutex.Unlock()

	// Initialize cacheRatioMap
	cacheRatioMapMutex.Lock()
	cacheRatioMap = defaultCacheRatio
	cacheRatioMapMutex.Unlock()

	// initialize imageRatioMap
	imageRatioMapMutex.Lock()
	imageRatioMap = defaultImageRatio
	imageRatioMapMutex.Unlock()

	// initialize audioRatioMap
	audioRatioMapMutex.Lock()
	audioRatioMap = defaultAudioRatio
	audioRatioMapMutex.Unlock()

	// initialize audioCompletionRatioMap
	audioCompletionRatioMapMutex.Lock()
	audioCompletionRatioMap = defaultAudioCompletionRatio
	audioCompletionRatioMapMutex.Unlock()
}

func GetModelPriceMap() map[string]float64 {
	modelPriceMapMutex.RLock()
	defer modelPriceMapMutex.RUnlock()
	return modelPriceMap
}

func ModelPrice2JSONString() string {
	modelPriceMapMutex.RLock()
	defer modelPriceMapMutex.RUnlock()

	jsonBytes, err := common.Marshal(modelPriceMap)
	if err != nil {
		common.SysError("error marshalling model price: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateModelPriceByJSONString(jsonStr string) error {
	modelPriceMapMutex.Lock()
	defer modelPriceMapMutex.Unlock()
	modelPriceMap = make(map[string]float64)
	err := json.Unmarshal([]byte(jsonStr), &modelPriceMap)
	if err == nil {
		InvalidateExposedDataCache()
	}
	return err
}

// GetModelPrice 返回模型的价格，如果模型不存在则返回-1，false
func GetModelPrice(name string, printErr bool) (float64, bool) {
	modelPriceMapMutex.RLock()
	defer modelPriceMapMutex.RUnlock()

	name = FormatMatchingModelName(name)

	price, ok := modelPriceMap[name]
	if !ok {
		if printErr {
			common.SysError("model price not found: " + name)
		}
		return -1, false
	}
	return price, true
}

func UpdateModelRatioByJSONString(jsonStr string) error {
	modelRatioMapMutex.Lock()
	defer modelRatioMapMutex.Unlock()
	modelRatioMap = make(map[string]float64)
	err := common.Unmarshal([]byte(jsonStr), &modelRatioMap)
	if err == nil {
		InvalidateExposedDataCache()
	}
	return err
}

// 处理带有思考预算的模型名称，方便统一定价
func handleThinkingBudgetModel(name, prefix, wildcard string) string {
	if strings.HasPrefix(name, prefix) && strings.Contains(name, "-thinking-") {
		return wildcard
	}
	return name
}

func GetModelRatio(name string) (float64, bool, string) {
	modelRatioMapMutex.RLock()
	defer modelRatioMapMutex.RUnlock()

	name = FormatMatchingModelName(name)

	ratio, ok := modelRatioMap[name]
	if !ok {
		return 37.5, operation_setting.SelfUseModeEnabled, name
	}
	return ratio, true, name
}

func DefaultModelRatio2JSONString() string {
	jsonBytes, err := common.Marshal(defaultModelRatio)
	if err != nil {
		common.SysError("error marshalling model ratio: " + err.Error())
	}
	return string(jsonBytes)
}

func GetDefaultModelRatioMap() map[string]float64 {
	return defaultModelRatio
}

func GetDefaultImageRatioMap() map[string]float64 {
	return defaultImageRatio
}

func GetDefaultAudioRatioMap() map[string]float64 {
	return defaultAudioRatio
}

func GetDefaultAudioCompletionRatioMap() map[string]float64 {
	return defaultAudioCompletionRatio
}

func GetCompletionRatioMap() map[string]float64 {
	CompletionRatioMutex.RLock()
	defer CompletionRatioMutex.RUnlock()
	return CompletionRatio
}

func CompletionRatio2JSONString() string {
	CompletionRatioMutex.RLock()
	defer CompletionRatioMutex.RUnlock()

	jsonBytes, err := json.Marshal(CompletionRatio)
	if err != nil {
		common.SysError("error marshalling completion ratio: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateCompletionRatioByJSONString(jsonStr string) error {
	CompletionRatioMutex.Lock()
	defer CompletionRatioMutex.Unlock()
	CompletionRatio = make(map[string]float64)
	err := common.Unmarshal([]byte(jsonStr), &CompletionRatio)
	if err == nil {
		InvalidateExposedDataCache()
	}
	return err
}

func GetCompletionRatio(name string) float64 {
	CompletionRatioMutex.RLock()
	defer CompletionRatioMutex.RUnlock()

	name = FormatMatchingModelName(name)

	if strings.Contains(name, "/") {
		if ratio, ok := CompletionRatio[name]; ok {
			return ratio
		}
	}
	hardCodedRatio, contain := getHardcodedCompletionModelRatio(name)
	if contain {
		return hardCodedRatio
	}
	if ratio, ok := CompletionRatio[name]; ok {
		return ratio
	}
	return hardCodedRatio
}

func getHardcodedCompletionModelRatio(name string) (float64, bool) {

	isReservedModel := strings.HasSuffix(name, "-all") || strings.HasSuffix(name, "-gizmo-*")
	if isReservedModel {
		return 2, false
	}

	if strings.HasPrefix(name, "gpt-") {
		if strings.HasPrefix(name, "gpt-4o") {
			if name == "gpt-4o-2024-05-13" {
				return 3, true
			}
			return 4, true
		}
		// gpt-5 匹配
		if strings.HasPrefix(name, "gpt-5") {
			return 8, true
		}
		// gpt-4.5-preview匹配
		if strings.HasPrefix(name, "gpt-4.5-preview") {
			return 2, true
		}
		if strings.HasPrefix(name, "gpt-4-turbo") || strings.HasSuffix(name, "gpt-4-1106") || strings.HasSuffix(name, "gpt-4-1105") {
			return 3, true
		}
		// 没有特殊标记的 gpt-4 模型默认倍率为 2
		return 2, false
	}
	if strings.HasPrefix(name, "o1") || strings.HasPrefix(name, "o3") {
		return 4, true
	}
	if name == "chatgpt-4o-latest" {
		return 3, true
	}

	if strings.Contains(name, "claude-3") {
		return 5, true
	} else if strings.Contains(name, "claude-sonnet-4") || strings.Contains(name, "claude-opus-4") {
		return 5, true
	} else if strings.Contains(name, "claude-instant-1") || strings.Contains(name, "claude-2") {
		return 3, true
	}

	if strings.HasPrefix(name, "gpt-3.5") {
		if name == "gpt-3.5-turbo" || strings.HasSuffix(name, "0125") {
			// https://openai.com/blog/new-embedding-models-and-api-updates
			// Updated GPT-3.5 Turbo model and lower pricing
			return 3, true
		}
		if strings.HasSuffix(name, "1106") {
			return 2, true
		}
		return 4.0 / 3.0, true
	}
	if strings.HasPrefix(name, "mistral-") {
		return 3, true
	}
	if strings.HasPrefix(name, "gemini-") {
		if strings.HasPrefix(name, "gemini-1.5") {
			return 4, true
		} else if strings.HasPrefix(name, "gemini-2.0") {
			return 4, true
		} else if strings.HasPrefix(name, "gemini-2.5-pro") { // 移除preview来增加兼容性，这里假设正式版的倍率和preview一致
			return 8, false
		} else if strings.HasPrefix(name, "gemini-2.5-flash") { // 处理不同的flash模型倍率
			if strings.HasPrefix(name, "gemini-2.5-flash-preview") {
				if strings.HasSuffix(name, "-nothinking") {
					return 4, false
				}
				return 3.5 / 0.15, false
			}
			if strings.HasPrefix(name, "gemini-2.5-flash-lite") {
				return 4, false
			}
			return 2.5 / 0.3, false
		} else if strings.HasPrefix(name, "gemini-robotics-er-1.5") {
			return 2.5 / 0.3, false
		}
		return 4, false
	}
	if strings.HasPrefix(name, "command") {
		switch name {
		case "command-r":
			return 3, true
		case "command-r-plus":
			return 5, true
		case "command-r-08-2024":
			return 4, true
		case "command-r-plus-08-2024":
			return 4, true
		default:
			return 4, false
		}
	}
	// hint 只给官方上4倍率，由于开源模型供应商自行定价，不对其进行补全倍率进行强制对齐
	if strings.HasPrefix(name, "ERNIE-Speed-") {
		return 2, true
	} else if strings.HasPrefix(name, "ERNIE-Lite-") {
		return 2, true
	} else if strings.HasPrefix(name, "ERNIE-Character") {
		return 2, true
	} else if strings.HasPrefix(name, "ERNIE-Functions") {
		return 2, true
	}
	switch name {
	case "llama2-70b-4096":
		return 0.8 / 0.64, true
	case "llama3-8b-8192":
		return 2, true
	case "llama3-70b-8192":
		return 0.79 / 0.59, true
	}
	return 1, false
}

func GetAudioRatio(name string) float64 {
	audioRatioMapMutex.RLock()
	defer audioRatioMapMutex.RUnlock()
	name = FormatMatchingModelName(name)
	if ratio, ok := audioRatioMap[name]; ok {
		return ratio
	}
	return 20
}

func GetAudioCompletionRatio(name string) float64 {
	audioCompletionRatioMapMutex.RLock()
	defer audioCompletionRatioMapMutex.RUnlock()
	name = FormatMatchingModelName(name)
	if ratio, ok := audioCompletionRatioMap[name]; ok {

		return ratio
	}
	return 2
}

func ModelRatio2JSONString() string {
	modelRatioMapMutex.RLock()
	defer modelRatioMapMutex.RUnlock()

	jsonBytes, err := common.Marshal(modelRatioMap)
	if err != nil {
		common.SysError("error marshalling model ratio: " + err.Error())
	}
	return string(jsonBytes)
}

var defaultImageRatio = map[string]float64{
	"gpt-image-1": 2,
}
var imageRatioMap map[string]float64
var imageRatioMapMutex sync.RWMutex
var (
	audioRatioMap      map[string]float64 = nil
	audioRatioMapMutex                    = sync.RWMutex{}
)
var (
	audioCompletionRatioMap      map[string]float64 = nil
	audioCompletionRatioMapMutex                    = sync.RWMutex{}
)

func ImageRatio2JSONString() string {
	imageRatioMapMutex.RLock()
	defer imageRatioMapMutex.RUnlock()
	jsonBytes, err := common.Marshal(imageRatioMap)
	if err != nil {
		common.SysError("error marshalling cache ratio: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateImageRatioByJSONString(jsonStr string) error {
	imageRatioMapMutex.Lock()
	defer imageRatioMapMutex.Unlock()
	imageRatioMap = make(map[string]float64)
	return common.Unmarshal([]byte(jsonStr), &imageRatioMap)
}

func GetImageRatio(name string) (float64, bool) {
	imageRatioMapMutex.RLock()
	defer imageRatioMapMutex.RUnlock()
	ratio, ok := imageRatioMap[name]
	if !ok {
		return 1, false // Default to 1 if not found
	}
	return ratio, true
}

func AudioRatio2JSONString() string {
	audioRatioMapMutex.RLock()
	defer audioRatioMapMutex.RUnlock()
	jsonBytes, err := common.Marshal(audioRatioMap)
	if err != nil {
		common.SysError("error marshalling audio ratio: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateAudioRatioByJSONString(jsonStr string) error {

	tmp := make(map[string]float64)
	if err := common.Unmarshal([]byte(jsonStr), &tmp); err != nil {
		return err
	}
	audioRatioMapMutex.Lock()
	audioRatioMap = tmp
	audioRatioMapMutex.Unlock()
	InvalidateExposedDataCache()
	return nil
}

func GetAudioRatioCopy() map[string]float64 {
	audioRatioMapMutex.RLock()
	defer audioRatioMapMutex.RUnlock()
	copyMap := make(map[string]float64, len(audioRatioMap))
	for k, v := range audioRatioMap {
		copyMap[k] = v
	}
	return copyMap
}

func AudioCompletionRatio2JSONString() string {
	audioCompletionRatioMapMutex.RLock()
	defer audioCompletionRatioMapMutex.RUnlock()
	jsonBytes, err := common.Marshal(audioCompletionRatioMap)
	if err != nil {
		common.SysError("error marshalling audio completion ratio: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateAudioCompletionRatioByJSONString(jsonStr string) error {
	tmp := make(map[string]float64)
	if err := common.Unmarshal([]byte(jsonStr), &tmp); err != nil {
		return err
	}
	audioCompletionRatioMapMutex.Lock()
	audioCompletionRatioMap = tmp
	audioCompletionRatioMapMutex.Unlock()
	InvalidateExposedDataCache()
	return nil
}

func GetAudioCompletionRatioCopy() map[string]float64 {
	audioCompletionRatioMapMutex.RLock()
	defer audioCompletionRatioMapMutex.RUnlock()
	copyMap := make(map[string]float64, len(audioCompletionRatioMap))
	for k, v := range audioCompletionRatioMap {
		copyMap[k] = v
	}
	return copyMap
}

func GetModelRatioCopy() map[string]float64 {
	modelRatioMapMutex.RLock()
	defer modelRatioMapMutex.RUnlock()
	copyMap := make(map[string]float64, len(modelRatioMap))
	for k, v := range modelRatioMap {
		copyMap[k] = v
	}
	return copyMap
}

func GetModelPriceCopy() map[string]float64 {
	modelPriceMapMutex.RLock()
	defer modelPriceMapMutex.RUnlock()
	copyMap := make(map[string]float64, len(modelPriceMap))
	for k, v := range modelPriceMap {
		copyMap[k] = v
	}
	return copyMap
}

func GetCompletionRatioCopy() map[string]float64 {
	CompletionRatioMutex.RLock()
	defer CompletionRatioMutex.RUnlock()
	copyMap := make(map[string]float64, len(CompletionRatio))
	for k, v := range CompletionRatio {
		copyMap[k] = v
	}
	return copyMap
}

// 转换模型名，减少渠道必须配置各种带参数模型
func FormatMatchingModelName(name string) string {

	if strings.HasPrefix(name, "gemini-2.5-flash-lite") {
		name = handleThinkingBudgetModel(name, "gemini-2.5-flash-lite", "gemini-2.5-flash-lite-thinking-*")
	} else if strings.HasPrefix(name, "gemini-2.5-flash") {
		name = handleThinkingBudgetModel(name, "gemini-2.5-flash", "gemini-2.5-flash-thinking-*")
	} else if strings.HasPrefix(name, "gemini-2.5-pro") {
		name = handleThinkingBudgetModel(name, "gemini-2.5-pro", "gemini-2.5-pro-thinking-*")
	}

	if strings.HasPrefix(name, "gpt-4-gizmo") {
		name = "gpt-4-gizmo-*"
	}
	if strings.HasPrefix(name, "gpt-4o-gizmo") {
		name = "gpt-4o-gizmo-*"
	}
	return name
}
