---
layout: post
title: "Building Docker Images: Pre-compiled Binaries and Multistage Builds"
date: 2024-06-28 20:49 +0300
mermaid: true
description: Discussing multistage and pre-compiled binaries for building Docker images.
categories: [DevOps]
tags: [ci/cd, docker, go, golang, images]
---

## Introduction

When it comes to building Docker images for Go applications (and many other languages), there are essentially two approaches you can take: building directly in the Docker image by copying the source code and using Go installed inside the image to compile the binary, or pre-compiling the binary outside the Docker image and only copying the final binary into the image.

With the introduction of multistage builds in Docker version [17.05](https://docs.docker.com/engine/release-notes/17.05/) in 2017, the process of compiling source code and generating smaller images has become significantly more efficient. However, it’s still common to see projects that either build images from scratch or rely on pre-compiled binaries without fully understanding the trade-offs.

In this post, I’ll break down the differences between these methods and discuss how multistage builds and pre-compiled binaries  can improve your workflow and the considerations to take when choosing which of these methods.

### The Old Days’ Approach: Setting Up the Entire Process

In the old days, in order to compile and a run a Go application during the Docker image build would included the sintallation of all necessary tools and dependencies to compile the source code, run it, and clean up. 

Here’s a simple example of how that looks:

```Dockerfile
FROM alpine:3.20

# Set up environment variables
ENV GO_VERSION=1.23.2 \
    GOPATH=/go \
    PATH=/go/bin:/usr/local/go/bin:$PATH

WORKDIR /app

# Install wget and download Go
RUN apk add --no-cache wget \
    && wget https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz \
    && mkdir -p "$GOPATH/src" "$GOPATH/bin" /app \
    && chmod -R 755 "$GOPATH"

# Copy project files
COPY main.go go.mod .

# Download Go modules and build the app
RUN go mod download \
    && go build -o godockerdemo .

# Clean up unnecessary files
RUN apk del wget \
    && rm -rf /usr/local/go \
    && rm -rf /go/pkg /go/src \
    && mv godockerdemo /usr/local/bin/

# Set the entry point to the binary
ENTRYPOINT ["/usr/local/bin/godockerdemo"]
```

This method requires installing tooling, copying source code, and downloading dependencies directly into the image. It often results in large images and slow build times. We also have to be careful when cleaning up unnecessary files and handling security aspects, such as ensuring sensitive files aren’t accidentally copied into the image `COPY . .` and *attack surface*.

### The Pre-compiled Binary

This method focuses on reducing image size and build time by compiling the Go application outside the Docker image, then copying only the binary. This approach is often used when developers need to run the application in Docker during development, especially when the source code is extensive. Moving only the compiled binary can be significantly faster than transferring all the source code and downloading dependencies.

Example of such Dockerfile:

```Dockerfile
FROM alpine:3.20

WORKDIR /app

COPY ../bin/godockerdemo /app/godockerdemo

RUN chmod +x /app/godockerdemo

ENTRYPOINT ["./godockerdemo"]
```

Example of extra tooling required to run the container:

```make
run: build
	docker-compose -f build/docker-compose.yml run --build -i app

.PHONY: build
build:
	GOOS=linux GOARCH=amd64 go build -o bin/godockerdemo main.go
```

This results in a small image, as we’re only including the pre-compiled binary, without the Go toolchain or dependencies. While it optimizes for smaller image sizes, it has drawbacks. By building the binary outside the container, you lose the ability to leverage Go’s automatic detection of host architecture and platform, and you may miss the chance to catch dependency issues or perform vulnerability checks on your code and dependencies during the build process.

### Multistage Builds

Although multistage builds have been available since 2017, many projects still follow old way. With multistage builds, we can compile our Go application in one stage and copy only the necessary files to a second stage, resulting into a minimal image since the final image is the one from the last stage. This method allows us to separate the environments used for compiling and running the application, reduce the attack surface, and smaller images.

Here’s an example of a multistage Dockerfile:

```Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY main.go go.mod .

RUN go mod download

RUN go build -o godockerdemo .

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/godockerdemo .

RUN chmod +x ./godockerdemo

ENTRYPOINT ["./godockerdemo"]
```

In this Dockerfile, we use the golang:1.23-alpine image to compile our Go app in the builder stage, and then copy the final binary into a clean alpine:3.20 image. This approach results in a much smaller final image compared with not using multistages, a clear separation of concerns between build and runtime, and avoid addicional tooling for building the binary outside the image.

## Conclusion

Aside from building images that compile the code from scratch, the choice between using multistage builds or pre-compiled binaries depends on the use case and how the integration and deployment processes are organized within a company. There are many ways to set up CI/CD pipelines, and the approach often depends on which aspects you want to optimize. Common goals include reducing build times and image sizes, but even these require careful consideration and sometimes more complex changes before altering how images are built.

On one hand, multistage builds allow for better delegation and control over how the binary is built, while also isolating the build and runtime environments, resulting in smaller image sizes. On the other hand, building images with pre-compiled binaries can optimize build times, particularly in projects where larger changes are difficult to implement. However, it’s important to understand the trade-offs, including the need to maintain information about the host machine and the loss of benefits like re-downloading dependencies and running vulnerability checks or static analysis on the source code.
