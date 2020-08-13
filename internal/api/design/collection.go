package design

import (
	. "goa.design/goa/v3/dsl"
)

var _ = Service("collection", func() {
	Description("The collection service manages packages being transferred to Archivematica.")
	HTTP(func() {
		Path("/collection")
	})
	Method("list", func() {
		Description("List all stored collections")
		Payload(func() {
			Attribute("name", String)
			Attribute("original_id", String)
			Attribute("transfer_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("aip_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("pipeline_id", String, func() {
				Format(FormatUUID)
			})
			Attribute("earliest_created_time", String, func() {
				Format(FormatDateTime)
			})
			Attribute("latest_created_time", String, func() {
				Format(FormatDateTime)
			})
			Attribute("status", String, func() {
				EnumCollectionStatus()
			})
			Attribute("cursor", String, "Pagination cursor")
		})
		Result(PaginatedCollectionOf(StoredCollection))
		HTTP(func() {
			GET("/")
			Response(StatusOK)
			Params(func() {
				Param("name")
				Param("original_id")
				Param("transfer_id")
				Param("aip_id")
				Param("pipeline_id")
				Param("earliest_created_time")
				Param("latest_created_time")
				Param("status")
				Param("cursor")
			})
		})
	})
	Method("show", func() {
		Description("Show collection by ID")
		Payload(func() {
			Attribute("id", UInt, "Identifier of collection to show")
			Required("id")
		})
		Result(StoredCollection)
		Error("not_found", NotFound, "Collection not found")
		HTTP(func() {
			GET("/{id}")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
		})
	})
	Method("delete", func() {
		Description("Delete collection by ID")
		Payload(func() {
			Attribute("id", UInt, "Identifier of collection to delete")
			Required("id")
		})
		Error("not_found", NotFound, "Collection not found")
		HTTP(func() {
			DELETE("/{id}")
			Response(StatusNoContent)
			Response("not_found", StatusNotFound)
		})
	})
	Method("cancel", func() {
		Description("Cancel collection processing by ID")
		Payload(func() {
			Attribute("id", UInt, "Identifier of collection to remove")
			Required("id")
		})
		Error("not_found", NotFound, "Collection not found")
		Error("not_running")
		HTTP(func() {
			POST("/{id}/cancel")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("not_running", StatusBadRequest)
		})
	})
	Method("retry", func() {
		Description("Retry collection processing by ID")
		Payload(func() {
			Attribute("id", UInt, "Identifier of collection to retry")
			Required("id")
		})
		Error("not_found", NotFound, "Collection not found")
		Error("not_running")
		HTTP(func() {
			POST("/{id}/retry")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("not_running", StatusBadRequest)
		})
	})
	Method("workflow", func() {
		Description("Retrieve workflow status by ID")
		Payload(func() {
			Attribute("id", UInt, "Identifier of collection to look up")
			Required("id")
		})
		Result(WorkflowStatus)
		Error("not_found", NotFound, "Collection not found")
		HTTP(func() {
			GET("/{id}/workflow")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
		})
	})
	Method("download", func() {
		Description("Download collection by ID")
		Payload(func() {
			Attribute("id", UInt, "Identifier of collection to look up")
			Required("id")
		})
		Result(Bytes)
		Error("not_found", NotFound, "Collection not found")
		HTTP(func() {
			GET("/{id}/download")
			Response(StatusOK)
			Response("not_found", StatusNotFound)
		})
	})
	Method("decide", func() {
		Description("Make decision for a pending collection by ID")
		Payload(func() {
			Attribute("id", UInt, "Identifier of collection to look up")
			Attribute("option", String, "Decision option to proceed with")
			Required("id", "option")
		})
		Error("not_found", NotFound, "Collection not found")
		Error("not_valid")
		HTTP(func() {
			POST("/{id}/decision")
			Body(func() {
				Attribute("option")
			})
			Response(StatusOK)
			Response("not_found", StatusNotFound)
			Response("not_valid", StatusBadRequest)
		})
	})
	Method("bulk", func() {
		Description("Bulk operations (retry, cancel...).")
		Payload(func() {
			Attribute("operation", String, func() {
				Enum("retry", "cancel", "abandon")
			})
			Attribute("status", String, func() {
				EnumCollectionStatus()
			})
			Attribute("size", UInt, func() {
				Default(100)
			})
			Required("operation", "status")
		})
		Result(BulkResult)
		Error("not_available")
		Error("not_valid")
		HTTP(func() {
			POST("/bulk")
			Response(StatusAccepted)
			Response("not_available", StatusConflict)
			Response("not_valid", StatusBadRequest)
		})
	})
	Method("bulk_status", func() {
		Description("Retrieve status of current bulk operation.")
		Result(BulkStatusResult)
		HTTP(func() {
			GET("/bulk")
			Response(StatusOK)
		})
	})
})

var EnumCollectionStatus = func() {
	Enum("new", "in progress", "done", "error", "unknown", "queued", "pending", "abandoned")
}

var Collection = Type("Collection", func() {
	Description("Collection describes a collection to be stored.")
	Attribute("name", String, "Name of the collection")
	Attribute("status", String, "Status of the collection", func() {
		EnumCollectionStatus()
		Default("new")
	})
	Attribute("workflow_id", String, "Identifier of processing workflow", func() {
		Format(FormatUUID)
	})
	Attribute("run_id", String, "Identifier of latest processing workflow run", func() {
		Format(FormatUUID)
	})
	Attribute("transfer_id", String, "Identifier of Archivematica transfer", func() {
		Format(FormatUUID)
	})
	Attribute("aip_id", String, "Identifier of Archivematica AIP", func() {
		Format(FormatUUID)
	})
	Attribute("original_id", String, "Identifier provided by the client")
	Attribute("pipeline_id", String, "Identifier of Archivematica pipeline", func() {
		Format(FormatUUID)
	})
	Attribute("created_at", String, "Creation datetime", func() {
		Format(FormatDateTime)
	})
	Attribute("started_at", String, "Start datetime", func() {
		Format(FormatDateTime)
	})
	Attribute("completed_at", String, "Completion datetime", func() {
		Format(FormatDateTime)
	})
	Required("id", "status", "created_at")
})

var StoredCollection = ResultType("application/vnd.enduro.stored-collection", func() {
	Description("StoredCollection describes a collection retrieved by the service.")
	Reference(Collection)
	Attributes(func() {
		Attribute("id", UInt, "Identifier of collection")
		Attribute("name")
		Attribute("status")
		Attribute("workflow_id")
		Attribute("run_id")
		Attribute("transfer_id")
		Attribute("aip_id")
		Attribute("original_id")
		Attribute("pipeline_id")
		Attribute("created_at")
		Attribute("started_at")
		Attribute("completed_at")
	})
	View("default", func() {
		Attribute("id")
		Attribute("name")
		Attribute("status")
		Attribute("workflow_id")
		Attribute("run_id")
		Attribute("transfer_id")
		Attribute("aip_id")
		Attribute("original_id")
		Attribute("pipeline_id")
		Attribute("created_at")
		Attribute("started_at")
		Attribute("completed_at")
	})
	Required("id", "status", "created_at")
})

var WorkflowStatus = ResultType("application/vnd.enduro.collection-workflow-status", func() {
	Description("WorkflowStatus describes the processing workflow status of a collection.")
	Attribute("status", String) // TODO
	Attribute("history", CollectionOf(WorkflowHistoryEvent))
})

var WorkflowHistoryEvent = ResultType("application/vnd.enduro.collection-workflow-history", func() {
	Description("WorkflowHistoryEvent describes a history event in Cadence.")
	Attributes(func() {
		Attribute("id", UInt, "Identifier of collection")
		Attribute("type", String, "Type of the event")
		Attribute("details", Any, "Contents of the event")
	})
})

var NotFound = Type("NotFound", func() {
	Description("NotFound is the type returned when attempting to operate with a collection that does not exist.")
	Attribute("message", String, "Message of error", func() {
		Meta("struct:error:name")
	})
	Attribute("id", UInt, "Identifier of missing collection")
	Required("message", "id")
})

var BulkResult = Type("BulkResult", func() {
	Attribute("workflow_id", String)
	Attribute("run_id", String)
	Required("workflow_id", "run_id")
})

var BulkStatusResult = Type("BulkStatusResult", func() {
	Attribute("running", Boolean)
	Attribute("started_at", String, func() {
		Format(FormatDateTime)
	})
	Attribute("closed_at", String, func() {
		Format(FormatDateTime)
	})
	Attribute("status", String)
	Attribute("workflow_id", String)
	Attribute("run_id", String)
	Required("running")
})
