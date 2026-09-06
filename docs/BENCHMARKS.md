---
layout: default
title: Performance Benchmarks
nav_order: 8
description: "Honest, reproducible performance benchmarks comparing sequential and concurrent HTTP serving in Gat."
permalink: /BENCHMARKS.html
---

# Gat HTTP Server Performance & Benchmarks

This document records honest, reproducible performance benchmarks comparing the **sequential blocking HTTP server** (`http_server_handle_one`) against the **concurrent thread-per-connection HTTP server** (`http_server_handle_one_concurrent`) in Gat standard library (`std/http.gat`).

---

## 1. Architecture & Concurrency Model

### Thread-Safety & Boundary Invariants
In Gat, classes are reference-counted (ARC) and heap-allocated within thread-local contexts. To prevent data races and reference count hazards without heavy global locks, the compiler strictly forbids passing `class` references across `thread_spawn`.

The concurrent HTTP server bridges this boundary safely:
1. `tcp_accept_raw(listener) -> i64`: Accepts client connections returning the raw OS socket handle (`fd: i64`).
2. `thread_spawn(http_conn_worker, ctx)`: Passes a simple context struct containing `fd: i64` and a raw pointer to the read-only `HttpRouter`.
3. `http_conn_worker`: In the spawned worker thread, a local `TcpStream { fd: client_fd }` is constructed, parses the request, dispatches the handler, flushes the HTTP response, and safely closes the socket via its local deinitializer.
4. **Connection Capping**: `http_server_handle_one_concurrent(server, tracker, max_concurrent)` accepts an atomic tracker pointer and throttles when active connections reach `max_concurrent` (default 64) with exponential micro-backoff to prevent OS thread exhaustion.

---

## 2. Test Environment & Methodology

- **Host Machine**: Windows 11 x64 / AMD Ryzen 7 / 32GB RAM
- **Gat Compiler**: `gatc` v0.3.0 (Self-hosted Stage 3 bitwise verified)
- **Target Backend**: Windows x64 native (PE32+ Winsock2) & Linux x64 (zero libc ELF direct syscalls)
- **Load Testing Tool**: Multithreaded Python benchmarking harness ([`tools/bench_http.py`](../tools/bench_http.py)) generating real TCP client threads with HTTP/1.1 keep-close requests.
- **Load Profile**:
  - **Concurrency**: 16 concurrent client threads
  - **Requests per thread**: 25 requests
  - **Total requests**: 400 requests per run

---

## 3. Results & Before/After Comparison

### Scenario A: In-Memory Micro-Benchmark (Zero I/O Delay)
Route immediately returns `200 OK` with short body `bench-ok`.

| Metric | Sequential (`http_server_handle_one`) | Concurrent (`http_server_handle_one_concurrent`) | Improvement |
|---|---|---|---|
| **Requests / Second** | **9,179.5 req/s** | **10,297.3 req/s** | **+12.2%** |
| **Elapsed Time** | 0.044 s | 0.039 s | -11.4% |
| **Mean Latency** | 1.60 ms | 1.41 ms | -11.9% |
| **Median Latency** | 1.57 ms | 1.44 ms | -8.3% |
| **95th Percentile (p95)** | 2.13 ms | 1.63 ms | -23.5% |
| **99th Percentile (p99)** | 2.30 ms | 1.74 ms | -24.3% |
| **Errors / Drops** | 0 / 400 (0.0%) | 0 / 400 (0.0%) | 100% reliable |

### Scenario B: Realistic Workload (10ms Simulated I/O / Query Delay)
Route handler performs 10ms simulated database query or downstream I/O before returning.

Under sequential processing, head-of-line blocking forces all 16 concurrent clients to queue synchronously.

| Metric | Sequential (`http_server_handle_one`) | Concurrent (`http_server_handle_one_concurrent`) | Improvement |
|---|---|---|---|
| **Requests / Second** | **61.9 req/s** | **975.4 req/s** | **15.75x Faster (1,475%)** |
| **Elapsed Time** | 6.467 s | 0.410 s | 15.7x reduction |
| **Mean Latency** | 254.21 ms | 16.13 ms | **93.7% lower latency** |
| **Median Latency** | 260.87 ms | 15.14 ms | **94.2% lower latency** |
| **95th Percentile (p95)** | 284.49 ms | 21.15 ms | **92.6% lower latency** |
| **99th Percentile (p99)** | 291.63 ms | 21.85 ms | **92.5% lower latency** |
| **Errors / Drops** | 0 / 400 (0.0%) | 0 / 400 (0.0%) | 100% reliable |

---

## 4. Honest Framing & Engineering Context

- **Systems-Language Baseline**: This is a hand-rolled HTTP/1.1 parser and server implemented in `gat` without epoll/kqueue or IOCP event loops.
- **Thread-per-Connection Limits**: Thread-per-connection creates an OS thread per active connection. For tens of thousands of idle keep-alive connections (C10k problem), an asynchronous event-driven loop (epoll / io_uring / IOCP) is required. For microservices, REST APIs, and embedded servers with moderate concurrency (tens to hundreds of concurrent requests), Gat's concurrent thread pool handles 1,000+ req/s with low latency.
- **Safety**: 100% memory safe, zero data races on the router, no cross-thread ARC reference leaks.

---

## 5. How to Reproduce

Run the benchmark scripts locally:

```powershell
# Compile benchmarks
.\bin\gatc.exe examples/bench_server_seq_io.gat -o bin/bench_server_seq_io.exe
.\bin\gatc.exe examples/bench_server_conc_io.gat -o bin/bench_server_conc_io.exe

# Run Sequential test
powershell -ExecutionPolicy Bypass -File .\tools\run_bench_seq_io.ps1

# Run Concurrent test
powershell -ExecutionPolicy Bypass -File .\tools\run_bench_conc_io.ps1
```
