#!/bin/bash

export VSC_WORKFLOW_ID="1926E4D2-CCE6-4356-A444-5C827ACCBD9D"

export ROOT_DIR=$(pwd)
export BUILD_DIR="$ROOT_DIR/build"

injectEnv() {
  OLD_IFS=$IFS

  # 使用换行符作为 for...in... 的空白符
  IFS=$'\n'

  for line in `cat $ROOT_DIR/$1`
  do 
    if [[ ! $line =~ ^(\#+).* ]]
    then
      export $line
    fi
  done

  IFS=$OLD_IFS
}

if [ ! -f "$ROOT_DIR/.release.env" ]; then
  echo "Can not find .release.env, use default config"
else
  injectEnv .release.env
fi

export VSC_WORLFLOW_DIR="/Alfred/Alfred.alfredpreferences/workflows/user.workflow.$VSC_WORKFLOW_ID"

