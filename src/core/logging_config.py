#!/usr/bin/env python3
"""
LocalFinance Logging Configuration
Centralized logging setup with file and console output.
"""

import logging
import sys
import hashlib
from pathlib import Path
from datetime import datetime
from logging.handlers import RotatingFileHandler

PROJECT_ROOT = Path(__file__).parent.parent.parent
LOG_DIR = PROJECT_ROOT / "logs"


def generate_error_code(error_type: str, details: str = "") -> str:
    """
    Generate a short error code for user to share.
    Format: LF-XXXX (4 hex chars based on error + timestamp)
    """
    timestamp = datetime.now().strftime("%Y%m%d%H%M")
    hash_input = f"{error_type}:{details}:{timestamp}"
    hash_hex = hashlib.md5(hash_input.encode()).hexdigest()[:4].upper()
    return f"LF-{hash_hex}"


def setup_logging(
    name: str = "localfinance",
    level: int = logging.INFO,
    log_to_file: bool = True,
    log_to_console: bool = True
) -> logging.Logger:
    """
    Set up logging for LocalFinance.
    
    Args:
        name: Logger name
        level: Logging level
        log_to_file: Whether to write to file
        log_to_console: Whether to write to console
    
    Returns:
        Configured logger
    """
    logger = logging.getLogger(name)
    logger.setLevel(level)
    
    # Clear existing handlers
    logger.handlers = []
    
    # Format
    formatter = logging.Formatter(
        '%(asctime)s | %(levelname)-8s | %(name)s | %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S'
    )
    
    if log_to_console:
        console_handler = logging.StreamHandler(sys.stdout)
        console_handler.setLevel(level)
        console_handler.setFormatter(formatter)
        logger.addHandler(console_handler)
    
    if log_to_file:
        LOG_DIR.mkdir(parents=True, exist_ok=True)
        
        # Main log file with rotation
        file_handler = RotatingFileHandler(
            LOG_DIR / f"{name}.log",
            maxBytes=5 * 1024 * 1024,  # 5 MB
            backupCount=3
        )
        file_handler.setLevel(level)
        file_handler.setFormatter(formatter)
        logger.addHandler(file_handler)
        
        # Error log (errors only)
        error_handler = RotatingFileHandler(
            LOG_DIR / f"{name}_errors.log",
            maxBytes=2 * 1024 * 1024,  # 2 MB
            backupCount=2
        )
        error_handler.setLevel(logging.ERROR)
        error_handler.setFormatter(formatter)
        logger.addHandler(error_handler)
    
    return logger


def get_logger(name: str = None) -> logging.Logger:
    """Get a logger, optionally as a child of the main logger."""
    if name:
        return logging.getLogger(f"localfinance.{name}")
    return logging.getLogger("localfinance")


# User-friendly error messages
ERROR_MESSAGES = {
    "file_not_found": "📁 File not found. Please send the file again.",
    "unsupported_format": "📄 Unsupported file format. Please send a PDF or CSV bank statement.",
    "parse_failed": "❌ Couldn't read this statement. Try downloading a CSV from your bank instead.",
    "no_transactions": "📭 No transactions found in this file. Make sure it's a bank statement.",
    "db_error": "💾 Database error. Your data is safe, but please try again.",
    "unknown_bank": "🏦 Unknown bank format. I'll try my best, but results may be incomplete.",
    "model_error": "🤖 AI model error. Try rephrasing your question.",
    "timeout": "⏱️ Request timed out. Please try again.",
    "generic": "Something went wrong. Please try again.",
}


def user_friendly_error(error_key: str, details: str = None) -> str:
    """
    Get a user-friendly error message with error code for support.
    
    Returns message like:
    ❌ Couldn't read this statement...
    
    📋 Error code: LF-A1B2
    Forward this to support if you need help.
    """
    msg = ERROR_MESSAGES.get(error_key, ERROR_MESSAGES["generic"])
    
    # Generate error code
    error_code = generate_error_code(error_key, details or "")
    
    # Log the error with code for correlation
    logger = logging.getLogger("localfinance.errors")
    logger.error(f"[{error_code}] {error_key}: {details}")
    
    # Build user message
    msg += f"\n\n📋 **Error code:** `{error_code}`"
    msg += f"\n_Forward this message to support if you need help._"
    
    if details:
        # Truncate long details
        short_details = details[:100] + "..." if len(details) > 100 else details
        msg += f"\n\n_Technical: {short_details}_"
    
    return msg


def log_error_with_code(error_key: str, details: str = None, exception: Exception = None) -> str:
    """
    Log an error and return the error code.
    Use this when you want to log but build your own message.
    """
    error_code = generate_error_code(error_key, details or "")
    
    logger = logging.getLogger("localfinance.errors")
    if exception:
        logger.exception(f"[{error_code}] {error_key}: {details}")
    else:
        logger.error(f"[{error_code}] {error_key}: {details}")
    
    return error_code
