"""
Утилита для маскирования VIN на публичных страницах.

Пример: 1HGCM82633A123456 → 1HG**********3456
"""

def mask_vin(vin: str | None):
    """Скрывает VIN для публичной страницы.

    Пример:
    1HGCM82633A123456 -> 1HG**********3456
    """
    if not vin:
        return None

    vin = vin.strip()

    if len(vin) <= 6:
        return vin

    return vin[:3] + "*" * (len(vin) - 7) + vin[-4:]