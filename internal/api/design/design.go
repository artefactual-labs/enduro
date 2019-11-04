package design

import (
	. "goa.design/goa/v3/dsl"
	cors "goa.design/plugins/v3/cors/dsl"
)

var _ = API("enduro", func() {
	Title("Enduro API")
	Server("enduro", func() {
		Services("collection", "swagger")
		Host("localhost", func() {
			URI("http://localhost:9000")
		})
	})
	cors.Origin("*", func() {
		cors.Methods("GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS")
	})
})

var _ = Service("collection", func() {
	Description("The collection service manages packages being transferred to Archivematica.")
	HTTP(func() {
		Path("/collection")
	})
	Method("list", func() {
		Description("List all stored collections")
		Payload(func() {
			Attribute("original_id", String, "ID of the original dataset")
			Attribute("cursor", String, "Pagination cursor")
		})
		Result(PaginatedCollectionOf(StoredCollection))
		HTTP(func() {
			GET("/")
			Response(StatusOK)
			Params(func() {
				Param("original_id")
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
})

var _ = Service("swagger", func() {
	Description("The swagger service serves the API swagger definition.")
	HTTP(func() {
		Path("/swagger")
	})
	Files("/swagger.json", "internal/api/gen/http/openapi.json", func() {
		Description("JSON document containing the API swagger definition.")
	})
})

var Collection = Type("Collection", func() {
	Description("Collection describes a collection to be stored.")
	Attribute("name", String, "Name of the collection")
	Attribute("status", String, "Status of the collection", func() {
		Enum("new", "in progress", "done", "error", "unknown")
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
	Attribute("created_at", String, "Creation datetime", func() {
		Format(FormatDateTime)
	})
	Attribute("completed_at", String, "Completion datetime", func() {
		Format(FormatDateTime)
	})
	Required("id", "status", "created_at")
})

var StoredCollection = ResultType("application/vnd.enduro.stored-collection", func() {
	Description("StoredPackage describes a collection retrieved by the service.")
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
		Attribute("created_at")
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
		Attribute("created_at")
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
