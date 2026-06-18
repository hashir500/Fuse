package cost

import "github.com/hashir500/Fuse/internal/config"

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

func Estimate(usage Usage, costs config.ModelCosts) float64 {
	return (float64(usage.PromptTokens)/1000.0)*costs.InputCostPer1K +
		(float64(usage.CompletionTokens)/1000.0)*costs.OutputCostPer1K
}
