[![Release](https://github.com/kneu-messenger-pigeon/healthcheck-pinger/actions/workflows/release.yaml/badge.svg)](https://github.com/kneu-messenger-pigeon/healthcheck-pinger/actions/workflows/release.yaml)
[![codecov](https://codecov.io/github/kneu-messenger-pigeon/healthcheck-pinger/branch/main/graph/badge.svg?token=yMS8HoUIPK)](https://codecov.io/github/kneu-messenger-pigeon/healthcheck-pinger)

### Run
```shell
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e COMPOSE_PROJECT_NAME=pigeon-app \
  -e HEALTHCHECK_PING_URL=https://hc-ping.com/cd205764-3489-43d2-9d26-41441830f67d \
  -e INTERVAL=10 \
  ghcr.io/kneu-messenger-pigeon/healthcheck-pinger:main
```

### Docker compose example
```yaml
services:
  healthcheck-ping:
    image: ghcr.io/kneu-messenger-pigeon/healthcheck-pinger:main
    restart: always
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - COMPOSE_PROJECT_NAME
      - HEALTHCHECK_PING_URL=https://hc-ping.com/cd205764-3489-43d2-9d26-41441830f67d
      - INTERVAL=90
```
