# Phony

[![Go Report Card](https://goreportcard.com/badge/github.com/Arceliar/phony)](https://goreportcard.com/report/github.com/Arceliar/phony)
[![GoDoc](https://godoc.org/github.com/Arceliar/phony?status.svg)](https://godoc.org/github.com/Arceliar/phony)

Phony is a [Pony](https://ponylang.io/)-inspired proof-of-concept implementation of shared-memory actor-model concurrency in the Go programming language. `Actor`s automatically manage goroutines and use asynchronous causal messaging (with backpressure) for communcation. This makes it easy to write programs that are free from deadlocks, goroutine leaks, and many of the `for` loops over `select` statements that show up in boilerplate code. The down side is that the code needs to be written in an asynchronous style, which is not idiomatic to Go, so it can take some getting used to.

## Benchmarks

```
goos: linux
goarch: amd64
pkg: github.com/Arceliar/phony
BenchmarkLoopActor-4                	15617646	        71.1 ns/op	      16 B/op	       1 allocs/op
BenchmarkLoopChannel-4              	14870767	        73.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkSendActor-4                	 3268095	       377 ns/op	      32 B/op	       2 allocs/op
BenchmarkSendChannel-4              	 2598151	       442 ns/op	       0 B/op	       0 allocs/op
BenchmarkRequestResponseActor-4     	 2256913	       527 ns/op	      48 B/op	       3 allocs/op
BenchmarkRequestResponseChannel-4   	 1257068	       869 ns/op	       0 B/op	       0 allocs/op
BenchmarkBlock-4                    	  780747	      1586 ns/op	     144 B/op	       3 allocs/op
PASS
ok  	github.com/Arceliar/phony	12.677s
```

These are microbenchmarks, but they seem to indicate that `Actor` messaging and goroutine+channel operations have comparable cost. I suspect that the difference is negligible in most applications.

## Implementation Details

The code base is short, under 100 source lines of code as of writing, so reading the code is probably the best way to see *what* it does, but that doesn't necessarily explain *why* certain design decisions were made. To elaborate on a few things:

- Phony only depends on packages from the standard library:
    - `runtime` for some scheduler manipulation (through `Goexit()` and `Gosched()`).
    - `sync/atomic` to implement the `Inbox`'s message queues.
    - `unsafe` to use `atomic`'s `unsafe.Pointer` operations, which the paranoid should audit themselves for correctness.

- Attempts were make to make embedding and composition work:
    - `Actor` is an `interface` satisfied by the `Inbox` `struct`.
    - The zero value of an `Inbox` is a fully initialized and ready-to-use `Actor`
    - This means any `struct` that anonymously embeds an `Inbox` is an `Actor`
    - `struct`s that don't want to export the `Actor` interface can embed it as a field instead.

- `Inbox` was implemented with scalability in mind:
    - The `Inbox` is basically an atomically updated single-consumer multiple-producer linked list.
    - Pushing a message is wait-free -- no locks, spinlocks, or `CompareAndSwap` loops.
    - Popping messages is wait-free in the normal case, with a busy loop (`LoadPointer`) if popping the last message lost a race with a push.
    - When backpressure is required, it's implemented by sending two extra messages (one to the receiver of the original message, and one to the sender).

- The implementation aims to be as lightweight as reasonably possible:
    - On `x86_64`, an empty `Inbox` is 24 bytes, and messages overhead is 16 bytes, or half that on `x86`.
    - An `Actor` with an empty `Inbox` has no goroutine.
    - An `Actor` that has stopped due to backpressure also has no goroutine.
    - This means that idle `Actor`s can be collected as garbage when they're no longer reachable, just like any other `struct`.

