# syntax=docker/dockerfile:1

# ---- Stage 1: build the Vue frontend ----
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
# Copy sources first, then run `npm ci`. Running it AFTER the copy is deliberate:
# npm ci deletes any node_modules that rode along (e.g. when the tree is
# folder-copied instead of git-cloned, a Windows/macOS node_modules would
# otherwise shadow vite and break the Linux build) and reinstalls deterministically
# from package-lock.json.
COPY frontend/ ./
RUN npm ci
RUN npm run build

# ---- Stage 2: build the Go binary with the frontend embedded ----
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY backend/ ./
# Replace the placeholder web/ with the freshly built SPA, then embed it.
COPY --from=frontend /app/frontend/dist/ ./web/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /security-toolbox .

# ---- Stage 3: minimal runtime ----
FROM scratch
COPY --from=build /security-toolbox /security-toolbox
EXPOSE 8080
USER 65534:65534
ENTRYPOINT ["/security-toolbox"]
