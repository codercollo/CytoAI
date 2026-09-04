# Deploying Cyto AI to Contabo — one-time setup

Your VPS: `169.58.236.118` (from the Contabo panel screenshot).
The UltraVNC connection error is unrelated — that's a remote-desktop GUI
tool, not needed for this. Everything below uses SSH, which GitHub Actions
also uses.

## 1. One-time server prep (SSH in once, manually)

```bash
ssh root@169.58.236.118

# Install Docker + Compose plugin
curl -fsSL https://get.docker.com | sh

# Clone your repo
mkdir -p /opt/cytoai
git clone https://github.com/OWNER/REPO.git /opt/cytoai
cd /opt/cytoai

# Create the real .env from the template, then EDIT the password
cp build/.env.example .env
nano .env   # set a strong POSTGRES_PASSWORD

# Open firewall (Contabo panel → Firewall, or ufw on the box) for 80/443/22
```

Replace the placeholder Caddyfile with the production one:

```bash
cp build/Caddyfile.prod build/Caddyfile
```

Edit `build/Caddyfile` later once you point a domain at the VPS (see the
comments inside it) — plain IP + HTTP is fine to get running today.

## 2. Create a GitHub PAT for the server to pull images

GHCR images are private by default. The VPS needs its own token to pull them
(the `GITHUB_TOKEN` used to _push_ during CI can't be used remotely).

1. GitHub → Settings → Developer settings → Personal access tokens →
   Tokens (classic) → Generate new token → scope: `read:packages` only.
2. Copy it — you'll paste it into a GitHub secret below, not onto the server.

## 3. Add these secrets to your GitHub repo

Repo → Settings → Secrets and variables → Actions → New repository secret:

| Secret            | Value                                                                                   |
| ----------------- | --------------------------------------------------------------------------------------- |
| `CONTABO_HOST`    | `169.58.236.118`                                                                        |
| `CONTABO_USER`    | `root`                                                                                  |
| `CONTABO_SSH_KEY` | the **private** key matching a public key already in the VPS's `~/.ssh/authorized_keys` |
| `GHCR_USERNAME`   | your GitHub username                                                                    |
| `GHCR_PAT`        | the token from step 2                                                                   |

If the VPS doesn't have your SSH key yet:

```bash
ssh-copy-id -i ~/.ssh/id_ed25519.pub root@169.58.236.118
```

## 4. Push to `main`

That's it — `.github/workflows/deploy.yml` builds both images, pushes them
to `ghcr.io`, SSHes in, and `scripts/deploy_contabo.sh` pulls + restarts the
stack + runs migrations.

## Known gap to check before first real deploy

`scripts/deploy_contabo.sh` calls `/app/cytoai migrate up` as its first
attempt, falling back to `scripts/migrate.sh` — I don't have `main.go`'s
actual CLI flags, so **confirm which one your binary really supports** and
delete the wrong branch so a real failure isn't silently swallowed by the
fallback.
