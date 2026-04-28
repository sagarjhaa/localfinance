package ai

import "fmt"

// RecommendByRAM picks a model name + size estimate based on host RAM (GB).
// Tiers come from the Phase 6 design doc.
func RecommendByRAM(ramGB uint64) (model string, sizeGB int, reason string) {
	switch {
	case ramGB >= 32:
		return "qwen2.5:7b", 5, fmt.Sprintf("Host has %d GB RAM — comfortable headroom for a 7B model.", ramGB)
	case ramGB >= 16:
		return "gemma3:4b", 3, fmt.Sprintf("Host has %d GB RAM — 4B vision-capable model fits comfortably.", ramGB)
	default:
		return "llama3.2:3b", 2, fmt.Sprintf("Host has %d GB RAM — 3B model recommended.", ramGB)
	}
}

// HasSweetSpotModel returns true if any installed model is in the 3-14B
// parameter sweet spot. Used by the setup state machine to skip the pull
// screen if the user already has a usable model.
func HasSweetSpotModel(models []string) bool {
	for _, m := range models {
		p := extractParamCount(m)
		if p >= 3 && p <= 14 {
			return true
		}
	}
	return false
}

// SelectFastestSweetSpotModel returns the smallest installed model in the
// 3-14B parameter sweet spot — i.e. fast enough for parse latency, large
// enough for solid PDF understanding. Falls back to the smallest model
// overall if nothing in the sweet spot is installed. Empty string if the
// list is empty.
func SelectFastestSweetSpotModel(installed []string) string {
	type cand struct {
		name   string
		params float64
	}
	var sweet []cand
	var any []cand
	for _, m := range installed {
		p := extractParamCount(m)
		if p > 0 {
			any = append(any, cand{m, p})
			if p >= 3 && p <= 14 {
				sweet = append(sweet, cand{m, p})
			}
		}
	}
	pickSmallest := func(xs []cand) string {
		if len(xs) == 0 {
			return ""
		}
		best := xs[0]
		for _, x := range xs[1:] {
			if x.params < best.params {
				best = x
			}
		}
		return best.name
	}
	if s := pickSmallest(sweet); s != "" {
		return s
	}
	return pickSmallest(any)
}
