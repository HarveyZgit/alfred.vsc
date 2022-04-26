ROOT_DIR=$(pwd)
BUILD_DIR="$ROOT_DIR/build"
WORLFLOW_DIR="/Alfred/Alfred.alfredpreferences/workflows/user.workflow.1926E4D2-CCE6-4356-A444-5C827ACCBD9D"
TARGET_DIR=/Users/bytedance/Library/Application\ Support/Alfred/Alfred.alfredpreferences/workflows/user.workflow.1926E4D2-CCE6-4356-A444-5C827ACCBD9D

cp -R $BUILD_DIR/ /Users/bytedance/Library/Application\ Support$WORLFLOW_DIR
cp -R $ROOT_DIR/assets/ /Users/bytedance/Library/Application\ Support$WORLFLOW_DIR/assets
cp $ROOT_DIR/package.json /Users/bytedance/Library/Application\ Support/$WORLFLOW_DIR/package.json
cp $ROOT_DIR/yarn.lock /Users/bytedance/Library/Application\ Support/$WORLFLOW_DIR/yarn.lock

cd /Users/bytedance/Library/Application\ Support$WORLFLOW_DIR
pwd
yarn
