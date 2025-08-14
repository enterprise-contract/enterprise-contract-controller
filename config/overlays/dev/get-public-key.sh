#!/bin/bash

# Get the public key from tekton-chains namespace and write to a file
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
kubectl get secret signing-secrets -n tekton-chains -o jsonpath='{.data.cosign\.pub}' | base64 --decode > "${SCRIPT_DIR}/cosign.pub" 