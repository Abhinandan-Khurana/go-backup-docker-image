<p align="center">
  <img src="./go-backup-docker-image.svg" width="300" height="300">
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/Abhinandan-Khurana/go-backup-docker-image"><img src="https://goreportcard.com/badge/github.com/Abhinandan-Khurana/go-backup-docker-image" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://golang.org/doc/devel/release.html"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8.svg" alt="Go Version"></a>
<img src="https://img.shields.io/badge/version-v1.0.0-blue.svg" alt="Version">
</p>

> A minimalist yet powerful utility for backing up and restoring Docker images with ease.

`go-backup-docker-image` helps you create portable archives of your Docker images, allowing you to store, transfer, and restore them when needed. Perfect for offline environments, backup strategies, or transferring images between airgapped systems.



## 🌟 Features

- **Flexible Input Methods**: Accept image names from stdin, text files, or command arguments
- **Concurrent Processing**: Utilize worker pools for efficient multi-image operations
- **Compression Support**: Save space with built-in gzip compression 
- **Rich Metadata**: Each backup includes detailed information about the image
- **Comprehensive Management**: Backup, restore, and list operations in one tool
- **Detailed Reporting**: Verbose output options for monitoring operations

## Direct Installation using go

```bash
go install -v github.com/Abhinandan-Khurana/go-backup-docker-image@latest
```

## 📦 Installation

### Prerequisites

- Go 1.23 or later
- Docker installed and running

### From Source

# Clone the repository
```bash
git clone https://github.com/Abhinandan-Khurana/go-backup-docker-image.git
cd go-backup-docker-image
```

# Build the binary
```bash
go build -o go-backup-docker-image main.go
```

# Optional: Move to a directory in your PATH
```bash
sudo mv go-backup-docker-image /usr/local/bin/
```

## 🚀 Quick Start

### Backup a Single Image

```bash
go-backup-docker-image backup nginx:latest
```

### Restore an Image

```bash
go-backup-docker-image restore docker-backups/nginx_latest-20230615-120530.tar.gz
```

### List Available Backups

```bash
go-backup-docker-image list
```

## 📖 Usage Guide

### Backup Command

Back up Docker images to compressed or uncompressed tarballs.

```bash
go-backup-docker-image backup [IMAGE_NAME...] [flags]
```

#### Flags

| Flag | Shorthand | Description |
|------|-----------|-------------|
| `--dir` | `-d` | Directory to store backups (default: "docker-backups") |
| `--workers` | `-w` | Maximum number of concurrent workers (default: 3) |
| `--verbose` | `-v` | Enable verbose logging |
| `--compress` | `-c` | Compression type (gzip, none) (default: "gzip") |
| `--file` | `-f` | Read image names from file |
| `--stdin` | `-s` | Read image names from stdin |

#### Examples

Backup multiple images:
```bash
go-backup-docker-image backup nginx:latest redis:alpine postgres:13
```

Backup images listed in a file:
```bash
go-backup-docker-image backup --file images.txt
```

Backup images from stdin:
```bash
cat images.txt | go-backup-docker-image backup --stdin
```

Use uncompressed format:
```bash
go-backup-docker-image backup --compress none nginx:latest
```

### Restore Command

Restore Docker images from tarballs.

```bash
go-backup-docker-image restore [TARBALL_PATH...] [flags]
```

#### Flags

| Flag | Shorthand | Description |
|------|-----------|-------------|
| `--verbose` | `-v` | Enable verbose logging |
| `--file` | `-f` | Read tarball paths from file |
| `--stdin` | `-s` | Read tarball paths from stdin |
| `--workers` | `-w` | Maximum number of concurrent workers (default: 3) |

#### Examples

Restore multiple image backups:
```bash
go-backup-docker-image restore backup1.tar.gz backup2.tar.gz
```

Restore images listed in a file:
```bash
go-backup-docker-image restore --file backups.txt
```

### List Command

Display available image backups.

```bash
go-backup-docker-image list [flags]
```

#### Flags

| Flag | Shorthand | Description |
|------|-----------|-------------|
| `--dir` | `-d` | Backup directory to list (default: "docker-backups") |
| `--verbose` | `-v` | Show detailed information |

## 🔄 Common Workflows

### Backup All Local Images

```bash
docker images --format "{{.Repository}}:{{.Tag}}" | grep -v "" > images.txt
go-backup-docker-image backup --file images.txt
```

### Transfer Images Between Machines

On source machine:
```bash
go-backup-docker-image backup nginx:latest
scp docker-backups/nginx_latest-*.tar.gz user@destination:/path/
```

On destination machine:
```bash
go-backup-docker-image restore /path/nginx_latest-*.tar.gz
```

### Regular Backup Strategy

```bash
#!/bin/bash
# Save this as backup-images.sh
DATE=$(date +%Y%m%d)
BACKUP_DIR="/backup/docker-images/$DATE"

mkdir -p $BACKUP_DIR
docker images --format "{{.Repository}}:{{.Tag}}" | grep -v "" > $BACKUP_DIR/images.txt
go-backup-docker-image backup --file $BACKUP_DIR/images.txt --dir $BACKUP_DIR
```
## 🔍 Troubleshooting

### Common Issues

**"Error: Cannot connect to the Docker daemon"**
- Ensure Docker is running with `docker ps`
- Check if your user has permissions to access the Docker socket

**"Error reading file: open images.txt: no such file or directory"**
- Verify the file path is correct
- Check that the file has correct permissions

**"Failed to save image: context deadline exceeded"**
- For large images, try increasing the timeout or using uncompressed format

## 💼 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the project
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

Made with ❤️ by [Abhinandan-Khurana](https://github.com/Abhinandan-Khurana) for Docker enthusiasts