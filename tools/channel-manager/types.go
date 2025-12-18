package main

import "github.com/brewgator/lightning-node-tools/internal/lnd"

// Type aliases for convenience - use shared types from pkg/lnd
type Channel = lnd.Channel
type ChannelFeeReport = lnd.ChannelFeeReport
type FeeReportResponse = lnd.FeeReportResponse
type ForwardingHistory = lnd.ForwardingHistory
type ForwardingEvent = lnd.ForwardingEvent
