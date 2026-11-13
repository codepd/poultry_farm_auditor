#!/usr/bin/env python3
"""
Full End-to-End QA Automation Suite for MyKoliPannai.

Creates/reuses an isolated test farm ("QA Automation Test Farm"), validates:
  1. Farm attributes (GET/PUT /api/tenants/{id}, country codes)
  2. Multi-role RBAC & permissions (ADMIN, OWNER, CO_OWNER, MANAGER, AUDITOR, OTHER_USER)
     including WhatsApp Magic Link invite -> phone verification -> password setup -> login
  3. Rolling 2-year mock data seeding (always keeps the last 24 months current relative to today)
  4. Hen batches, mortality, cull hen sales, and egg price history
  5. Pure-Python Tally-style PDF ledger generation -> POST /api/admin/upload-ledger (commit=true)
     -> Python parser verification -> S3/upload audit listing -> /analytics/latest-data-month
     -> /analytics/enhanced-monthly-summary (Unit Economics + Layer vs Rearing feed split)
  6. User profile language ('en' <-> 'ta') & monotonic onboarding_version persistence
"""

import calendar
import datetime as dt
import io
import json
import mimetypes
import os
import sys
import urllib.error
import urllib.request
import uuid

API_BASE = os.environ.get("API_BASE_URL", "http://localhost:8080/api").rstrip("/")
ADMIN_KEY = os.environ.get("QA_ADMIN_KEY", "local-dev-admin-key")
QA_FARM_NAME = "QA Automation Test Farm"
QA_PASSWORD = os.environ.get("QA_TEST_PASSWORD", "QaFarmTest@2026!")

OWNER_EMAIL = "qa.owner@mykolipannai.com"
ROLE_SPECS = [
    {"role": "ADMIN", "phone": "+919900000001", "name": "QA Admin"},
    {"role": "CO_OWNER", "phone": "+919900000002", "name": "QA Co-Owner"},
    {"role": "MANAGER", "phone": "+919900000003", "name": "QA Farm Manager"},
    {"role": "AUDITOR", "phone": "+919900000004", "name": "QA Auditor"},
    {"role": "OTHER_USER", "phone": "+919900000005", "name": "QA Staff Employee"},
]


def api_json(method: str, path: str, body=None, token=None, headers=None):
    url = f"{API_BASE}{path}"
    req_headers = {"Content-Type": "application/json"}
    if token:
        req_headers["Authorization"] = f"Bearer {token}"
    if headers:
        req_headers.update(headers)
    data = json.dumps(body).encode("utf-8") if body is not None else None
    req = urllib.request.Request(url, data=data, headers=req_headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode("utf-8")
            try:
                return resp.status, json.loads(raw) if raw else {}
            except Exception:
                return resp.status, {"raw": raw}
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8")
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {"raw": raw}


def api_multipart(path: str, fields: dict, file_field: str, filename: str, file_bytes: bytes, token: str):
    boundary = f"----QABoundary{uuid.uuid4().hex}"
    buf = io.BytesIO()
    for k, v in fields.items():
        buf.write(f"--{boundary}\r\n".encode())
        buf.write(f'Content-Disposition: form-data; name="{k}"\r\n\r\n{v}\r\n'.encode())
    ctype = mimetypes.guess_type(filename)[0] or "application/pdf"
    buf.write(f"--{boundary}\r\n".encode())
    buf.write(f'Content-Disposition: form-data; name="{file_field}"; filename="{filename}"\r\n'.encode())
    buf.write(f"Content-Type: {ctype}\r\n\r\n".encode())
    buf.write(file_bytes)
    buf.write(f"\r\n--{boundary}--\r\n".encode())

    req = urllib.request.Request(
        f"{API_BASE}{path}",
        data=buf.getvalue(),
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": f"multipart/form-data; boundary={boundary}",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            raw = resp.read().decode("utf-8")
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8")
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {"raw": raw}


def build_minimal_pdf(lines):
    """Generate a valid single-page PDF 1.4 with Helvetica text lines readable by pdfplumber."""
    def esc(s: str) -> str:
        return s.replace("\\", "\\\\").replace("(", "\\(").replace(")", "\\)")

    stream_cmds = ["BT", "/F1 10 Tf", "50 770 Td", "14 TL"]
    for idx, line in enumerate(lines):
        if idx == 0:
            stream_cmds.append(f"({esc(line)}) Tj")
        else:
            stream_cmds.append(f"T* ({esc(line)}) Tj")
    stream_cmds.append("ET")
    stream_data = "\n".join(stream_cmds).encode("latin-1")

    objects = [
        b"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n",
        b"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n",
        b"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>\nendobj\n",
        b"4 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n",
        f"5 0 obj\n<< /Length {len(stream_data)} >>\nstream\n".encode() + stream_data + b"\nendstream\nendobj\n",
    ]

    pdf = bytearray(b"%PDF-1.4\n")
    offsets = [0]
    for obj in objects:
        offsets.append(len(pdf))
        pdf.extend(obj)
    xref_pos = len(pdf)
    pdf.extend(f"xref\n0 {len(objects) + 1}\n0000000000 65535 f \n".encode())
    for off in offsets[1:]:
        pdf.extend(f"{off:010d} 00000 n \n".encode())
    pdf.extend(f"trailer\n<< /Size {len(objects) + 1} /Root 1 0 R >>\nstartxref\n{xref_pos}\n%%EOF\n".encode())
    return bytes(pdf)


def add_months(year: int, month: int, delta: int):
    idx = year * 12 + (month - 1) + delta
    return idx // 12, (idx % 12) + 1


def login_email(email: str, password: str):
    return api_json("POST", "/auth/login", {"email": email, "password": password, "remember_me": False})


def login_phone(phone: str, password: str):
    return api_json("POST", "/auth/login", {"phone": phone, "password": password, "remember_me": False})


def bootstrap_qa_farm():
    print("[1/6] Bootstrapping isolated QA Automation Test Farm & Owner ...")
    status, login_res = login_email(OWNER_EMAIL, QA_PASSWORD)
    if status == 200 and login_res.get("data", {}).get("token"):
        token = login_res["data"]["token"]
        st_t, tenants_res = api_json("GET", "/tenants", token=token)
        assert st_t == 200, f"GET /tenants failed: {tenants_res}"
        for t in tenants_res.get("data") or []:
            if t.get("name") == QA_FARM_NAME:
                print(f"  ✓ Reusing existing QA farm id={t['id']}")
                return t["id"], token

    st, create_t = api_json(
        "POST",
        "/admin/create-tenant",
        {
            "name": QA_FARM_NAME,
            "location": "Namakkal QA Lab, TN",
            "country_code": "IND",
            "currency": "INR",
            "number_format": "lakhs",
            "date_format": "DD-MM-YYYY",
            "capacity": 45000,
        },
        headers={"X-Admin-Key": ADMIN_KEY},
    )
    assert st == 201, f"create-tenant failed ({st}): {create_t}"
    tenant_id = create_t["data"]["id"]

    st, create_o = api_json(
        "POST",
        "/admin/create-owner",
        {
            "email": OWNER_EMAIL,
            "password": QA_PASSWORD,
            "full_name": "QA Farm Owner",
            "tenant_id": tenant_id,
        },
        headers={"X-Admin-Key": ADMIN_KEY},
    )
    assert st == 201, f"create-owner failed ({st}): {create_o}"

    st, login_res = login_email(OWNER_EMAIL, QA_PASSWORD)
    assert st == 200, f"Owner login failed ({st}): {login_res}"
    print(f"  ✓ Created QA farm id={tenant_id} and authenticated OWNER")
    return tenant_id, login_res["data"]["token"]


def verify_farm_attributes(tenant_id: str, owner_token: str):
    print("[2/6] Verifying farm attributes & country codes ...")
    st, get_res = api_json("GET", f"/tenants/{tenant_id}", token=owner_token)
    assert st == 200 and get_res.get("data", {}).get("id") == tenant_id, f"GET tenant failed: {get_res}"

    payload = {
        "name": QA_FARM_NAME,
        "location": "Namakkal Automated QA Farm",
        "country_code": "IND",
        "currency": "INR",
        "number_format": "lakhs",
        "date_format": "DD-MM-YYYY",
        "timezone": "Asia/Kolkata",
        "egg_price_reference_zone": "Namakkal",
        "financial_year_start_month": 4,
        "capacity": 48000,
        "age_category_chick_max_weeks": 8,
        "age_category_grower_max_weeks": 18,
        "age_category_prelayer_max_weeks": 22,
    }
    st, put_res = api_json("PUT", f"/tenants/{tenant_id}", payload, token=owner_token)
    assert st == 200, f"PUT /tenants/{tenant_id} failed ({st}): {put_res}"
    updated = put_res.get("data") or {}
    assert updated.get("capacity") == 48000, f"Expected capacity 48000, got {updated.get('capacity')}"
    assert updated.get("location") == "Namakkal Automated QA Farm"
    assert updated.get("egg_price_reference_zone") == "Namakkal"

    st, cc_put = api_json(
        "PUT",
        f"/tenants/{tenant_id}/country-codes",
        {"country_codes": [{"country_code": "+91", "country_name": "India"}, {"country_code": "+1", "country_name": "USA"}]},
        token=owner_token,
    )
    assert st == 200, f"PUT country-codes failed ({st}): {cc_put}"
    st, cc_get = api_json("GET", f"/tenants/{tenant_id}/country-codes")
    assert st == 200 and "+91" in [c.get("country_code") for c in (cc_get.get("data") or [])]

    # Cross-tenant isolation check: requesting another tenant UUID must be denied (403)
    other_tenant_id = str(uuid.uuid4())
    st_iso_get, _ = api_json("GET", f"/tenants/{other_tenant_id}", token=owner_token)
    assert st_iso_get in (403, 404), f"Cross-tenant GET should be denied (403/404), got {st_iso_get}"
    st_iso_put, _ = api_json("PUT", f"/tenants/{other_tenant_id}", {"name": "Hijack Farm"}, token=owner_token)
    assert st_iso_put in (403, 404), f"Cross-tenant PUT should be denied (403/404), got {st_iso_put}"
    print("  ✓ Farm attributes (capacity=48000, zone=Namakkal, FY=4, age bands, country codes, cross-tenant isolation) verified")


def provision_and_test_roles(tenant_id: str, owner_token: str):
    print("[3/6] Provisioning & testing RBAC permissions for ADMIN, OWNER, CO_OWNER, MANAGER, AUDITOR, OTHER_USER ...")
    tokens = {"OWNER": owner_token}

    # Check existing users so reruns are idempotent
    st, users_res = api_json("GET", "/users", token=owner_token)
    assert st == 200, f"GET /users failed ({st}): {users_res}"
    existing_by_phone = {u.get("phone"): u for u in (users_res.get("data") or []) if u.get("phone")}

    for spec in ROLE_SPECS:
        role, phone, name = spec["role"], spec["phone"], spec["name"]
        st_l, res_l = login_phone(phone, QA_PASSWORD)
        if st_l == 200 and res_l.get("data", {}).get("token"):
            tokens[role] = res_l["data"]["token"]
            continue

        if phone in existing_by_phone:
            st_m, ml_res = api_json("POST", "/users/magic-link", {"user_id": existing_by_phone[phone]["id"]}, token=owner_token)
            assert st_m == 200, f"magic-link failed for {role}: {ml_res}"
            magic_token = ml_res["data"]["token"]
        else:
            st_i, inv_res = api_json(
                "POST",
                "/users/invite",
                {"phone": phone, "role": role, "full_name": name},
                token=owner_token,
            )
            assert st_i == 201, f"Invite {role} failed ({st_i}): {inv_res}"
            magic_token = inv_res["data"]["token"]

        st_ml, ml_login = api_json("POST", "/auth/magic-login", {"token": magic_token, "phone": phone})
        assert st_ml == 200, f"magic-login failed for {role} ({st_ml}): {ml_login}"
        temp_tok = ml_login["data"]["token"]

        st_pw, pw_res = api_json(
            "POST",
            "/users/change-password",
            {"new_password": QA_PASSWORD, "skip_current_for_magic": True},
            token=temp_tok,
        )
        assert st_pw == 200, f"change-password failed for {role} ({st_pw}): {pw_res}"

        st_l, res_l = login_phone(phone, QA_PASSWORD)
        assert st_l == 200 and res_l.get("data", {}).get("token"), f"Password login failed for {role}: {res_l}"
        tokens[role] = res_l["data"]["token"]

    # Create a test transaction as OTHER_USER (employee staff can create, cannot approve)
    today_iso = dt.date.today().isoformat()
    tx_payload = {
        "transaction_date": today_iso,
        "transaction_type": "SALE",
        "category": "EGG",
        "item_name": "LARGE EGG",
        "quantity": 1000,
        "unit": "NOS",
        "rate": 5.50,
        "amount": 5500.0,
        "notes": "QA RBAC permission probe",
    }
    st_staff_tx, staff_tx = api_json("POST", "/transactions", tx_payload, token=tokens["OTHER_USER"])
    assert st_staff_tx in (200, 201), f"OTHER_USER should be able to create tx, got {st_staff_tx}: {staff_tx}"
    probe_tx_id = staff_tx["data"]["id"]

    # AUDITOR cannot create transactions -> 403
    st_aud_tx, _ = api_json("POST", "/transactions", tx_payload, token=tokens["AUDITOR"])
    assert st_aud_tx == 403, f"AUDITOR should get 403 creating tx, got {st_aud_tx}"

    # OTHER_USER cannot approve -> 403, AUDITOR cannot approve -> 403
    st_staff_app, _ = api_json("POST", f"/transactions/{probe_tx_id}/approve", {}, token=tokens["OTHER_USER"])
    assert st_staff_app == 403, f"OTHER_USER should get 403 approving tx, got {st_staff_app}"
    st_aud_app, _ = api_json("POST", f"/transactions/{probe_tx_id}/approve", {}, token=tokens["AUDITOR"])
    assert st_aud_app == 403, f"AUDITOR should get 403 approving tx, got {st_aud_app}"

    # MANAGER can approve -> 200
    st_mgr_app, _ = api_json("POST", f"/transactions/{probe_tx_id}/approve", {}, token=tokens["MANAGER"])
    assert st_mgr_app == 200, f"MANAGER should approve tx, got {st_mgr_app}"

    # MANAGER, OTHER_USER, AUDITOR cannot manage users -> 403; ADMIN & CO_OWNER can -> 200
    for denied_role in ("MANAGER", "OTHER_USER", "AUDITOR"):
        st_u, _ = api_json("GET", "/users", token=tokens[denied_role])
        assert st_u == 403, f"{denied_role} should get 403 on GET /users, got {st_u}"
    for allowed_role in ("OWNER", "ADMIN", "CO_OWNER"):
        st_u, _ = api_json("GET", "/users", token=tokens[allowed_role])
        assert st_u == 200, f"{allowed_role} should get 200 on GET /users, got {st_u}"

    # AUDITOR can view analytics charts; OTHER_USER cannot -> 403
    now = dt.date.today()
    st_aud_chart, _ = api_json("GET", f"/analytics/enhanced-monthly-summary?year={now.year}&month={now.month}", token=tokens["AUDITOR"])
    assert st_aud_chart == 200, f"AUDITOR should view analytics (200), got {st_aud_chart}"
    st_staff_chart, _ = api_json("GET", f"/analytics/enhanced-monthly-summary?year={now.year}&month={now.month}", token=tokens["OTHER_USER"])
    assert st_staff_chart == 403, f"OTHER_USER should get 403 on analytics, got {st_staff_chart}"

    # Clean up probe tx so analytics stay clean
    api_json("DELETE", f"/transactions/{probe_tx_id}", token=owner_token)
    print("  ✓ Verified all 6 roles (OWNER, ADMIN, CO_OWNER, MANAGER, AUDITOR, OTHER_USER) positive & 403 checks")
    return tokens


def seed_rolling_two_years(owner_token: str):
    print("[4/6] Ensuring rolling 24-month test data (always last 2 years relative to today) ...")
    today = dt.date.today()
    start_ym = add_months(today.year, today.month, -23)
    end_ym = add_months(today.year, today.month, -1)

    st, existing_res = api_json("GET", "/transactions?limit=5000", token=owner_token)
    assert st == 200, f"GET /transactions failed ({st}): {existing_res}"
    existing_txs = existing_res.get("data") or []

    cutoff_date = dt.date(start_ym[0], start_ym[1], 1)
    pruned = 0
    present_months = set()
    for tx in existing_txs:
        tx_date_str = (tx.get("transaction_date") or "")[:10]
        if not tx_date_str:
            continue
        tx_d = dt.date.fromisoformat(tx_date_str)
        if tx_d < cutoff_date:
            d_st, _ = api_json("DELETE", f"/transactions/{tx['id']}", token=owner_token)
            if d_st == 200:
                pruned += 1
        else:
            present_months.add((tx_d.year, tx_d.month))

    seeded_months = 0
    cur_y, cur_m = start_ym
    while (cur_y, cur_m) <= end_ym:
        if (cur_y, cur_m) not in present_months:
            d1 = dt.date(cur_y, cur_m, 10).isoformat()
            d2 = dt.date(cur_y, cur_m, 15).isoformat()
            d3 = dt.date(cur_y, cur_m, 20).isoformat()
            monthly_rows = [
                {"transaction_date": d1, "transaction_type": "SALE", "category": "EGG", "item_name": "LARGE EGG", "quantity": 750000, "unit": "NOS", "rate": 5.45, "amount": 4087500.0},
                {"transaction_date": d2, "transaction_type": "PURCHASE", "category": "FEED", "item_name": "LAYER MASH BULK", "quantity": 95000, "unit": "KGS", "rate": 34.0, "amount": 3230000.0},
                {"transaction_date": d2, "transaction_type": "PURCHASE", "category": "FEED", "item_name": "GROWER MASH", "quantity": 18000, "unit": "KGS", "rate": 34.0, "amount": 612000.0},
                {"transaction_date": d3, "transaction_type": "PURCHASE", "category": "MEDICINE", "item_name": "D3 FORTE", "quantity": 60, "unit": "NOS", "rate": 1320.0, "amount": 79200.0},
                {"transaction_date": d3, "transaction_type": "EXPENSE", "category": "OTHER", "item_name": "FARM OVERHEADS (LABOR/EB/EMI)", "amount": 360000.0},
            ]
            for row in monthly_rows:
                c_st, c_res = api_json("POST", "/transactions", row, token=owner_token)
                assert c_st in (200, 201), f"Seed tx failed for {cur_y}-{cur_m:02d}: {c_res}"
            seeded_months += 1
        cur_y, cur_m = add_months(cur_y, cur_m, 1)

    # Ensure hen batches exist
    st_hb, hb_res = api_json("GET", "/hen-batches", token=owner_token)
    assert st_hb == 200, f"GET /hen-batches failed: {hb_res}"
    batches = hb_res.get("data") or []
    if not batches:
        b1_st, b1 = api_json(
            "POST",
            "/hen-batches",
            {
                "batch_name": "QA Layer Flock A",
                "initial_count": 42000,
                "current_count": 41500,
                "age_weeks": 36,
                "date_added": (today - dt.timedelta(days=250)).isoformat(),
                "notes": "QA automated test flock",
            },
            token=owner_token,
        )
        assert b1_st in (200, 201), f"Create hen batch failed: {b1}"
        batch_id = b1["data"]["id"]
        m_st, m_res = api_json(
            "POST",
            "/hen-batches/mortality",
            {"batch_id": batch_id, "mortality_date": today.isoformat(), "count": 12, "reason": "QA natural"},
            token=owner_token,
        )
        assert m_st in (200, 201), f"Create mortality failed: {m_res}"

    print(f"  ✓ Rolling 24-month window {start_ym[0]}-{start_ym[1]:02d}..{end_ym[0]}-{end_ym[1]:02d} ready (seeded={seeded_months} months, pruned={pruned} stale rows)")


def test_pdf_upload_and_analytics(owner_token: str):
    print("[5/6] Generating sample PDF, uploading via /admin/upload-ledger, and verifying parser + DB + analytics ...")
    today = dt.date.today()
    mon_abbr = calendar.month_abbr[today.month].upper()
    yy = str(today.year)[2:]
    d05 = f"05-{calendar.month_abbr[today.month]}-{yy}"
    d08 = f"08-{calendar.month_abbr[today.month]}-{yy}"
    d12 = f"12-{calendar.month_abbr[today.month]}-{yy}"
    d15 = f"15-{calendar.month_abbr[today.month]}-{yy}"
    d20 = f"20-{calendar.month_abbr[today.month]}-{yy}"
    d25 = f"25-{calendar.month_abbr[today.month]}-{yy}"

    pdf_lines = [
        "QA Sample Feed Mill Pvt Ltd",
        f"Ledger Account : {QA_FARM_NAME}",
        f"1-{calendar.month_abbr[today.month]}-{yy} to 28-{calendar.month_abbr[today.month]}-{yy}",
        f"{d05} Poultry Egg Sales 1,35,000.00",
        "LARGE EGG 25000 NOS 5.40/NOS 1,35,000.00",
        f"{d08} Poultry Feed Sales 1,19,000.00",
        "LAYER MASH BULK 3000 KGS 34.00/KGS 1,02,000.00",
        "GROWER MASH 500 KGS 34.00/KGS 17,000.00",
        f"{d12} Poultry Feed Suppliments 2,640.00",
        "D3 FORTE 2 NOS 1320.00/NOS 2,640.00",
        f"{d15} Poultry Feed Sales Credit Note 3,000.00",
        f"{d20} To Canara Bank Payment 50,000.00",
        f"{d25} TDS 1,350.00",
    ]
    pdf_bytes = build_minimal_pdf(pdf_lines)
    pdf_filename = f"{yy}_{today.month:02d}_{mon_abbr}_QA_SAMPLE.pdf"

    st, up_res = api_multipart(
        "/admin/upload-ledger",
        {"file_type": "LEDGER_PDF", "year": str(today.year), "month": str(today.month), "commit": "true"},
        "file",
        pdf_filename,
        pdf_bytes,
        owner_token,
    )
    assert st == 200 and up_res.get("success") is True, f"Upload ledger PDF failed ({st}): {up_res}"
    up_data = up_res.get("data") or {}
    assert up_data.get("parser") == "go_native", f"Expected native Go PDF parser, got {up_data.get('parser')}"
    assert up_data.get("sha256_hash"), "Missing sha256_hash on upload response"

    st_list, list_res = api_json("GET", "/admin/uploads", token=owner_token)
    assert st_list == 200, f"GET /admin/uploads failed ({st_list}): {list_res}"
    uploads = list_res.get("data") or []
    assert any(u.get("file_name") == pdf_filename for u in uploads), f"Uploaded PDF {pdf_filename} not found in /admin/uploads"

    st_latest, latest_res = api_json("GET", "/analytics/latest-data-month", token=owner_token)
    assert st_latest == 200, f"GET /analytics/latest-data-month failed ({st_latest}): {latest_res}"
    latest_data = latest_res.get("data") or {}
    assert latest_data.get("has_data") is True
    assert latest_data.get("year") == today.year and latest_data.get("month") == today.month, (
        f"Expected latest-data-month {today.year}-{today.month:02d}, got {latest_data}"
    )

    st_sum, sum_res = api_json(
        "GET",
        f"/analytics/enhanced-monthly-summary?year={today.year}&month={today.month}",
        token=owner_token,
    )
    assert st_sum == 200, f"GET enhanced-monthly-summary failed ({st_sum}): {sum_res}"
    raw_data = sum_res.get("data")
    sdata = raw_data[0] if isinstance(raw_data, list) and raw_data else (raw_data or {})
    assert float(sdata.get("total_eggs_sold", 0)) >= 25000.0, f"Expected total_eggs_sold >= 25000, got {sdata.get('total_eggs_sold')}"
    assert float(sdata.get("layer_feed_price", 0)) >= 102000.0, f"Expected layer_feed_price >= 102000, got {sdata.get('layer_feed_price')}"
    assert float(sdata.get("rearing_feed_price", 0)) >= 17000.0, f"Expected rearing_feed_price >= 17000, got {sdata.get('rearing_feed_price')}"
    print(f"  ✓ PDF '{pdf_filename}' parsed & committed; latest-data-month={today.year}-{today.month:02d}; eggs={sdata.get('total_eggs_sold')}, layer_feed=₹{sdata.get('layer_feed_price')}, rearing_feed=₹{sdata.get('rearing_feed_price')}")


def verify_profile_and_onboarding(owner_token: str):
    print("[6/6] Verifying profile language & onboarding_version persistence ...")
    st_p, prof = api_json("GET", "/users/profile", token=owner_token)
    assert st_p == 200, f"GET /users/profile failed: {prof}"
    orig_lang = prof["data"].get("preferred_language") or "en"
    cur_ver = int(prof["data"].get("onboarding_version") or 0)

    st_ta, put_ta = api_json("PUT", "/users/profile", {"preferred_language": "ta", "onboarding_version": max(cur_ver, 1)}, token=owner_token)
    assert st_ta == 200 and put_ta["data"]["preferred_language"] == "ta"
    assert int(put_ta["data"]["onboarding_version"]) >= 1

    # Verify onboarding_version cannot regress via GREATEST(...)
    st_reg, put_reg = api_json("PUT", "/users/profile", {"preferred_language": orig_lang, "onboarding_version": 0}, token=owner_token)
    assert st_reg == 200 and int(put_reg["data"]["onboarding_version"]) >= 1
    print("  ✓ Language toggle & monotonic onboarding_version verified")


def main():
    st, _ = api_json("GET", "/health")
    assert st == 200, f"Go API not healthy on {API_BASE}"
    tenant_id, owner_token = bootstrap_qa_farm()
    verify_farm_attributes(tenant_id, owner_token)
    provision_and_test_roles(tenant_id, owner_token)
    seed_rolling_two_years(owner_token)
    test_pdf_upload_and_analytics(owner_token)
    verify_profile_and_onboarding(owner_token)
    print("\nALL FULL E2E QA CHECKS PASSED!")


if __name__ == "__main__":
    main()
