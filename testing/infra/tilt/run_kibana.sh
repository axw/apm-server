#!/usr/bin/bash
set -xe

# Run this script from the kibana directory.

SCRIPTDIR=$(dirname "$0")
SCRIPTDIR=$(cd "$SCRIPTDIR" && pwd)

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm

kubectl get --template '{{index .data "kibana.yml"}}' secret/kibana-kb-config | base64 -d > $SCRIPTDIR/kibana.yml
sed -i 's/elasticsearch-es-http.default.svc/localhost/' $SCRIPTDIR/kibana.yml
sed -i '/certificateAuthorities/d' $SCRIPTDIR/kibana.yml

nvm use
yarn start -c $SCRIPTDIR/kibana.yml --no-dev-credentials --no-base-path
