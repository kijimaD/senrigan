#!/bin/bash

set -eu

cd $(dirname $0);
cd ../..

mkdir -p ./lib/generated

docker run --rm \
       -v $PWD:/workdir \
       -w /workdir \
       -u "$(id -u):$(id -g)" \
       redocly/cli \
       bundle oas/openapi.yml -o oas/output/openapi.gen.yml

docker run --rm \
       -v $PWD:/workdir \
       -w /workdir \
       -u "$(id -u):$(id -g)" \
       -e XDG_CACHE_HOME=/tmp/.cache \
       golang:1.24-bookworm \
       bash -c "go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 && \
         oapi-codegen --config oas/config/models.yml oas/output/openapi.gen.yml && \
         oapi-codegen --config oas/config/server.yml oas/output/openapi.gen.yml && \
         oapi-codegen --config oas/config/spec.yml oas/output/openapi.gen.yml"
