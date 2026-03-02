#!/usr/bin/env python3
"""
LocalFinance AI - Model Testing Script
Tests the fine-tuned model with sample finance queries.

Usage:
    cd /Users/sagarjha/projects/localfinance
    source .venv/bin/activate
    python training/test_model.py
"""

import time
import torch
from pathlib import Path
from transformers import AutoTokenizer, AutoModelForCausalLM, StoppingCriteria, StoppingCriteriaList

# Model path
MODEL_DIR = Path(__file__).parent.parent / "models" / "localfinance-v1"


class SQLStoppingCriteria(StoppingCriteria):
    """
    Stop generation when we hit SQL statement terminators.
    
    Stops on:
    - Semicolon (;) - standard SQL terminator
    - Double newline - end of statement block
    - Period followed by non-digit - natural language ending
    """
    
    def __init__(self, tokenizer, prompt_length: int):
        self.tokenizer = tokenizer
        self.prompt_length = prompt_length
        # Pre-compute token IDs for stop sequences
        self.stop_tokens = set()
        for char in [';', '\n']:
            tokens = tokenizer.encode(char, add_special_tokens=False)
            self.stop_tokens.update(tokens)
    
    def __call__(self, input_ids: torch.LongTensor, scores: torch.FloatTensor, **kwargs) -> bool:
        # Only check tokens we generated (not the prompt)
        if input_ids.shape[1] <= self.prompt_length:
            return False
        
        # Check the last generated token
        last_token = input_ids[0, -1].item()
        if last_token in self.stop_tokens:
            return True
        
        # Also check if we've generated a complete SQL statement by decoding
        generated = self.tokenizer.decode(input_ids[0, self.prompt_length:], skip_special_tokens=True)
        
        # Stop if we have a semicolon anywhere in generated text
        if ';' in generated:
            return True
        
        # Stop if we have multiple newlines (statement complete)
        if '\n\n' in generated:
            return True
            
        return False

# Sample finance queries to test
TEST_QUERIES = [
    "How much did I spend on dining last month?",
    "Show me all Amazon purchases",
    "What's my total spending this week?",
    "Find all grocery transactions",
    "How much did I spend on entertainment in January?",
    "Show me transactions over $100",
    "What did I pay for subscriptions?",
    "Set my dining budget to $400",
]


def format_prompt(question: str) -> str:
    """Format a question using the training prompt format."""
    return f"""Convert this finance question to SQL.

Question: {question}
SQL:"""


def extract_sql(generated_text: str, prompt: str) -> str:
    """Extract clean SQL from the generated text."""
    import re
    
    # Remove the prompt from the output
    sql = generated_text[len(prompt):].strip()
    
    # Take only the first line (the SQL statement)
    sql = sql.split('\n')[0].strip()
    
    # Stop at first semicolon (include it)
    if ';' in sql:
        sql = sql.split(';')[0] + ';'
    
    # Remove trailing garbage patterns
    # Pattern: ". " followed by anything (natural language intrusion)
    sql = re.sub(r'\.\s+.*$', ';', sql)
    
    # Pattern: ")" followed by garbage that's not part of SQL
    # Look for unbalanced parentheses followed by junk
    paren_count = 0
    clean_end = len(sql)
    for i, char in enumerate(sql):
        if char == '(':
            paren_count += 1
        elif char == ')':
            paren_count -= 1
            if paren_count < 0:
                # Unbalanced - cut here
                clean_end = i
                break
    sql = sql[:clean_end]
    
    # Pattern: Multiple WHERE/AND/ORDER BY clauses (repetition)
    for keyword in ['WHERE', 'ORDER BY', 'GROUP BY', 'LIMIT']:
        parts = sql.split(keyword)
        if len(parts) > 2:
            # Keep only first occurrence
            sql = keyword.join(parts[:2])
    
    # Clean up and ensure proper ending
    sql = sql.strip()
    if sql and not sql.endswith(';'):
        sql += ';'
    
    return sql


def main():
    print("=" * 60)
    print("🏦 LocalFinance AI - Model Testing")
    print("=" * 60)
    print(f"📁 Loading model from: {MODEL_DIR}")
    
    # Load model and tokenizer
    load_start = time.time()
    tokenizer = AutoTokenizer.from_pretrained(MODEL_DIR)
    model = AutoModelForCausalLM.from_pretrained(
        MODEL_DIR,
        torch_dtype=torch.float32,
        low_cpu_mem_usage=True,
    )
    model.eval()
    load_time = time.time() - load_start
    
    print(f"✅ Model loaded in {load_time:.2f}s")
    print(f"📊 Parameters: {model.num_parameters():,}")
    print()
    
    # Test each query
    print("-" * 60)
    print("🧪 Running test queries...")
    print("-" * 60)
    
    inference_times = []
    
    for i, query in enumerate(TEST_QUERIES, 1):
        prompt = format_prompt(query)
        
        # Tokenize
        inputs = tokenizer(prompt, return_tensors="pt")
        prompt_length = inputs['input_ids'].shape[1]
        
        # Create stopping criteria for this prompt
        stopping_criteria = StoppingCriteriaList([
            SQLStoppingCriteria(tokenizer, prompt_length)
        ])
        
        # Generate with stop tokens
        start_time = time.time()
        with torch.no_grad():
            outputs = model.generate(
                **inputs,
                max_new_tokens=150,  # Allow more tokens, stopping criteria will cut off
                num_beams=1,  # Greedy for speed
                do_sample=False,
                pad_token_id=tokenizer.pad_token_id,
                eos_token_id=tokenizer.eos_token_id,
                stopping_criteria=stopping_criteria,
            )
        inference_time = time.time() - start_time
        inference_times.append(inference_time)
        
        # Decode and extract SQL
        generated_text = tokenizer.decode(outputs[0], skip_special_tokens=True)
        sql = extract_sql(generated_text, prompt)
        
        print(f"\n[{i}/{len(TEST_QUERIES)}] Query: {query}")
        print(f"    SQL: {sql}")
        print(f"    ⏱️  {inference_time:.3f}s")
    
    # Summary statistics
    print("\n" + "=" * 60)
    print("📈 Inference Statistics")
    print("=" * 60)
    print(f"Total queries:     {len(TEST_QUERIES)}")
    print(f"Total time:        {sum(inference_times):.2f}s")
    print(f"Avg time/query:    {sum(inference_times) / len(inference_times):.3f}s")
    print(f"Min time:          {min(inference_times):.3f}s")
    print(f"Max time:          {max(inference_times):.3f}s")
    print()
    print("✅ Testing complete!")


if __name__ == "__main__":
    main()
