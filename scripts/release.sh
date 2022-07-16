#!/bin/bash

ROOT_DIR=$(pwd)

source "$ROOT_DIR/scripts/injectEnv.sh"

echo VSC_WORKFLOW_ID $VSC_WORKFLOW_ID
echo VSC_WORLFLOW_DIR $VSC_WORLFLOW_DIR

cp -R $BUILD_DIR/ /Users/bytedance/Library/Application\ Support$VSC_WORLFLOW_DIR

cd /Users/bytedance/Library/Application\ Support$VSC_WORLFLOW_DIR
pwd
yarn --production
