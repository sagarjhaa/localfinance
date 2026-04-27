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
