---
layout: post
title: "Go Templator"
date: 2025-01-30 09:30 +0300
mermaid: true
description: "Getting compile-time safety with Go templates"
categories: [Go, Templating]
tags: [go, golang, templating, html/template, compile-time, safety]
---

## Introduction

Recently I've been working on a project that required me to build a bunch of dashboards using Go templates. I've been using Go for a while now and I've always been a fan of the compile-time safety that it provides. However, when it comes to templates, it's a different story. The Go templating engine is very powerful, but it's also very dynamic, which means that you can easily make mistakes that won't be caught until runtime. This got me thinking: is there a way to get compile-time safety with Go templates?

## The Problem

Let's have a look at this code:

```go
package main

import (
    "bytes"
    "fmt"
    "html/template"
    "os"
)

var tmpl = "Foo value is: '{{ .Foo }}'"

func main() {
    t, err := template.New("example").Parse(tmpl)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing template: %s", err)
        os.Exit(1)
    }

    // We're passing a struct with a field named Bar,
    // but the template is expecting a field named Foo
    data := struct {
        Bar string
    }{
        Bar: "Bar",
    }

    var out bytes.Buffer
    _ = t.Execute(&out, data)

    fmt.Println(out.String())
}
```

What would you expect to happen when you run this code? I give you a hint: it won't be a compile-time error.

The code will compile just fine, but when you run it, you'll get an empty string as output. More precisely, the output will be

```shell
Foo value is: '
```

*Note the missing quotes at the end.*

You can run it yourself [here](https://go.dev/play/p/cuhE4gY0ZgN).

Why that? Because the template is expecting a field named `Foo`, but we're passing a struct with a field named `Bar`.

Now, what if we modify the code to check the error returned by `t.Execute`?

```go
package main

import (
    "bytes"
    "fmt"
    "html/template"
    "os"
)

var tmpl = "Foo value is: '{{ .Foo }}'"

func main() {
    t, err := template.New("example").Parse(tmpl)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing template: %s", err)
        os.Exit(1)
    }

    // Same as before
    data := struct {
        Bar string
    }{
        Bar: "Bar",
    }

    var out bytes.Buffer

    // But now we're checking the error
    if err := t.Execute(&out, data); err != nil {
        fmt.Fprintf(os.Stderr, "Error executing template: %s", err)
        os.Exit(2)
    }

    fmt.Println(out.String())
}
```

You guessed it! The output will be:

```shell
Error executing template: template: example:1:18: executing "example" at <.Foo>: can't evaluate field Foo in type struct { Bar string }exit status 1
```

Again, you can check it yourself at: [Go Playground](https://go.dev/play/p/Z9bsvFovRWf).

So, we get a runtime error and some unexpected value injected in the template.

This is not ideal, especially when working on a large project with many templates expecting different data structures.

## The Solution

What I want is to get a compile-time error whenever I try to pass a data structure that doesn't match the template.

One way to this, is to when initializing the template we pass (or register) the data structure that we want to bind to the template. This way, when we execute the template, the Go compiler will check if the data structure matches the one we passed when initializing the template.

Note: Since templates are quite powerful, we don't want to have an interface on top that would get in our way of the expressiveness of the template. So my goal is to a lightweight implementation.

## Baking the Cake

Disclaimer: *The cake is a lie*.

This implementation is driven by a problem that I wanted to have fun solving. One, because initially I wanted to reduce the boilerplate code that I had to write when managing my HTML templates, two because I thought compile-time safety would be a nice feature to have during the development of my projects, and three, because I wanted to learn and refresh my knowledge about templates, code generation, and generics in Go. You can find the full code in the [go-templator](https://github.com/alesr/templator) repository.

### The Registry

The plan is to have some sort of registry where I can assign templates with their respective data structures. This registry will receive the `struct` I want to use with the template and a file system where the HTML template is stored.

This registry should allow me then to access a function (I call it handler) that enables me to execute the template enforcing the data structure that I've registered.

Here's how it looks like:

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/alesr/templator"
)

// Define your template data
type HomeData struct {
    Title   string
    Content string
}

func main() {
    // Use the filesystem of your choice
    fs := os.DirFS(".")

    // Initialize registry with your data type
    reg, _ := templator.NewRegistry[HomeData](fs)

    // Get type-safe handler for home template
    home, _ := reg.GetHome()

    // Execute template with proper data
    _ = home.Execute(context.TODO(), os.Stdout, HomeData{
        Title:   "Welcome",
        Content: "Hello, World!",
    })
}
```

Again, the `templator.NewRegistry` function will return a registry that will allow to access the handler for the template that we've registered.

With the registry in hand, we can now get the handler for the template we want to execute, in this case, the `home` template.
This handler will enforce the data structure that we've registered.

If we try to pass a different data structure, we'll get a compile-time error.

```go
// Define different data types for different templates

type HomeData struct {
    Title    string
    Content  string
}

type AboutData struct {
    Company  string
    Year     int
}

// Create registries for the different template types
homeReg := templator.NewRegistry[HomeData](fs)
aboutReg := templator.NewRegistry[AboutData](fs)

// Get handlers
home, _ := homeReg.GetHome()
about, _ := aboutReg.GetAbout()

// Type safety enforced at compile time
home.Execute(ctx, w, HomeData{...})  // Compiles

// Tries to pass the AboutData struct to the home template
home.Execute(ctx, w, AboutData{...}) // Compile error
```

### HTML Template Validation

Additionally, the registry allows passing an optional function to validate if the HTML template uses only the fields of the data structure that we've registered.

```go
type ArticleData struct {
    Title    string
    Content  string    
}

// Enable validation during registry creation
reg := templator.NewRegistry[ArticleData](
    fs,
    templator.WithFieldValidation(ArticleData{}),
)

// Example templates:

// valid.html:
// <h1>{{.Title}}</h1>           // OK - Title exists in ArticleData
// <p>{{.Content}}</p>           // OK - Content exists in ArticleData

// invalid.html:
// <h1>{{.Author}}</h1>          // Error - Author field doesn't exist
// <p>{{.PublishedAt}}</p>       // Error - PublishedAt field doesn't exist

// Using the templates:
handler, err := reg.Get("valid")    // OK - all fields exist
if err != nil {
    log.Fatal(err)
}

handler, err := reg.Get("invalid")  // Error: "template 'invalid' validation error: Author - field 'Author' not found in type ArticleData"

// The validation error provides:
// - Template name
// - Invalid field path
// - Detailed error message
if validErr, ok := err.(*templator.ValidationError); ok {
    fmt.Printf("Template: %s\n", validErr.TemplateName)
    fmt.Printf("Invalid field: %s\n", validErr.FieldPath)
    fmt.Printf("Error: %v\n", validErr.Err)
}
```

### Template Functions

As we could normally do with the `html/template` package, we can also pass custom functions to the registry.

```go
// Define your custom functions
funcMap := template.FuncMap{
    "upper": strings.ToUpper,
    "lower": strings.ToLower,
}

// Create registry with functions
reg, err := templator.NewRegistry[PageData](fs, templator.WithTemplateFuncs[PageData](funcMap))

// Use functions in your templates:
// <h1>{{ .Title | upper }}</h1>
```

### Other Features of the Registry

- **Lazy loading**. Templates are loaded only when requested.
- **Templates are parsed once and cached in the registry**. Subsequent requests use the cache which improves performance.
- **Safe for concurrent use**. We can get and execute templates concurrently.

### Code Generation

As we registered the templates with their respective data structures, we want the register to give us those nice handlers so we can interact with the templates. This is where code generation comes in.

```zsh
templates/
├── home.html           -> reg.GetHome()
├── about.html          -> reg.GetAbout()
└── components/
    └── header.html     -> reg.GetComponentsHeader()

# Generate methods
go generate ./...
```

The `go generate` command will automatically discover all the HTML files and generate the methods for the registry.

## Conclusion

This has been a fun project to work on, and it has been meaningful to me because it's actually solving a problem that I've encountered in my day-to-day work. I don't expect this to be a silver bullet for everyone, but I hope it can be useful for some people, of if anything, that this reading has been interesting show how we can achieve compile-time safety with Go templates.

Please feel free to check the full code in the [go-templator](https://github.com/alesr/templator) repository, and let me know what you think. I'm always open to feedback and contributions.
