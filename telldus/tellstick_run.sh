#!/bin/bash

/etc/init.d/telldusd start
sleep 1
exec /telldus-arm
