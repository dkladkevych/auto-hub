# Auto-Hub Deal Validation
Version: v0.1 (Manual)

## Overview

This document defines how to quickly evaluate used car listings under $15k and assign a risk level.

### Goal:

- filter out scams and bad deals
- identify solid listings
- provide buyers with safer options

---

## Risk Model

Each listing is evaluated using a point-based system.

- 0–3 → LOW RISK
- 4–7 → MEDIUM RISK
- 8+ → HIGH RISK (avoid)

---

## Validation Checklist

### 1. Year vs Mileage

#### Rule:

- ~15–20k km/year is normal

#### Score:

normal → 0
slightly high → +1
very high → +2

---

### 2. Photos

#### Check:

- multiple angles
- interior visible
- no stock images

#### Score:

- clear, multiple → 0
- average → +1
- poor / few → +2

---

### 3. Description Quality

#### Check:

- detailed info
- condition mentioned

#### Score:

- clear description → 0
- minimal → +1
- vague / hype (“urgent sale”) → +2

---

### 4. Seller Profile

#### Check:

- account age
- activity

#### Score:

- normal profile → 0
- slightly suspicious → +1
- new / empty → +2

---

### 5. Price Sanity

#### Check:

- compared to market

#### Score:

- realistic → 0
- slightly under → +1
- too cheap → +2

---

### 6. VIN Availability

#### Check:

- VIN provided or available

#### Score:

- provided → 0
- “later” → +1
- refuses → +2

---

### 7. Visual Condition

#### Check:

- rust
- interior wear
- overall condition

#### Score:

- clean → 0
- some issues → +1
- visibly poor → +2

---

### 8. Communication

#### Check:

- response clarity
- willingness to answer

#### Score:

- clear → 0
- short / vague → +1
- evasive → +2

---

## Red Flags (Immediate Reject)
- seller claims to be “out of country”
- asks for deposit before meeting
- refuses inspection
- inconsistent story
- stock / copied images

---

## Notes

- Condition > mileage
- Trust your intuition (add +1 if unsure)
- Do not overanalyze — quick filtering only