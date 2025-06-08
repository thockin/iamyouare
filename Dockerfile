FROM busybox
COPY iamyouare /
ENTRYPOINT ["/iamyouare"]
