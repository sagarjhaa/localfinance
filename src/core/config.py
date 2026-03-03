#!/usr/bin/env python3
"""
LocalFinance User Configuration
Handles user preferences including date formats, categories, etc.
"""

import json
from pathlib import Path
from typing import Optional

PROJECT_ROOT = Path(__file__).parent.parent.parent
DEFAULT_CONFIG_PATH = PROJECT_ROOT / "data" / "config.json"


# Supported date formats with examples
DATE_FORMATS = {
    "us": {
        "format": "%m/%d/%Y",
        "example": "03/02/2026",
        "description": "US (MM/DD/YYYY)"
    },
    "eu": {
        "format": "%d/%m/%Y",
        "example": "02/03/2026",
        "description": "European (DD/MM/YYYY)"
    },
    "iso": {
        "format": "%Y-%m-%d",
        "example": "2026-03-02",
        "description": "ISO (YYYY-MM-DD)"
    },
    "us_dash": {
        "format": "%m-%d-%Y",
        "example": "03-02-2026",
        "description": "US with dashes (MM-DD-YYYY)"
    },
    "eu_dash": {
        "format": "%d-%m-%Y",
        "example": "02-03-2026",
        "description": "European with dashes (DD-MM-YYYY)"
    }
}


DEFAULT_CONFIG = {
    "date_format": "us",  # Default to US format
    "currency": "USD",
    "currency_symbol": "$",
    "categories": {
        # Shopping
        "AMAZON": "Shopping",
        "TARGET": "Shopping",
        "WALMART": "Shopping",
        "COSTCO": "Shopping",
        # Groceries
        "WHOLE FOODS": "Groceries",
        "TRADER JOE": "Groceries",
        "SAFEWAY": "Groceries",
        "GROCERY": "Groceries",
        "KROGER": "Groceries",
        # Dining
        "UBER EATS": "Dining",
        "DOORDASH": "Dining",
        "GRUBHUB": "Dining",
        "RESTAURANT": "Dining",
        "STARBUCKS": "Dining",
        "MCDONALD": "Dining",
        "CHIPOTLE": "Dining",
        # Transport
        "UBER": "Transport",
        "LYFT": "Transport",
        "PARKING": "Transport",
        # Gas
        "SHELL": "Gas",
        "CHEVRON": "Gas",
        "EXXON": "Gas",
        "GAS": "Gas",
        # Subscriptions
        "NETFLIX": "Subscriptions",
        "SPOTIFY": "Subscriptions",
        "HULU": "Subscriptions",
        "DISNEY": "Subscriptions",
        "APPLE.COM/BILL": "Subscriptions",
        # Healthcare
        "PHARMACY": "Healthcare",
        "CVS": "Healthcare",
        "WALGREENS": "Healthcare",
        "DOCTOR": "Healthcare",
        "HOSPITAL": "Healthcare",
        # Utilities
        "ELECTRIC": "Utilities",
        "WATER": "Utilities",
        "INTERNET": "Utilities",
        "PHONE": "Utilities",
    },
    "ignore_patterns": [
        "PAYMENT THANK YOU",
        "AUTOPAY",
        "CREDIT CARD PAYMENT",
    ]
}


class UserConfig:
    """Manages user configuration."""
    
    def __init__(self, config_path: Optional[Path] = None):
        self.config_path = config_path or DEFAULT_CONFIG_PATH
        self._config = None
        self._load()
    
    def _load(self):
        """Load config from file or create default."""
        if self.config_path.exists():
            try:
                with open(self.config_path) as f:
                    self._config = json.load(f)
            except json.JSONDecodeError:
                self._config = DEFAULT_CONFIG.copy()
        else:
            self._config = DEFAULT_CONFIG.copy()
    
    def save(self):
        """Save config to file."""
        self.config_path.parent.mkdir(parents=True, exist_ok=True)
        with open(self.config_path, 'w') as f:
            json.dump(self._config, f, indent=2)
    
    @property
    def date_format(self) -> str:
        """Get the strftime format string for dates."""
        fmt_key = self._config.get("date_format", "us")
        return DATE_FORMATS.get(fmt_key, DATE_FORMATS["us"])["format"]
    
    @property
    def date_format_name(self) -> str:
        """Get human-readable date format name."""
        fmt_key = self._config.get("date_format", "us")
        return DATE_FORMATS.get(fmt_key, DATE_FORMATS["us"])["description"]
    
    def set_date_format(self, format_key: str) -> bool:
        """Set date format by key (us, eu, iso, etc.)."""
        if format_key not in DATE_FORMATS:
            return False
        self._config["date_format"] = format_key
        self.save()
        return True
    
    @property
    def categories(self) -> dict:
        """Get category mapping."""
        return self._config.get("categories", DEFAULT_CONFIG["categories"])
    
    def add_category_rule(self, keyword: str, category: str):
        """Add a category rule."""
        if "categories" not in self._config:
            self._config["categories"] = {}
        self._config["categories"][keyword.upper()] = category
        self.save()
    
    @property
    def ignore_patterns(self) -> list:
        """Get patterns to ignore during import."""
        return self._config.get("ignore_patterns", DEFAULT_CONFIG["ignore_patterns"])
    
    def categorize(self, description: str) -> str:
        """Categorize a transaction description."""
        desc_upper = description.upper()
        for keyword, category in self.categories.items():
            if keyword in desc_upper:
                return category
        return "Other"
    
    def should_ignore(self, description: str) -> bool:
        """Check if transaction should be ignored."""
        desc_upper = description.upper()
        return any(pattern in desc_upper for pattern in self.ignore_patterns)


# Singleton instance
_config_instance = None

def get_config(config_path: Optional[Path] = None) -> UserConfig:
    """Get or create config instance."""
    global _config_instance
    if _config_instance is None:
        _config_instance = UserConfig(config_path)
    return _config_instance


def list_date_formats() -> str:
    """Return formatted list of available date formats."""
    lines = ["**Available Date Formats:**\n"]
    for key, info in DATE_FORMATS.items():
        lines.append(f"• `{key}` — {info['description']} (e.g., {info['example']})")
    return "\n".join(lines)
