#!/usr/bin/env python3
import pdfplumber
p = 'JULY_25.pdf'
with pdfplumber.open(p) as pdf:
    i=0
    for page in pdf.pages:
        text = page.extract_text() or ''
        for line in text.splitlines():
            if 'TDS' in line.upper():
                print(page.page_number, line)
                i+=1
print('Found',i,'TDS lines')
