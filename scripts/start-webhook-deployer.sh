#!/bin/bash
export WEBHOOK_SECRET=$(cat /home/brewgator/lightning-node-tools/secrets/webhook.secret)
exec /home/brewgator/lightning-node-tools/bin/webhook-deployer \
  --port=9000 \
  --repo=/home/brewgator/lightning-node-tools \
  --branch=main \
  --script=./scripts/auto-deploy.sh
