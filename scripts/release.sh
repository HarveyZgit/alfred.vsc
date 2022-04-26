ROOT_DIR=$(pwd)
BUILD_DIR="$ROOT_DIR/build"
WORLFLOW_DIR="/Alfred/Alfred.alfredpreferences/workflows/user.workflow.D35CAA4D-0B5B-536F-9382-5102111FCB18"
TARGET_DIR=/Users/example/Library/Application\ Support/Alfred/Alfred.alfredpreferences/workflows/user.workflow.D35CAA4D-0B5B-536F-9382-5102111FCB18

cp -R $BUILD_DIR/ /Users/example/Library/Application\ Support$WORLFLOW_DIR
cp -R $ROOT_DIR/assets/ /Users/example/Library/Application\ Support$WORLFLOW_DIR/assets
cp $ROOT_DIR/package.json /Users/example/Library/Application\ Support/$WORLFLOW_DIR/package.json
cp $ROOT_DIR/yarn.lock /Users/example/Library/Application\ Support/$WORLFLOW_DIR/yarn.lock

cd /Users/example/Library/Application\ Support$WORLFLOW_DIR
pwd
yarn
