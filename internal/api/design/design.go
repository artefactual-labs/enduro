/*
Package design is the single source of truth of Enduro's API. It uses the Goa
design language (https://goa.design) which is a Go DSL.

We describe multiple services (collection, pipeline) which map to resources in
REST or service declarations in gRPC. Services define their own methods, errors,
etc...
*/
package design

import (
	. "goa.design/goa/v3/dsl"
	"goa.design/goa/v3/expr"
	cors "goa.design/plugins/v3/cors/dsl"
)

var _ = API("enduro", func() {
	Title("Enduro API")
	Randomizer(expr.NewDeterministicRandomizer())
	Server("enduro", func() {
		Services("pipeline", "batch", "collection", "swagger")
		Host("localhost", func() {
			URI("http://localhost:9000")
		})
	})
	cors.Origin("*", func() {
		cors.Methods("GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS")
	})
})
