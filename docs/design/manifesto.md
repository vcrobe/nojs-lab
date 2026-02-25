# The NoJS Manifesto: A Commitment to Simplicity

Every year, teams spend thousands of engineer-hours installing toolchains, debugging bundlers, and chasing breaking changes in dependencies — before writing a single line of business logic. The JavaScript ecosystem has industrialized complexity and called it progress.

**NoJS** is an intentional rebellion against framework fatigue. We believe software should be a tool that serves the developer, not a puzzle that consumes them. We build in Go, we compile to WebAssembly, and we ship software that developers can fully understand and own.

### I. The 48-Hour Mastery

A framework should not be a career-long study. If a professional developer cannot build a data-bound, routed application with NoJS in one or two days, we have failed. We prioritize the **lowest possible learning curve** so you can spend your time building your business, not learning our abstractions. When you finish the quick guide, you should know everything you need.

### II. The Brain is the Best IDE

The most powerful tool a developer has is their mental model of the code. NoJS is built on the principle of **Memorability**. We use very few rules and even fewer abstractions. Our goal is for you to remember exactly how the framework works without ever needing to open our documentation a second time. If it's too complex to remember, it's too complex to exist.

### III. Radical Minimalism

Most frameworks grow until they are bloated with "nice-to-have" features that 90% of users never touch. **NoJS is small by design.** We do not include code just because it is trendy.

* We only include what is strictly necessary to build robust, interactive applications.
* We reject "feature creep" to ensure that the framework remains fast, secure, and understandable.

### IV. Go-First Development

We believe in the power of **Go**. Your application logic is written entirely in Go — compiled Ahead-of-Time to WebAssembly and executed natively in the browser. A thin JavaScript bootstrap layer handles the WASM runtime; everything else is Go.

* **Go as the Universal Language:** One language for backend, frontend, and tooling.
* **Type Safety:** Catch errors at compile-time, not in the user's browser.
* **AOT Compilation:** HTML templates generate type-safe Go `Render()` methods at build time, not at runtime.

### V. Performance by Architecture

WebAssembly runs at near-native speed. NoJS is designed to exploit this from the ground up with a Virtual DOM that minimises real DOM operations through diff/patch cycles, keeping UI updates fast without sacrificing correctness.

* **Virtual DOM:** In-memory tree diffing reduces expensive browser reflows.
* **AOT, not runtime parsing:** Template compilation happens once at build time, never in the user's browser.
* **No runtime overhead from dynamic interpretation.** What runs is what was compiled.

### VI. Own Your Stack

Your application should belong entirely to you.

* **No `node_modules` supply chain.** Dependencies are Go modules — auditable, versioned, and deterministic.
* **No telemetry, no SaaS lock-in, no cloud requirement.** Deploy a static file and a WASM binary. That is all.
* **Data sovereignty by default.** NoJS is built for teams and enterprises that cannot afford to depend on third-party infrastructure they do not control.

### VII. Stability Over Hype

We do not follow a "fixed release" schedule. We do not ship to satisfy a marketing calendar.

* **Quality is the only deadline.** A feature is released only when it is tested, stable, and fits our philosophy of simplicity.
* We are not competing with the giants; we are building a sanctuary for those who want software that just works.
* The framework grows deliberately — each phase fully proven before the next begins.

> **Our Promise:** We will never prioritize a new feature over the simplicity of the existing framework. We are building NoJS for the long haul — for the enterprises that value data sovereignty and the developers who value their own sanity.

---

*This philosophy is intentional and is built incrementally, one stable phase at a time.*
