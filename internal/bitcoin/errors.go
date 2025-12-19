package bitcoin

import "errors"

var (
	// ErrInvalidAddress indicates an invalid Bitcoin address
	ErrInvalidAddress = errors.New("invalid Bitcoin address")
	
	// ErrNodeNotConnected indicates Bitcoin node is not accessible
	ErrNodeNotConnected = errors.New("Bitcoin node not connected")
	
	// ErrAddressNotImported indicates address is not imported in wallet
	ErrAddressNotImported = errors.New("address not imported in wallet")
	
	// ErrInsufficientIndex indicates txindex is not enabled
	ErrInsufficientIndex = errors.New("Bitcoin node requires txindex=1 for full functionality")
)