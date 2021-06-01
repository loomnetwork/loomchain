#!/bin/bash
count=`grep ".go" lintreport | wc -l`
echo "The number of errors is $count"
if [ $count -le 54 ] #Set this to a value higher than the current number of linter errors. We should lower this number over time
then
  echo "The number of errors is within the threshold."
  exit 0
else
  echo "The number of errors exceeds the threshold." >&2
  exit 1
fi
