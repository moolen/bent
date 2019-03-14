#!/bin/bash
/bin/echo-server &
/sbin/my_init -- /usr/local/bin/entrypoint.sh
