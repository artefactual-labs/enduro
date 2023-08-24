package design

import (
	. "goa.design/goa/v3/dsl"
)

var _ = Service("batch", func() {
	Description("The batch service manages batches of collections.")
	HTTP(func() {
		Path("/batch")
	})
	Method("submit", func() {
		Description("Submit a new batch")
		Payload(func() {
			Attribute("path", String)
			Attribute("pipeline", String)
			Attribute("processing_config", String)
			Attribute("completed_dir", String)
			Attribute("retention_period", String)
			Attribute("reject_duplicates", Boolean, func() { Default(false) })
			Attribute("transfer_type", String)
			Attribute("process_name_metadata", Boolean, func() { Default(false) })
			Required("path")
		})
		Result(BatchResult)
		Error("not_available")
		Error("not_valid")
		HTTP(func() {
			POST("/")
			Response(StatusAccepted)
			Response("not_available", StatusConflict)
			Response("not_valid", StatusBadRequest)
		})
	})
	Method("status", func() {
		Description("Retrieve status of current batch operation.")
		Result(BatchStatusResult)
		HTTP(func() {
			GET("/")
			Response(StatusOK)
		})
	})
	Method("hints", func() {
		Description("Retrieve form hints")
		Result(BatchHintsResult)
		HTTP(func() {
			GET("/hints")
			Response(StatusOK)
		})
	})
})

var BatchResult = Type("BatchResult", func() {
	Attribute("workflow_id", String)
	Attribute("run_id", String)
	Required("workflow_id", "run_id")
})

var BatchStatusResult = Type("BatchStatusResult", func() {
	Attribute("running", Boolean)
	Attribute("status", String)
	Attribute("workflow_id", String)
	Attribute("run_id", String)
	Required("running")
})

var BatchHintsResult = Type("BatchHintsResult", func() {
	Attribute("completed_dirs", ArrayOf(String), "A list of known values of completedDir used by existing watchers.")
})
