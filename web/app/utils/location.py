"""
Нормализация локаций для гибкого поиска.

Преобразует строку локации в набор токенов с алиасами:
- провинции Канады (on ↔ ontario, bc ↔ british columbia)
- города GTA (mississauga, brampton → добавляет gta, toronto)

Используется при поиске объявлений (services/listings.py).
"""

import re


def build_location_search(location: str) -> str:
    """Нормализует location для более гибкого поиска.

    Добавляет алиасы провинций и простые локальные расширения.
    """
    if not location:
        return ""

    raw = location.strip().lower()

    province_aliases = {
        "on": "ontario",
        "ontario": "on",
        "bc": "british columbia",
        "british columbia": "bc",
        "ab": "alberta",
        "alberta": "ab",
        "qc": "quebec",
        "quebec": "qc",
        "mb": "manitoba",
        "manitoba": "mb",
        "sk": "saskatchewan",
        "saskatchewan": "sk",
        "ns": "nova scotia",
        "nova scotia": "ns",
        "nb": "new brunswick",
        "new brunswick": "nb",
        "pe": "prince edward island",
        "prince edward island": "pe",
        "nl": "newfoundland labrador",
        "newfoundland": "nl",
        "newfoundland labrador": "nl",
    }

    clean = re.sub(r"[^a-z0-9\s,]", " ", raw)
    parts = [p for p in re.split(r"[\s,]+", clean) if p]

    expanded = set(parts)

    joined = " ".join(parts)
    if joined in province_aliases:
        expanded.add(province_aliases[joined])

    for part in parts:
        if part in province_aliases:
            expanded.add(province_aliases[part])

    if {"mississauga", "toronto", "brampton", "vaughan"} & expanded:
        expanded.add("gta")
        expanded.add("greater toronto area")

    return " ".join(sorted(expanded))