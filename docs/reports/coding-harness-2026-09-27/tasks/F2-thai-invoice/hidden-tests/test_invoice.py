"""Hidden tests for F2. Money rules: VAT 7% and WHT on the pre-VAT amount, each
rounded half-up to the satang (ROUND_HALF_UP, as the Revenue Department's own
calculators do); baht text in the cheque form (สิบเอ็ด, ยี่สิบ, หนึ่งร้อยเอ็ด,
"บาทถ้วน" for whole amounts); Buddhist-era dates; the 13-digit tax id checksum."""
import datetime as dt
import importlib.util
import os
from decimal import Decimal
from pathlib import Path

import pytest

WS = Path(os.environ["BENCH_WORKSPACE"])
spec = importlib.util.spec_from_file_location("invoice", WS / "invoice.py")
inv = importlib.util.module_from_spec(spec)
spec.loader.exec_module(inv)
D = Decimal


@pytest.mark.parametrize("amount,text", [
    ("0", "ศูนย์บาทถ้วน"),
    ("1", "หนึ่งบาทถ้วน"),
    ("11", "สิบเอ็ดบาทถ้วน"),
    ("21", "ยี่สิบเอ็ดบาทถ้วน"),
    ("101", "หนึ่งร้อยเอ็ดบาทถ้วน"),
    ("1234.50", "หนึ่งพันสองร้อยสามสิบสี่บาทห้าสิบสตางค์"),
    ("1000000", "หนึ่งล้านบาทถ้วน"),
    ("21000011.21", "ยี่สิบเอ็ดล้านสิบเอ็ดบาทยี่สิบเอ็ดสตางค์"),
    ("0.25", "ยี่สิบห้าสตางค์"),
])
def test_baht_text(amount, text):
    assert inv.baht_text(D(amount)) == text


def test_baht_text_accepts_str():
    assert inv.baht_text("15") == "สิบห้าบาทถ้วน"


def test_compute_with_wht():
    r = inv.compute([{"desc": "ออกแบบโลโก้", "qty": 1, "unit": "15000"}, {"desc": "นามบัตร", "qty": 2, "unit": "1250.50"}], 3)
    assert r["subtotal"] == D("17501.00")
    assert r["vat"] == D("1225.07")
    assert r["total"] == D("18726.07")
    assert r["wht"] == D("525.03")
    assert r["net"] == D("18201.04")


def test_compute_rounding_half_up():
    r = inv.compute([{"desc": "x", "qty": 1, "unit": "0.50"}], 3)
    assert r["vat"] == D("0.04")
    assert r["wht"] == D("0.02")


def test_compute_types():
    r = inv.compute([{"desc": "x", "qty": 3, "unit": D("10.10")}], 0)
    assert all(isinstance(v, Decimal) for v in r.values() if not isinstance(v, (list, str)))
    assert r["wht"] == D("0.00") and r["net"] == r["total"]


@pytest.mark.parametrize("d,text", [(dt.date(2026, 9, 25), "25 กันยายน 2569"), (dt.date(2027, 1, 1), "1 มกราคม 2570"),
                                     (dt.date(2024, 2, 29), "29 กุมภาพันธ์ 2567")])
def test_thai_date(d, text):
    assert inv.thai_date(d) == text


def _tax_id(first12):
    s = sum(int(c) * (13 - i) for i, c in enumerate(first12))
    return first12 + str((11 - s % 11) % 10)


def test_tax_id():
    good = _tax_id("010555012345")
    assert inv.valid_tax_id(good)
    bad = good[:-1] + str((int(good[-1]) + 1) % 10)
    assert not inv.valid_tax_id(bad)
    assert not inv.valid_tax_id("12345")
    assert not inv.valid_tax_id("abcdefghijklm")
