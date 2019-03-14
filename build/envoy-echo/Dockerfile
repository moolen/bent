FROM jmalloc/echo-server:latest as echo
FROM moolen/bent-envoy:latest

COPY --from=echo /bin/echo-server /bin/echo-server
COPY build/envoy-echo/entrypoint.sh /bin/entrypoint.sh
ENTRYPOINT [ "/bin/entrypoint.sh" ]
