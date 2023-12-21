[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/token-server/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/token-server)](https://goreportcard.com/report/github.com/udhos/token-server)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/token-server.svg)](https://pkg.go.dev/github.com/udhos/token-server)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/token-server)](https://artifacthub.io/packages/search?repo=token-server)
[![Docker Pulls](https://img.shields.io/docker/pulls/udhos/token-server)](https://hub.docker.com/r/udhos/token-server)

# token-server

# Usage

## Default

```bash
token-server

curl -s localhost:8080/token | gojq
```

## Client Credentials

```bash
CLIENT_CREDENTIALS=true token-server

curl -s -H 'content-type: application/x-www-form-urlencoded' \
  -d grant_type=client_credentials \
  -d client_id=admin \
  -d client_secret=admin \
  -d audience=YOUR_API_IDENTIFIER \
  localhost:8080/token | gojq
``````
