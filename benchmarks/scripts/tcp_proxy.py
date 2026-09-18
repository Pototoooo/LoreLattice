#!/usr/bin/env python3
"""Expose a Docker-only TCP service on loopback for integration tests."""

import argparse
import selectors
import socket
import threading


def relay(left: socket.socket, right: socket.socket) -> None:
    selector = selectors.DefaultSelector()
    selector.register(left, selectors.EVENT_READ, right)
    selector.register(right, selectors.EVENT_READ, left)
    try:
        while True:
            for key, _ in selector.select(timeout=60):
                data = key.fileobj.recv(65536)
                if not data:
                    return
                key.data.sendall(data)
    finally:
        selector.close()
        left.close()
        right.close()


def serve(listen_host: str, listen_port: int, target_host: str, target_port: int) -> None:
    family = socket.AF_INET6 if ":" in listen_host else socket.AF_INET
    with socket.create_server((listen_host, listen_port), family=family) as listener:
        print(
            f"PROXY_READY {listen_host}:{listen_port} -> {target_host}:{target_port}",
            flush=True,
        )
        while True:
            client, _ = listener.accept()
            upstream = socket.create_connection((target_host, target_port), timeout=10)
            threading.Thread(target=relay, args=(client, upstream), daemon=True).start()


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--listen-host", default="127.0.0.1")
    parser.add_argument("--listen-port", type=int, default=50051)
    parser.add_argument("--target-host", required=True)
    parser.add_argument("--target-port", type=int, default=50051)
    args = parser.parse_args()
    serve(args.listen_host, args.listen_port, args.target_host, args.target_port)
