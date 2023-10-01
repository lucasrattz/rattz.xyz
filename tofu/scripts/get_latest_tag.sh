#!/bin/bash
PROJECT=$1
REGION=$2
REPOSITORY=$3
IMAGE=$4

LATEST=$(gcloud container images describe ${REGION}-docker.pkg.dev/${PROJECT}/${REPOSITORY}/${IMAGE}:latest  --format="value(image_summary.fully_qualified_digest)" | tr -d '\n')
echo "{\"image\": \"${LATEST}\"}"
