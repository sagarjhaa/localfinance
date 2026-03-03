#!/usr/bin/env python3
"""
LocalFinance AI Inference Module
Loads the fine-tuned model and generates SQL from natural language queries.
"""

import re
import time
import torch
from pathlib import Path
from transformers import AutoTokenizer, AutoModelForCausalLM, StoppingCriteria, StoppingCriteriaList

# Default model path
DEFAULT_MODEL_DIR = Path(__file__).parent.parent / "models" / "localfinance-v1"


class SQLStoppingCriteria(StoppingCriteria):
    """Stop generation at SQL terminators."""
    
    def __init__(self, tokenizer, prompt_length: int):
        self.tokenizer = tokenizer
        self.prompt_length = prompt_length
        self.stop_tokens = set()
        for char in [';', '\n']:
            tokens = tokenizer.encode(char, add_special_tokens=False)
            self.stop_tokens.update(tokens)
    
    def __call__(self, input_ids: torch.LongTensor, scores: torch.FloatTensor, **kwargs) -> bool:
        if input_ids.shape[1] <= self.prompt_length:
            return False
        
        last_token = input_ids[0, -1].item()
        if last_token in self.stop_tokens:
            return True
        
        generated = self.tokenizer.decode(input_ids[0, self.prompt_length:], skip_special_tokens=True)
        if ';' in generated or '\n\n' in generated:
            return True
            
        return False


def clean_sql(sql: str) -> str:
    """Clean up generated SQL."""
    # Take first line
    sql = sql.split('\n')[0].strip()
    
    # Stop at first semicolon
    if ';' in sql:
        sql = sql.split(';')[0] + ';'
    
    # Remove trailing garbage
    sql = re.sub(r'\.\s+.*$', ';', sql)
    
    # Fix unbalanced parens
    paren_count = 0
    clean_end = len(sql)
    for i, char in enumerate(sql):
        if char == '(':
            paren_count += 1
        elif char == ')':
            paren_count -= 1
            if paren_count < 0:
                clean_end = i
                break
    sql = sql[:clean_end]
    
    # Remove repeated clauses
    for keyword in ['WHERE', 'ORDER BY', 'GROUP BY', 'LIMIT']:
        parts = sql.split(keyword)
        if len(parts) > 2:
            sql = keyword.join(parts[:2])
    
    sql = sql.strip()
    if sql and not sql.endswith(';'):
        sql += ';'
    
    return sql


class LocalFinanceAI:
    """LocalFinance AI model for SQL generation."""
    
    def __init__(self, model_dir=None):
        self.model_dir = Path(model_dir) if model_dir else DEFAULT_MODEL_DIR
        self.model = None
        self.tokenizer = None
        self._loaded = False
    
    def load(self):
        """Load the model and tokenizer."""
        if self._loaded:
            return
        
        print(f"🔄 Loading model from {self.model_dir}...")
        start = time.time()
        
        self.tokenizer = AutoTokenizer.from_pretrained(self.model_dir)
        self.model = AutoModelForCausalLM.from_pretrained(
            self.model_dir,
            torch_dtype=torch.float32,
            low_cpu_mem_usage=True,
        )
        self.model.eval()
        
        self._loaded = True
        print(f"✅ Model loaded in {time.time() - start:.2f}s")
    
    def generate_sql(self, question: str, max_tokens: int = 150) -> tuple[str, float]:
        """
        Generate SQL from a natural language question.
        
        Returns: (sql_query, inference_time_seconds)
        """
        if not self._loaded:
            self.load()
        
        prompt = f"""Convert this finance question to SQL.

Question: {question}
SQL:"""
        
        inputs = self.tokenizer(prompt, return_tensors="pt")
        prompt_length = inputs['input_ids'].shape[1]
        
        stopping_criteria = StoppingCriteriaList([
            SQLStoppingCriteria(self.tokenizer, prompt_length)
        ])
        
        start = time.time()
        with torch.no_grad():
            outputs = self.model.generate(
                **inputs,
                max_new_tokens=max_tokens,
                num_beams=1,
                do_sample=False,
                pad_token_id=self.tokenizer.pad_token_id,
                eos_token_id=self.tokenizer.eos_token_id,
                stopping_criteria=stopping_criteria,
            )
        inference_time = time.time() - start
        
        generated_text = self.tokenizer.decode(outputs[0], skip_special_tokens=True)
        sql = generated_text[len(prompt):].strip()
        sql = clean_sql(sql)
        
        return sql, inference_time
    
    def query(self, question: str) -> dict:
        """
        High-level query interface.
        Returns dict with sql, time, and success status.
        """
        try:
            sql, inference_time = self.generate_sql(question)
            return {
                'success': True,
                'sql': sql,
                'inference_time': inference_time,
                'question': question
            }
        except Exception as e:
            return {
                'success': False,
                'error': str(e),
                'question': question
            }


# Singleton instance for reuse
_model_instance = None

def get_model(model_dir=None) -> LocalFinanceAI:
    """Get or create the model instance (singleton)."""
    global _model_instance
    if _model_instance is None:
        _model_instance = LocalFinanceAI(model_dir)
    return _model_instance


def query(question: str) -> dict:
    """Convenience function to query the model."""
    model = get_model()
    return model.query(question)


if __name__ == "__main__":
    # Test the module
    print("Testing LocalFinance AI inference module...")
    
    test_questions = [
        "How much did I spend on dining last month?",
        "Show me all Amazon purchases",
        "What's my total spending this week?",
    ]
    
    model = get_model()
    model.load()
    
    for q in test_questions:
        result = model.query(q)
        print(f"\nQ: {q}")
        if result['success']:
            print(f"SQL: {result['sql']}")
            print(f"Time: {result['inference_time']:.2f}s")
        else:
            print(f"Error: {result['error']}")
