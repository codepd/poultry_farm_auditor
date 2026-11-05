#!/usr/bin/env python3
"""
One-time migration script to upload all existing local ledgers, feed price sheets,
and parsed Excel workbooks in this repository to AWS S3 for auditing and async workflows,
and register them in the PostgreSQL `document_uploads` table.

Usage:
  # Preview migration manifest (dry run):
  python3 scripts/migrate_documents_to_s3.py --dry-run

  # Create bucket (if needed), upload all files to S3, and seed DB / SQL file:
  python3 scripts/migrate_documents_to_s3.py --create-bucket --profile AdminAccess-Pradeep
"""

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Dict, List, Optional, Tuple
from urllib.parse import urlencode

PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(PROJECT_ROOT))
sys.path.insert(0, str(PROJECT_ROOT / "python_backend"))

DEFAULT_TENANT_ID = "8d7939f7-b716-4eb0-98d4-544c18c8dfb8"
DEFAULT_BUCKET = "kolipannai-dev-documents"
DEFAULT_REGION = "ap-south-1"
DEFAULT_PROFILE = "AdminAccess-Pradeep"

SOURCE_DIRS = [
    "ledgers",
    "feed_price_history",
    "2025_ledgers_excel",
    "2025_parsed_data",
]

MONTH_MAP = {
    "jan": 1,
    "feb": 2,
    "fed": 2,
    "mar": 3,
    "apr": 4,
    "may": 5,
    "jun": 6,
    "jul": 7,
    "aug": 8,
    "sep": 9,
    "oct": 10,
    "nov": 11,
    "dec": 12,
}


def compute_sha256(file_path: Path) -> str:
    h = hashlib.sha256()
    with open(file_path, "rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def detect_file_type_and_mime(file_path: Path) -> Optional[Tuple[str, str]]:
    ext = file_path.suffix.lower()
    if ext == ".pdf":
        return "LEDGER_PDF", "application/pdf"
    if ext == ".xlsx":
        return (
            "PARSED_EXCEL",
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        )
    if ext == ".xls":
        return "PARSED_EXCEL", "application/vnd.ms-excel"
    if ext in (".jpg", ".jpeg"):
        return "FEED_PRICE_IMAGE", "image/jpeg"
    if ext == ".png":
        return "FEED_PRICE_IMAGE", "image/png"
    return None


def infer_year_and_month(rel_path: str) -> Tuple[int, Optional[int]]:
    lower_name = os.path.basename(rel_path).lower()
    lower_rel = rel_path.lower()

    year = 2025
    month: Optional[int] = None

    # 1. Explicit 4-digit year in filename or path
    m_year = re.search(r"(202[0-9])", lower_name) or re.search(r"(202[0-9])", lower_rel)
    if m_year:
        year = int(m_year.group(1))

    # 2. Short year prefix like 25_01_JAN or parsed_26_04_APR
    m_short = re.search(r"(?:^|parsed_)(2[4-9])_(0[1-9]|1[0-2])", lower_name)
    if m_short:
        year = 2000 + int(m_short.group(1))
        month = int(m_short.group(2))

    # 3. Month name in filename (word/token boundary so 'summary' doesn't match 'mar')
    if month is None:
        for k, v in MONTH_MAP.items():
            if re.search(rf"(?:^|[^a-z]){k}(?:[a-z]*)(?:$|[^a-z])", lower_name):
                if k == "mar" and "summary" in lower_name:
                    continue
                month = v
                break

    # 4. Date pattern like 11_10_2025.jpeg or 04_01_30_2026.jpeg (MM_DD_YYYY)
    if month is None:
        m_date = re.match(r"^(0[1-9]|1[0-2])_[0-9]{2}", lower_name)
        if m_date:
            month = int(m_date.group(1))

    return year, month


def build_s3_key(tenant_id: str, file_type: str, year: int, filename: str, subfolder_hint: str = "") -> str:
    if file_type == "LEDGER_PDF":
        category_folder = "ledgers"
    elif file_type == "FEED_PRICE_IMAGE":
        category_folder = "feed_price_history"
    else:
        category_folder = "parsed_excels"

    if subfolder_hint:
        return f"{tenant_id}/{category_folder}/{year}/{subfolder_hint}/{filename}"
    return f"{tenant_id}/{category_folder}/{year}/{filename}"


def discover_documents(tenant_id: str) -> List[Dict]:
    manifest: List[Dict] = []
    seen_key_hashes: Dict[str, str] = {}

    for src_dir in SOURCE_DIRS:
        abs_dir = PROJECT_ROOT / src_dir
        if not abs_dir.exists():
            continue

        for root, _, files in os.walk(abs_dir):
            for fname in sorted(files):
                if fname.startswith("."):
                    continue
                fpath = Path(root) / fname
                rel_path = str(fpath.relative_to(PROJECT_ROOT))
                detected = detect_file_type_and_mime(fpath)
                if not detected:
                    continue

                file_type, content_type = detected
                year, month = infer_year_and_month(rel_path)
                sha256_hex = compute_sha256(fpath)
                file_size = fpath.stat().st_size

                s3_key = build_s3_key(tenant_id, file_type, year, fname)

                # If identical S3 key already seen:
                if s3_key in seen_key_hashes:
                    if seen_key_hashes[s3_key] == sha256_hex:
                        # Exact duplicate file content in another local folder, skip duplicate upload
                        continue
                    else:
                        # Different file content with same name: namespace by source folder
                        s3_key = build_s3_key(tenant_id, file_type, year, fname, subfolder_hint=src_dir)

                seen_key_hashes[s3_key] = sha256_hex

                upload_status = "ARCHIVED" if file_type == "FEED_PRICE_IMAGE" else "COMMITTED"

                manifest.append(
                    {
                        "local_path": str(fpath),
                        "rel_path": rel_path,
                        "file_name": fname,
                        "file_type": file_type,
                        "content_type": content_type,
                        "file_size": file_size,
                        "sha256_hash": sha256_hex,
                        "year": year,
                        "month": month,
                        "s3_key": s3_key,
                        "upload_status": upload_status,
                    }
                )

    return manifest


def ensure_bucket_exists(bucket: str, region: str, profile: str) -> None:
    print(f"Checking/creating S3 bucket '{bucket}' in {region} (profile={profile})...")
    head_cmd = ["aws", "s3api", "head-bucket", "--bucket", bucket, "--region", region, "--profile", profile]
    res = subprocess.run(head_cmd, capture_output=True, text=True)
    if res.returncode != 0:
        print(f"Creating bucket '{bucket}'...")
        create_cmd = [
            "aws",
            "s3api",
            "create-bucket",
            "--bucket",
            bucket,
            "--region",
            region,
            "--create-bucket-configuration",
            f"LocationConstraint={region}",
            "--profile",
            profile,
        ]
        subprocess.run(create_cmd, check=True)

    # Enable versioning for audit trail
    subprocess.run(
        [
            "aws",
            "s3api",
            "put-bucket-versioning",
            "--bucket",
            bucket,
            "--versioning-configuration",
            "Status=Enabled",
            "--region",
            region,
            "--profile",
            profile,
        ],
        check=True,
    )

    # Block public access
    subprocess.run(
        [
            "aws",
            "s3api",
            "put-public-access-block",
            "--bucket",
            bucket,
            "--public-access-block-configuration",
            "BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true",
            "--region",
            region,
            "--profile",
            profile,
        ],
        check=True,
    )
    print(f"Bucket '{bucket}' is ready with Versioning=Enabled and PublicAccess=Blocked.")


def upload_file_to_s3(item: Dict, bucket: str, region: str, profile: str, tenant_id: str) -> Optional[str]:
    tagging = urlencode(
        {
            "tenant_id": tenant_id,
            "file_type": item["file_type"],
            "upload_source": "REPO_MIGRATION",
            "original_path": item["rel_path"],
            "sha256": item["sha256_hash"][:32],
        }
    )
    cmd = [
        "aws",
        "s3api",
        "put-object",
        "--bucket",
        bucket,
        "--key",
        item["s3_key"],
        "--body",
        item["local_path"],
        "--content-type",
        item["content_type"],
        "--tagging",
        tagging,
        "--region",
        region,
        "--profile",
        profile,
        "--output",
        "json",
    ]
    res = subprocess.run(cmd, capture_output=True, text=True)
    if res.returncode != 0:
        raise RuntimeError(res.stderr.strip() or res.stdout.strip())

    try:
        out = json.loads(res.stdout)
        return out.get("VersionId")
    except Exception:
        return None


def write_sql_seed_file(manifest: List[Dict], bucket: str, tenant_id: str, output_sql: Path) -> None:
    lines = [
        "-- Auto-generated by scripts/migrate_documents_to_s3.py",
        "-- Registers migrated repository ledgers, feed price sheets, and parsed excels in document_uploads",
        "",
        "CREATE TABLE IF NOT EXISTS document_uploads (",
        "    id SERIAL PRIMARY KEY,",
        "    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,",
        "    s3_bucket VARCHAR(255) NOT NULL,",
        "    s3_key TEXT NOT NULL,",
        "    s3_version_id VARCHAR(255),",
        "    file_name VARCHAR(255) NOT NULL,",
        "    file_type VARCHAR(50) NOT NULL,",
        "    content_type VARCHAR(100),",
        "    file_size BIGINT,",
        "    sha256_hash VARCHAR(64),",
        "    document_year INTEGER,",
        "    document_month INTEGER,",
        "    upload_source VARCHAR(50) DEFAULT 'WEB_PORTAL',",
        "    upload_status VARCHAR(50) DEFAULT 'UPLOADED',",
        "    parse_result JSONB,",
        "    error_message TEXT,",
        "    uploaded_by_user_id INTEGER REFERENCES users(id),",
        "    ledger_parse_id INTEGER REFERENCES ledger_parses(id) ON DELETE SET NULL,",
        "    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,",
        "    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,",
        "    UNIQUE(tenant_id, s3_key)",
        ");",
        "",
    ]

    for item in manifest:
        ver_sql = f"'{item['s3_version_id']}'" if item.get("s3_version_id") else "NULL"
        month_sql = str(item["month"]) if item.get("month") else "NULL"
        fname_esc = item["file_name"].replace("'", "''")
        s3_key_esc = item["s3_key"].replace("'", "''")
        lines.append(
            f"INSERT INTO document_uploads ("
            f"tenant_id, s3_bucket, s3_key, s3_version_id, file_name, file_type, "
            f"content_type, file_size, sha256_hash, document_year, document_month, "
            f"upload_source, upload_status"
            f") VALUES ("
            f"'{tenant_id}', '{bucket}', '{s3_key_esc}', {ver_sql}, '{fname_esc}', '{item['file_type']}', "
            f"'{item['content_type']}', {item['file_size']}, '{item['sha256_hash']}', {item['year']}, {month_sql}, "
            f"'REPO_MIGRATION', '{item['upload_status']}'"
            f") ON CONFLICT (tenant_id, s3_key) DO UPDATE SET "
            f"s3_version_id = COALESCE(EXCLUDED.s3_version_id, document_uploads.s3_version_id), "
            f"file_size = EXCLUDED.file_size, "
            f"sha256_hash = EXCLUDED.sha256_hash, "
            f"updated_at = CURRENT_TIMESTAMP;"
        )

    output_sql.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"Wrote SQL audit seed file to: {output_sql}")


def try_seed_database(manifest: List[Dict], bucket: str, tenant_id: str) -> None:
    try:
        from database.connection import get_db_cursor

        sql_seed_path = PROJECT_ROOT / "database" / "seed_migrated_documents.sql"
        sql_text = sql_seed_path.read_text(encoding="utf-8")
        with get_db_cursor() as cur:
            cur.execute(sql_text)
        print(f"Successfully registered {len(manifest)} documents in PostgreSQL `document_uploads` table.")
    except Exception as exc:
        print(
            f"[INFO] Could not connect to PostgreSQL directly ({exc}). "
            f"You can apply `database/seed_migrated_documents.sql` once the RDS tunnel is active."
        )


def main():
    parser = argparse.ArgumentParser(description="Migrate local ledgers & feed price history to AWS S3")
    parser.add_argument("--bucket", default=DEFAULT_BUCKET, help="Target S3 bucket name")
    parser.add_argument("--region", default=DEFAULT_REGION, help="AWS region")
    parser.add_argument("--profile", default=DEFAULT_PROFILE, help="AWS CLI profile")
    parser.add_argument("--tenant-id", default=DEFAULT_TENANT_ID, help="Tenant UUID")
    parser.add_argument("--create-bucket", action="store_true", help="Create S3 bucket if it does not exist")
    parser.add_argument("--dry-run", action="store_true", help="List files and target S3 keys without uploading")
    parser.add_argument("--seed-db-only", action="store_true", help="Only apply database/seed_migrated_documents.sql to PostgreSQL")
    args = parser.parse_args()

    manifest = discover_documents(args.tenant_id)
    if args.seed_db_only:
        try_seed_database(manifest, args.bucket, args.tenant_id)
        return
    total_bytes = sum(x["file_size"] for x in manifest)

    by_type: Dict[str, int] = {}
    for item in manifest:
        by_type[item["file_type"]] = by_type.get(item["file_type"], 0) + 1

    print("=" * 80)
    print(f"S3 Document Migration Summary (Bucket: s3://{args.bucket})")
    print(f"Unique Documents Discovered: {len(manifest)} ({total_bytes / (1024 * 1024):.2f} MB)")
    for ftype, count in sorted(by_type.items()):
        print(f"  - {ftype}: {count} files")
    print("=" * 80)

    sql_seed_path = PROJECT_ROOT / "database" / "seed_migrated_documents.sql"
    write_sql_seed_file(manifest, args.bucket, args.tenant_id, sql_seed_path)

    if args.dry_run:
        for idx, item in enumerate(manifest, 1):
            period = f"{item['year']}-{item['month']:02d}" if item["month"] else f"{item['year']}"
            print(
                f"[{idx:02d}/{len(manifest):02d}] {item['file_type']:16s} | {period:7s} | "
                f"{item['rel_path']:42s} -> s3://{args.bucket}/{item['s3_key']}"
            )
        print("\nDry run complete. Run without --dry-run to upload to AWS S3.")
        return

    if args.create_bucket:
        ensure_bucket_exists(args.bucket, args.region, args.profile)

    uploaded = 0
    failed = 0
    for idx, item in enumerate(manifest, 1):
        try:
            ver_id = upload_file_to_s3(item, args.bucket, args.region, args.profile, args.tenant_id)
            item["s3_version_id"] = ver_id
            uploaded += 1
            print(f"[{idx:02d}/{len(manifest):02d}] ✓ Uploaded s3://{args.bucket}/{item['s3_key']}")
        except Exception as exc:
            failed += 1
            print(f"[{idx:02d}/{len(manifest):02d}] ✗ FAILED {item['rel_path']}: {exc}")

    # Rewrite SQL seed file with S3 VersionIds included
    write_sql_seed_file(manifest, args.bucket, args.tenant_id, sql_seed_path)
    try_seed_database(manifest, args.bucket, args.tenant_id)

    print("=" * 80)
    print(f"Migration finished: {uploaded} uploaded, {failed} failed.")
    if failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
