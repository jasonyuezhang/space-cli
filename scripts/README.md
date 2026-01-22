# Space CLI Setup Scripts

## DNS Setup Script

### `setup-dns.sh`

Automates the macOS DNS resolver configuration for `*.space.local` domains.

#### What it does:
1. Creates `/etc/resolver` directory (if it doesn't exist)
2. Configures macOS to resolve `*.space.local` queries to the Space CLI DNS server
3. Points DNS queries to `127.0.0.1:5353` (Space CLI embedded DNS server)

#### Usage:

```bash
# Run the setup script (will prompt for sudo password)
./scripts/setup-dns.sh
```

#### What you'll see:

```
=========================================
  Space CLI DNS Resolver Setup
=========================================

This script will configure macOS to resolve *.space.local domains
to the Space CLI embedded DNS server running on localhost:5353.

Note: This requires sudo access to modify /etc/resolver/

Creating /etc/resolver directory...
Password: [enter your password]
Creating DNS resolver configuration for space.local...
✓ DNS resolver configured successfully!

Configuration:
  nameserver 127.0.0.1
  port 5353

Setup complete!

Now you can run 'space up' and access your services at:
  • http://postgres.space.local:5432
  • http://app.space.local:3000
  • etc.
```

#### After setup:

1. The DNS resolver is configured permanently (survives reboots)
2. `space up` will no longer require sudo (DNS server starts without sudo)
3. No host port bindings needed - no more port conflicts!
4. Access services at clean URLs like `postgres.space.local:5432`

#### Troubleshooting:

If you get permission errors:
```bash
# Make sure the script is executable
chmod +x scripts/setup-dns.sh
```

To verify the resolver is configured:
```bash
cat /etc/resolver/space.local
```

To test DNS resolution:
```bash
# After running 'space up'
dig @127.0.0.1 -p 5353 postgres.space.local
```

#### Uninstall:

To remove the DNS resolver configuration:
```bash
sudo rm /etc/resolver/space.local
```
