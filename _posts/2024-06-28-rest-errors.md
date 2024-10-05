---
layout: post
title: Effective RESTful Error Handling in Go
date: 2024-06-28 20:49 +0300
mermaid: true
description: Streamline application error logging and output standardized RESTful responses.
categories: [Go]
tags: [go, golang, error, error-handling, resterr, rest]
---

## Introduction

Go’s approach to error handling tends to spark strong opinions. Whether you love it or hate it, many developers struggle with propagating and processing errors while ensuring meaningful responses are returned to clients, all while keeping application details secure.

In this post, I’ll explain the approach I typically use to turn application errors into JSON responses, along with the benefits of this method.

When working on applications that expose APIs, it’s crucial to decide how to handle errors before sending them to clients:

1. The internal application’s error details should be accessible to developers and operations teams for metrics collection and troubleshooting.
2. Any error that occurs during request processing should be translated into a format supported by the API, providing useful information to the client.
3. By default, we should prevent internal implementation details from being exposed to external users in error responses.

For example, if an API request fails because a record doesn’t exist, we should convey this with an HTTP Not Found status to the client. On the other hand, unexpected issues like connection errors should result in an Internal Server Error (500) to avoid exposing sensitive information.

You can access the implementation of this error handler [here](https://github.com/alesr/resterr). A complete demonstration of an application using this error handler can be found [here](https://github.com/alesr/resterrdemo).

## Implementation

To achieve this, the idea is to implement an **error handler** that can be attached to the transport layer. Each time the transport layer receives a request, it calls the service (or domain) layer to process it. When a result is returned, the handler logs the original error and sends a JSON payload mapped to the error returned by the service.

In this setup, the service is responsible for listing all errors that could occur while processing the business logic and providing the transport version of each error. The transport layer then communicates the error according to its API specification.

If the service returns an error that hasn’t been mapped to a transport error, we treat it as unexpected and return a *500 Internal Server Error*.

After explaining the error handler implementation, I’ll show how to use it within your application, and finally, we’ll discuss how to make the most of Go’s error handling when propagating errors through your code.

### Step 1: Setting up the Error Handler

It’s widely accepted that dependency injection, combined with some form of layered architecture, is beneficial—particularly for web applications where layers such as transport, domain (business), and storage are well-separated. In this approach, dependency injection allows us to easily add functionality across layers, minimizing changes to critical code.

For our case, we’ll create a small service that logs errors and returns compatible JSON error responses.

We’ll begin by implementing a handler, which will have the necessary dependencies to process and handle errors.

```go
// Handler handles standard errors by logging them and looking for an equivalent REST error in the error map.
// Errors that are not mapped result in internal server errors.
type Handler struct {
	logger          *slog.Logger
	internalErrJSON []byte
	errorMap        sync.Map
	validationFn    func(restErr RESTErr) error
}
```
{: file="resterr/handler.go" .nolineno }

The `Handler` struct holds a logger for logging the original errors, a JSON representation of a 500 internal response (used when we want to hide the real cause of the error), and an error map that holds the mappings of expected errors to REST JSON responses. It can also contain a validation function that checks the error map’s completeness during runtime.

The internal `errorMap` is not the same as the one passed by the caller. It contains the original mappings plus JSON-marshaled byte slices, so we don’t have to marshal Go structs into JSON for every response.

```go
func NewHandler(logger *slog.Logger, errorMap map[error]RESTErr, opts ...Option) (*Handler, error) {
	internalErrJSON, err := json.Marshal(internalErr)
	if err != nil {
		return nil, fmt.Errorf("could not marshal internal err: %w", err)
	}

	h := Handler{
		logger:          logger.WithGroup("resterr-handler"),
		errorMap:        make(map[error]*RESTErr, len(errorMap)),
		internalErrJSON: internalErrJSON,
	}

	for _, o := range opts {
		o(&h)
	}

	for k, e := range errorMap {
		if h.validationFn != nil {
			if err := h.validationFn(e); err != nil {
				return nil, fmt.Errorf("validation failed for REST error '%v': %w", e, err)
			}
		}

		res, err := json.Marshal(&e)
		if err != nil {
			return nil, fmt.Errorf("could not marshal REST error '%v': %w", e, err)
		}
		e.json = res

		h.errorMap[k] = &e
	}
	return &h, nil
}
```
{: file="resterr/handler.go" .nolineno }

Next, we implement the method that will handle errors:

```go
func (h *Handler) Handle(ctx context.Context, w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")

	var restErr RESTErr
	if errors.As(err, &restErr) {
		h.logger.ErrorContext(ctx, "Handling REST error.", slog.String("error", err.Error()))
		h.write(ctx, w, restErr)
		return
	}

	for k, v := range h.errorMap {
		if errors.Is(err, k) {
			h.logger.ErrorContext(ctx, "Handling mapped error.", slog.String("error", err.Error()))
			h.write(ctx, w, *v)
			return
		}
	}

	h.logger.ErrorContext(ctx, "Handling unmapped error.", slog.String("source-error", err.Error()))
	h.writeInternalErr(ctx, w)
}
```
{: file="resterr/handler.go" .nolineno }

If the error we receive is already a `RESTErr`, we log it and use it as the response. Errors not present in the map can be sent directly by middleware running before the HTTP handler processes the request. Be aware that unmapped errors won’t be covered by validation during handler initialization.

If the error we received isn't a `RESTErr`, we use `errors.Is()` to check if the error belongs to any of the keys in the map. If it is, we log the error and respond with the equivalent JSON data.

If the error isn't a `RESTErr` and is not a mapped error, we log the original error and return a 500 internal server error.

You can try this error handler by importing it from this [repository](https://github.com/alesr/resterr). All you need to do is provide an error map containing the errors your logic might return, along with the corresponding client response as you'll see below.

### Step 2: Using the Error Handler

To use the error handler, we need to ensure that our HTTP handler has access to it and that it calls the handler whenever an error occurs. Call the `Handle` method, passing the request context and response writer, and wrap the error with `%w` to provide as much information as possible.

```go
type fooService interface {
	Fetch() error
}

type errHandler interface {
	Handle(ctx context.Context, w http.ResponseWriter, err error)
}

// FooHandler implements HTTP handlers and processes requests related to the foo resource.
type FooHandler struct {
	logger     *slog.Logger
	fooSvc     fooService
	errHandler errHandler
}

// NewHandler instantiates a new FooHandler struct.
func NewHandler(logger *slog.Logger, fooSvc fooService, errHandler errHandler) (*FooHandler, error) {
	return &FooHandler{
		logger:     logger.WithGroup("foo-rest-handler"),
		fooSvc:     fooSvc,
		errHandler: errHandler,
	}, nil
}

// Get mimics an HTTP handler for fetching a foo resource.
func (fh *FooHandler) Get(w http.ResponseWriter, r *http.Request) {
	if err := fh.fooSvc.Fetch(); err != nil {
		fh.errHandler.Handle(r.Context(), w, fmt.Errorf("could not get foo from service: %w", err))
		return
	}
	w.WriteHeader(http.StatusOK)
}
```
{: file="resterrdemo/app/rest/handlers/foo/foo.go" .nolineno }

Here’s how to initialize and inject the error handler into the HTTP handler:

```go
var (
	ErrNotFound = errors.New("not found")
	ErrBadRequest = errors.New("bad request")
)

// Here we map possible errors that can be returned by our application's 
// logic to the JSON errors we would like to transport.
errorMap := map[error]resterr.RESTErr{
    ErrNotFound: {
        StatusCode: http.StatusNotFound,
        Message:    "The requested resource was not found",
    },
    ErrBadRequest: {
        StatusCode: http.StatusBadRequest,
        Message:    "The request was invalid",
    },
}

// Initialize foo storage, service (business) and transport error handler.

fooRepo := foorepo.NewPostgres()
fooSvc := foo.New(fooRepo)

fooErrHandler, err := resterr.NewHandler(logger, errorMap)
if err != nil {
	logger.Error("Failed to initialize foo error handler.", errAttr(err))
	os.Exit(1)
}

fooHandler, err := foohandler.NewHandler(logger, fooSvc, fooErrHandler)
if err != nil {
	logger.Error("Failed to initialize foo handler.", errAttr(err))
	os.Exit(2)
}
```

You can check the complete demo [here](https://github.com/alesr/resterrdemo).

### Step 3: Propagating Errors

The main features that enable us to handle errors this way is Go's ability to wrap errors `%w` and to use `errors.Is` and to check if a given error is contained in a chain of errors.

```go
package main

import (
	"errors"
	"fmt"
)

var errFoo = errors.New("foo error")

func main() {
	// wrap errFoo into another err:
	errBar := fmt.Errorf("could not do bar: %w", errFoo)

	// new err contain chained errors
	fmt.Println(errBar.Error())

	// errFoo is part of the chain of errors.
	fmt.Println(errors.Is(errBar, errFoo))
}

output:
	could not do bar: foo error
	true
```


For example, if we receive an error from the storage layer, we can wrap it and pass it to the service layer. By wrapping it, we retain all the original information about the error while adding context to the higher layers:

```go
var (
	ErrFooNotFound = errors.New("foo not found")
	ErrDatabaseError = errors.New("database error")
)

// FooRepo represents a storage repository for the foo resource.
type FooRepo struct{}

// Fetch retrieves a foo resource from the storage.
func (fr *FooRepo) Fetch() error {
	// Mimic some storage operation that fails.
	return fmt.Errorf("could not fetch foo from the database: %w", ErrFooNotFound)
}
```

```go
// FooService represents a service for the foo resource.
type FooService struct {
	repo FooRepo
}

// Fetch attempts to fetch the foo resource from the storage layer.
func (fs *FooService) Fetch() error {
	err := fs.repo.Fetch()
	if errors.Is(err, ErrFooNotFound) {
		return ErrFooNotFound
	}
	return fmt.Errorf("an unexpected error occurred: %w", err)
}
```

We delegate the responsibility of deciding if an error should be exposed to the client to the service layer. If the service return a mapped error, an equivalent JSON error is used. Otherwise, a 500 is returned.

## Conclusion

By wrapping errors and distributing error handling across layers, we gain flexibility in returning meaningful, actionable responses to clients while protecting internal application details. This way, the business logic decides which errors should be translated for clients, and any API can use a map to handle those translations. At the same time, the domain layer can work independently of the transport details, allowing us to swap transport layers without worrying about how data is passed over the wire.

This approach makes it easy to see what errors a service can return by checking the list of exposed errors, and we can track the errors an API can return by looking at the mappings between service errors and transport errors. Original errors are always logged before sending back the translated ones to the client. Since we might want to return well-scoped and actionable error messages to the client, we keep all the context related to the error, adding extra information using the `%w` wrap verb as needed.

In the end, the transport layer is responsible for logging and mapping errors, keeping the core service logic clean and focused.

The repository with the error handler can be found [here](https://github.com/alesr/resterr), and a demo application is available [here](https://github.com/alesr/resterrdemo).
