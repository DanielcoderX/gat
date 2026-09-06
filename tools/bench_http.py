import socket
import time
import threading
import sys
import statistics

def worker(host, port, n_requests, latencies, error_count):
    req = b"GET /bench HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n"
    for _ in range(n_requests):
        t0 = time.perf_counter()
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.connect((host, port))
            s.sendall(req)
            resp = b""
            while True:
                chunk = s.recv(1024)
                if not chunk:
                    break
                resp += chunk
            s.close()
            t1 = time.perf_counter()
            if b"200 OK" in resp:
                latencies.append((t1 - t0) * 1000.0) # ms
            else:
                error_count[0] += 1
        except Exception as e:
            error_count[0] += 1

def run_bench(name, port, concurrency=16, requests_per_thread=25):
    total_req = concurrency * requests_per_thread
    threads = []
    latencies = []
    errors = [0]
    
    # Warmup
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.connect(("127.0.0.1", port))
        s.sendall(b"GET /bench HTTP/1.1\r\nHost: 127.0.0.1\r\nConnection: close\r\n\r\n")
        s.recv(1024)
        s.close()
    except:
        pass

    start_time = time.perf_counter()
    for _ in range(concurrency):
        t = threading.Thread(target=worker, args=("127.0.0.1", port, requests_per_thread, latencies, errors))
        threads.append(t)
        t.start()
    for t in threads:
        t.join()
    end_time = time.perf_counter()

    elapsed = end_time - start_time
    rps = len(latencies) / elapsed if elapsed > 0 else 0
    latencies.sort()

    print(f"=== {name} ===")
    print(f"Concurrency:       {concurrency}")
    print(f"Total Requests:    {total_req}")
    print(f"Successes:         {len(latencies)}")
    print(f"Errors:            {errors[0]}")
    print(f"Elapsed Time:      {elapsed:.3f} s")
    print(f"Requests / Second: {rps:.1f} req/s")
    if latencies:
        print(f"Latency Mean:      {statistics.mean(latencies):.2f} ms")
        print(f"Latency Median:    {statistics.median(latencies):.2f} ms")
        p95 = latencies[int(len(latencies) * 0.95)]
        p99 = latencies[int(len(latencies) * 0.99)]
        print(f"Latency p95:       {p95:.2f} ms")
        print(f"Latency p99:       {p99:.2f} ms")
    print()

if __name__ == "__main__":
    server_type = sys.argv[1] # seq or conc
    port = int(sys.argv[2])
    run_bench(f"Gat HTTP Server ({server_type})", port, concurrency=16, requests_per_thread=25)
