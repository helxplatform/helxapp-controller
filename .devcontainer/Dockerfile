FROM mcr.microsoft.com/devcontainers/go:0-1.20

RUN mkdir -p /usr/local/bin
RUN curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
RUN chmod +x kubebuilder && mv kubebuilder /usr/local/bin/