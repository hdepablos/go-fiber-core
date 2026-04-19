package queue

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var pollingDigits = regexp.MustCompile(`\b\d+\b`)

type PollingErrorEvent struct {
	Fingerprint string
	Count       int
	Threshold   int
	Triggered   bool
}

type PollingErrorGuard struct {
	threshold int
	lastKey   string
	count     int
}

func NewPollingErrorGuard(threshold int) *PollingErrorGuard {
	if threshold <= 0 {
		threshold = defaultPollingErrorThreshold()
	}
	return &PollingErrorGuard{threshold: threshold}
}

func (g *PollingErrorGuard) Record(err error) PollingErrorEvent {
	if err == nil {
		g.Reset()
		return PollingErrorEvent{}
	}

	key := normalizePollingError(err.Error())
	if key == g.lastKey {
		g.count++
	} else {
		g.lastKey = key
		g.count = 1
	}

	return PollingErrorEvent{
		Fingerprint: key,
		Count:       g.count,
		Threshold:   g.threshold,
		Triggered:   g.threshold > 0 && g.count >= g.threshold,
	}
}

func (g *PollingErrorGuard) Reset() {
	g.lastKey = ""
	g.count = 0
}

func DefaultPollingErrorCooldown() time.Duration {
	seconds := 60
	if raw := strings.TrimSpace(os.Getenv("SQS_CONSUMER_RECEIVE_ERROR_COOLDOWN_SECONDS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			seconds = parsed
		}
	}
	return time.Duration(seconds) * time.Second
}

func defaultPollingErrorThreshold() int {
	threshold := 10
	if raw := strings.TrimSpace(os.Getenv("SQS_CONSUMER_RECEIVE_ERROR_THRESHOLD")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			threshold = parsed
		}
	}
	return threshold
}

func normalizePollingError(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = pollingDigits.ReplaceAllString(raw, ":n")
	raw = strings.Join(strings.Fields(raw), " ")
	return raw
}
