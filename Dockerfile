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

FROM golang:1.24-alpine AS builder

ARG TARGETPLATFORM
ARG TARGETARCH
ARG MODULE
ARG MODULE_RUN_PATH
ARG MODE=debug

WORKDIR /app

COPY go.work go.work.sum ./
COPY vendor/ ./vendor/
COPY durablefuture/go.mod durablefuture/go.sum ./durablefuture/
COPY usage/go.mod ./usage/

RUN mkdir -p durablefuture usage

RUN go work sync

COPY durablefuture/ ./durablefuture
COPY usage/ ./usage

WORKDIR /app/${MODULE}

RUN go build -ldflags="-w -s" -o /app/server ${MODULE_RUN_PATH}/main.go

RUN chmod +x /app/server

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/server /server

EXPOSE 8080