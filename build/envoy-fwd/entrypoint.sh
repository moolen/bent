#!/bin/bash
/bin/trace-fwd &
/sbin/my_init -- /usr/local/bin/entrypoint.sh
