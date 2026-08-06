package tokenizer

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderQwen      Provider = "qwen"
	ProviderKimi      Provider = "kimi"
	ProviderDeepSeek  Provider = "deepseek"
	ProviderGrok      Provider = "grok"
	ProviderMistral   Provider = "mistral"
	ProviderGemini    Provider = "gemini"
)
