#!/usr/bin/env python3
"""
LocalFinance Statement Parser
Handles PDF and CSV bank statement imports.
"""

import re
import csv
import logging
from pathlib import Path
from datetime import datetime
from typing import Optional
from dataclasses import dataclass

# Setup logging
logger = logging.getLogger(__name__)

# Try to import pdfplumber
try:
    import pdfplumber
    PDF_SUPPORT = True
except ImportError:
    PDF_SUPPORT = False
    logger.warning("pdfplumber not installed. PDF support disabled.")


@dataclass
class Transaction:
    """A single financial transaction."""
    date: str
    description: str
    amount: float
    category: str
    source: str
    
    def to_dict(self) -> dict:
        return {
            "date": self.date,
            "description": self.description,
            "amount": self.amount,
            "category": self.category,
            "source": self.source
        }


@dataclass 
class ParseResult:
    """Result of parsing a statement."""
    success: bool
    transactions: list
    source: str
    error: Optional[str] = None
    warnings: list = None
    
    def __post_init__(self):
        if self.warnings is None:
            self.warnings = []


class StatementParser:
    """Parses bank statements (PDF/CSV) into transactions."""
    
    # Supported banks
    SUPPORTED_BANKS = [
        "Chase",
        "Capital One",
        "American Express",
        "Bank of America",
        "Wells Fargo",
        "Discover",
        "Citi",
        "Generic CSV"
    ]
    
    def __init__(self, categorize_fn=None):
        """
        Initialize parser.
        
        Args:
            categorize_fn: Function to categorize transactions (desc -> category)
        """
        self.categorize = categorize_fn or (lambda x: "Other")
    
    def parse_file(self, file_path: Path) -> ParseResult:
        """
        Parse a statement file (PDF or CSV).
        
        Returns ParseResult with success status and transactions.
        """
        file_path = Path(file_path)
        
        if not file_path.exists():
            return ParseResult(
                success=False,
                transactions=[],
                source="Unknown",
                error=f"File not found: {file_path.name}"
            )
        
        suffix = file_path.suffix.lower()
        
        if suffix == '.pdf':
            if not PDF_SUPPORT:
                return ParseResult(
                    success=False,
                    transactions=[],
                    source="Unknown",
                    error="PDF support not available. Install pdfplumber: pip install pdfplumber"
                )
            return self._parse_pdf(file_path)
        
        elif suffix == '.csv':
            return self._parse_csv(file_path)
        
        else:
            return ParseResult(
                success=False,
                transactions=[],
                source="Unknown",
                error=f"Unsupported file type: {suffix}. Use PDF or CSV."
            )
    
    def _parse_pdf(self, file_path: Path) -> ParseResult:
        """Parse a PDF statement."""
        try:
            with pdfplumber.open(file_path) as pdf:
                # Read first page to detect bank
                first_page = pdf.pages[0].extract_text() or ""
                
                # Detect bank and use appropriate parser
                if "Chase" in first_page:
                    return self._parse_chase_pdf(pdf, file_path)
                elif "Capital One" in first_page:
                    return self._parse_capital_one_pdf(pdf, file_path)
                elif "American Express" in first_page or "AMEX" in first_page:
                    return self._parse_amex_pdf(pdf, file_path)
                elif "Bank of America" in first_page:
                    return self._parse_bofa_pdf(pdf, file_path)
                else:
                    # Try generic parsing
                    return self._parse_generic_pdf(pdf, file_path)
                    
        except Exception as e:
            logger.exception(f"Error parsing PDF: {e}")
            return ParseResult(
                success=False,
                transactions=[],
                source="Unknown",
                error=f"Failed to parse PDF: {str(e)}"
            )
    
    def _parse_chase_pdf(self, pdf, file_path: Path) -> ParseResult:
        """Parse Chase credit card statement."""
        transactions = []
        warnings = []
        source = "Chase"
        
        # Detect card type from first page
        first_page_text = pdf.pages[0].extract_text() or ""
        if "Freedom" in first_page_text:
            source = "Chase Freedom"
        elif "Sapphire" in first_page_text:
            source = "Chase Sapphire"
        
        # Extract year from filename or statement
        year = self._extract_year(file_path, first_page_text)
        
        for page in pdf.pages:
            text = page.extract_text() or ""
            
            # Chase pattern: MM/DD Description Amount
            pattern = r'(\d{2}/\d{2})\s+(.+?)\s+(-?[\d,]+\.\d{2})$'
            
            for line in text.split('\n'):
                match = re.match(pattern, line.strip())
                if match:
                    date_str, description, amount = match.groups()
                    
                    # Parse date
                    month, day = date_str.split('/')
                    full_date = f"{year}-{month}-{day}"
                    
                    # Parse amount (negative = charge)
                    amount_float = abs(float(amount.replace(',', '')))
                    
                    # Skip payments
                    if "PAYMENT" in description.upper() or "AUTOPAY" in description.upper():
                        continue
                    
                    transactions.append(Transaction(
                        date=full_date,
                        description=description.strip(),
                        amount=amount_float,
                        category=self.categorize(description),
                        source=source
                    ))
        
        if not transactions:
            warnings.append("No transactions found. Statement format may not be recognized.")
        
        return ParseResult(
            success=len(transactions) > 0,
            transactions=[t.to_dict() for t in transactions],
            source=source,
            warnings=warnings
        )
    
    def _parse_capital_one_pdf(self, pdf, file_path: Path) -> ParseResult:
        """Parse Capital One credit card statement."""
        transactions = []
        warnings = []
        source = "Capital One"
        
        first_page_text = pdf.pages[0].extract_text() or ""
        
        # Detect card type
        if "Savor" in first_page_text:
            source = "Capital One Savor"
        elif "Quicksilver" in first_page_text:
            source = "Capital One Quicksilver"
        elif "Venture" in first_page_text:
            source = "Capital One Venture"
        
        year = self._extract_year(file_path, first_page_text)
        
        for page in pdf.pages:
            text = page.extract_text() or ""
            
            # Capital One pattern: Mon DD Mon DD Description $Amount
            pattern = r'(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d+)\s+(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d+)\s+(.+?)\s+\$?([\d,]+\.\d{2})'
            
            matches = re.findall(pattern, text)
            
            for match in matches:
                trans_month, trans_day, _, _, description, amount = match
                
                month_map = {"Jan": "01", "Feb": "02", "Mar": "03", "Apr": "04",
                            "May": "05", "Jun": "06", "Jul": "07", "Aug": "08",
                            "Sep": "09", "Oct": "10", "Nov": "11", "Dec": "12"}
                
                month = month_map[trans_month]
                full_date = f"{year}-{month}-{int(trans_day):02d}"
                
                amount_float = float(amount.replace(',', ''))
                
                # Skip payments
                if "AUTOPAY" in description.upper() or "PAYMENT" in description.upper():
                    continue
                
                transactions.append(Transaction(
                    date=full_date,
                    description=description.strip(),
                    amount=amount_float,
                    category=self.categorize(description),
                    source=source
                ))
        
        if not transactions:
            warnings.append("No transactions found. Statement format may not be recognized.")
        
        return ParseResult(
            success=len(transactions) > 0,
            transactions=[t.to_dict() for t in transactions],
            source=source,
            warnings=warnings
        )
    
    def _parse_amex_pdf(self, pdf, file_path: Path) -> ParseResult:
        """Parse American Express statement."""
        transactions = []
        warnings = []
        source = "American Express"
        
        first_page_text = pdf.pages[0].extract_text() or ""
        year = self._extract_year(file_path, first_page_text)
        
        for page in pdf.pages:
            text = page.extract_text() or ""
            
            # Amex pattern varies, try common format
            pattern = r'(\d{2}/\d{2}/\d{2,4})\s+(.+?)\s+\$?([\d,]+\.\d{2})'
            
            for line in text.split('\n'):
                match = re.search(pattern, line)
                if match:
                    date_str, description, amount = match.groups()
                    
                    # Parse various date formats
                    try:
                        if len(date_str) == 8:  # MM/DD/YY
                            dt = datetime.strptime(date_str, "%m/%d/%y")
                        else:  # MM/DD/YYYY
                            dt = datetime.strptime(date_str, "%m/%d/%Y")
                        full_date = dt.strftime("%Y-%m-%d")
                    except ValueError:
                        continue
                    
                    amount_float = float(amount.replace(',', ''))
                    
                    transactions.append(Transaction(
                        date=full_date,
                        description=description.strip(),
                        amount=amount_float,
                        category=self.categorize(description),
                        source=source
                    ))
        
        if not transactions:
            warnings.append("No transactions found. Try CSV export instead.")
        
        return ParseResult(
            success=len(transactions) > 0,
            transactions=[t.to_dict() for t in transactions],
            source=source,
            warnings=warnings
        )
    
    def _parse_bofa_pdf(self, pdf, file_path: Path) -> ParseResult:
        """Parse Bank of America statement."""
        # Similar structure, simplified for now
        return ParseResult(
            success=False,
            transactions=[],
            source="Bank of America",
            error="Bank of America PDF parsing not yet implemented. Please use CSV export."
        )
    
    def _parse_generic_pdf(self, pdf, file_path: Path) -> ParseResult:
        """Try to parse unknown PDF format."""
        transactions = []
        warnings = ["Unknown bank format. Results may be incomplete."]
        
        first_page_text = pdf.pages[0].extract_text() or ""
        year = self._extract_year(file_path, first_page_text)
        
        # Try common patterns
        patterns = [
            r'(\d{2}/\d{2}/\d{4})\s+(.+?)\s+\$?([\d,]+\.\d{2})',
            r'(\d{2}/\d{2})\s+(.+?)\s+\$?([\d,]+\.\d{2})',
            r'(\d{4}-\d{2}-\d{2})\s+(.+?)\s+\$?([\d,]+\.\d{2})',
        ]
        
        for page in pdf.pages:
            text = page.extract_text() or ""
            
            for pattern in patterns:
                matches = re.findall(pattern, text)
                for match in matches:
                    date_str, description, amount = match
                    
                    # Normalize date
                    if len(date_str) == 5:  # MM/DD
                        full_date = f"{year}-{date_str.replace('/', '-')}"
                    elif '/' in date_str:
                        parts = date_str.split('/')
                        if len(parts[2]) == 2:
                            full_date = f"20{parts[2]}-{parts[0]}-{parts[1]}"
                        else:
                            full_date = f"{parts[2]}-{parts[0]}-{parts[1]}"
                    else:
                        full_date = date_str
                    
                    amount_float = float(amount.replace(',', ''))
                    
                    transactions.append(Transaction(
                        date=full_date,
                        description=description.strip(),
                        amount=amount_float,
                        category=self.categorize(description),
                        source="Unknown Bank"
                    ))
        
        if not transactions:
            return ParseResult(
                success=False,
                transactions=[],
                source="Unknown",
                error="Could not extract transactions. Try CSV export from your bank."
            )
        
        return ParseResult(
            success=True,
            transactions=[t.to_dict() for t in transactions],
            source="Unknown Bank",
            warnings=warnings
        )
    
    def _parse_csv(self, file_path: Path) -> ParseResult:
        """Parse a CSV statement."""
        transactions = []
        warnings = []
        source = "CSV Import"
        
        try:
            with open(file_path, 'r', encoding='utf-8-sig') as f:
                # Detect delimiter
                sample = f.read(1024)
                f.seek(0)
                
                dialect = csv.Sniffer().sniff(sample)
                reader = csv.DictReader(f, dialect=dialect)
                
                # Normalize column names
                if reader.fieldnames:
                    fieldnames_lower = [fn.lower().strip() for fn in reader.fieldnames]
                else:
                    return ParseResult(
                        success=False,
                        transactions=[],
                        source=source,
                        error="CSV has no headers. Expected columns: date, description, amount"
                    )
                
                # Find columns
                date_col = self._find_column(fieldnames_lower, ['date', 'trans date', 'transaction date', 'posted date'])
                desc_col = self._find_column(fieldnames_lower, ['description', 'desc', 'merchant', 'name', 'payee'])
                amount_col = self._find_column(fieldnames_lower, ['amount', 'debit', 'charge', 'transaction amount'])
                
                if not all([date_col, desc_col, amount_col]):
                    missing = []
                    if not date_col: missing.append("date")
                    if not desc_col: missing.append("description")
                    if not amount_col: missing.append("amount")
                    return ParseResult(
                        success=False,
                        transactions=[],
                        source=source,
                        error=f"Missing required columns: {', '.join(missing)}"
                    )
                
                # Re-read with proper column mapping
                f.seek(0)
                reader = csv.DictReader(f, dialect=dialect)
                
                for row in reader:
                    try:
                        # Get values using original fieldnames
                        date_val = row.get(reader.fieldnames[date_col])
                        desc_val = row.get(reader.fieldnames[desc_col])
                        amount_val = row.get(reader.fieldnames[amount_col])
                        
                        if not all([date_val, desc_val, amount_val]):
                            continue
                        
                        # Parse date (try multiple formats)
                        full_date = self._parse_date(date_val)
                        if not full_date:
                            warnings.append(f"Skipped row with unparseable date: {date_val}")
                            continue
                        
                        # Parse amount
                        amount_str = amount_val.replace('$', '').replace(',', '').strip()
                        if amount_str.startswith('(') and amount_str.endswith(')'):
                            amount_str = '-' + amount_str[1:-1]
                        amount_float = abs(float(amount_str))
                        
                        transactions.append(Transaction(
                            date=full_date,
                            description=desc_val.strip(),
                            amount=amount_float,
                            category=self.categorize(desc_val),
                            source=source
                        ))
                    except (ValueError, KeyError) as e:
                        logger.debug(f"Skipped row: {e}")
                        continue
                
        except Exception as e:
            logger.exception(f"Error parsing CSV: {e}")
            return ParseResult(
                success=False,
                transactions=[],
                source=source,
                error=f"Failed to parse CSV: {str(e)}"
            )
        
        if not transactions:
            return ParseResult(
                success=False,
                transactions=[],
                source=source,
                error="No transactions found in CSV."
            )
        
        return ParseResult(
            success=True,
            transactions=[t.to_dict() for t in transactions],
            source=source,
            warnings=warnings
        )
    
    def _find_column(self, headers: list, candidates: list) -> Optional[int]:
        """Find column index matching any candidate name."""
        for i, header in enumerate(headers):
            if any(c in header for c in candidates):
                return i
        return None
    
    def _parse_date(self, date_str: str) -> Optional[str]:
        """Try to parse date string into YYYY-MM-DD format."""
        formats = [
            "%m/%d/%Y",
            "%m/%d/%y",
            "%Y-%m-%d",
            "%d/%m/%Y",
            "%m-%d-%Y",
            "%Y/%m/%d",
        ]
        
        for fmt in formats:
            try:
                dt = datetime.strptime(date_str.strip(), fmt)
                return dt.strftime("%Y-%m-%d")
            except ValueError:
                continue
        
        return None
    
    def _extract_year(self, file_path: Path, text: str) -> int:
        """Extract year from filename or statement text."""
        # Try filename first
        basename = file_path.name
        
        # YYYYMMDD format
        match = re.match(r'^(\d{4})\d{4}', basename)
        if match:
            return int(match.group(1))
        
        # YYYY-MM-DD format
        match = re.match(r'^(\d{4})-\d{2}-\d{2}', basename)
        if match:
            return int(match.group(1))
        
        # Try to find year in text
        match = re.search(r'(202[0-9])', text)
        if match:
            return int(match.group(1))
        
        # Default to current year
        return datetime.now().year


# Convenience function
def parse_statement(file_path: Path, categorize_fn=None) -> ParseResult:
    """Parse a statement file."""
    parser = StatementParser(categorize_fn)
    return parser.parse_file(file_path)
