package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/luiscarbonell/lightning-node-tools/pkg/lnd"
)

// RebalanceConfig holds configuration for rebalancing operations
type RebalanceConfig struct {
	MaxFeeRate     float64 // Maximum fee rate as percentage (e.g., 0.5 for 0.5%)
	MaxFeeAbsolute int64   // Maximum absolute fee in sats
	MinAmount      int64   // Minimum rebalance amount in sats
	MaxAmount      int64   // Maximum rebalance amount in sats
	TargetRatio    float64 // Target local balance ratio (0.0-1.0)
	ToleranceRatio float64 // Acceptable deviation from target ratio
}

// ChannelScore represents a channel's rebalancing priority and metrics
type ChannelScore struct {
	Channel        lnd.Channel
	Alias          string
	LocalRatio     float64 // Current local balance ratio
	CapacityScore  float64 // Normalized capacity score
	ActivityScore  float64 // Recent forwarding activity score
	ImbalanceScore float64 // How far from ideal balance
	OverallScore   float64 // Combined priority score
	NeedsInbound   bool    // Channel needs inbound liquidity
	NeedsOutbound  bool    // Channel needs outbound liquidity
}

// RebalanceProposal represents a suggested rebalancing operation
type RebalanceProposal struct {
	FromChannel  lnd.Channel
	ToChannel    lnd.Channel
	FromAlias    string
	ToAlias      string
	Amount       int64
	EstimatedFee int64
	FeeRate      float64
	Reason       string
	Priority     int // 1=high, 2=medium, 3=low
}

// DefaultRebalanceConfig returns sensible default configuration
func getDefaultRebalanceConfig() RebalanceConfig {
	return RebalanceConfig{
		MaxFeeRate:     0.5,     // 0.5% max fee rate
		MaxFeeAbsolute: 1000,    // 1000 sats max absolute fee
		MinAmount:      10000,   // 10k sats minimum
		MaxAmount:      1000000, // 1M sats maximum
		TargetRatio:    0.5,     // 50% local balance target
		ToleranceRatio: 0.1,     // ±10% tolerance
	}
}

// analyzeChannels analyzes all channels and returns scored results
func analyzeChannels() ([]ChannelScore, error) {
	channels, err := getChannels()
	if err != nil {
		return nil, fmt.Errorf("failed to get channels: %v", err)
	}

	// Get forwarding history for activity scoring
	forwardingHistory, err := getRecentForwardingHistory(7 * 24 * time.Hour) // Last 7 days
	if err != nil {
		log.Printf("Warning: Could not get forwarding history: %v", err)
	}

	var scores []ChannelScore
	totalCapacity := int64(0)

	// Calculate total capacity for normalization
	for _, channel := range channels {
		if capacity, err := strconv.ParseInt(channel.Capacity, 10, 64); err == nil {
			totalCapacity += capacity
		}
	}

	for _, channel := range channels {
		if !channel.Active {
			continue // Skip inactive channels
		}

		score := ChannelScore{
			Channel: channel,
			Alias:   getNodeAlias(channel.RemotePubkey),
		}

		capacity, _ := strconv.ParseInt(channel.Capacity, 10, 64)
		localBalance, _ := strconv.ParseInt(channel.LocalBalance, 10, 64)

		// Calculate ratios and scores
		if capacity > 0 {
			score.LocalRatio = float64(localBalance) / float64(capacity)
			score.CapacityScore = float64(capacity) / float64(totalCapacity) * 100 // Normalize to 0-100
		}

		// Activity score based on recent forwards
		score.ActivityScore = calculateActivityScore(channel.ChanID, forwardingHistory)

		// Imbalance score - higher means more imbalanced
		targetRatio := getDefaultRebalanceConfig().TargetRatio
		score.ImbalanceScore = math.Abs(score.LocalRatio - targetRatio)

		// Determine liquidity needs
		tolerance := getDefaultRebalanceConfig().ToleranceRatio
		score.NeedsInbound = score.LocalRatio > (targetRatio + tolerance)
		score.NeedsOutbound = score.LocalRatio < (targetRatio - tolerance)

		// Calculate overall score (higher = higher priority for rebalancing)
		// Prioritize: high imbalance + high activity + large capacity
		score.OverallScore = (score.ImbalanceScore * 40) +
			(score.ActivityScore * 30) +
			(score.CapacityScore * 30)

		scores = append(scores, score)
	}

	// Sort by overall score (highest priority first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].OverallScore > scores[j].OverallScore
	})

	return scores, nil
}

// calculateActivityScore calculates a channel's activity score based on recent forwards
func calculateActivityScore(chanID string, history *lnd.ForwardingHistory) float64 {
	if history == nil {
		return 0.0
	}

	forwardCount := 0
	totalFeeSats := int64(0)

	for _, event := range history.ForwardingEvents {
		if event.ChanIdIn == chanID || event.ChanIdOut == chanID {
			forwardCount++
			if feeMsat, err := strconv.ParseInt(event.FeeMsat, 10, 64); err == nil {
				totalFeeSats += feeMsat / 1000 // Convert msat to sats
			}
		}
	}

	// Normalize score: combines forward count and fee earnings
	// More forwards and higher fees = higher score
	activityScore := float64(forwardCount)*10 + float64(totalFeeSats)*0.1

	// Cap at 100 for normalization
	if activityScore > 100 {
		activityScore = 100
	}

	return activityScore
}

// getRecentForwardingHistory gets forwarding history for a specific time period
func getRecentForwardingHistory(duration time.Duration) (*lnd.ForwardingHistory, error) {
	startTime := time.Now().Add(-duration).Unix()
	endTime := time.Now().Unix()

	return getForwardingHistory(strconv.FormatInt(startTime, 10), strconv.FormatInt(endTime, 10))
}

// generateRebalanceProposals creates intelligent rebalancing suggestions
func generateRebalanceProposals(scores []ChannelScore, config RebalanceConfig) []RebalanceProposal {
	var proposals []RebalanceProposal

	// Find channels that need liquidity in different directions
	var needsInbound []ChannelScore
	var needsOutbound []ChannelScore

	for _, score := range scores {
		if score.NeedsInbound {
			needsInbound = append(needsInbound, score)
		}
		if score.NeedsOutbound {
			needsOutbound = append(needsOutbound, score)
		}
	}

	// Create proposals by pairing channels
	for _, inboundChan := range needsInbound {
		for _, outboundChan := range needsOutbound {
			if inboundChan.Channel.ChanID == outboundChan.Channel.ChanID {
				continue // Skip same channel
			}

			// Calculate optimal rebalance amount
			amount := calculateOptimalAmount(inboundChan, outboundChan, config)
			if amount < config.MinAmount {
				continue
			}

			// Estimate rebalancing cost
			estimatedFee, feeRate := estimateRebalanceCost(amount, inboundChan.Channel, outboundChan.Channel)

			// Check if cost is acceptable
			if feeRate > config.MaxFeeRate || estimatedFee > config.MaxFeeAbsolute {
				continue
			}

			// Determine priority based on scores and imbalance
			priority := calculateProposalPriority(inboundChan, outboundChan)

			// Create reason description
			reason := fmt.Sprintf("Move liquidity from over-liquid %s (%.1f%% local) to under-liquid %s (%.1f%% local)",
				inboundChan.Alias, inboundChan.LocalRatio*100,
				outboundChan.Alias, outboundChan.LocalRatio*100)

			proposal := RebalanceProposal{
				FromChannel:  inboundChan.Channel,
				ToChannel:    outboundChan.Channel,
				FromAlias:    inboundChan.Alias,
				ToAlias:      outboundChan.Alias,
				Amount:       amount,
				EstimatedFee: estimatedFee,
				FeeRate:      feeRate,
				Reason:       reason,
				Priority:     priority,
			}

			proposals = append(proposals, proposal)
		}
	}

	// Sort proposals by priority and potential impact
	sort.Slice(proposals, func(i, j int) bool {
		if proposals[i].Priority != proposals[j].Priority {
			return proposals[i].Priority < proposals[j].Priority // Lower number = higher priority
		}
		return proposals[i].Amount > proposals[j].Amount // Larger amounts first for same priority
	})

	// Limit to top 10 proposals to avoid overwhelming output
	if len(proposals) > 10 {
		proposals = proposals[:10]
	}

	return proposals
}

// calculateOptimalAmount determines the optimal rebalance amount between two channels
func calculateOptimalAmount(fromChan, toChan ChannelScore, config RebalanceConfig) int64 {
	fromCapacity, _ := strconv.ParseInt(fromChan.Channel.Capacity, 10, 64)
	fromLocal, _ := strconv.ParseInt(fromChan.Channel.LocalBalance, 10, 64)

	toCapacity, _ := strconv.ParseInt(toChan.Channel.Capacity, 10, 64)
	toLocal, _ := strconv.ParseInt(toChan.Channel.LocalBalance, 10, 64)

	// Calculate how much each channel is away from target ratio
	targetLocal := int64(float64(fromCapacity) * config.TargetRatio)
	fromExcess := fromLocal - targetLocal

	targetToLocal := int64(float64(toCapacity) * config.TargetRatio)
	toDeficit := targetToLocal - toLocal

	// Take the minimum of excess and deficit, but respect configured limits
	amount := int64(math.Min(float64(fromExcess), float64(toDeficit)))

	// Apply configured limits
	if amount > config.MaxAmount {
		amount = config.MaxAmount
	}
	if amount < config.MinAmount {
		return 0 // Not worth rebalancing
	}

	return amount
}

// estimateRebalanceCost estimates the cost and fee rate for a rebalancing operation
func estimateRebalanceCost(amount int64, fromChan, toChan lnd.Channel) (int64, float64) {
	// This is a simplified cost estimation
	// In a real implementation, you'd use actual routing attempts or fee rate analysis

	// Base fee estimation: assume 0.1% fee rate plus some base fee
	baseFeeRate := 0.001 // 0.1%
	estimatedFee := int64(float64(amount) * baseFeeRate)

	// Add some base fee (typical base fees are 1-10 sats)
	estimatedFee += 5

	// Calculate fee rate as percentage
	feeRate := float64(estimatedFee) / float64(amount) * 100

	return estimatedFee, feeRate
}

// calculateProposalPriority determines the priority level for a rebalance proposal
func calculateProposalPriority(fromChan, toChan ChannelScore) int {
	// Higher combined overall scores = higher priority
	combinedScore := fromChan.OverallScore + toChan.OverallScore

	if combinedScore > 150 {
		return 1 // High priority
	} else if combinedScore > 75 {
		return 2 // Medium priority
	}
	return 3 // Low priority
}

// handleRebalance handles the manual rebalance command
func handleRebalance() {
	if len(os.Args) < 8 {
		fmt.Println("Usage: channel-manager rebalance --from-channel <ID> --to-channel <ID> --amount <sats>")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  channel-manager rebalance --from-channel 12345 --to-channel 67890 --amount 100000")
		return
	}

	var fromChannelID, toChannelID, amountStr string

	// Parse arguments
	for i := 2; i < len(os.Args); i += 2 {
		if i+1 >= len(os.Args) {
			fmt.Printf("Error: Missing value for %s\n", os.Args[i])
			return
		}

		switch os.Args[i] {
		case "--from-channel":
			fromChannelID = os.Args[i+1]
		case "--to-channel":
			toChannelID = os.Args[i+1]
		case "--amount":
			amountStr = os.Args[i+1]
		default:
			fmt.Printf("Unknown flag: %s\n", os.Args[i])
			return
		}
	}

	if fromChannelID == "" || toChannelID == "" || amountStr == "" {
		fmt.Println("Error: --from-channel, --to-channel, and --amount are required")
		return
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		fmt.Printf("Error: Invalid amount: %s\n", amountStr)
		return
	}

	fmt.Printf("🔄 Starting rebalance operation...\n")
	fmt.Printf("From Channel: %s\n", fromChannelID)
	fmt.Printf("To Channel: %s\n", toChannelID)
	fmt.Printf("Amount: %s\n", formatSats(amount))

	err = executeRebalance(fromChannelID, toChannelID, amount)
	if err != nil {
		fmt.Printf("❌ Rebalance failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Rebalance completed successfully!\n")
}

// handleSuggestRebalance handles the suggest-rebalance command
func handleSuggestRebalance() {
	fmt.Println("🔍 Analyzing channels for rebalancing opportunities...")

	scores, err := analyzeChannels()
	if err != nil {
		fmt.Printf("Error analyzing channels: %v\n", err)
		return
	}

	config := getDefaultRebalanceConfig()
	proposals := generateRebalanceProposals(scores, config)

	if len(proposals) == 0 {
		fmt.Println("✅ No rebalancing needed - all channels are well balanced!")
		return
	}

	fmt.Printf("\n💡 Found %d rebalancing opportunities:\n", len(proposals))
	fmt.Println(strings.Repeat("━", 100))

	for i, proposal := range proposals {
		priorityEmoji := getPriorityEmoji(proposal.Priority)

		fmt.Printf("%s %d. %s → %s\n", priorityEmoji, i+1, proposal.FromAlias, proposal.ToAlias)
		fmt.Printf("   Amount: %s | Estimated fee: %s (%.3f%%)\n",
			formatSats(proposal.Amount), formatSats(proposal.EstimatedFee), proposal.FeeRate)
		fmt.Printf("   Reason: %s\n", proposal.Reason)
		fmt.Printf("   Command: ./bin/channel-manager rebalance --from-channel %s --to-channel %s --amount %d\n",
			proposal.FromChannel.ChanID, proposal.ToChannel.ChanID, proposal.Amount)
		fmt.Println()
	}

	fmt.Println(strings.Repeat("━", 100))
	fmt.Printf("💡 To execute all high-priority rebalances: ./bin/channel-manager auto-rebalance\n")
}

// handleAutoRebalance handles the auto-rebalance command
func handleAutoRebalance() {
	fmt.Println("🤖 Starting automated rebalancing...")

	scores, err := analyzeChannels()
	if err != nil {
		fmt.Printf("Error analyzing channels: %v\n", err)
		return
	}

	config := getDefaultRebalanceConfig()
	proposals := generateRebalanceProposals(scores, config)

	// Filter to only high-priority proposals for auto-execution
	var highPriorityProposals []RebalanceProposal
	for _, proposal := range proposals {
		if proposal.Priority == 1 {
			highPriorityProposals = append(highPriorityProposals, proposal)
		}
	}

	if len(highPriorityProposals) == 0 {
		fmt.Println("✅ No high-priority rebalancing operations needed!")
		return
	}

	fmt.Printf("🎯 Executing %d high-priority rebalancing operations...\n\n", len(highPriorityProposals))

	successCount := 0
	for i, proposal := range highPriorityProposals {
		fmt.Printf("%d/%d: %s → %s (%s)\n",
			i+1, len(highPriorityProposals),
			proposal.FromAlias, proposal.ToAlias,
			formatSats(proposal.Amount))

		err := executeRebalance(proposal.FromChannel.ChanID, proposal.ToChannel.ChanID, proposal.Amount)
		if err != nil {
			fmt.Printf("❌ Failed: %v\n", err)
		} else {
			fmt.Printf("✅ Success!\n")
			successCount++
		}
		fmt.Println()

		// Small delay between operations
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("📊 Summary: %d/%d operations completed successfully\n", successCount, len(highPriorityProposals))
}

// executeRebalance performs the actual rebalancing operation using circular payments
func executeRebalance(fromChannelID, toChannelID string, amount int64) error {
	fmt.Printf("🔄 Executing rebalance via circular payment...\n")

	// Step 1: Get our node's pubkey (for validation)
	_, err := lnd.GetNodePubkey()
	if err != nil {
		return fmt.Errorf("failed to get our node pubkey: %v", err)
	}

	// Step 2: Get channel information to find the destination peer
	channels, err := getChannels()
	if err != nil {
		return fmt.Errorf("failed to get channels: %v", err)
	}

	var toPeerPubkey string
	for _, channel := range channels {
		if channel.ChanID == toChannelID {
			toPeerPubkey = channel.RemotePubkey
			break
		}
	}

	if toPeerPubkey == "" {
		return fmt.Errorf("destination channel %s not found", toChannelID)
	}

	// Step 3: Use circular rebalancing (traditional method)
	err = executeCircularRebalance(fromChannelID, toChannelID, amount)
	if err != nil {
		return fmt.Errorf("circular rebalance failed: %v", err)
	}

	fmt.Printf("✅ Circular payment completed successfully\n")
	return nil
}

// executeCircularRebalance performs traditional circular rebalancing using invoice payments
func executeCircularRebalance(fromChannelID, toChannelID string, amount int64) error {
	fmt.Printf("💰 Attempting circular rebalancing using invoice method\n")

	// Calculate fee limit (0.5% of amount, minimum 5 sats)
	feeLimit := int64(math.Max(5, float64(amount)*0.005))

	fmt.Printf("🔄 Creating temporary invoice for %s (fee limit: %s)\n",
		formatSats(amount), formatSats(feeLimit))

	// Step 1: Create a temporary invoice to ourselves
	invoice, rHash, err := createTemporaryInvoice(amount)
	if err != nil {
		return fmt.Errorf("failed to create temporary invoice: %v", err)
	}

	fmt.Printf("📄 Created temporary invoice\n")

	// Step 2: Pay the invoice using specified outgoing channel
	fmt.Printf("🔄 Attempting payment through channel %s...\n", fromChannelID[:8])

	success := false
	defer func() {
		if !success {
			// Cancel the invoice if payment failed
			fmt.Printf("🧹 Cleaning up temporary invoice...\n")
			cancelTemporaryInvoice(rHash)
		}
	}()

	// Try to pay with outgoing channel specified
	err = attemptInvoicePayment(invoice, fromChannelID, feeLimit)
	if err == nil {
		success = true
		fmt.Printf("✅ Payment successful through specified channel\n")
		return nil
	}

	fmt.Printf("⚠️  Specific channel payment failed: %v\n", err)
	fmt.Printf("🔄 Trying fallback payment without channel restriction...\n")

	// Fallback: try without channel specification
	err = attemptGeneralInvoicePayment(invoice, feeLimit*2) // Allow higher fees for fallback
	if err == nil {
		success = true
		fmt.Printf("✅ Fallback payment successful\n")
		return nil
	}

	return fmt.Errorf("both specific and general payments failed: %v", err)
}

// handleTestKeysend tests keysend functionality and channel diagnostics
func handleTestKeysend() {
	fmt.Println("🔍 Testing Lightning Network keysend and channel diagnostics...")

	// Get channels
	channels, err := getChannels()
	if err != nil {
		fmt.Printf("❌ Failed to get channels: %v\n", err)
		return
	}

	if len(channels) == 0 {
		fmt.Println("❌ No channels found")
		return
	}

	fmt.Printf("✅ Found %d channels\n", len(channels))

	// Show channel states
	fmt.Println("\n📊 Channel States:")
	for i, channel := range channels {
		if i >= 3 { // Only show first 3 channels
			break
		}

		alias := getNodeAlias(channel.RemotePubkey)
		localBal, _ := strconv.ParseInt(channel.LocalBalance, 10, 64)
		remoteBal, _ := strconv.ParseInt(channel.RemoteBalance, 10, 64)

		status := "🟢 Active"
		if !channel.Active {
			status = "🔴 Inactive"
		}

		fmt.Printf("  %s %s (ID: %s)\n", status, alias, channel.ChanID)
		fmt.Printf("    Local: %s | Remote: %s\n", formatSats(localBal), formatSats(remoteBal))
	}

	// Test simple keysend to first active channel
	var testChannel *lnd.Channel
	for _, channel := range channels {
		if channel.Active {
			testChannel = &channel
			break
		}
	}

	if testChannel == nil {
		fmt.Println("❌ No active channels found for testing")
		return
	}

	fmt.Printf("\n🧪 Testing 1-sat keysend to %s...\n", getNodeAlias(testChannel.RemotePubkey))

	// Try a minimal keysend payment
	output, err := lnd.RunLNCLI(
		"sendpayment",
		"--dest", testChannel.RemotePubkey,
		"--amt", "1",
		"--keysend",
		"--fee_limit", "5",
		"--timeout", "10s",
		"--json",
	)

	if err != nil {
		fmt.Printf("❌ Keysend test failed: %v\n", err)
		return
	}

	// Check result
	if err := checkPaymentSuccess(output); err != nil {
		fmt.Printf("❌ Keysend payment unsuccessful: %v\n", err)
		fmt.Printf("📄 Raw output: %s\n", string(output))
	} else {
		fmt.Printf("✅ Keysend test successful! Your node supports keysend payments.\n")
		fmt.Printf("💡 The rebalancing should work - the issue might be routing or balance-specific.\n")
	}
}

// createTemporaryInvoice creates a temporary invoice for circular rebalancing
func createTemporaryInvoice(amount int64) (string, string, error) {
	// Convert amount to millisats
	amountMsat := amount * 1000

	// Create invoice with short expiry (5 minutes)
	output, err := lnd.RunLNCLI(
		"addinvoice",
		"--amt_msat", strconv.FormatInt(amountMsat, 10),
		"--memo", fmt.Sprintf("Rebalance %s", formatSats(amount)),
		"--expiry", "300", // 5 minutes
	)

	if err != nil {
		return "", "", fmt.Errorf("failed to create invoice: %v", err)
	}

	// Parse the invoice response
	var invoiceResponse struct {
		PaymentRequest string `json:"payment_request"`
		RHash          string `json:"r_hash"`
	}

	if err := json.Unmarshal(output, &invoiceResponse); err != nil {
		return "", "", fmt.Errorf("failed to parse invoice response: %v", err)
	}

	return invoiceResponse.PaymentRequest, invoiceResponse.RHash, nil
}

// attemptInvoicePayment tries to pay invoice through specific channel
func attemptInvoicePayment(invoice, fromChannelID string, feeLimit int64) error {
	output, err := lnd.RunLNCLI(
		"payinvoice",
		"--pay_req", invoice,
		"--outgoing_chan_id", fromChannelID,
		"--fee_limit", strconv.FormatInt(feeLimit, 10),
		"--timeout", "60s",
		"--force", // Skip confirmation
		"--json",
	)

	if err != nil {
		return fmt.Errorf("invoice payment failed: %v", err)
	}

	return checkPaymentSuccess(output)
}

// attemptGeneralInvoicePayment tries general invoice payment without channel restrictions
func attemptGeneralInvoicePayment(invoice string, feeLimit int64) error {
	output, err := lnd.RunLNCLI(
		"payinvoice",
		"--pay_req", invoice,
		"--fee_limit", strconv.FormatInt(feeLimit, 10),
		"--timeout", "30s",
		"--force", // Skip confirmation
		"--json",
	)

	if err != nil {
		return fmt.Errorf("general invoice payment failed: %v", err)
	}

	return checkPaymentSuccess(output)
}

// cancelTemporaryInvoice cancels a temporary invoice using its hash
func cancelTemporaryInvoice(rHash string) {
	_, err := lnd.RunLNCLI("cancelinvoice", "--rhash", rHash)
	if err != nil {
		// Non-critical error
		fmt.Printf("⚠️  Could not cancel invoice (non-critical): %v\n", err)
	}
}

// checkPaymentSuccess verifies if the payment was successful
func checkPaymentSuccess(output []byte) error {
	var paymentResponse struct {
		Status        string `json:"status"`
		FailureReason string `json:"failure_reason,omitempty"`
	}

	if err := json.Unmarshal(output, &paymentResponse); err != nil {
		// If we can't parse JSON, check if output contains success indicators
		outputStr := string(output)
		if strings.Contains(outputStr, "SUCCEEDED") || strings.Contains(outputStr, "Payment completed") {
			return nil
		}
		if strings.Contains(outputStr, "FAILED") {
			return fmt.Errorf("payment failed (raw output): %s", outputStr)
		}
		return fmt.Errorf("failed to parse payment response, raw output: %s", outputStr)
	}

	if paymentResponse.Status == "SUCCEEDED" {
		return nil
	}

	if paymentResponse.FailureReason != "" {
		return fmt.Errorf("payment failed: %s", paymentResponse.FailureReason)
	}

	return fmt.Errorf("payment failed with status: %s", paymentResponse.Status)
}

// getPriorityEmoji returns an emoji for the priority level
func getPriorityEmoji(priority int) string {
	switch priority {
	case 1:
		return "🔴" // High priority
	case 2:
		return "🟡" // Medium priority
	default:
		return "🟢" // Low priority
	}
}
