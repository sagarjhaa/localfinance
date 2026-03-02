#!/usr/bin/env python3
"""
LocalFinance AI - GGUF Conversion Script
Converts the fine-tuned safetensors model to GGUF format for llama.cpp.

Usage:
    cd /Users/sagarjha/projects/localfinance
    source .venv/bin/activate
    python scripts/convert_to_gguf.py

Requirements:
    pip install llama-cpp-python
    # Or clone llama.cpp and use convert_hf_to_gguf.py
"""

import os
import sys
import subprocess
from pathlib import Path

# Paths
PROJECT_ROOT = Path(__file__).parent.parent
MODEL_DIR = PROJECT_ROOT / "models" / "localfinance-v1"
OUTPUT_FILE = PROJECT_ROOT / "models" / "localfinance-v1.gguf"
QUANTIZED_FILE = PROJECT_ROOT / "models" / "localfinance-v1-q4_k_m.gguf"


def find_llama_cpp():
    """Try to find llama.cpp installation."""
    # Common locations
    possible_paths = [
        Path.home() / "llama.cpp",
        Path.home() / "projects" / "llama.cpp",
        Path("/opt/llama.cpp"),
        Path("/usr/local/llama.cpp"),
    ]
    
    for path in possible_paths:
        convert_script = path / "convert_hf_to_gguf.py"
        if convert_script.exists():
            return path
    
    return None


def convert_with_llama_cpp(llama_cpp_path: Path):
    """Convert using llama.cpp's script."""
    convert_script = llama_cpp_path / "convert_hf_to_gguf.py"
    
    print(f"📍 Using llama.cpp at: {llama_cpp_path}")
    print(f"📄 Convert script: {convert_script}")
    
    cmd = [
        sys.executable,
        str(convert_script),
        str(MODEL_DIR),
        "--outfile", str(OUTPUT_FILE),
        "--outtype", "f16",  # Full precision first
    ]
    
    print(f"🔄 Running: {' '.join(cmd)}")
    result = subprocess.run(cmd, capture_output=True, text=True)
    
    if result.returncode != 0:
        print(f"❌ Conversion failed:")
        print(result.stderr)
        return False
    
    print(result.stdout)
    return True


def convert_with_transformers():
    """Convert using transformers' GGUF export (if available)."""
    try:
        from transformers import AutoModelForCausalLM
        
        print("📥 Loading model...")
        model = AutoModelForCausalLM.from_pretrained(MODEL_DIR)
        
        # Check if GGUF export is available (transformers 4.36+)
        if hasattr(model, 'save_gguf'):
            print(f"💾 Exporting to GGUF: {OUTPUT_FILE}")
            model.save_gguf(str(OUTPUT_FILE))
            return True
        else:
            print("⚠️  transformers GGUF export not available")
            return False
            
    except Exception as e:
        print(f"❌ Error: {e}")
        return False


def try_pip_install_converter():
    """Try installing and using a standalone converter."""
    print("📦 Attempting to use llama-cpp-python converter...")
    
    # Try importing
    try:
        import llama_cpp
        print(f"   llama-cpp-python version: {llama_cpp.__version__}")
    except ImportError:
        print("   llama-cpp-python not installed")
        print("   Install with: pip install llama-cpp-python")
        return False
    
    # llama-cpp-python doesn't include the converter script
    # Need to use the standalone llama.cpp
    return False


def main():
    print("=" * 60)
    print("🏦 LocalFinance AI - GGUF Conversion")
    print("=" * 60)
    print(f"📁 Input model: {MODEL_DIR}")
    print(f"📁 Output GGUF: {OUTPUT_FILE}")
    print()
    
    # Check model exists
    if not (MODEL_DIR / "model.safetensors").exists():
        print("❌ Model not found! Train the model first.")
        sys.exit(1)
    
    # Try different conversion methods
    success = False
    
    # Method 1: llama.cpp convert script
    llama_cpp_path = find_llama_cpp()
    if llama_cpp_path:
        print("🔧 Method 1: Using llama.cpp converter")
        success = convert_with_llama_cpp(llama_cpp_path)
    else:
        print("⚠️  llama.cpp not found at common locations")
    
    # Method 2: transformers native export
    if not success:
        print("\n🔧 Method 2: Trying transformers GGUF export")
        success = convert_with_transformers()
    
    # Method 3: Standalone tool
    if not success:
        print("\n🔧 Method 3: Checking for llama-cpp-python")
        try_pip_install_converter()
    
    # Result
    print()
    if success and OUTPUT_FILE.exists():
        size_mb = OUTPUT_FILE.stat().st_size / (1024 * 1024)
        print("=" * 60)
        print("✅ Conversion successful!")
        print(f"📄 Output: {OUTPUT_FILE}")
        print(f"📦 Size: {size_mb:.1f} MB")
        print()
        print("To quantize (smaller/faster), install llama.cpp and run:")
        print(f"  ./quantize {OUTPUT_FILE} {QUANTIZED_FILE} Q4_K_M")
    else:
        print("=" * 60)
        print("❌ GGUF conversion not completed")
        print()
        print("To manually convert, install llama.cpp:")
        print("  git clone https://github.com/ggerganov/llama.cpp ~/llama.cpp")
        print("  cd ~/llama.cpp && pip install -r requirements.txt")
        print()
        print("Then run:")
        print(f"  python ~/llama.cpp/convert_hf_to_gguf.py {MODEL_DIR} \\")
        print(f"    --outfile {OUTPUT_FILE} --outtype f16")
        sys.exit(1)


if __name__ == "__main__":
    main()
