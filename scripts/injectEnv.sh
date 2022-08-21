#!/bin/bash

WORKFLOW_ID="harvey.vsc.development"

if [ "$VSC_RELEASE_MODE" = "1" ]; then
  WORKFLOW_ID="harvey.vsc.production"
fi

export VSC_WORKFLOW_ID=$WORKFLOW_ID
export VSC_WORLFLOW_DIR="/Alfred/Alfred.alfredpreferences/workflows/user.workflow.$WORKFLOW_ID"
export ROOT_DIR=$(pwd)
export BUILD_DIR="$ROOT_DIR/build"
