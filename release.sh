#/bin/bash

if [ -z "$1" ]
  then
    echo "$0 patch|minor|major [/path/to/credentials]"
    exit 1
fi

BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$BRANCH" != "master" ]]; then
  echo 'Not on master branch, aborting script';
  exit 1;
fi

# only works for ssh url
OWNER=$(git remote get-url origin | cut -d: -f 2 | cut -d/ -f 1 | tr '[:upper:]' '[:lower:]')
REPO=$(git remote get-url origin | cut -d: -f 2 | cut -d/ -f 2 | cut -f 1 -d '.' | tr '[:upper:]' '[:lower:]')

read -p "Repository [$OWNER/$REPO]: " TMP
if [ ! -z "$TMP" ]; then
  OWNER=$(echo $TMP | cut -d/ -f 1)
  REPO=$(echo $TMP | cut -sd/ -f 2)
fi

if [ -z "$OWNER" ] || [ -z "$REPO" ]; then
  echo "Expected github owner/repo, got '$OWNER/$REPO' - Aborting."
  exit 1
fi

if [ -z "$2" ]; then
  CONFIG_FILE="$HOME/.githubrepotoken"
else
  CONFIG_FILE=$2
fi

if [ -f $CONFIG_FILE ]; then
  CREDIENTIALS=$(cat "$CONFIG_FILE")
  GITHUB_USER=$(echo $CREDIENTIALS | cut -d/ -f 1)
  GITHUB_TOKEN=$(echo $CREDIENTIALS | cut -d/ -f 2-)
else
  echo "No credientials file found at '$CONFIG_FILE', add 'github-user/github-token' to this file to skip questions."
fi

if [ -z "$GITHUB_USER" ]; then
  read -p "Your Github username: " GITHUB_USER
fi

if [ -z "$GITHUB_TOKEN" ]; then
  read -p "Your Github Token: " GITHUB_TOKEN
  echo -en "\033[1A\033[2K"
  echo "Your Github Token: ******************"
fi

URL="https://api.github.com/repos/$OWNER/$REPO/collaborators/$GITHUB_USER/permission"
HTTP_RESPONSE=$(curl -s -w "HTTPSTATUS:%{http_code}" -X GET -u $GITHUB_USER:$GITHUB_TOKEN $URL)
HTTP_BODY=$(echo $HTTP_RESPONSE | sed -e 's/HTTPSTATUS\:.*//g')
HTTP_STATUS=$(echo $HTTP_RESPONSE | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

PERMISSION=""
if [ $HTTP_STATUS -eq 200  ]; then
  PERMISSION=$(echo $HTTP_BODY | python2 -c 'import json,sys;res=json.load(sys.stdin); print res["permission"]')
fi

if [ "$PERMISSION" != "admin" ] && [ "$PERMISSION" != "write" ]; then
  echo "Missing write/admin permission to repo for github user '$GITHUB_USER', Aborting."
  exit 1
fi

URL="https://api.github.com/repos/$OWNER/$REPO/releases/latest"
HTTP_RESPONSE=$(curl -s -w "HTTPSTATUS:%{http_code}" -X GET -u $GITHUB_USER:$GITHUB_TOKEN $URL)
HTTP_BODY=$(echo $HTTP_RESPONSE | sed -e 's/HTTPSTATUS\:.*//g')
HTTP_STATUS=$(echo $HTTP_RESPONSE | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

if [ $HTTP_STATUS -eq 200  ]; then
  CURRENT_RELEASE=$(echo $HTTP_BODY | python2 -c 'import json,sys;res=json.load(sys.stdin); print res["tag_name"]')

  TARGET_COMMITISH=$(echo $HTTP_BODY | python2 -c 'import json,sys;res=json.load(sys.stdin); print res["target_commitish"]')
fi

NEW_RELEASE="0.0.0"
if [ $1 == "major"  ]; then
  NEW_RELEASE=$(echo "${CURRENT_RELEASE:-0.0.-1}" | awk -F'[.]' '{print $1+1"."0"."0}')
elif [ $1 == "minor" ]; then
  NEW_RELEASE=$(echo "${CURRENT_RELEASE:-0.0.-1}" | awk -F'[.]' '{print $1"."$2+1"."0}')
elif [ $1 == "patch" ]; then
  NEW_RELEASE=$(echo "${CURRENT_RELEASE:-0.0.-1}" | awk -F'[.]' '{print $1"."$2"."$3+1}')
fi

LAST_COMMIT=$(git rev-parse HEAD)
if [ ! -z $TARGET_COMMITISH ] && [ $TARGET_COMMITISH == $LAST_COMMIT ]; then
  echo "Latest commit in master is the same as the latest release, '$TARGET_COMMITISH', Aborting."
  exit 1
fi

echo "Current release : $CURRENT_RELEASE"
echo "New release     : $NEW_RELEASE"

read -p "Continue? [y/n]: " CONTINUE
if [ $CONTINUE != "y" ]; then
  echo "Aborting."
  exit 1
fi

URL="https://api.github.com/repos/$OWNER/$REPO/releases"
DATA="{\"tag_name\":\"$NEW_RELEASE\",\"target_commitish\":\"$LAST_COMMIT\"}"
HTTP_RESPONSE=$(curl -s -d "$DATA" -w "HTTPSTATUS:%{http_code}" -X POST -u $GITHUB_USER:$GITHUB_TOKEN $URL)
HTTP_BODY=$(echo $HTTP_RESPONSE | sed -e 's/HTTPSTATUS\:.*//g')
HTTP_STATUS=$(echo $HTTP_RESPONSE | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')

echo $HTTP_BODY
