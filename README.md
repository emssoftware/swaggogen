# Swaggogen

Swaggogen is a tool for extracting Go (golang) type information from an
application and combining it with code comments to generate a Swagger/OpenAPI
2.0 specification document.


## Operation

Swaggogen takes one parameter, `pkg`. This parameter should be the package path
of the application you want to document.

Example:

```
swaggogen -pkg github.com/foo/bar
```

The application will generate the Swagger/OpenAPI document as JSON and print it
to stdout.

It is acknowledged that there are some unavoidable warnings that are printed to
stderr, and it's not pretty. The author(s) know this, and it is preferred that
end users be aware of the limitations as they exist. Because these warnings are
printed to stderr (not stdout), they should not affect the output of any JSON.
Furthermore, in practical settings, these warnings have never indicated a
failure to generate a complete Swagger specification document. Please feel free
to submit a merge request as appropriate.

## Annotations

Swaggogen observes two kinds of code blocks, **API definitions** and **Route
Definitions**. The lines that are parsed for use in the Swagger document must
contain a keyword, which is a marker beginning with an '@'. The format of each
line depends on the keyword.

These annotations are intended to be compatible with Yuriy Vasiyarov's project,
found at github.com/yvasiyarov/swagger.

For the sake of simplicity, a **Route Definition**  combines the necessary
information to generate Paths and Operations in Swagger terminology. For this
reason, throughout the documentation, a *Route* will be in reference to a
Swagger *Operation* in combination with its respective *Path*.

### API Definitions

**API Definitions** are comprised of lines beginning with the following keywords:

* @APIVersion
* @APITitle
* @APIDescription
* @BasePath
* @SubApi

Any comment block containing the `@APITitle` tag is considered an **API
Definition**. Multiple API definitions are allowed, but they will be combined
without any guarantees of precedence.

#### @APIVersion

The `@APIVersion` tag defines the API version of your application. It is
followed by a single argument, a version number. Any combination of contiguous
digits and periods is acceptable.

Example:

```
@APIVersion 1.0.0
```

#### @APITitle

The `@APITitle` tag defines the title of your API. Any text after the tag is
accepted as your title.

Example:

```
@APITitle REST API
```

#### @APIDescription

The `@APIDescription` tag defines the description of your API. Any text after
the tag is accepted as your description.

Example:

```
@APIDescription My API is awesome!
```

#### @BasePath

The `@BasePath` tag defines the base path of your API. Per the Swagger
specification, this path is prepended to all paths defined in the Paths of your
specification. An acceptable path (URL component) should begin with a forward
slash and contain letters, numbers, periods, slashes, hyphens, and underscores.

Example:

```
@BasePath /api/v1
```

#### @SubApi

The `@SubApi` tag defines a logic grouping of paths/routes. If the path of an
operation begins with the route defined by this tag, then the operation will be
tagged with the name defined by this tag.

The `@SubApi` tag should be followed by two arguments. The first is the name of
the Sub-API. The second, enclosed is square brackets, is the path segment that
defines the sub-API.

Multiple `@SubApi` tags can be defined.

Example:

```
@SubApi Contacts [/contacts]
```

### Route Definitions

*Route Definitions* are comprised of lines beginning with the following keywords:

* @Accept
* @Description
* @Param
* @Success
* @Failure
* @Router
* @Title

Any comment block containing the `@Router` tag is considered an **Route
Definition**. Multiple API route definitions are allowed.

#### @Accept

The `@Accept` tag defines the set of MIME types that this Route consumes and
produces; symmetry in this regard is assumed.

The `@Accept` tag should be followed by a comma-separated list of MIME types.

Admittedly, this tool takes some liberties and simply checks for the presence
of `json` or `xml`. Accordingly, the **Produces** and **Consumes** properties
of the corresponding Swagger Operation object is populated with standard
`application/json` and `application/xml` strings. This behavior will likely
change as greater sophistication is required. Feel free to submit a merge
request with more sophisticated behavior.

Example:

```
@Accept  json
```


#### @Description

The `@Description` tag defines a human readable description for the Swagger Operation.

Everything after the `@Description` tag is assumed to be the description.

Example:

```
@Description This route is a good one.
```

#### @Param

The `@Param` tag defines a request parameter.

This tag expects five arguments in order: parameter name, parameter location
(such as 'path', 'body', etc. per the Swagger spec), parameter type, a boolean
that indicates if the parameter is required, and a double-quote delimited
description of the parameter.

The type argument can be a Swagger-defined primitive type (int, string, boolean,
etc.) or a Go type. If the argument references a Go type, it must be specified
exactly how it would be referenced in code. For example, if the struct type
**Foo** is in the local package, then the argument can be referenced simply with
`Foo`. If it is defined in another package that is imported with an alias
(`import f "/github.com/emssoftware/fooness"`), then the type argument should be
referenced with the alias, `f.Foo`.

Example:

```
@Param   id	path    int true    "Thing ID"
```

#### @Success

The `@Success` tag defines a response.

This tag expects four arguments in order: HTTP status code, a largely ignored
argument, a type, and a description. The second argument is kept for backwards
compatibility with yvasiyarov's annotations. The description must be
double-quote delimited.

The type argument can be a Swagger-defined primitive type (int, string, boolean,
etc.) or a Go type. If the argument references a Go type, it must be specified
exactly how it would be referenced in code. For example, if the struct type
**Foo** is in the local package, then the argument can be referenced simply with
`Foo`. If it is defined in another package that is imported with an alias
(`import f "/github.com/emssoftware/fooness"`), then the type argument should be
referenced with the alias, `f.Foo`.

Example:

```
@Success 200 {object} model.ThingViewModel "Success"
```

#### @Failure

`@Failure` is parsed in the same way as the `@Success` tag. They're effectively
synonyms.

```
@Failure 400 {object} apicommon.ErrorResponse "Bad Request"
```

#### @Router

The `@Router` tag defines the path for our Route (Operation/Path combination).

This tag expects two arguments, a route and an HTTP method (PUT, GET, POST,
DELETE, OPTIONS, HEAD, PATCH) enclosed in square brackets. The HTTP method is
not case sensitive. The route may contain identifiers in curly braces, but
Gorilla Web Toolkit style regular expression expressions are not supported.

Example:

```
@Router /bookviews/{id} [get]
```

#### @Title

The `@Title` tag defines a title for the Swagger operation.

All the text after the tag is considered the title.

Example:

```
@Title Get Thing
```


# Creditation

This tool was written blind with respect to other similar tools.

While this tool is intended to utilize the same annotations as the yvasiyarov
project, the original parsing algorithms were not copied (or even used as
reference). Therefore, exact parsing behavior is not expected to be the same.

Also, the availability of the `github.com/go-openapi/spec` library is greatly
appreciated.