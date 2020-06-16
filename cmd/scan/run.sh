#!/bin/sh

ulimit -n 65536
h=$1
./scan -start $(( 430000000  * h )) -count 430000000
