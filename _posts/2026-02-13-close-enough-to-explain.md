---
layout: post
title: "Close Enough to Explain"
date: 2026-02-13 23:00 +0200
mermaid: false
description: "Reflections on work these days: less pairing, more AI, and staying close to the code"
categories: [Software Engineering, Work, Practices]
tags: [engineering, collaboration, pair-programming, ai, llm, code-review, testing, legibility, go]
---

## The Setup

Back in the day when I used to pair program with actual friends, there was often a new cool CLI tool one of us would pull out that did exactly the job we needed. It was always a nice little thing that made work more interesting. Most of the terminal tools I know today, I discovered at work, especially from more experienced developers.

That habit was not just about tools, though. It was about shared knowledge, and about staying close to understanding.

These days feel different. Over the last two or three years, I've met fewer colleagues who are inclined to jump into pair programming sessions, and we've all drifted toward self-(in)sufficiency, one way or another. No judgment.

## The Low

AI can support understanding, but a lot of AI-driven development quietly trades it away.

You slowly lose track of what's going on, stop being sure what the code does, why it works, and how you can interact with it.

The failure modes usually cluster around a few familiar areas.

**Ownership and legibility**

PRs full of generated code that nobody fully owns, with descriptions that explain why the feature ended up in the product backlog rather than pointing you to the right place to look at the important bits. Patterns copied across services without respecting invariants. Stale docs left around like landmines, and then turned into "the knowledge base".

**Design and invariants**

The design gets flatter. Instead of constraints being explicit, they become vibes. Retries without budgets. Timeouts without reasoning. Idempotency assumed, not enforced. Surprise global state. Things work until they do not, and then nobody can tell you what the system is supposed to guarantee.

**Tests and verification**

Tests that, when they exist, assert implementation details instead of behavior. Snapshots of today’s structure, not checks for what must remain true. And when it goes wrong, you get a team that is fast at producing change but slow at explaining what changed.

## The High

But let's be honest: technology isn't inherently bad. The tools of our time, Stack Overflow then, LLMs now, are here either to enhance the quality and cadence of development or to amplify already poor engineering practices.

Yes, an LLM can write code, but it misses many nuances that make a codebase cohesive. You can get close to cohesion with enough time, learning, experimentation, and failure. It is not free, and it is not automatic.

This is even more difficult for mid-sized companies with an existing codebase, where constraints are real, context is spread out, and the system was not built with AI usage in mind.

If a company fails to remove outdated documentation from GitHub repositories before thinking about creating a knowledge base, it only reinforces the point. If data races are left in the codebase for years, it only reinforces the point.

*Human engineering, supported by tested standards and responsibility, remains the foundation for designing complex systems.*

That's the part I see companies missing in practice: how your company will succeed in the AI world is deeply connected to how human engineering has worked before AI.

AI doesn't fix engineering cultures; it scales them.

If your human engineering was solid, AI accelerates it.

If it wasn't, AI helps you ship faster into a wall.

And some of the same practices that lead companies into unnecessary trouble and inefficiency are usually those that make the company unable to understand its own issues.

![XKCD: For the Sake of Simplicity](https://imgs.xkcd.com/comics/for_the_sake_of_simplicity.png)
*[Source: XKCD #2587](https://xkcd.com/2587/)*

Conversely, the best thing about software engineering today is that it empowers the engineer to do what has always been its highest point: **to apply good taste when solving problems.**

When this new capability is adopted within a context of already solid engineering and professional practices, it becomes not only a tool for innovation but also makes software engineering more rewarding, interesting, and fun.

## The Good

If you want AI to help instead of harm, the basics still apply.

**Treat AI output as untrusted input. Require tests, require review, require invariants.**

Assume AI code is wrong until it proves otherwise. Make it pass the same bar as any human change: tests that cover behavior, review that checks design and edge cases, and invariants (explicit rules the system must always preserve, like idempotency, authorization guarantees, ordering, monotonic counters, “no goroutine leaks,” etc.). If a change can’t be tested or reasoned about, it’s not “done,” it’s cargo shipped.

**Invest in legibility first. Kill stale docs, reduce footguns, make ownership real.**

The fastest team isn’t the one typing fastest, it’s the one where the next person can understand and safely change things. Legibility means: clear module boundaries, boring naming, consistent patterns, up-to-date docs, and code that explains itself. “Footguns” are the sharp edges that keep causing incidents (hidden global state, surprising defaults, magical config, unclear retries/timeouts). “Ownership” means someone is accountable for a domain/service and its standards, so PRs don’t become anonymous AI paste bins.

**Optimize for lead time to safe change, not for volume of code shipped.**

Measure “how quickly can we deliver a change that won’t blow up prod,” not “how many lines/PRs did we crank out.” AI can inflate output. That’s not progress if it increases review time, regressions, on-call load, or makes future changes harder. Practical signals: time from idea to production with confidence, rollback frequency, incident rate, PR cycle time, and how often changes require spelunking.

**Use AI to support understanding, not to avoid it.**

Best use-cases are the ones that make engineers more aware of the system: summarizing a subsystem, mapping call graphs, explaining a test failure, generating hypotheses, drafting a design doc, proposing test cases, translating domain rules into acceptance criteria. Worst use-cases are “write the feature for me” when nobody can explain it afterward. If AI helped, you should end up with more clarity than you started with.

---

And maybe that's why I keep thinking about pair programming, and the experiences around real collaboration and tools. Not because `ripgrep` was nice, or because we were somehow better engineers back then, but because those moments pulled us closer to the work and gave us a great time while doing it.

Someone would show up with a small tool, a small trick, a small mental model, and the problem felt clearer. The code felt touchable, and the problem solvable in a way that we built toward and actually understood.

I like that feeling. I like being close enough to the code that I can explain it without squinting. Close enough that when it breaks, I know where to look. Close enough that I can change it with intention, not by accident.

And that is why tooling that improves visibility still matters, especially in an AI-shaped workflow.

## Tool of the Day

All this, of course, brings us to the Tool of the Day =]

[Lacune](https://github.com/alesr/lacune) (French for "gap") is a small TUI app for accessing test coverage in Go code. It's not a replacement for VSCode inline coverage, but if you don't have it in your editor, as I don't, you might make good use of Lacune.

![Lacune GIF](/assets/img/lacune.gif)

Go check it out, and if you find it useful, add it to your toolbelt and contribute to the project!

Peace.

---
