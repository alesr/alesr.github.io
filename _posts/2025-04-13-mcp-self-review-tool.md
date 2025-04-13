---
layout: post
title: "Building a Claude-Powered GitHub Reviewer with Anthropic MCP"
date: 2025-04-13 17:56 +0300
mermaid: true
description: "An AI Code Review Assistant for Solo Projects"
categories: [Tooling]
tags: [code review, MCP, Anthropic, AI]
---

I just built `gh-self-reviewer` this weekend, a small Go tool connecting Claude AI to GitHub using Anthropic's Model Control Protocol (MCP). It serves two purposes: providing a second perspective on my solo project code and helping me gain practical experience with MCP.

## The Solo Developer's Dilemma

When you're the only developer on a project, you lack the benefit of team code reviews. While you can certainly catch many issues yourself, it's still valuable to have a different perspective examine your code. We all have our own blind spots and preconceptions that can affect how we evaluate our work. Having another viewpoint often reveals things we hadn't considered.

## Exploring MCP Through Practical Application

I wanted to understand MCP beyond just reading documentation. Building a real tool that solves an actual problem seemed like the perfect approach to learn the protocol.

Last year, I spent months working on a project using OpenAI's Function Calling. That approach required significant setup - designing a schema for function calling tools, and handling all the plumbing between the AI and my application. While powerful, it was complex and required maintaining multiple pieces just to enable basic AI tool text completion capabilities.

The MCP implementation was surprisingly straightforward:

```go
server := mcp.NewServer(stdio.NewStdioServerTransport())

// Register tools with the server
server.RegisterTool("list_my_pull_requests", "List my pull requests",
    func(arguments gh.PRListRequest) (*mcp.ToolResponse, error) {
        // Implementation
    })

// Start the server
if err := server.Serve(); err != nil {
    log.Printf("MCP server error: %v", err)
}
```

## GitHub PR Review Implementation

The key feature of this tool is its ability to post reviews directly to GitHub PRs:

Here's how the review functionality works:

```go
func (h *GithubToolHandler) SubmitPullRequestReview(ctx context.Context, prURLStr string, reviewBody string) (*PRReview, error) {
    // ... URL parsing logic ...

    // Create a review request
    reviewRequest := &github.PullRequestReviewRequest{
        Body:  &reviewBody,
        Event: github.String("COMMENT"), // Submit as a comment
    }

    // Submit the review
    review, _, err := h.client.PullRequests.CreateReview(ctx, owner, repo, prNumber, reviewRequest)
    if err != nil {
        return nil, fmt.Errorf("failed to submit PR review: %w", err)
    }
    
    return &PRReview{
        Body:    review.GetBody(),
        HTMLURL: review.GetHTMLURL(),
    }, nil
}
```

This function is exposed through the MCP tools interface:

```go
server.RegisterTool("review_pr", "Submit a review on a pull request",
    func(arguments gh.PRReviewSubmitRequest) (*mcp.ToolResponse, error) {
        review, err := githubToolHandler.SubmitPullRequestReview(ctx, arguments.PRURL, arguments.ReviewBody)
        // ... handle response ...
    });
```

## How It Functions as a Second Opinion

With this tool, Claude acts as my buddy code reviewer by:

1. Identifying potential edge cases I might not have thought of
2. Suggesting alternative implementations
3. Questioning design decisions from a different angle
4. Pointing out possible improvements in structure or organization
5. Recommending documentation enhancements

This happens through natural conversation:

```shell
Could you list my open GitHub pull requests?
```

And then:

```shell
Please review my PR at https://github.com/alesr/my-repo/pull/42
```

## The Tool in Action

I tested the tool on an old, forgotten PR in one of my repositories: [https://github.com/alesr/mp4srt/pull/1](https://github.com/alesr/mp4srt/pull/1). This PR had been sitting around for ages without any review.

When I asked Claude to review this PR, it:

1. Retrieved the PR content using the `get_pr_content` tool
2. Analyzed the code changes and context
3. Posted a detailed review directly on the PR using the `review_pr` tool

The review appeared directly in the GitHub PR interface, just like reviews from a human reviewer. This made it easy to see Claude's suggestions right where I needed them, alongside the code.

Having this kind of feedback directly in GitHub's interface is much more useful than just getting it in a chat window, as it allows me to address issues while looking at the actual code.

## Setup

### GitHub Token Requirements

You'll need a GitHub personal access token with the following scopes:

- `repo` - Full control of private repositories
  - This is needed to list PRs and add reviews

### Claude Desktop Configuration

This tool is designed to work with the Claude Desktop app, which supports MCP. Add the following to your Claude Desktop configuration file (typically found in `~/.config/Claude/config.json` on macOS/Linux or the equivalent Windows location):

```json
{
  "mcpServers": {
    "github_tools": {
      "command": "/path/to/gh-self-reviewer",
      "args": [],
      "env": {
        "GITHUB_TOKEN_MCP_APP_REVIEW": "your_github_token_here"
      }
    }
  }
}
```

Replace `/path/to/gh-self-reviewer` with the actual path to your compiled binary and `your_github_token_here` with your GitHub personal access token.

After adding this configuration, restart the Claude Desktop app for the changes to take effect.

## Future Improvements

My next step is to add structured review guidelines for Claude to follow. I want to implement a set of criteria that I personally find important when reviewing code:

- SOLID principles adherence
- Security considerations
- Code duplication and complexity
- Performance implications of algorithmic choices
- Idiomatic Go patterns and best practices
- Test coverage and quality
- Consistent error handling approaches

This would make the reviews more thorough and aligned with my own coding standards. It would also help ensure that even my hobby projects maintain a certain level of quality that I'd expect in professional settings.

## Conclusion

This utility is a nice additon to my hobby projects workflow by providing a second perspective while simultaneously helping me learn about MCP.

The code is [available on GitHub](https://github.com/alesr/gh-self-reviewer) if you want to try it for your own projects or learn more about MCP implementation.
