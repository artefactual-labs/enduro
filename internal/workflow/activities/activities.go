// Package activities implements Enduro's workflow activities.
package activities

const (
	AcquirePipelineActivityName  = "acquire-pipeline-activity"
	DownloadActivityName         = "download-activity"
	BundleActivityName           = "bundle-activity"
	TransferActivityName         = "transfer-activity"
	PollTransferActivityName     = "poll-transfer-activity"
	PollIngestActivityName       = "poll-ingest-activity"
	CleanUpActivityName          = "clean-up-activity"
	HidePackageActivityName      = "hide-package-activity"
	DeleteOriginalActivityName   = "delete-original-activity"
	DisposeOriginalActivityName  = "dispose-original-activity"
	ValidateTransferActivityName = "validate-transfer-activity"
)
