// Package models contains data structures used by the JungleBus client
package models

// StatusMessage represents a status message from the JungleBus server
type StatusMessage struct {
	Code    uint   `json:"code"`
	Message string `json:"message"`
}
