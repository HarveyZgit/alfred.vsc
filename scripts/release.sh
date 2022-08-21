#!/bin/bash

ROOT_DIR=$(pwd)

source "$ROOT_DIR/scripts/injectEnv.sh"

echo VSC_WORKFLOW_ID $VSC_WORKFLOW_ID
echo VSC_WORLFLOW_DIR $VSC_WORLFLOW_DIR


if [ "$VSC_RELEASE_MODE" = "1" ]; then
  rm -rf ~/Library/Application\ Support$VSC_WORLFLOW_DIR
fi

cp -R $BUILD_DIR/ ~/Library/Application\ Support$VSC_WORLFLOW_DIR

cd ~/Library/Application\ Support$VSC_WORLFLOW_DIR
pwd
pnpm install --prod
