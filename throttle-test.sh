#!/bin/bash

killLoomProcess()
{
    cd builtin
    if [ -s nohup.pid ]
    then
            echo "killing already running loom process"
            sudo kill -9 `cat nohup.pid`
            rm -rf nohup.pid
            touch nohup.pid
            echo > nohup.out
    fi
    cd ..

}

compileLoom()
{
    PKG=github.com/loomnetwork/loomchain
    LOOM_SRC=$GOPATH/src/$PKG
    cd $LOOM_SRC

    make clean
    make deps
    make

}

runLoom()
{
    cd builtin
    ../loom init -f
    nohup ../loom run &
    echo $! > nohup.pid

    echo "waiting 5 seconds for loom to run"
    sleep 5

    cd ..

}

customContract()
{
    echo ""
    echo "Custom Contract:"
    ./loom genkey -a publicKeyFilename -k privateKeyFilename
    CONTRACT_ADDRESS=`./loom deploy -a publicKeyFilename -k privateKeyFilename -b contractBytecode.bin | awk '{ print $6}' | sed  -n '1 p' | awk -F ':' '{print $2}'`
    echo "Contract Address: "$CONTRACT_ADDRESS

    start=`date +%s`
    loopcount=1
    while [ $loopcount -le $loopmax ]
    do
        echo "loop iteration: $loopcount"
        (( loopcount=loopcount+1 ))
        ./loom call  -a publicKeyFilename -k privateKeyFilename -i inputDataFile -c $CONTRACT_ADDRESS
    done
    end=`date +%s`
    runtime=$((end-start))
    echo "Total runtime: "$runtime
}

coinContract()
{
    echo ""
    echo "Coin Contract:"
    ./loom genkey -a publicKeyFilename -k privateKeyFilename

    start=`date +%s`
    loopcount=1
    while [ $loopcount -le $loopmax ]
    do
        echo "loop iteration: $loopcount"
        (( loopcount=loopcount+1 ))
        ./loom call  -a publicKeyFilename -k privateKeyFilename -i inputDataFile -n coin
    done
    end=`date +%s`
    runtime=$((end-start))
    echo "Total runtime: "$runtime
}

dposContract()
{
    echo ""
    echo "DPOS Contract:"
    ./loom genkey -a publicKeyFilename -k privateKeyFilename

    start=`date +%s`

    loopcount=1
    while [ $loopcount -le $loopmax ]
    do
       echo "loop iteration: $loopcount"
       (( loopcount=loopcount+1 ))
       ./loom call  -a publicKeyFilename -k privateKeyFilename -i inputDataFile -n dpos
    done
    end=`date +%s`
    runtime=$((end-start))
    echo "Total runtime: "$runtime
}

loopmax=2

killLoomProcess
compileLoom
runLoom
#customContract
#coinContract
#dposContract

