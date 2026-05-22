# Docker Setup in Sprite Environment

**Important:** This environment runs with restricted disk space. Do NOT create large VFS files. Disk images must be sparse.

## 1. Install Docker

```bash
sudo apt-get update
sudo apt-get install -y docker.io docker-buildx
```

## 2. Configure Docker Storage

Configure Docker to use the overlay2 storage driver. The Sprite environment runs on overlay, but kernel 6.12+ supports nested overlay with native diff:

```bash
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json > /dev/null <<EOF
{
  "storage-driver": "overlay2"
}
EOF
```

## 3. Start Docker as a Sprite Service

Since Sprite environments don't use systemd, Docker needs to be registered as a Sprite service:

```bash
sprite-env services create docker --cmd /usr/bin/sudo --args /usr/bin/dockerd
```

## 4. Verify Docker is Running

```bash
sudo docker ps
```

## 5. Create a Checkpoint

After installation, create a checkpoint to preserve the Docker setup across reboots:

```bash
sprite-env checkpoints create
```

## 6. Using Docker

```bash
sudo docker run --rm alpine echo "Hello from Docker!"
sudo docker images
sudo docker pull nginx
```

## Service Management Commands

```bash
# Check Docker service status
sprite-env services get docker

# List all services
sprite-env services list

# View Docker logs
tail -f /.sprite/logs/services/docker.stderr.log
```
