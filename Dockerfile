# ---- Build stage ----
# Match the Go version declared in go.mod so local builds and the container
# build behave identically.
FROM golang:1.27-alpine AS builder

WORKDIR /src

# Copy dependency manifests first so Docker can cache the download step.
# Source code changes far more often than dependencies do, so this ordering
# avoids re-downloading modules on every code change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 produces a statically linked binary, which is what lets us
# run it in a minimal, C-library-free base image below.
RUN CGO_ENABLED=0 GOOS=linux go build -o /calculator ./cmd/api

# ---- Runtime stage ----
# distroless/static contains no shell, no package manager, and no OS
# libraries beyond what a static Go binary needs — a small attack surface
# for a service that's meant to run unattended for a long time.
FROM gcr.io/distroless/static-debian12

COPY --from=builder /calculator /calculator

EXPOSE 8080

ENTRYPOINT ["/calculator"]