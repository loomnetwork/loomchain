@ECHO OFF
runas /user:kevin "C:\msys64\usr\bin\bash.exe -l -c 'id && cd /c/jenkins/workspace/loom-sdk-pipeline-test && ./jenkins.sh'"
