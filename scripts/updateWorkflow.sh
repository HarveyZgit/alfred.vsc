#!/bin/bash

ROOT_DIR=$(pwd)
PUBLIC_DIR="$ROOT_DIR/public"

source "$ROOT_DIR/scripts/injectEnv.sh"


cp ~/Library/Application\ Support/$VSC_WORLFLOW_DIR/info.plist $PUBLIC_DIR/info.plist 
