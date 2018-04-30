@ECHO OFF
C:\msys64\usr\bin\bash.exe -l -c "id && cd /c/jenkins/workspace/loom-sdk-pipeline-test && ssh -o accept_key git@github.com && ./jenkins.sh"
