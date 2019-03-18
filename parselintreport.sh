#!/bin/bash
count=`grep ".go" lintreport | wc -l`
echo "Number of errors $count"
