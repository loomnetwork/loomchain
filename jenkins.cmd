@ECHO OFF
dir
SET PATH="c:\Program Files\Git\bin";"c:\Program Files\rsync\bin";%PATH%
echo .
echo Path=%PATH$
echo .
"c:\Program Files\Git\bin\bash" jenkins.sh
