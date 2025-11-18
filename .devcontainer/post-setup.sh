#!/bin/bash
set -e

# Activate mise environment
eval "$(mise activate bash)"

# Setup kubectl config
mkdir -p ~/.kube
kind get kubeconfig --name kind > ~/.kube/config 2>/dev/null || true
