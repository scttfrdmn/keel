# Project records

These pages are not documentation for using keel. They are the project's own
records, kept for a different reader: someone auditing a number, reproducing a
measurement, working on the code, or reading keel as a field report on Go's
experimental SIMD support.

They are served from the repository's own markdown, unedited. Nothing on this
site summarises them, because a summary is a copy with nothing to keep it honest.

| record | what it is |
| --- | --- |
| [Design document](design.md) | The contract this project is built against: architecture, the phase plan and its gates, the testing philosophy, and the standing orders. |
| [Testing methodology](methodology.md) | Section 5 of the design document on its own page — the numbered rules every gate enforces, including what a measurement has to carry before it can be published. |
| [Toolchain field notes](toolchain-notes.md) | Every surprise `GOEXPERIMENT=simd` produced, each with a minimal repro and, where one was filed, the upstream issue and CL. |
| [Microkernel notes](kernel.md) | The tile shapes that ship, the register-allocation ceiling they were shaped around, and the per-host measurements behind the choice. |
| [Changelog](changelog.md) | Every user-visible change, with the reasoning that produced it. |
| [Contributing](contributing.md) | The code rules: differential tests before any vector op, one tolerance model, no numbers without their denominator. |

## Why the field notes exist

keel is built on `simd` and `archsimd`, which are experimental. Every lowering
surprise, missed optimisation and instrumentation conflict found along the way is
written down with a repro rather than worked around quietly, and several are
carried upstream. The
[toolchain field notes](toolchain-notes.md) are that record, and they have value
independent of keel.
