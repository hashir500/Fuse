package budget

import (
	"fmt"
	"strings"

	"github.com/hashir500/Fuse/internal/config"
	"github.com/hashir500/Fuse/internal/money"
	"github.com/hashir500/Fuse/internal/store"
)

type Decision struct {
	Allowed      bool
	SoftWarnings []Warning
	HardHit      *HardHit
}

type Warning struct {
	Period string
	Soft   float64
	Spend  float64
}

type HardHit struct {
	Period       string
	CapAmount    float64
	CurrentSpend float64
	RequestCost  float64
}

func Check(cfg config.BudgetConfig, spend store.PeriodSpend, requestCost float64, onHardCap string) Decision {
	decision := Decision{Allowed: true}
	checks := []struct {
		name   string
		budget config.Budget
		spend  float64
	}{
		{"daily", cfg.Daily, spend.Daily},
		{"weekly", cfg.Weekly, spend.Weekly},
		{"monthly", cfg.Monthly, spend.Monthly},
	}

	for _, item := range checks {
		projected := item.spend + requestCost
		if item.budget.Soft > 0 && item.spend < item.budget.Soft && projected >= item.budget.Soft {
			decision.SoftWarnings = append(decision.SoftWarnings, Warning{
				Period: item.name,
				Soft:   item.budget.Soft,
				Spend:  projected,
			})
		}
		if item.budget.Hard > 0 && projected > item.budget.Hard && onHardCap == "block" {
			decision.Allowed = false
			decision.HardHit = &HardHit{
				Period:       item.name,
				CapAmount:    item.budget.Hard,
				CurrentSpend: item.spend,
				RequestCost:  requestCost,
			}
			return decision
		}
	}
	return decision
}

func (h HardHit) Message() string {
	return fmt.Sprintf("%s hard cap of %s exceeded. Current: %s, Request estimated max cost: %s.",
		title(h.Period), money.Dollars(h.CapAmount), money.Dollars(h.CurrentSpend), money.Dollars(h.RequestCost))
}

func title(value string) string {
	if len(value) == 0 {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
