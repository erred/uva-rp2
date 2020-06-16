#!/usr/bin/env bash

for h in {0..9}; do
    rsync -rP scan scan-2$h:scan
    rsync -rP run.sh scan-2$h:run.sh
    ssh scan-2$h screen -dmS scan ./run.sh $h
done
