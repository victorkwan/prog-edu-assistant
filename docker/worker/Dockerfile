# Start nsjail build stage.
FROM debian:testing
RUN apt-get -y update
# The requirements for building nsjail.
RUN apt-get install -y \
  autoconf \
  bison \
  flex \
  gcc \
  g++ \
  git \
  libprotobuf-dev \
  libnl-route-3-dev \
  libtool \
  make \
  pkg-config \
  protobuf-compiler \
  bash

# Install the python environment (and nsjail runtime dependencies).
# TODO(salikh): Move this to a separate stage.
RUN apt-get install -y \
    python3 python3-dev python3-scipy python3-pip python-virtualenv \
    libprotobuf17  libnl-route-3-200 \
    freetype* bash \
  && ln -s /usr/bin/pip3 /usr/bin/pip \
  && ln -sf /usr/bin/python3 /usr/bin/python \
  && pip install --no-cache-dir \
    six arrow scikit-learn pandas numpy scikit-image

# Build nsjail
RUN git clone https://github.com/google/nsjail.git /root/nsjail \
  && cd /root/nsjail && make && mv /root/nsjail/nsjail /usr/local/bin
# Build dependencies for worker daemon
RUN apt-get install -y \
  golang-go
RUN export GOPATH=/go && go get -u -v github.com/streadway/amqp
RUN git clone https://github.com/google/prog-edu-assistant /root/prog-edu-assistant \
  && cd /root/prog-edu-assistant \
  && mkdir -p bin && go build -o bin/worker worker/cmd/worker.go \
  && mv bin/worker /usr/local/bin/worker

# Start the python worker stage.
#FROM debian:testing
#RUN apt-get -y update

# TODO(salikh): Install python environment and nsjail runtime dependencies.

# Copy nsjail.
COPY --from=0 /usr/local/bin/nsjail /bin/nsjail

CMD ["/bin/bash", "-i"]
