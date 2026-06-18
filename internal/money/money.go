package money

import "fmt"

func Dollars(value float64) string {
	if value > 0 && value < 0.01 {
		return fmt.Sprintf("$%.5f", value)
	}
	return fmt.Sprintf("$%.2f", value)
}
