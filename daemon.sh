#!/bin/bash

./bin/kwan_daemon -command="./bin/kwan -c ./etc/config.xml" -directory="./etc/vhost/" -pattern="(.+\\.xml)$"
