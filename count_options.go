package tokenizer

type CountConfig struct {
	AllowCalibratedFallback bool
	AllowHeuristicFallback  bool
	ExactOnly               bool
}

type CountOption func(*CountConfig)

func DefaultCountConfig() CountConfig {
	return CountConfig{
		AllowCalibratedFallback: true,
		AllowHeuristicFallback:  false,
		ExactOnly:               false,
	}
}

func WithExactOnly() CountOption {
	return func(cfg *CountConfig) {
		cfg.ExactOnly = true
		cfg.AllowCalibratedFallback = false
		cfg.AllowHeuristicFallback = false
	}
}

func WithCalibratedFallback() CountOption {
	return func(cfg *CountConfig) {
		cfg.AllowCalibratedFallback = true
	}
}

func WithHeuristicFallback() CountOption {
	return func(cfg *CountConfig) {
		cfg.AllowHeuristicFallback = true
	}
}
