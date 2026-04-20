"""
Utility for masking VIN on public pages.

Example: 1HGCM82633A123456 → 1HG**********3456
"""

def mask_vin(vin: str | None):
    """Hides VIN for the public page.

    Example:
    1HGCM82633A123456 -> 1HG**********3456
    """
    if not vin:
        return None

    vin = vin.strip()

    if len(vin) <= 6:
        return vin

    return vin[:3] + "*" * (len(vin) - 7) + vin[-4:]