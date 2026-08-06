package tokenizer

import "runtime"

type Options struct {
	MaxWorkers            int
	ParallelByteThreshold int
	BatchByteThreshold    int
}

func DefaultOptions() Options {
	return Options{
		MaxWorkers:            runtime.GOMAXPROCS(0),
		ParallelByteThreshold: 16384,
		BatchByteThreshold:    16384,
	}
}

func sanitizeOptions(opts Options) Options {
	def := DefaultOptions()
	if opts.MaxWorkers <= 0 {
		opts.MaxWorkers = def.MaxWorkers
	}
	if opts.ParallelByteThreshold <= 0 {
		opts.ParallelByteThreshold = def.ParallelByteThreshold
	}
	if opts.BatchByteThreshold <= 0 {
		opts.BatchByteThreshold = def.BatchByteThreshold
	}
	return opts
}
