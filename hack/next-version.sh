#!/usr/bin/env bash
# Copyright The Enterprise Contract Contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: Apache-2.0

# TODO: Add usage

set -o errexit
set -o pipefail
set -o nounset

function debug() {
    >&2 echo "DEBUG: ${1}"
}

# E.g. api/v0.1.33
latest_version="$(git tag | sort -V -r | head -n 1)"
debug "Latest version: ${latest_version}"

latest_patch_version="$(echo -n ${latest_version} | cut -d. -f3)"
debug "Latest patch version: ${latest_patch_version}"

version_prefix="$(echo -n ${latest_version} | cut -d. -f1-2)"
debug "Version prefix: ${version_prefix}"

next_patch_version="$((${latest_patch_version}+1))"
debug "Next patch version: ${latest_patch_version}"

next_version="${version_prefix}.${next_patch_version}"
debug "Next version is: ${next_version}"

echo -n "${next_version}"
