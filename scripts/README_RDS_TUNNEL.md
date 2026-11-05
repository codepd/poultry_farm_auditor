# RDS tunnel and monthly feed prices

Shared by the load-feed-prices skill and the parse-ledger skill. Both start the database the same way. Feed rates are loaded here; the ledger skill then parses and validates against those rates.

## 1. Open the tunnel

From the repo root:

```bash
scripts/start-rds-port-forward.sh
```

Leave that terminal open. Wait until it prints `Waiting for connections...`.

What it does:

1. `scripts/aws-sso-ensure.sh` checks profile `AdminAccess-Pradeep`. If the SSO token is still valid, it does nothing.
2. If the token is expired, it starts `aws sso login` and opens the approval URL in the Chrome profile named **codepd** (directory `Profile 8`, account codepd3@gmail.com). Approve that page. The token is stored in `~/.aws/sso/cache` (mode 600). There is no saved password to renew this.
3. SSM port-forward through bastion `kolipannai-dev-bastion` (`i-0a5c2678494e93072`) to `poultry-farm-dev` on local port `15432`.

If 15432 is already in use:

```bash
LOCAL_PORT=15433 scripts/start-rds-port-forward.sh
```

Use that same port in every later `DB_PORT`.

### If Chrome shows the AWS app portal and the script is still waiting

The portal home (`https://d-9f676c1851.awsapps.com/start/#/`) is not the CLI callback. The script is waiting for a redirect to `http://127.0.0.1:<port>/oauth/callback`. Copy the `https://oidc.ap-south-1.amazonaws.com/authorize?...` URL from the login output and open it again in the codepd window:

```bash
scripts/open_aws_sso_chrome.sh '<authorize-url>'
```

Do not start a second Chrome with `open -n` or `--new-window`. That opens a stray window instead of the codepd profile.

## 2. Load monthly feed bag rates

`scripts/load_monthly_feed_prices.py` holds the daily bands transcribed from `feed_price_history/<YEAR>/MM_DD_DD_YYYY.jpeg`. Transcribed months: `2026-04` through `2026-09`. Prices are rupees per bag. The sheet footer (total, Per bag, Per Kg) is a checksum and is not stored.

The tunnel must already be up. `DB_PASSWORD` is the `poultry_admin` password recorded in `.cursor/skills/load-feed-prices/SKILL.md`.

```bash
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm \
DB_USER=poultry_admin DB_PASSWORD="$DB_PASSWORD" \
python_backend/venv/bin/python scripts/load_monthly_feed_prices.py
```

That prints unchanged, missing, and different rows and does not write. To write:

```bash
DB_HOST=localhost DB_PORT=15432 DB_NAME=poultry_farm \
DB_USER=poultry_admin DB_PASSWORD="$DB_PASSWORD" \
python_backend/venv/bin/python scripts/load_monthly_feed_prices.py --apply
```

Limit months with `--months 2026-07,2026-08`. A full month must end as `days × 4` rows in `price_history` for tenant Pradeep Farm (`8d7939f7-b716-4eb0-98d4-544c18c8dfb8`).

A new screenshot is not loaded by filename alone. Read the image, check the four column totals, add the bands and expected totals to the script, then run the two commands above.

Weekly rate cards (`MM_DD_YYYY.jpeg`) still go through `python_backend/cli/process_feed_price_images.py`, and only for dates a monthly grid does not cover. Do not point that OCR script at a monthly grid.
