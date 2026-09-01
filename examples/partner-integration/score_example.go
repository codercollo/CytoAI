package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/codercollo/cytoai/pkg/cytoaiclient"
)

func main() {
	base := envOr("CYTOAI_API_BASE", "http://localhost:8080")
	key := os.Getenv("CYTOAI_API_KEY")
	riderID := os.Getenv("CYTOAI_RIDER_ID")
	batteryID := os.Getenv("CYTOAI_BATTERY_ID")
	if key == "" || riderID == "" || batteryID == "" {
		fmt.Fprintln(os.Stderr, "set CYTOAI_API_KEY, CYTOAI_RIDER_ID, and CYTOAI_BATTERY_ID")
		os.Exit(1)
	}

	client := cytoaiclient.New(base, key)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	score, err := client.Score(ctx, riderID, batteryID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "score failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("CytoScore=%.2f  BHI=%.2f  RRI=%.2f\n", score.CytoScore, score.BatteryHealthIndex, score.RepaymentRiskIndex)
	fmt.Printf("Anomaly flags: %d\n", len(score.AnomalyFlags))
	for _, f := range score.AnomalyFlags {
		fmt.Printf("  - [%s] %s: %s (entity=%s/%s)\n", f.Severity, f.RuleOrModel, f.Reason, f.EntityType, f.EntityID)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
