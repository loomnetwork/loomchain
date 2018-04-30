@ECHO OFF
C:\msys64\usr\bin\bash.exe -l -c "ssh -o StrictHostKeyChecking=false github.com && cd /c/jenkins/workspace/loom-sdk-pipeline-test && ./jenkins.sh"
