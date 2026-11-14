from datetime import datetime, timezone

from signalops_dagster.definitions import _scheduled_session_date


def test_scheduled_session_date_uses_new_york_calendar_day_after_utc_midnight() -> None:
    # 00:15 UTC is still the prior ET trading session during daylight time.
    tick = datetime(2026, 11, 10, 0, 15, tzinfo=timezone.utc)
    assert _scheduled_session_date(tick) == "2026-11-09"


def test_scheduled_session_date_uses_same_et_day_before_midnight() -> None:
    tick = datetime(2026, 11, 9, 23, 45, tzinfo=timezone.utc)
    assert _scheduled_session_date(tick) == "2026-11-09"


def test_manual_session_date_fallback_is_explicitly_available() -> None:
    # A manual run with no scheduled tick remains deterministic at invocation
    # time; production retries should carry the original run tag instead.
    assert isinstance(_scheduled_session_date(None), str)
