#!/bin/bash
set -e

# Setup mise
mise trust
mise install
echo 'eval "$(mise activate bash)"' | sudo tee -a $HOME/.bashrc /root/.bashrc

sudo mkdir -p /root/.local/share
sudo ln -sf $HOME/.local/share/mise /root/.local/share/mise

sudo ln -sf /run/docker/containerd/containerd.sock /run/containerd/containerd.sock

# Setup nerdctl
sudo ln -sf $(mise which nerdctl) /usr/local/bin/nerdctl

# Setup Docker mirrors
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json > /dev/null << EOF
{
  "registry-mirrors": [
    "$MIRROR_CONTAINER_REGISTRY"
  ]
}
EOF
sudo killall -SIGHUP dockerd

# Docker login
echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USERNAME --password-stdin
