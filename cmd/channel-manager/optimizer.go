package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Maximum reasonable fee per million (PPM)
const MaxReasonablePPM int64 = 1000
// Minimum viable fee per million (PPM)
const MinViablePPM int64 = 10

// ChannelAnalysis contains analysis data for fee optimization
type ChannelAnalysis struct {
	Channel          Channel
	LocalBalance     int64
	RemoteBalance    int64
	Capacity         int64
	LocalRatio       float64
	Category         string
	CurrentBaseFee   int64
	CurrentPPM       int64
	RecommendedPPM   int64
	RecommendedBase  int64
	ForwardingCount  int
	EarningsLast30d  int64
	DaysSinceForward int
	Reasoning        string
	Priority         string // high, medium, low
}

// ChannelCategory constants
const (
	CategoryHighCapOutbound = "high-cap-outbound" // >500K local, good for sending large payments
	CategoryHighCapInbound  = "high-cap-inbound"  // >500K remote, liquidity sink
	CategoryBalanced        = "balanced"          // 30-70% local ratio
	CategoryLowLiquidity    = "low-liquidity"     // <500K total capacity
)

// handleSuggestFees analyzes channels and suggests optimal fees without applying changes
func handleSuggestFees() {
	fmt.Println("🔍 Analyzing channels for fee optimization opportunities...")

	analyses, err := analyzeChannelsForFeeOptimization()
	if err != nil {
		log.Fatal("Failed to analyze channels:", err)
	}

	if len(analyses) == 0 {
		fmt.Println("No active channels found for analysis")
		return
	}

	// Sort by priority (high first) then by potential revenue impact
	sort.Slice(analyses, func(i, j int) bool {
		if analyses[i].Priority != analyses[j].Priority {
			priorityOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
			return priorityOrder[analyses[i].Priority] > priorityOrder[analyses[j].Priority]
		}
		return analyses[i].Capacity > analyses[j].Capacity
	})

	displayFeeOptimizationSuggestions(analyses)
}

// handleFeeOptimizer automatically applies optimized fees based on analysis
func handleFeeOptimizer() {
	var dryRun bool

	// Check for --dry-run flag
	for _, arg := range os.Args[2:] {
		if arg == "--dry-run" {
			dryRun = true
			break
		}
	}

	if dryRun {
		fmt.Println("🧪 Running fee optimizer in dry-run mode (no changes will be applied)...")
	} else {
		fmt.Println("🤖 Running automatic fee optimizer...")
	}

	analyses, err := analyzeChannelsForFeeOptimization()
	if err != nil {
		log.Fatal("Failed to analyze channels:", err)
	}

	if len(analyses) == 0 {
		fmt.Println("No active channels found for optimization")
		return
	}

	// Filter for high and medium priority changes
	var toUpdate []ChannelAnalysis
	for _, analysis := range analyses {
		if analysis.Priority == "high" || analysis.Priority == "medium" {
			// Only update if there's a meaningful change
			if math.Abs(float64(analysis.RecommendedPPM-analysis.CurrentPPM)) >= 10 {
				toUpdate = append(toUpdate, analysis)
			}
		}
	}

	if len(toUpdate) == 0 {
		fmt.Println("✅ All channels already have optimal fees")
		return
	}

	fmt.Printf("📊 Found %d channels that would benefit from fee optimization:\n\n", len(toUpdate))

	successCount := 0
	for _, analysis := range toUpdate {
		alias := getNodeAlias(analysis.Channel.RemotePubkey)

		if dryRun {
			fmt.Printf("🔧 Would update %s: %d → %d ppm (%s priority)\n",
				alias, analysis.CurrentPPM, analysis.RecommendedPPM, analysis.Priority)
			successCount++
		} else {
			err := setChannelFees(
				analysis.Channel.ChanID,
				strconv.FormatInt(analysis.RecommendedBase, 10),
				strconv.FormatInt(analysis.RecommendedPPM, 10),
			)

			if err != nil {
				fmt.Printf("❌ Failed to update %s: %v\n", alias, err)
			} else {
				fmt.Printf("✅ Updated %s: %d → %d ppm (%s priority)\n",
					alias, analysis.CurrentPPM, analysis.RecommendedPPM, analysis.Priority)
				successCount++
			}
		}
	}

	if dryRun {
		fmt.Printf("\n🧪 Dry run complete: %d channels would be updated\n", successCount)
		fmt.Println("💡 Run without --dry-run to apply changes")
	} else {
		fmt.Printf("\n📊 Optimization complete: %d/%d channels updated successfully\n", successCount, len(toUpdate))
	}
}

// analyzeChannelsForFeeOptimization performs comprehensive channel analysis for fee optimization
func analyzeChannelsForFeeOptimization() ([]ChannelAnalysis, error) {
	channels, err := getChannels()
	if err != nil {
		return nil, err
	}

	feeReport, err := getFeeReport()
	if err != nil {
		return nil, err
	}

	// Get forwarding history for the last 30 days
	endTime := time.Now().Unix()
	startTime := endTime - (30 * 24 * 60 * 60) // 30 days ago

	forwardingHistory, err := getForwardingHistory(
		strconv.FormatInt(startTime, 10),
		strconv.FormatInt(endTime, 10),
	)
	if err != nil {
		// Continue without forwarding history if unavailable
		forwardingHistory = &ForwardingHistory{ForwardingEvents: []ForwardingEvent{}}
	}

	// Create fee lookup map
	feeMap := make(map[string]ChannelFeeReport)
	for _, fee := range feeReport.ChannelFees {
		feeMap[fee.ChanID] = fee
	}

	// Create forwarding lookup maps
	outgoingForwards := make(map[string][]ForwardingEvent)
	incomingForwards := make(map[string][]ForwardingEvent)

	for _, event := range forwardingHistory.ForwardingEvents {
		outgoingForwards[event.ChanIdOut] = append(outgoingForwards[event.ChanIdOut], event)
		incomingForwards[event.ChanIdIn] = append(incomingForwards[event.ChanIdIn], event)
	}

	var analyses []ChannelAnalysis
	for _, channel := range channels {
		if !channel.Active {
			continue // Skip inactive channels
		}

		analysis := analyzeChannel(channel, feeMap, outgoingForwards, incomingForwards)
		analyses = append(analyses, analysis)
	}

	return analyses, nil
}

// analyzeChannel performs detailed analysis of a single channel
func analyzeChannel(channel Channel, feeMap map[string]ChannelFeeReport, outgoingForwards, incomingForwards map[string][]ForwardingEvent) ChannelAnalysis {
	localBalance, _ := strconv.ParseInt(channel.LocalBalance, 10, 64)
	remoteBalance, _ := strconv.ParseInt(channel.RemoteBalance, 10, 64)
	capacity, _ := strconv.ParseInt(channel.Capacity, 10, 64)

	localRatio := float64(localBalance) / float64(capacity)

	// Get current fees
	currentBase := int64(1000) // default
	currentPPM := int64(1)     // default

	if feeInfo, exists := feeMap[channel.ChanID]; exists {
		if baseFee, err := strconv.ParseInt(feeInfo.BaseFeeMsat, 10, 64); err == nil {
			currentBase = baseFee
		}
		if ppm, err := strconv.ParseInt(feeInfo.FeePerMil, 10, 64); err == nil {
			currentPPM = ppm
		}
	}

	// Calculate forwarding metrics
	forwardingCount := len(outgoingForwards[channel.ChanID])
	var earningsLast30d int64
	var lastForwardTime time.Time

	for _, forward := range outgoingForwards[channel.ChanID] {
		fee, _ := strconv.ParseInt(forward.FeeMsat, 10, 64)
		earningsLast30d += fee / 1000 // Convert msat to sat

		timestamp, _ := strconv.ParseInt(forward.Timestamp, 10, 64)
		forwardTime := time.Unix(timestamp, 0)
		if forwardTime.After(lastForwardTime) {
			lastForwardTime = forwardTime
		}
	}

	daysSinceForward := int(time.Since(lastForwardTime).Hours() / 24)
	if forwardingCount == 0 {
		daysSinceForward = 999 // Large number for channels that never forwarded
	}

	// Categorize channel
	category := categorizeChannel(localBalance, remoteBalance, capacity, localRatio)

	// Calculate recommended fees
	recommendedPPM, recommendedBase, reasoning, priority := calculateOptimalFees(
		category, localBalance, remoteBalance, capacity, localRatio,
		forwardingCount, earningsLast30d, daysSinceForward,
	)

	return ChannelAnalysis{
		Channel:          channel,
		LocalBalance:     localBalance,
		RemoteBalance:    remoteBalance,
		Capacity:         capacity,
		LocalRatio:       localRatio,
		Category:         category,
		CurrentBaseFee:   currentBase,
		CurrentPPM:       currentPPM,
		RecommendedPPM:   recommendedPPM,
		RecommendedBase:  recommendedBase,
		ForwardingCount:  forwardingCount,
		EarningsLast30d:  earningsLast30d,
		DaysSinceForward: daysSinceForward,
		Reasoning:        reasoning,
		Priority:         priority,
	}
}

// categorizeChannel determines the channel category for fee optimization
func categorizeChannel(localBalance, remoteBalance, capacity int64, localRatio float64) string {
	if capacity < 500000 {
		return CategoryLowLiquidity
	}

	if localBalance > 500000 && localRatio > 0.6 {
		return CategoryHighCapOutbound
	}

	if remoteBalance > 500000 && localRatio < 0.4 {
		return CategoryHighCapInbound
	}

	if localRatio >= 0.3 && localRatio <= 0.7 {
		return CategoryBalanced
	}

	// Default fallback
	if localBalance > remoteBalance {
		return CategoryHighCapOutbound
	}
	return CategoryHighCapInbound
}

// calculateOptimalFees calculates the optimal fee structure for a channel
func calculateOptimalFees(category string, localBalance, remoteBalance, capacity int64, localRatio float64,
	forwardingCount int, earningsLast30d int64, daysSinceForward int) (ppm, baseFee int64, reasoning, priority string) {

	baseFee = 1000 // Standard base fee

	// Base PPM by category
	var basePPM int64
	switch category {
	case CategoryHighCapOutbound:
		basePPM = 25 // Competitive for large outbound payments
		reasoning = "High-capacity outbound channel - competitive fees to attract large payments"
	case CategoryBalanced:
		basePPM = 100 // Moderate fees for balanced routing
		reasoning = "Well-balanced channel - moderate fees for optimal routing"
	case CategoryHighCapInbound:
		basePPM = 300 // Higher fees to prevent excessive draining
		reasoning = "High-capacity inbound channel - higher fees to manage liquidity drain"
	case CategoryLowLiquidity:
		basePPM = 500 // Premium fees for limited liquidity
		reasoning = "Low-liquidity channel - premium fees due to limited capacity"
	default:
		basePPM = 100
		reasoning = "Standard channel - moderate fees"
	}

	// Performance adjustments
	priority = "medium"

	// Recent forwarding activity (last 7 days)
	if daysSinceForward <= 7 && forwardingCount > 0 {
		basePPM = int64(float64(basePPM) * 0.8) // 20% discount for active channels
		reasoning += " (reduced for recent activity)"
		priority = "high"
	} else if daysSinceForward > 30 {
		basePPM = int64(float64(basePPM) * 1.5) // 50% increase for inactive channels
		reasoning += " (increased for inactivity)"
		priority = "low"
	}

	// High-earning channels get moderate increase
	avgDailyEarnings := earningsLast30d / 30
	if avgDailyEarnings > 10 { // More than 10 sats per day average
		basePPM = int64(float64(basePPM) * 1.25) // 25% increase
		reasoning += " (premium for high earnings)"
		priority = "high"
	}

	// Ensure minimum viable fees
	if basePPM < MinViablePPM {
		basePPM = MinViablePPM
	}

	// Cap maximum fees to reasonable levels
	if basePPM > MaxReasonablePPM {
		basePPM = MaxReasonablePPM
	}

	return basePPM, baseFee, reasoning, priority
}

// displayFeeOptimizationSuggestions shows the analysis results in a formatted table
func displayFeeOptimizationSuggestions(analyses []ChannelAnalysis) {
	fmt.Println("\n💡 Fee Optimization Suggestions:")
	fmt.Println(strings.Repeat("─", 120))

	highPriorityCount := 0
	mediumPriorityCount := 0

	for _, analysis := range analyses {
		alias := getNodeAlias(analysis.Channel.RemotePubkey)
		if len(alias) > 25 {
			alias = alias[:22] + "..."
		}

		// Priority indicator
		priorityIcon := "🟢"
		switch analysis.Priority {
		case "high":
			priorityIcon = "🔴"
			highPriorityCount++
		case "medium":
			priorityIcon = "🟡"
			mediumPriorityCount++
		case "low":
			priorityIcon = "🟢"
		}

		// Fee change indicator
		feeChangeIcon := "→"
		feeChange := analysis.RecommendedPPM - analysis.CurrentPPM
		if feeChange > 0 {
			feeChangeIcon = "↗"
		} else if feeChange < 0 {
			feeChangeIcon = "↘"
		}

		fmt.Printf("%s %-25s │ %s %3d %s %3d ppm │ %s │ %7s │ %2dd │ %s\n",
			priorityIcon,
			alias,
			feeChangeIcon,
			analysis.CurrentPPM,
			feeChangeIcon,
			analysis.RecommendedPPM,
			formatSats(analysis.Capacity),
			fmt.Sprintf("%.1f%%", analysis.LocalRatio*100),
			analysis.DaysSinceForward,
			analysis.Category,
		)

		// Show reasoning for high priority items
		if analysis.Priority == "high" {
			fmt.Printf("   └─ %s\n", analysis.Reasoning)
		}
	}

	fmt.Println(strings.Repeat("─", 120))
	fmt.Printf("📊 Summary: %d high priority, %d medium priority changes suggested\n",
		highPriorityCount, mediumPriorityCount)

	if highPriorityCount > 0 || mediumPriorityCount > 0 {
		fmt.Println("\n💡 Commands:")
		fmt.Println("   ./bin/channel-manager fee-optimizer --dry-run    # Preview changes")
		fmt.Println("   ./bin/channel-manager fee-optimizer             # Apply optimizations")
	}

	fmt.Println("\n🔑 Legend:")
	fmt.Println("   🔴 High priority  🟡 Medium priority  🟢 Low priority")
	fmt.Println("   ↗ Increase fees  ↘ Decrease fees  → No change")
	fmt.Printf("   Categories: %s, %s, %s, %s\n",
		CategoryHighCapOutbound, CategoryBalanced, CategoryHighCapInbound, CategoryLowLiquidity)
}
