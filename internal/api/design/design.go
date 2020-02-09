package design

import (
	. "goa.design/goa/v3/dsl"
	cors "goa.design/plugins/v3/cors/dsl"
)

var _ = API("enduro", func() {
	Title("Enduro API")
	Server("enduro", func() {
		Services("pipeline", "collection", "swagger")
		Host("localhost", func() {
			URI("http://localhost:9000")
		})
	})
	cors.Origin("*", func() {
		cors.Methods("GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS")
	})
})
