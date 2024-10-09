# Advanced Tips for Geranos

This file contains advanced tips and tricks for optimizing your workflow with Geranos. Check back regularly as new tips will be added!

## 1. Optimized Pushing Between Repositories with `--mount`

There is a `--mount` option for optimized pushing of images between repositories. Imagine a case where you push an image to one repository and later want to modify it slightly and push it to another repository. If you have access to both repositories, you can use the `--mount` option to mount blobs from the first repository to the second. This avoids the need to re-push the blobs entirely.

While registries would deduplicate blobs even without the `--mount` option, using `--mount` can save network traffic and reduce upload time.

### Example:
```bash
./geranos push ghcr.io/macvmio/vm-image:macos-15.0.1-base --mount ghcr.io/macvmio/registry:macos-15.0.1-base
```

## 2. Managing Contexts to Avoid Repetition

Geranos provides a `context` command to manage registry contexts, similar to how `kubectl` manages contexts for Kubernetes clusters. This feature allows you to avoid repeatedly specifying the registry name or other connection details.

By setting a context, you can easily switch between different registries and configurations without needing to input the registry details each time you push or pull images.

### Example Usage:
```bash
# Set a new context
geranos context set myregistry --registry ghcr.io/macvmio

# Use a different context
geranos context use myregistry

# List all available contexts
geranos context list

# Unset the current context
geranos context unset
```

Using contexts simplifies working with multiple registries and reduces the need to repeatedly enter the same information.

---

### More tips coming soon...

Feel free to add more tips below this section as you discover new ways to optimize and enhance your use of Geranos.

