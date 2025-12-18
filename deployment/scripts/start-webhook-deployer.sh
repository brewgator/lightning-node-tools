#!/bin/bash
export WEBHOOK_SECRET=$(cat $HOME/lightning-node-tools/secrets/webhook.secret)
exec $HOME/lightning-node-tools/bin/webhook-deployer \
  --port=9000 \
  --repo=$HOME/lightning-node-tools \
  --branch=main \
  --script=./deployment/scripts/auto-deploy.sh
