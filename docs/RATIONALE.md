# space-cli: Rationale and Comparison

## The Problem

When using git worktrees for parallel development, you often have:

```
/project/main/          # main branch
/project/_wt1/          # feature-A worktree
/project/_wt2/          # feature-B worktree
```

All directories share the **same** `docker-compose.yml` with services like:
- postgres on port 5432
- api-server on port 6060
- app on port 3000

**Running `docker compose up` in multiple worktrees causes:**

| Problem | What Happens |
|---------|--------------|
| Port conflicts | Second `up` fails: "port 5432 already in use" |
| Project name collision | Containers get recreated/overwritten |
| DNS collision | Multiple `postgres.*.orb.local` - which one? |

## Existing Solutions

### OrbStack Alone

- ✅ Provides `*.orb.local` DNS automatically
- ❌ Does not solve port conflicts
- ❌ No automatic project isolation

### DDEV

- ✅ Supports git worktrees (omit project name, uses directory)
- ✅ Automatic port allocation (v1.23.5+)
- ✅ Router-based DNS (`*.ddev.site`)
- ✅ Mature, battle-tested, large community
- ⚠️ Requires `ddev config` initialization per worktree
- ⚠️ Uses separate config (`.ddev/config.yaml`), not your docker-compose.yml
- ⚠️ Heavier overhead (router, extra containers)
- ⚠️ Originally CMS-focused (Drupal, WordPress)

### Lando / Docksal

- ✅ Docker-based local development
- ⚠️ Framework/CMS focused
- ⚠️ Heavier abstraction layers
- ❌ Not specifically designed for worktree workflows

### Manual Approach

```bash
# Set unique project name per directory
COMPOSE_PROJECT_NAME=project-$(basename $PWD) docker compose up
```

- ✅ Fixes container name conflicts
- ❌ Still have port conflicts
- ❌ Need to modify docker-compose.yml or use overrides
- ❌ DNS not automatic

### Traefik + DNSMasq

- ✅ Flexible routing
- ⚠️ Complex setup
- ❌ Still need unique ports internally
- ❌ Manual configuration per project

## What space-cli Does

```bash
# Worktree 1
cd /project/main
space up
# → postgres-a1b2c3.space.local:5432
# → api-server-a1b2c3.space.local:6060

# Worktree 2 (same docker-compose.yml, zero changes)
cd /project/_wt1
space up
# → postgres-d4e5f6.space.local:5432
# → api-server-d4e5f6.space.local:6060
```

**Key features:**
1. Auto-generates unique project names from directory hash
2. Removes host port bindings (no conflicts)
3. Provides DNS with hash-based isolation (`*.space.local`)
4. Works with existing `docker-compose.yml` (no separate config)
5. Detects provider (OrbStack, Docker Desktop)
6. External script hooks (any language)

## Comparison Matrix

| Feature | space-cli | DDEV | OrbStack | Manual |
|---------|-----------|------|----------|--------|
| Zero config for worktrees | ✅ | ⚠️ needs init | ❌ | ❌ |
| Uses existing docker-compose.yml | ✅ | ❌ separate config | ✅ | ✅ |
| No port conflicts | ✅ | ✅ | ❌ | ❌ |
| Automatic DNS isolation | ✅ | ✅ | ⚠️ conflicts | ❌ |
| Lightweight | ✅ | ❌ | ✅ | ✅ |
| Mature/battle-tested | ❌ | ✅ | ✅ | ✅ |
| Community support | ❌ | ✅ | ✅ | N/A |

## When to Use What

### Use DDEV if:
- You want mature, battle-tested tooling
- You don't mind the extra configuration overhead
- You're working with CMS/PHP projects
- Community support is important to you

### Use space-cli if:
- You want zero changes to existing docker-compose.yml
- You prefer lightweight tooling
- You need simple worktree isolation without learning new config formats
- You're already using OrbStack and want to enhance it

### Use OrbStack alone if:
- You only work on one branch at a time
- Port conflicts aren't an issue for your workflow

## Conclusion

**Is space-cli reinventing the wheel?**

Partially. DDEV solves the same core problem and is more mature. However, space-cli fills a specific niche:

> **Zero-config worktree isolation for existing docker-compose.yml workflows**

The key differentiator is that space-cli works with your existing `docker-compose.yml` without requiring:
- A separate configuration format
- Initialization commands per worktree
- Additional abstractions

Whether this trade-off is worth using a less mature tool depends on your workflow preferences.

## References

- [DDEV Config Options](https://docs.ddev.com/en/stable/users/configuration/config/)
- [DDEV Auto Port Assignment](https://ddev.com/blog/release-v1235-auto-port-assignment/)
- [DDEV + Git Worktree (Florida DrupalCamp)](https://www.fldrupal.camp/session/use-git-worktree-ddev-run-multiple-versions-same-site)
- [DDEV Multiple Projects Discussion](https://github.com/orgs/ddev/discussions/6769)
- [Local Docker Development DNS (ldddns)](https://ldddns.arnested.dk/)
- [dnsdock](https://github.com/aacebedo/dnsdock)
