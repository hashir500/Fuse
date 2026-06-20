package spark

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/hashir500/Fuse/internal/money"
)

var (
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Red    = "\033[31m"
	Reset  = "\033[0m"
	Bold   = "\033[1m"

	output io.Writer = os.Stderr
	quiet  bool
	noArt  bool
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func SetOutput(w io.Writer) {
	if w == nil {
		output = io.Discard
		return
	}
	output = w
}

func SetQuiet(value bool) {
	quiet = value
}

func SetNoMascot(value bool) {
	noArt = value
}

func Quiet() bool {
	return quiet || os.Getenv("FUSE_QUIET") == "1"
}

func NoMascot() bool {
	return noArt || os.Getenv("FUSE_NO_MASCOT") == "1"
}

func Colorize(s string, color string) string {
	return colorize(s, color)
}

func colorize(s string, color string) string {
	if !isTerminal() {
		return stripANSI(s)
	}
	return color + s + Reset
}

func isTerminal() bool {
	stat, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func Greet() {
	if Quiet() {
		return
	}
	printArt()
	fmt.Fprintln(output, line("Spark is watching your spend..."))
}

func ProxyStarted(addr string) {
	if Quiet() {
		return
	}
	printArt()
	fmt.Fprintf(output, "Fuse proxy on %s\n", addr)
	fmt.Fprintln(output, line("Spark is watching your spend..."))
}

func SoftCapWarning(spend, cap float64, period string) {
	if Quiet() {
		return
	}
	fmt.Fprintf(output, "%s Spark SOFT CAP: %s/%s %s spend. Careful.\n",
		prefix(), money.Dollars(spend), money.Dollars(cap), strings.ToLower(period))
}

func HardCapBlocked(reqCost, current, cap float64, period string) {
	if Quiet() {
		return
	}
	fmt.Fprintln(output, colorize("SPARK: HARD CAP TRIGGERED!", Red))
	printArt()
	fmt.Fprintf(output, "%s hard cap: %s\n", title(period), money.Dollars(cap))
	fmt.Fprintf(output, "Current spend:  %s\n", money.Dollars(current))
	fmt.Fprintf(output, "This request:   %s\n", money.Dollars(reqCost))
	fmt.Fprintln(output, "Result:         BLOCKED. You're safe for now.")
}

func StatusLine(spend, cap float64, period string, pct string, bar string) string {
	base := fmt.Sprintf("%-7s %s / %s  %s %s", title(period)+":", money.Dollars(spend), money.Dollars(cap), bar, pct)
	if Quiet() {
		return base
	}
	return fmt.Sprintf("%s %s", prefix(), base)
}

func CompactArt() string {
	if Quiet() || NoMascot() {
		return ""
	}
	return colorize(strings.Trim(SparkLarge, "\n"), Green)
}

func ConfigLoaded() {
	if Quiet() {
		return
	}
	fmt.Fprintln(output, line("Spark has your budgets. Don't make me use them."))
}

func ConfigInvalid(err error) {
	if Quiet() {
		return
	}
	fmt.Fprintf(output, "%s Spark is confused. Fix this: %v\n", prefix(), err)
}

func HistoryEmpty() {
	if Quiet() {
		return
	}
	printCompact()
	fmt.Fprintln(output, line("Spark sees nothing. No requests logged."))
	fmt.Fprintln(output, "Either you're not using AI, or I'm broken.")
}

func Shutdown() {
	if Quiet() {
		return
	}
	fmt.Fprintln(output, line("Spark going dark. Spend safe."))
}

func Tip() {
	if Quiet() {
		return
	}
	fmt.Fprintln(output, line("Tip: keep max output tokens tight when testing caps."))
}

func printArt() {
	if NoMascot() {
		return
	}
	fmt.Fprintln(output, colorize(strings.Trim(SparkLarge, "\n"), Green))
}

func printCompact() {
	if NoMascot() {
		return
	}
	fmt.Fprintln(output, CompactArt())
}

func prefix() string {
	return colorize("⚡", Green)
}

func line(message string) string {
	return fmt.Sprintf("%s Spark: %s", prefix(), message)
}

func title(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}
