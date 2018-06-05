#!/usr/bin/env bash
rm -rf etherboy.log;
CURRENT_PATH=`pwd`
COMMAND="./loom run 2>&1 | tee -a etherboy.log"
(ps aux | grep $COMMAND | awk {' print $2 '} | xargs kill -9 ) || true
(ps aux | grep etherboycore.0.0.1 | awk {' print $2 '} | xargs kill -9 ) || true


make || exit 1

cp loom ../etherboy-core/run/

cd ../etherboy-core/run;
rm -rf app.db chaindata
./loom init -f

cp ../genesis.json .;

$COMMAND

cd $CURRENT_PATH;