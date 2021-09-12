#!/bin/bash

docker build -t togettoyou/jaeger-example:latest -f Dockerfile .
docker build -t togettoyou/jaeger-example:opentelemetry -f Dockerfile.opentelemetry .
#docker push togettoyou/jaeger-example:latest
#docker push togettoyou/jaeger-example:opentelemetry