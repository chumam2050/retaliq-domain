# development container for the retaliq-domain helper
# includes Go toolchain and Debian packaging tools

FROM debian:bookworm-slim

# install build dependencies
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        build-essential \
        git \
        ca-certificates \
        curl \
        gnupg \
        golang-go \
        make \
        debhelper \
        dpkg-dev \
        devscripts \
        dh-make \
        fakeroot \
    && rm -rf /var/lib/apt/lists/*

# workspace
WORKDIR /workspace

# copy source and build script if present (mounted over in development)
COPY . /workspace

# set GOPATH and add bin to path
ENV GOPATH=/workspace
ENV PATH=$PATH:/workspace/bin

# default command shows help
CMD ["bash"]
