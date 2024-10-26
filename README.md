# Geranos

[![Build Status](https://github.com/macvmio/geranos/actions/workflows/main.yml/badge.svg)](https://github.com/macvmio/geranos/actions)
[![Build Status](https://github.com/macvmio/geranos/actions/workflows/release.yml/badge.svg)](https://github.com/macvmio/geranos/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Introduction

Geranos is a command-line tool written in Go for efficiently transferring **macOS virtual machine images** to and from OCI-compliant container registries. Specifically designed for macOS VMs utilizing the **APFS Copy-on-Write filesystem**, Geranos optimizes both bandwidth and disk usage by leveraging sparse files and filesystem cloning capabilities.

Geranos integrates seamlessly with [Curie](https://github.com/macvmio/curie), a macOS VM virtualization program, allowing users to pull VM images and run them with minimal effort.

## Features

- **Efficient Transfer of Large VM Images**: Handles VM images typically over 30GB in size.
- **Bandwidth Optimization**: Verifies local hashes in `disk.img` files to minimize data transfer.
- **Disk Usage Optimization**: Utilizes clone operations for efficient cloning and skips writing zeros to save disk space.
- **Integration with Curie**: Easily pull VM images with Geranos and run them using Curie.
- **OCI Registry Support**: Push and pull VM images from any OCI-compliant container registry.
- **Familiar Interface**: Command-line interface similar to Docker and `crane`, making it easy for users familiar with these tools.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
    - [Configuration](#configuration)
    - [Pulling a VM Image](#pulling-a-vm-image)
    - [Running a Pulled VM Image with Curie](#running-a-pulled-vm-image-with-curie)
    - [Available Commands](#available-commands)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)
- [Future Plans](#future-plans)
- [Contact and Support](#contact-and-support)

## Installation

Geranos can be downloaded from the [GitHub releases page](https://github.com/macvmio/geranos/releases).

### Prerequisites

- **Go**: If you plan to build Geranos from source, ensure you have Go installed.
- **Curie**: For running pulled VM images, install [Curie](https://github.com/macvmio/curie).

### Download Binary

1. **Visit the Releases Page**: Go to the [Geranos Releases](https://github.com/macvmio/geranos/releases) page.
2. **Download the Binary**: Choose the appropriate binary for your operating system.
3. **Install the Binary**:
    - Move the binary to a directory in your `$PATH`, such as `/usr/local/bin`.
    - Make the binary executable:
      ```bash
      chmod +x /usr/local/bin/geranos
      ```

### Build from Source (Optional)

If you prefer to build from source:

```bash
git clone https://github.com/macvmio/geranos.git
cd geranos
go build -o geranos main.go
```

## Usage

### Configuration

Geranos requires a configuration file located at `~/.geranos/config.yaml`. This file specifies where images are stored locally.

**Example `~/.geranos/config.yaml`:**

```yaml
images_directory: /Users/yourusername/.curie/.images
```

Replace `/Users/yourusername` with your actual username or the path where Curie stores images.

NOTE: For curie up to 3.0, you have to specify ".curie/images" (without a dot)

### Pulling a VM Image

To pull a macOS VM image from an OCI registry:

```bash
geranos pull ghcr.io/macvmio/macos-sonoma:14.5-agent-v1.6
```

This command downloads the VM image while optimizing bandwidth and disk usage.

### Running a Pulled VM Image with Curie

After pulling the image, run it using Curie:

```bash
curie run ghcr.io/macvmio/macos-sonoma:14.5-agent-v1.6
```

### Available Commands

Geranos provides several commands:

- **adopt**: Adopt a directory as an image under the current local registry.
- **clone**: Locally clone one reference to another name.
- **completion**: Generate the autocompletion script for the specified shell.
- **context**: Manage contexts.
- **help**: Help about any command.
- **inspect**: Inspect details of a specific OCI image.
- **list**: List all OCI images in a specific local registry.
- **login**: Log in to a registry.
- **logout**: Log out of a registry.
- **pull**: Pull an OCI image from a registry and extract the file.
- **push**: Push a large file as an OCI image to a registry.
- **remote**: Manipulate remote repositories.
- **remove**: Remove locally stored images.
- **version**: Print the version.

**General Flags:**

- `-h`, `--help`: Help for Geranos.
- `-v`, `--verbose`: Enable verbose output.
- `--version`: Show Geranos version.

**Get Help for a Command:**

```bash
geranos [command] --help
```

### Examples

- **List remote images**
  
  ```bash
  geranos remote images ghcr.io/macvmio/macos-sonoma
  ```

- **Push an Image to a Registry:**

  ```bash
  geranos push registry.example.com/namespace/myimage:tag
  ```

- **List Images in Local Registry:**

  ```bash
  geranos list
  ```

## Contributing

Contributions are welcome! Please see the [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

- **Reporting Issues**: Use the [issue tracker](https://github.com/macvmio/geranos/issues) to report bugs or request features.
- **Pull Requests**: Submit pull requests to the `main` branch.

## License

Geranos is licensed under the Apache 2.0 License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [go-containerregistry](https://github.com/google/go-containerregistry).
- Utilizes [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper) for command-line interface and configuration management.

## Future Plans

- Integration with the upcoming [macvm.io](https://macvm.io) website.
- Enhanced filesystem optimization features.
- Support for additional VM formats and platforms.


## Contact and Support

- **GitHub Repository**: [github.com/macvmio/geranos](https://github.com/macvmio/geranos)
- **Issues**: [github.com/macvmio/geranos/issues](https://github.com/macvmio/geranos/issues)
- **Email**: [contact@macvm.io](mailto:contact@macvm.io)

