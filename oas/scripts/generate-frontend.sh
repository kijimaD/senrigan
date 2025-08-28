#!/bin/bash
set -eux

####################
# generate frontend
####################

cd `dirname $0`
cd ../..

docker run --rm \
       -v $PWD:/workdir \
       -w /workdir \
       -u "$(id -u):$(id -g)" \
       openapitools/openapi-generator-cli:v7.11.0 generate \
         -i oas/openapi.yml \
         -g typescript-axios \
         -o oas/output/ts-axios

mkdir -p frontend/generated
mv oas/output/ts-axios/*.ts frontend/generated/
