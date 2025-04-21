FROM golang:1.24.2-bookworm@sha256:00eccd446e023d3cd9566c25a6e6a02b90db3e1e0bbe26a48fc29cd96e800901 AS builder

# Set GOMODCACHE explicitly
ENV GOMODCACHE=/go/pkg/mod

RUN apt update && apt install -y unzip wget git \
  && wget https://github.com/cli/cli/releases/download/v2.69.0/gh_2.69.0_linux_amd64.deb \
  && dpkg -i gh_2.69.0_linux_amd64.deb && rm gh_2.69.0_linux_amd64.deb \
  && wget https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_linux_amd64.zip \
  && unzip terraform_1.5.7_linux_amd64.zip && rm terraform_1.5.7_linux_amd64.zip \
  && mv terraform /usr/local/bin/terraform && chmod +x /usr/local/bin/terraform \
  && wget https://github.com/opentofu/opentofu/releases/download/v1.9.0/tofu_1.9.0_amd64.deb \
  && dpkg -i tofu_1.9.0_amd64.deb && rm tofu_1.9.0_amd64.deb

WORKDIR /app

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download
# RUN go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build scripts/build-dev.sh
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build scripts/help-docker.sh

# RUN scripts/build-dev.sh
# RUN scripts/help-docker.sh

FROM builder AS test-stage

RUN mkdir -p /app/coverdata

WORKDIR /app

ENV GOCOVERDIR=/app/coverdata

CMD ["/bin/sh", "-c", "go test -covermode=atomic -coverprofile=/app/coverdata/coverage.out ./... && echo 'Coverage data collected'"]
