---
title: "API changes"
linkTitle: "API changes"
weight: 2
description: >
  Working with the API.
---

Enduro uses the [Goa framework][goa] to build the API. Goa provides a
[design language][goa-dsl] which is our source of truth from which both
behavior and docs are derived.

Goa generates all the code that relates to the communication transport as well
as the API specification using OpenAPI. Enduro provides the implementation of
the services where we include the business logic. This idea is described by
Robert C. Martin in [The Clean Architecture][clean-arch].

See also:

- [Dynamic documentation in our website][gen-api-hugo]
- [OpenAPI document (openapi.json)][gen-api-spec]

## API design

Our API definition can be found in the [api/design][design-pkg] package. The
following example shows how we've described the `DELETE /collection` API.

```go
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
```

With this design language we describe our API services, methods and types. It
is also possible to implement some advanced features such different
authentication methods or streaming of contents.

## Backend development

After making new changes to the API design, the developer should run:
`make goagen` which generates all the code under `internal/api/gen`, including
the OpenAPI description of the API for the HTTP transport:
[`openapi.json`][openapi-json].

In the example above, we added a `Delete` method to the `Collection` service.
The corresponding Go interface gets a new method:

```go
Delete(context.Context, *DeletePayload) (err error)
```

It's now up to the developer to implement the expected functionality.

## Frontend development

We use `openapi-generator-cli` to generate the client code after the OpenAPI
description of the API. Run `make ui-client` to generate all the TypeScript
code under `ui/src/client` which is used by the Enduro frontend.

You should see new models and methods added, like the new `collectionDelete`
method:

```ts
async collectionDelete(requestParameters: CollectionDeleteRequest): Promise<void> {
    await this.collectionDeleteRaw(requestParameters);
}
```


[goa]: https://goa.design/
[goa-dsl]: https://godoc.org/goa.design/goa/dsl
[design-pkg]: https://github.com/artefactual-labs/enduro/tree/main/internal/api/design
[clean-arch]: https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html
[openapi-json]: https://github.com/artefactual-labs/enduro/blob/main/internal/api/gen/http/openapi.json
[gen-api-hugo]: {{< ref "/docs/api" >}}
[gen-api-spec]: /openapi.json
