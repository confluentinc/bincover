#!/usr/bin/env bash
while read line
do
  echo "$line"
done < /dev/stdin
echo START_BINCOVER_METADATA
echo "{\"cover_mode\":\"\",\"exit_code\":1}"
echo END_BINCOVER_METADATA