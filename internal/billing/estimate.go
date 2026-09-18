package billing

import (
	"encoding/binary"
	"math"
	"unicode/utf8"
)

// EstimateTokens is intentionally deterministic across providers. Exact token
// usage replaces it whenever a provider returns usage; embedding and rerank
// APIs commonly omit usage, so their events retain estimated=true.
func EstimateTokens(values ...string) int {
	total := 0
	for _, value := range values {
		if value == "" {
			continue
		}
		runes := utf8.RuneCountInString(value)
		asciiApprox := (len(value) + 3) / 4
		if runes > asciiApprox {
			total += runes
		} else {
			total += asciiApprox
		}
	}
	return total
}

// EstimateAudioSeconds reads canonical PCM WAV metadata when available and
// otherwise uses a conservative 16 kHz mono 16-bit estimate.
func EstimateAudioSeconds(audio []byte) float64 {
	if len(audio) >= 44 && string(audio[:4]) == "RIFF" && string(audio[8:12]) == "WAVE" {
		byteRate := binary.LittleEndian.Uint32(audio[28:32])
		dataSize := binary.LittleEndian.Uint32(audio[40:44])
		if byteRate > 0 && dataSize > 0 {
			return math.Max(1, math.Ceil(float64(dataSize)/float64(byteRate)))
		}
	}
	return math.Max(1, math.Ceil(float64(len(audio))/(16000*2)))
}
