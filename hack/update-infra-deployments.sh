#!/usr/bin/env bash
# Copyright 2022 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: Apache-2.0

# Updates a local clone of redhat-appstudio/infra-deployments to use the latest
# packages produced by this repository.
# Usage:
#   update-infra-deployments.sh <PATH_TO_INFRA_DEPLOYMENTS> [<REVISION>]

set -o errexit
set -o pipefail
set -o nounset

REVISION="${2-$(git rev-parse HEAD)}"

TARGET_DIR="${1}"
cd "${TARGET_DIR}" || exit 1

echo "Updating infra-deployments to revision ${REVISION}..."
sed -i \
  -e 's|\(https://github.com/enterprise-contract/enterprise-contract-controller/.*?ref=\)\(.*\)|\1'${REVISION}'|' \
  -e 's|\(https://raw.githubusercontent.com/enterprise-contract/enterprise-contract-controller/\)\([[:alnum:]]*\)\(.*\)|\1'${REVISION}'\3|' \
  -e 's/\(newTag: \).*/\1'${REVISION}'/' \
  components/enterprise-contract/kustomization.yaml
echo 'infra-deployments updated successfully'
