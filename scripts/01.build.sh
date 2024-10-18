#!/bin/bash

# Docker build
docker build --target go-dev -t seicheese:go --no-cache ./Docker && docker build --target db-dev -t seicheese:db --no-cache ./Docker && \
docker build --target admin -t seicheese:admin --no-cache ./Docker