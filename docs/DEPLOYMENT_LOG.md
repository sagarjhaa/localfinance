# Deployment Log

## 2026-03-03: Playground Deployment Attempt

### Goal
Deploy LocalFinance model to `tinys-playground-2.local` for testing.

### Machine Specs
- **Host:** tinys-playground-2.local
- **CPU:** Intel 6-Core i7 @ 2.6GHz
- **RAM:** 16GB (mostly used by system processes)
- **GPU:** None (Intel integrated)
- **OS:** macOS Sequoia 15.2

### What We Tried

#### Attempt 1: F16 (Full Precision) Model
- **File:** `localfinance-v1.gguf` (948MB)
- **Result:** ❌ FAILED
- **Issue:** OOM killed during inference. Model loads (~1GB in RAM) but inference gets killed by system.
- **Root cause:** 16GB RAM nearly full (system processes using 15GB+). Model needs headroom for inference.

#### Attempt 2: Q4_K_M (Quantized) Model
- **File:** `localfinance-v1-q4_k_m.gguf` (379MB)
- **Result:** ❌ FAILED (practical use)
- **Issue:** Model loads fine (~480MB RAM) but inference takes 8+ minutes for a simple query, then gets OOM killed.
- **Root cause:** 
  1. No GPU acceleration (MLX library fails to load on Intel Mac)
  2. CPU-only inference at 100% utilization
  3. Memory pressure from other processes causes eventual OOM

### Technical Details

**MLX Error (appears on all runs):**
```
ERROR Failed to load MLX dynamic library symbols path=/Applications/Ollama.app/Contents/Resources/libmlxc.dylib
WARN MLX dynamic library not available
```
MLX requires Apple Silicon. Intel Macs fall back to CPU-only inference.

**Memory State:**
```
PhysMem: 16G used (2650M wired, 840M compressor), 47M unused
```
System has almost no free memory before model even loads.

**Process Conflicts:**
- Found old LocalFinance Telegram bot consuming 2GB RAM (killed)
- Various system processes (mediaanalysisd, photoanalysisd) using significant RAM

### Models Created on Playground

| Model Name | File | Size | Status |
|------------|------|------|--------|
| `localfinance:latest` | localfinance-v1.gguf | 994MB | Created, unusable (OOM) |
| `localfinance-q4:latest` | localfinance-v1-q4_k_m.gguf | 392MB | Created, too slow |

### Learnings

1. **Intel Macs are not suitable for LLM inference** — No MLX, no GPU acceleration, CPU-only is impractical
2. **16GB RAM is marginal** — With macOS system overhead, ~2GB free is not enough
3. **Quantization helps memory but not speed** — Q4 model uses half the RAM but is still painfully slow on CPU
4. **Target hardware matters** — Need Apple Silicon or dedicated GPU for acceptable inference times

### Recommendations

For LocalFinance deployment:

1. **Apple Silicon Mac Mini (M2/M3)** — MLX acceleration, 8GB+ unified memory, ~$500-600
2. **Raspberry Pi 5 + NPU HAT** — If staying on ARM, Hailo-8L AI HAT could help
3. **Cloud GPU (dev only)** — RunPod/Vast.ai for testing, not for privacy-focused product
4. **Consider smaller model** — TinyLlama (1.1B) or Phi-2 (2.7B) might work better on constrained hardware

### Next Steps

- [ ] Test on Sagar's main Mac (Apple Silicon if available)
- [ ] Research Raspberry Pi 5 + AI HAT options
- [ ] Consider hybrid: simple queries on-device, complex to cloud (opt-in)
- [ ] Evaluate if prompt engineering can make model responses shorter/faster

---

*Last updated: 2026-03-03 by Tiny*
