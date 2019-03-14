FROM moolen/http-egress:latest as egress
FROM moolen/bent-envoy:latest

COPY --from=egress /entrypoint.sh /bin/egress.sh
COPY build/envoy-egress/entrypoint.sh /bin/entrypoint.sh
ENTRYPOINT [ "/bin/entrypoint.sh" ]
