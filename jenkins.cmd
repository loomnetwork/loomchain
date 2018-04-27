@ECHO OFF
dir
SET PATH="c:\\Program Files\\Git\\bin";"c:\\Program Files\\rsync\\bin";%PATH%
echo Got here
"c:\Program Files\Git\bin\bash" jenkins.sh
