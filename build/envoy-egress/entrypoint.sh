#!/bin/bash
/bin/egress.sh &
/sbin/my_init -- /usr/local/bin/entrypoint.sh
