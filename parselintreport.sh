#!/bin/bash
count=`grep ".go" lintreport | wc -l`
echo "Number of errors $count"
if [ $count -le 95 ] #Set this to one higher then current number of lints, lower this number over time
then
  echo "Errors within threshold"
  exit 0
else
  echo "Errors have exceeded threshold limit" >&2
  exit 1
fi
