FROM docker.yylt.gq/library/busybox:latest

COPY /bin/node-problem-detector /usr/bin/node-problem-detector
COPY /bin/net-checker /usr/bin/net-checker

ENTRYPOINT ["/usr/bin/node-problem-detector"]