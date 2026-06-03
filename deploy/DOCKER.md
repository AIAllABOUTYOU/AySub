# AySub Docker Image

AySub Docker images are built from the current AySub repository source. The default local image tag used by Docker Compose is `aysub:latest`.

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

Recommended local-directory deployment:

```bash
cd deploy
cp .env.example .env
nano .env
mkdir -p data postgres_data redis_data
docker compose -f docker-compose.local.yml up -d --build
docker compose -f docker-compose.local.yml logs -f aysub
```

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

## Defaults

| Item | Default |
|------|---------|
| Image | `aysub:latest` |
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
