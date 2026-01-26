#!/usr/bin/env python3
"""List all services with their URLs.

Usage: space run services
"""

import json
import sys

def main():
    # Read context from stdin
    context = json.load(sys.stdin)

    print(f"Project: {context['project_name']}")
    print(f"Hash: {context['hash']}")
    print(f"Domain: {context['base_domain']}")
    print()

    services = context.get('services', {})
    if not services:
        print("No services configured.")
        return

    print("Services:")
    for name, svc in sorted(services.items()):
        print(f"  {name}:")
        print(f"    DNS:  {svc['dns_name']}")
        print(f"    Port: {svc['internal_port']}")
        print(f"    URL:  {svc['url']}")
        print()

if __name__ == "__main__":
    main()
