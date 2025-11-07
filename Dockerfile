# Copyright 2025 Nguyen Nhat Nguyen
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.25.1-alpine AS builder

ARG TARGETBIN=server
ARG TARGETPKG=./cmd/durablefuture-server
ARG TARGETARCH

WORKDIR /src

ENV GOFLAGS="-mod=vendor"

COPY go.mod go.sum ./
COPY internal/third_party/chronicle ./internal/third_party/chronicle
COPY vendor ./vendor

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-s -w" -o /out/${TARGETBIN} ${TARGETPKG}

FROM gcr.io/distroless/static-debian12

ARG TARGETBIN=server

COPY --from=builder /out/${TARGETBIN} /bin/app

ENTRYPOINT ["/bin/app"]
