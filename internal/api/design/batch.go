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
			Attribute("exclude_hidden_files", Boolean, func() { Default(false) })
			Attribute("transfer_type", String)
			Attribute("process_name_metadata", Boolean, func() { Default(false) })
			Attribute("depth", Int, func() {
				Default(0)
				Minimum(0)
			})
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
	Method("browse", func() {
		Description("Browse batch source directories")
		Payload(func() {
			Attribute("path", String, "Root-relative directory path to list")
		})
		Result(BatchBrowseResult)
		Error("not_available")
		Error("not_valid")
		HTTP(func() {
			GET("/browser")
			Response(StatusOK)
			Response("not_available", StatusNotFound)
			Response("not_valid", StatusBadRequest)
			Params(func() {
				Param("path")
			})
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
	Attribute("browser_enabled", Boolean, "Whether the batch source directory browser is configured.", func() {
		Default(false)
	})
})

var BatchBrowseResult = Type("BatchBrowseResult", func() {
	Attribute("path", String, "Root-relative path of the listed directory.")
	Attribute("absolute_path", String, "Absolute path of the listed directory.")
	Attribute("entries", ArrayOf(BatchBrowseEntry), "Immediate child directories.")
	Attribute("truncated", Boolean, "Whether the result was truncated because it exceeded the entry limit.", func() {
		Default(false)
	})
	Required("path", "absolute_path", "entries", "truncated")
})

var BatchBrowseEntry = Type("BatchBrowseEntry", func() {
	Attribute("name", String, "Directory name.")
	Attribute("path", String, "Root-relative path of the directory.")
	Attribute("absolute_path", String, "Absolute path of the directory.")
	Attribute("modified_at", String, "Directory modification time.", func() {
		Format(FormatDateTime)
	})
	Required("name", "path", "absolute_path")
})
