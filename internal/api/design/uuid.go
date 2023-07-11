// # UUID attributes
//
// Use [AttributeUUID] or [TypedAttributeUUID] to declare UUID attributes.
//
// These attributes produce consistent example UUIDs.
package design

import (
	. "goa.design/goa/v3/dsl"
)

// AttributeUUID describes a string typed field that must be a valid UUID.
// The desc is a short description of the field's purpose.
//
// AttributeUUID's example value is a deterministic UUID.
func AttributeUUID(name, desc string) {
	Attribute(name, String, desc, func() {
		Format(FormatUUID)
		Example("d1845cb6-a5ea-474a-9ab8-26f9bcd919f5")
	})
}

// TypedAttributeUUID describes a [uuid.UUID] typed field. The desc is a short
// description of the field's purpose.
//
// TypedAttributeUUID's example value is a deterministic UUID.
//
// [uuid.UUID]: https://github.com/google/uuid
func TypedAttributeUUID(name, desc string) {
	Attribute(name, String, desc, func() {
		Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
		Example("d1845cb6-a5ea-474a-9ab8-26f9bcd919f5")
	})
}
