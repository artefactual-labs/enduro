/*
Package design is the single source of truth of Enduro's API. It uses the Goa
design language (https://goa.design) which is a Go DSL.

We describe multiple services (collection) which map to resources in
REST or service declarations in gRPC. Services define their own methods, errors,
etc...
*/
package design

import (
	. "goa.design/goa/v3/dsl"
	cors "goa.design/plugins/v3/cors/dsl"
)

var _ = API("enduro", func() {
	Title("Enduro API")
	Server("enduro", func() {
		Services("batch", "collection", "swagger")
		Host("localhost", func() {
			URI("http://localhost:9000")
		})
	})
	cors.Origin("*", func() {
		cors.Methods("GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS")
	})
})
