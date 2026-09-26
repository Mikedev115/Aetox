"""Hidden tests for F1 (PromptPay payload). Structural: any tag order, any amount
spelling the standard allows, is accepted; what must hold is what a bank app reads.

Run with BENCH_WORKSPACE pointing at the folder holding the run's promptpay.py.
"""
from __future__ import annotations

import importlib.util
import os
import re
import subprocess
import sys
from decimal import Decimal
from pathlib import Path

import pytest

WS = Path(os.environ["BENCH_WORKSPACE"]).resolve()
AID = "A000000677010111"


def load():
    spec = importlib.util.spec_from_file_location("promptpay_under_test", WS / "promptpay.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


@pytest.fixture(scope="module")
def pp():
    return load()


def crc16(data: str) -> str:
    crc = 0xFFFF
    for byte in data.encode("ascii"):
        crc ^= byte << 8
        for _ in range(8):
            crc = ((crc << 1) ^ 0x1021) if crc & 0x8000 else (crc << 1)
            crc &= 0xFFFF
    return f"{crc:04X}"


def tlv(text: str) -> dict:
    out, i = {}, 0
    while i < len(text):
        tag, size = text[i:i + 2], int(text[i + 2:i + 4])
        out[tag] = text[i + 4:i + 4 + size]
        i += 4 + size
    return out


def check(payload: str, kind: str, value: str, amount: Decimal | None):
    assert re.fullmatch(r"[0-9A-Za-z.]+", payload), payload
    assert payload[-8:-4] == "6304", "CRC field must close the payload"
    assert payload[-4:] == crc16(payload[:-4]), "CRC-16/CCITT-FALSE over everything before it, upper-case hex"
    fields = tlv(payload[:-8])
    assert fields.get("00") == "01"
    assert fields.get("01") == ("11" if amount is None else "12")
    assert fields.get("58") == "TH"
    assert fields.get("53") == "764"
    inner = tlv(fields["29"])
    assert inner.get("00") == AID
    assert inner.get(kind) == value
    if amount is None:
        assert "54" not in fields
    else:
        assert re.fullmatch(r"\d+(\.\d{1,2})?", fields["54"]), fields["54"]
        assert Decimal(fields["54"]) == amount


def valid_id(first12: str) -> str:
    total = sum(int(d) * (13 - i) for i, d in enumerate(first12))
    return first12 + str((11 - total % 11) % 10)


@pytest.mark.parametrize("target", ["0812345678", "081-234-5678", "+66 81 234 5678", "+66812345678"])
def test_phone_spellings_give_one_payload(pp, target):
    check(pp.build_payload(target), "01", "0066812345678", None)


def test_personal_id_with_valid_check_digit(pp):
    pid = valid_id("110170023070")
    check(pp.build_payload(pid), "02", pid, None)
    dashed = f"{pid[0]}-{pid[1:5]}-{pid[5:10]}-{pid[10:12]}-{pid[12]}"
    check(pp.build_payload(dashed), "02", pid, None)


@pytest.mark.parametrize("bad", ["1111111111111", "3100500123456"])
def test_personal_id_with_wrong_check_digit_is_refused(pp, bad):
    if valid_id(bad[:12]) == bad:
        pytest.skip("fixture accidentally valid")
    with pytest.raises(ValueError):
        pp.build_payload(bad)


def test_juristic_tax_id_is_not_held_to_the_personal_check(pp):
    check(pp.build_payload("0123456789012"), "02", "0123456789012", None)


def test_ewallet(pp):
    check(pp.build_payload("012345678901234"), "03", "012345678901234", None)


@pytest.mark.parametrize("amount,expected", [
    (4.22, Decimal("4.22")), ("1,250.50", Decimal("1250.50")), (100, Decimal("100")), ("0.5", Decimal("0.5")),
])
def test_amounts(pp, amount, expected):
    check(pp.build_payload("0812345678", amount), "01", "0066812345678", expected)


@pytest.mark.parametrize("amount", [0, -5, "12.345", "abc"])
def test_bad_amounts_are_refused(pp, amount):
    with pytest.raises(ValueError):
        pp.build_payload("0812345678", amount)


@pytest.mark.parametrize("target", ["12345", "abcdefghij", "", "08123456789", "+1 202 555 0100"])
def test_bad_targets_are_refused(pp, target):
    with pytest.raises(ValueError):
        pp.build_payload(target)


@pytest.mark.parametrize("target,amount,kind,value", [
    ("081-234-5678", None, "phone", "0812345678"),
    ("0123456789012", "99.5", "id", "0123456789012"),
    ("012345678901234", 20, "ewallet", "012345678901234"),
])
def test_parse_round_trip(pp, target, amount, kind, value):
    parsed = pp.parse_payload(pp.build_payload(target, amount))
    assert parsed["type"] == kind
    assert parsed["target"] == value
    if amount is None:
        assert parsed["amount"] is None
    else:
        assert Decimal(str(parsed["amount"])) == Decimal(str(amount))


def test_parse_refuses_a_broken_crc(pp):
    good = pp.build_payload("0812345678", 10)
    bad = good[:-4] + ("0000" if good[-4:] != "0000" else "FFFF")
    with pytest.raises(ValueError):
        pp.parse_payload(bad)


def test_cli_prints_the_payload():
    run = subprocess.run([sys.executable, str(WS / "promptpay.py"), "0812345678", "50"],
                         capture_output=True, text=True, timeout=30, cwd=WS)
    assert run.returncode == 0, run.stderr
    check(run.stdout.strip().splitlines()[-1], "01", "0066812345678", Decimal("50"))


def test_cli_refuses_with_exit_code_2():
    # Python itself exits 2 when the file is missing; that is not a refusal.
    assert (WS / "promptpay.py").is_file()
    run = subprocess.run([sys.executable, str(WS / "promptpay.py"), "12345"],
                         capture_output=True, text=True, timeout=30, cwd=WS)
    assert run.returncode == 2
    assert run.stderr.strip()
