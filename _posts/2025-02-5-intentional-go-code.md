---
layout: post
title: "Pointer Initialization: Semantics Matter"
date: 2025-02-5 11:30 +0300
mermaid: true
description: "Writing Intentional Go Code Through Mindful Pointer Initialization"
categories: [Best Practices, Go, Escape Analysis, Code Quality]
tags: [go, golang, escape analysis, code quality, code readability, variable initialization, programming practices]
---

## Introduction

I've been working with Go for a while now, and during this time, I've helped onboard new colleagues to the language and worked at companies that were migrating from other languages to Go. Whenever someone starts learning a new language, we usually focus on the syntax and on getting things working (hopefully without bugs), especially if we're doing this at work. I think this is a natural way to get started, and there's absolutely nothing wrong with it. However, as we get more comfortable with the language, we start to realize that there's more to it than just committing code. We start to understand the semantics of the language and how to write idiomatic code. This is what I call writing intentional code.

Writing idiomatic code is not about being picky about small details, but about writing your code in a way that is quickly identifiable by other developers regardless of their background.

Quoting a good friend and mentor: 

> *Writing good code is about read speed.*

> *Writing good code is about scaling yourself over people over time.*

## Deliberate Abstractions

Recently Go started to receive more quality-of-life improvements like the [sort package](https://pkg.go.dev/sort), [range over function types](https://go.dev/blog/range-functions), soon [test context](https://pkg.go.dev/testing@master#T.Context), and the list goes on... But despite all these improvements, and being a language with a small set of keywords, Go can still be a bit tricky to master. If you have been helping colleagues coming from other languages, you've probably heard complaints about the lack of direction when structuring a project, the verbosity of error handling, or seen new colleagues overusing generics, empty interfaces, goroutines everywhere, or even solutions using the `reflect` package (["reflection is not for you"](https://youtu.be/PAAkCSZUG1c?si=O1vqib7jheETfTBO&t=922)) only because the developer hasn't gotten their head around the type system yet.

Among all those things, I find escape analysis one of the most interesting topics to discuss and a frequent source of confusion for many of us. In this post, I want to focus especially on a particular aspect of escape analysis that is often overlooked but can have a significant impact on how the code is understood.

Let's say we have these two functions:

```go
package escapeanalysis

type Foo struct{ val string }

func foo1() *Foo {
    f := &Foo{"foo2"}
    return f
}

func foo2() *Foo {
    f := Foo{"foo1"}
    return &f
}
```

Right away, I can tell you that these cases `foo1` and `foo2` allocate memory *more or less* in the same way. Both are escaping the value of `f` to the heap.

To verify this, we can use the `-gcflags="-m"` flag to see the escape analysis in action:

```go
package escapeanalysis

import "testing"

func TestFoo1(t *testing.T) {
	_ = foo1()
}

func TestFoo2(t *testing.T) {
	_ = foo2()
}
```

```shell
escapeanalysis ❱ go test -gcflags="-m -l" -run TestFoo1
# github.com/alesr/escapeanalysis [github.com/alesr/escapeanalysis.test]
./escapeanalysis.go:6:7: &Foo{...} escapes to heap # <- focus on this line
./escapeanalysis_test.go:5:15: t does not escape
# github.com/alesr/escapeanalysis.test
_testmain.go:43:42: testdeps.TestDeps{} escapes to heap
PASS
ok      github.com/alesr/escapeanalysis 0.204s
```

```shell
escapeanalysis ❱ go test -gcflags="-m -l" -run TestFoo2
# github.com/alesr/escapeanalysis [github.com/alesr/escapeanalysis.test]
./escapeanalysis.go:11:2: moved to heap: ff # <- focus on this line
./escapeanalysis_test.go:9:15: t does not escape
# github.com/alesr/escapeanalysis.test
_testmain.go:43:42: testdeps.TestDeps{} escapes to heap
PASS
ok      github.com/alesr/escapeanalysis 0.547s
```

While it's important to know that both functions are escaping their values to the heap, we're not going to discuss why in this post, but rather the semantics of the code and how it can affect readability.

## Syntax vs Semantics

```go
func foo1() *Foo {
    f := &Foo{"foo2"}
    return f
}

func foo2() *Foo {
    f := Foo{"foo1"}
    return &f
}
```

In both functions, we are returning a pointer to a `Foo` struct. However, the way we're initializing the `f` variable is different. In `foo1`, we're initializing `f` with a pointer to a `Foo` struct, and in `foo2`, we're initializing `f` with a `Foo` struct.

The difference between these two functions is not only in the syntax but in the semantics. If you're feeling confused, think about it this way: syntax is how we write the code, and semantics is the meaning of the code.

In the case of `foo1`, we are binding together how the variable is being initialized with the way it's being used in the code. This means that we are assuming the caller of the function is going to use the value of `f` as a pointer to a `Foo` struct. In some cases, this can be exactly what you want to do:

```go
// Good - & groups with allocation
func NewFoo() *Foo {
    return &Foo{val: "foo"}
}
```

The problem with `foo1` is that separating the initialization of a variable from its usage is a crucial practice in programming, as it enhances code readability and communicates intent more clearly to the reader.

It's not uncommon to see code like this:

```go
func foo1() *Foo {
    f := &Foo{"foo1"}
    // line
    // line
    // line
    // many lines of code have passed since the initialization of f
    // so many things have happened along the way
    // this is real life code, you know how it can be
    // line
    // bar(f) <- possibly modifying f
    // quz(*f) <- we are dereferencing f just because of the way it was initialized
    // line
    // line
    return f
}
```

Imagine you're reading real production code that wasn't written just yesterday. When you get to the last line, do you remember that `f` was initialized as a pointer? Yes, I know your IDE can help you with that, but what if you're on the metro, on your phone doing some code review? Of course, you're going to get back to it on your big screen later, but you know, you're curious, you want to know what's going on.

Now, let's see the `foo2` function:

```go
func foo2() *Foo {
    f := Foo{"foo1"}
    // line
    // line
    // line
    // many lines of code have passed since the initialization of f
    // ...
    return &f
}
```

Yep. The code probably works the same way as `foo1`, but the way it's written makes it easier to understand. When `f` is declared, the reader knows just that—`f` is a `Foo` struct. The reader doesn't have to remember that `f` is a pointer to a `Foo` struct, or that it must be dereferenced if they want to prevent an accidental modification of the value of `f` because of the way it was initialized.

## Conclusion

This is kind of a small detail that is often overlooked by many developers, especially when they're starting with Go. But as you get more comfortable with the language, you start to realize that writing idiomatic code is not about being picky about small details, but about writing your code in a way that is quickly identifiable and aligns with the idea that separation of concerns and being intentional about the way you write your code is important.

I hope this was an interesting read for you and that you can take something from it.
