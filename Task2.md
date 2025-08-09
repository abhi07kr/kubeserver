
# Containerd Snapshotters – An Overview

## 1. Introduction  

In containerd, a **snapshotter** is the component responsible for managing the filesystem snapshots used by containers. When you pull an image, it’s stored as a series of layers. The snapshotter knows how to stack these layers efficiently, apply copy-on-write (COW) semantics, and provide a writable filesystem to the container runtime.  

Think of it as the “storage engine” for containerd — different snapshotters use different underlying technologies, and your choice affects performance, disk usage, and compatibility.  

---

## 2. Built-in Snapshotters  

Containerd ships with several snapshotters by default. Each is suited for different environments and filesystems.

### **Overlayfs** (default on most Linux hosts)  
- Uses Linux’s OverlayFS kernel feature to stack read-only image layers and add a thin writable layer for changes.  
- **Pros**: Fast startup, minimal disk usage, works well for Kubernetes workloads.  
- **Cons**: Requires a compatible kernel; some limitations with file attributes and special files.  

---

### **Native**  
- A simple snapshotter that uses plain directories and `cp` for creating layers.  
- **Pros**: Works anywhere, no special kernel support needed.  
- **Cons**: No real copy-on-write — consumes more disk space, slower on large images.  
- **Use case**: Testing or debugging where kernel features aren’t available.  

---

### **Btrfs**  
- Uses the Btrfs filesystem’s built-in snapshot and subvolume features.  
- **Pros**: True COW at the filesystem level, efficient cloning and rollback.  
- **Cons**: Requires Btrfs as the host filesystem; Btrfs maturity varies across distros.  

---

### **ZFS**  
- Similar to Btrfs in concept, but leverages ZFS datasets and snapshots.  
- **Pros**: Very reliable, great for large-scale persistent storage.  
- **Cons**: Requires ZFS setup on host; higher memory usage.  

---

### **Devmapper**  
- Uses device-mapper thin provisioning for COW layers.  
- **Pros**: Good for environments where OverlayFS isn’t an option.  
- **Cons**: More complex setup; slower in some workloads compared to OverlayFS.  

---

## 3. Plugin-Based Snapshotters  

Snapshotters in containerd follow a plugin model — you can swap or extend them without patching containerd itself. Two notable ones:

### **Nydus**  
- FUSE-based remote snapshotter optimized for large-scale image distribution.  
- Instead of pulling full layers first, it can stream data on demand from a registry.  
- Useful for reducing startup latency in massive Kubernetes clusters.  

### **stargz** (eStargz format)  
- Supports **lazy-loading** image layers: containers can start running while data is still being fetched in the background.  
- Great for speeding up cold starts in CI/CD or serverless environments.  

Both integrate into containerd via the same gRPC plugin interface that built-in snapshotters use. You configure them in `containerd.toml` and point images to use the correct handler.

---

## 4. Performance & Trade-offs  

- **Overlayfs vs. Native**  
  - Overlayfs is much faster and more space-efficient for production. Native is more of a fallback.  

- **Btrfs/ZFS**  
  - Shine when you want fast rollback, efficient snapshots, or are already using those filesystems.  
  - Require host setup — not always viable in cloud VMs with managed disks.  

- **Plugin Snapshotters (Nydus/Stargz)**  
  - Can drastically cut startup time for large images.  
  - Extra moving parts (FUSE, remote fetching) add operational complexity.  

---

### **Rule of Thumb**  
- **Default choice**: Overlayfs — stable and supported almost everywhere.  
- **Optimizing image pulls**: Stargz or Nydus.  
- **Advanced storage needs**: Btrfs or ZFS, if your environment supports them.  

---

*In short, snapshotters are a key part of containerd’s performance story. Picking the right one can mean the difference between a container starting in milliseconds or in seconds, especially at scale.*

---

If you want, I can also make a quick **layer-stack diagram** in Markdown so it’s more visual for your submission. That could make it look even more authentic. Would you like me to add that?
