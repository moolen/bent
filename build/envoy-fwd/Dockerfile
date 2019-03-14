FROM moolen/bent-trace-fwd:latest as fwd
FROM moolen/bent-envoy:latest

COPY --from=fwd /trace-fwd /bin/trace-fwd
COPY build/envoy-fwd/entrypoint.sh /bin/entrypoint.sh
ENTRYPOINT [ "/bin/entrypoint.sh" ]
