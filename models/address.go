package models

// Address struct
type Address struct {
	ID            string `json:"id"` // unique od this address record = sha256(Address + TransactionID + BlockIndex)
	Address       string `json:"address"`
	TransactionID string `json:"transaction_id"`
	BlockHeight   uint32 `json:"block_height"`
	BlockHash     string `json:"block_hash"`
	BlockIndex    uint64 `json:"block_index"`
}
