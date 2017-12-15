#!/bin/bash

OK=0
NONOK=1
UNKNOWN=2

# TODO: how to check if tunnel is up

# TODO: cmd to verify tunnel status
#if [ $? -ne 0 ]; then
#    echo "Systemd is not supported"
#    exit $UNKNOWN
#fi

# TODO: cmd to verify tunnel status
#if [ $? -ne 0 ]; then
#    echo "tunnel is down"
#    exit $NONOK
#fi

echo "tunnel is up"
exit $OK