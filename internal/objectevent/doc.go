// Package objectevent provides an internal webhook bridge for object storage
// event producers.
//
// The package receives provider-specific webhook payloads, normalizes
// object-created file events into watcher.EnduroEvent values, and publishes
// them to Redis. The S3 watcher can then consume those Redis messages with
// eventFormat = "enduro".
//
// This package does not start processing workflows directly. Workflow startup
// remains owned by the watcher loop so all object event sources use the same
// ingestion path.
package objectevent
