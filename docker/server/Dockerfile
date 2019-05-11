FROM alpine

COPY bin/uploadserver /usr/local/bin/uploadserver
RUN mkdir /tmp/uploads

ENTRYPOINT ["/usr/local/bin/uploadserver", "--upload_dir=/tmp/uploads", "--logtostderr", "--v=3", "--queue_spec", "amqp://guest:guest@rabbitmq:5672/"]
