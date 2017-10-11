#!/bin/sh
set -x
iptables -t nat -A OUTPUT -p tcp --dport 8080 --dest moo-repo.wdf.sap.corp -j REDIRECT --to-ports 9191
guttle server --proxy-addr spiegel.de:80
