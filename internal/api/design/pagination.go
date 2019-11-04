package design

import "goa.design/goa/v3/dsl"

func PaginatedCollectionOf(v interface{}, adsl ...func()) interface{} {
	return func() {
		dsl.Attribute("items", dsl.CollectionOf(v, adsl...))
		dsl.Attribute("next_cursor", dsl.String)
		dsl.Required("items")
	}
}
