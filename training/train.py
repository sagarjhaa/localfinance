#!/usr/bin/env python3
"""
LocalFinance AI - Model Training Script
Fine-tunes a small model on our finance Q&A data.

Usage:
    cd /Users/sagarjha/projects/localfinance
    source .venv/bin/activate
    python training/train.py
"""

import json
import torch
from pathlib import Path
from datetime import datetime
from transformers import (
    AutoTokenizer,
    AutoModelForCausalLM,
    TrainingArguments,
    Trainer,
    DataCollatorForLanguageModeling,
)
from datasets import Dataset

# Configuration
BASE_MODEL = "Qwen/Qwen2-0.5B-Instruct"  # Small, fast, good at code/SQL
OUTPUT_DIR = Path(__file__).parent.parent / "models" / "localfinance-v1"
DATA_PATH = Path(__file__).parent / "data" / "combined.jsonl"

# Training parameters (optimized for CPU/small GPU)
EPOCHS = 3
BATCH_SIZE = 4
LEARNING_RATE = 2e-5
MAX_LENGTH = 512

def load_training_data():
    """Load and format training data."""
    print(f"📂 Loading data from {DATA_PATH}")
    
    examples = []
    with open(DATA_PATH) as f:
        for line in f:
            ex = json.loads(line)
            # Format as instruction-following
            prompt = f"""Convert this finance question to SQL.

Question: {ex['instruction']}
SQL: {ex['output']}"""
            examples.append({"text": prompt})
    
    print(f"   Loaded {len(examples):,} examples")
    return Dataset.from_list(examples)

def tokenize_data(dataset, tokenizer):
    """Tokenize the dataset."""
    def tokenize_fn(examples):
        return tokenizer(
            examples["text"],
            truncation=True,
            max_length=MAX_LENGTH,
            padding="max_length",
        )
    
    return dataset.map(tokenize_fn, batched=True, remove_columns=["text"])

def main():
    print("=" * 60)
    print("🏦 LocalFinance AI - Model Training")
    print("=" * 60)
    print(f"⏰ Started: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"🤖 Base model: {BASE_MODEL}")
    print(f"📁 Output: {OUTPUT_DIR}")
    
    # Use CPU for Intel Mac (MPS has compatibility issues)
    device = "cpu"
    print(f"🖥️  Device: {device}")
    print("⚠️  Training on CPU - this will be slow but works!")
    
    # Load tokenizer and model
    print("\n📥 Loading model and tokenizer...")
    tokenizer = AutoTokenizer.from_pretrained(BASE_MODEL)
    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token
    
    model = AutoModelForCausalLM.from_pretrained(
        BASE_MODEL,
        torch_dtype=torch.float32,
        low_cpu_mem_usage=True,
    )
    model = model.to("cpu")
    
    print(f"   Model loaded: {model.num_parameters():,} parameters")
    
    # Load and tokenize data
    print("\n📊 Preparing training data...")
    dataset = load_training_data()
    tokenized_dataset = tokenize_data(dataset, tokenizer)
    
    # Split train/eval
    split = tokenized_dataset.train_test_split(test_size=0.1, seed=42)
    train_dataset = split["train"]
    eval_dataset = split["test"]
    
    print(f"   Train: {len(train_dataset):,} examples")
    print(f"   Eval: {len(eval_dataset):,} examples")
    
    # Training arguments
    training_args = TrainingArguments(
        output_dir=str(OUTPUT_DIR),
        num_train_epochs=EPOCHS,
        per_device_train_batch_size=BATCH_SIZE,
        per_device_eval_batch_size=BATCH_SIZE,
        learning_rate=LEARNING_RATE,
        weight_decay=0.01,
        logging_steps=10,
        eval_strategy="epoch",
        save_strategy="epoch",
        load_best_model_at_end=True,
        report_to="none",  # No wandb
        fp16=False if device == "cpu" else True,
        dataloader_num_workers=0,  # Avoid multiprocessing issues
    )
    
    # Data collator
    data_collator = DataCollatorForLanguageModeling(
        tokenizer=tokenizer,
        mlm=False,  # Causal LM, not masked
    )
    
    # Trainer
    trainer = Trainer(
        model=model,
        args=training_args,
        train_dataset=train_dataset,
        eval_dataset=eval_dataset,
        data_collator=data_collator,
    )
    
    # Train!
    print("\n🚀 Starting training...")
    print("-" * 40)
    
    trainer.train()
    
    print("-" * 40)
    print("\n✅ Training complete!")
    
    # Save final model
    print(f"\n💾 Saving model to {OUTPUT_DIR}")
    trainer.save_model(str(OUTPUT_DIR))
    tokenizer.save_pretrained(str(OUTPUT_DIR))
    
    # Save training info
    info = {
        "base_model": BASE_MODEL,
        "training_examples": len(train_dataset),
        "epochs": EPOCHS,
        "trained_at": datetime.now().isoformat(),
    }
    with open(OUTPUT_DIR / "training_info.json", "w") as f:
        json.dump(info, f, indent=2)
    
    print("\n🎉 Done!")
    print(f"⏰ Finished: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")

if __name__ == "__main__":
    main()
