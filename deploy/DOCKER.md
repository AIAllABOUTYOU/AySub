# AySub Docker Image

AySub Docker images are built from the current AySub repository source. Local source builds use `aysub:latest`; GitHub Actions publishes prebuilt images to `ghcr.io/aiallaboutyou/aysub`.

## GitHub Actions

The workflow in `.github/workflows/docker.yml` builds and pushes a multi-platform image for `linux/amd64` and `linux/arm64`.

Docker image publishing runs on:

- Pushes to `main`: publishes `ghcr.io/aiallaboutyou/aysub:latest`
- Tags matching `v*`: publishes the matching version tag
- Manual Docker `workflow_dispatch`: runs from the GitHub Actions UI

GitHub Releases are handled by `.github/workflows/release.yml`. A Release is created only when a `v*` tag is pushed, or when the `Release` workflow is run manually with an existing `v*` tag. Normal commits to `main` do not create GitHub Releases.

Create a release tag:

```bash
git tag -a v0.1.134 -m "AySub v0.1.134"
git push origin v0.1.134
```

The Release workflow uploads binary archives named `aysub_<version>_<os>_<arch>.tar.gz` plus `checksums.txt`. Those assets are used by `deploy/install.sh` and the in-app update check. Docker images are still published by the Docker workflow.

Use a published image:

```bash
docker pull ghcr.io/aiallaboutyou/aysub:latest
```

## Build

From the repository root:

```bash
docker build -t aysub:latest .
```

Or use the helper script:

```bash
./deploy/build_image.sh
```

## Docker Compose

Prebuilt GHCR image deployment:

```bash
cd deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.image.yml pull
docker compose -f docker-compose.image.yml up -d
docker compose -f docker-compose.image.yml logs -f aysub
```

Recommended local-directory deployment:

```bash
cd deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.local.yml up -d --build
docker compose -f docker-compose.local.yml logs -f aysub
```

Fixed-directory GitHub source updates:

```bash
# First-time setup on a server that already has deploy/.env and data directories:
sudo ./deploy/update-from-github.sh \
  --root /opt/AySub-current \
  --data-source /opt/AySub/deploy

# Later updates:
sudo /opt/AySub-current/deploy/update-from-github.sh
```

The script pulls `main` from GitHub, builds `aysub:latest`, recreates the Compose services, waits for container health, and smoke-tests `/health`, `/status`, `/models`, and `/playground`. It keeps `.env`, `data`, `postgres_data`, and `redis_data` in place. If the fixed deployment directory has tracked local code changes, the script stops unless `--force` is passed.

Named-volume deployment:

```bash
cd deploy
cp .env.example .env
nano .env
docker compose up -d --build
docker compose logs -f aysub
```

Standalone app-only deployment with external PostgreSQL and Redis:

```bash
cd deploy
cp .env.example .env
nano .env
docker compose -f docker-compose.standalone.yml up -d --build
docker compose -f docker-compose.standalone.yml logs -f aysub
```

## Optional Grok WARP / FlareSolverr Stack

`deploy/docker-compose.proxy-profiles.yml` provides optional proxy services for Grok Cookie accounts:

- `warp`: SOCKS5 egress proxy at `socks5://warp:1080` inside the Compose network.
- `flaresolverr`: Cloudflare clearance helper at `http://flaresolverr:8191` inside the Compose network.
- `privoxy`: optional HTTP proxy helper at `http://privoxy:8118`.

Enable FlareSolverr for automatic Cloudflare clearance refresh:

```bash
cd deploy
GROK_FLARESOLVERR_URL=http://flaresolverr:8191 \
docker compose -f docker-compose.image.yml -f docker-compose.proxy-profiles.yml --profile flaresolverr up -d
```

Enable WARP and bind it to a Grok Cookie account:

```bash
cd deploy
docker compose -f docker-compose.image.yml -f docker-compose.proxy-profiles.yml --profile warp up -d
```

Then create an active proxy in the admin proxy page:

```text
socks5://warp:1080
```

Bind that proxy to the Grok Cookie account. When both WARP and FlareSolverr are enabled, AySub passes the account proxy to FlareSolverr so the clearance cookie is solved from the same egress path used by the Grok request.

## Defaults

| Item | Default |
|------|---------|
| Local build image | `aysub:latest` |
| Prebuilt image | `ghcr.io/aiallaboutyou/aysub:latest` |
| Service | `aysub` |
| Container | `aysub` |
| PostgreSQL container | `aysub-postgres` |
| Redis container | `aysub-redis` |
| Network | `aysub-network` |
| Named data volume | `aysub_data` |

The runtime binary inside the image is `/app/aysub`.

## Links

- [GitHub Repository](https://github.com/AIAllABOUTYOU/AySub)
- [Documentation](https://github.com/AIAllABOUTYOU/AySub#readme)
