// Code generated by goa v3.13.0, DO NOT EDIT.
//
// HTTP request path constructors for the collection service.
//
// Command:
// $ goa gen github.com/artefactual-labs/enduro/internal/api/design -o
// internal/api

package client

import (
	"fmt"
)

// MonitorCollectionPath returns the URL path to the collection service monitor HTTP endpoint.
func MonitorCollectionPath() string {
	return "/collection/monitor"
}

// ListCollectionPath returns the URL path to the collection service list HTTP endpoint.
func ListCollectionPath() string {
	return "/collection"
}

// ShowCollectionPath returns the URL path to the collection service show HTTP endpoint.
func ShowCollectionPath(id uint) string {
	return fmt.Sprintf("/collection/%v", id)
}

// DeleteCollectionPath returns the URL path to the collection service delete HTTP endpoint.
func DeleteCollectionPath(id uint) string {
	return fmt.Sprintf("/collection/%v", id)
}

// CancelCollectionPath returns the URL path to the collection service cancel HTTP endpoint.
func CancelCollectionPath(id uint) string {
	return fmt.Sprintf("/collection/%v/cancel", id)
}

// RetryCollectionPath returns the URL path to the collection service retry HTTP endpoint.
func RetryCollectionPath(id uint) string {
	return fmt.Sprintf("/collection/%v/retry", id)
}

// WorkflowCollectionPath returns the URL path to the collection service workflow HTTP endpoint.
func WorkflowCollectionPath(id uint) string {
	return fmt.Sprintf("/collection/%v/workflow", id)
}

// DownloadCollectionPath returns the URL path to the collection service download HTTP endpoint.
func DownloadCollectionPath(id uint) string {
	return fmt.Sprintf("/collection/%v/download", id)
}

// DecideCollectionPath returns the URL path to the collection service decide HTTP endpoint.
func DecideCollectionPath(id uint) string {
	return fmt.Sprintf("/collection/%v/decision", id)
}

// BulkCollectionPath returns the URL path to the collection service bulk HTTP endpoint.
func BulkCollectionPath() string {
	return "/collection/bulk"
}

// BulkStatusCollectionPath returns the URL path to the collection service bulk_status HTTP endpoint.
func BulkStatusCollectionPath() string {
	return "/collection/bulk"
}
