#!/usr/bin/env bash
set -uexo pipefail

source ./script/common.bash
jenkins_setup

make update-beats
COMMIT_MESSAGE="Update to elastic/beats@$(go list -m -f {{.Version}} github.com/elastic/beats/... | cut -d- -f3)"

git config user.email
git checkout -b "update-beats-$(date "+%Y%m%d%H%M%S")"
git add go.mod go.sum NOTICE.txt testing/environments
git commit -m "$COMMIT_MESSAGE"
git --no-pager log -1
