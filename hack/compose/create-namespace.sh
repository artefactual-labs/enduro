#!/usr/bin/env bash

set -eu

temporal operator namespace create --namespace "${DEFAULT_NAMESPACE:-default}" --retention 72h
