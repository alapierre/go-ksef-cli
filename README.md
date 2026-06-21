# go-ksef-cli

`go-ksef-cli` is a command-line client for working with KSeF from a terminal. It is useful both as a practical day-to-day tool
and as a small reference application showing how to use the [`go-ksef`](https://github.com/alapierre/go-ksef) KSeF client library from Go.

The CLI helps you work with KSeF without building a full integration platform first. It can be used by developers testing KSeF flows,
accounting teams preparing data for Excel, small businesses that need a lightweight operational tool,
and anyone who wants scriptable access to invoices, metadata, exports, and reports.

What you can do with it:

| Area               | Capabilities                                                                                                                                                |
|--------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Authentication     | Initialize encrypted local storage, store KSeF authorisation tokens, log in, reuse session tokens automatically, and log out.                                  |
| Multi-context work | Work with multiple taxpayer contexts by NIP. Stored tokens are scoped by KSeF environment and identifier, so `TEST`, `DEMO`, and `PROD` data stay separate. |
| Invoice sending    | Send XML invoice files online or as a batch, including processing directories with invoice XML files.                                                       |
| Invoice lookup     | Query invoice metadata by date range, subject type, date type, pagination, and sorting.                                                                     |
| Data export        | Export query results to CSV, download KSeF invoice export ZIP packages, and create CSV reports from exported ZIP files.                                     |
| Reporting          | Turn an export ZIP into `invoices.csv` and `invoice_rows.csv`, with invoice line items linked back to invoice metadata by `ksef_number`.                    |
| QR                 | Generate KSeF QR Code II verification links and QR images (`png` or `bmp`) using a certificate and private key.                                             |

Tokens are stored encrypted on disk. The encryption key is kept in the selected keystore, with the default `desktop` keystore using the operating system keyring.
This makes regular use more convenient while avoiding plain-text token files in project directories or shell history.

## Installation

Download and unpack a release archive for your operating system from the project releases page:

```shell
https://github.com/alapierre/go-ksef-cli/releases
```

Then put the `ksef-cli` binary on your `PATH`, or run it directly from the unpacked directory.

## Configuration model

All options are passed as command-line flags or environment variables. Global options are available on every command:

| Flag         | Environment variable | Default   | Description                                  |
|--------------|----------------------|-----------|----------------------------------------------|
| `--env`      | `KSEF_ENVIRONMENT`   | `TEST`    | KSeF environment: `TEST`, `DEMO`, or `PROD`. |
| `--keystore` | `KSEF_KEYSTORE_TYPE` | `desktop` | Keystore type for the local encryption key.  |

Example:

```shell
ksef-cli --env DEMO query --identifier 1234567890 --date-from 2026-06-01T00:00:00
```

The same environment can be selected with an environment variable:

```shell
export KSEF_ENVIRONMENT=DEMO
ksef-cli query --identifier 1234567890 --date-from 2026-06-01T00:00:00
```

## KSeF authorisation token handling

Commands that need a KSeF authorisation token accept it with `--token` / `-t` or the `KSEF_TOKEN` environment variable.

```shell
ksef-cli query \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__" \
  --date-from 2026-06-01T00:00:00
```

For regular use, store the authorisation token once and run `login` once.  
After that, operational commands automatically try the stored session token pair first.  
If the pair is missing or expired, the CLI falls back to full authentication using:
1. `--token` / `KSEF_TOKEN`, then
2. stored encrypted authorisation token from `store`.

Token storage must be initialized first:

```shell
ksef-cli init
```

`init` generates an encryption key and saves it in the selected keystore. With the default `desktop` keystore, the key is stored in the system keyring.

Then store the KSeF authorisation token for a given NIP and environment:

```shell
ksef-cli --env TEST store \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__"
```

The encrypted token file is stored under:

```text
$HOME/.go-ksef-cli/<ENV>/.authorisation_token_<NIP>
```

The encrypted file is bound to the environment and NIP. Tokens for `TEST`, `DEMO`, and `PROD` are stored separately.

You can also provide the token through an environment variable when storing it:

```shell
export KSEF_TOKEN="__ksef_authorisation_token__"
ksef-cli --env TEST store --identifier 1234567890
```

## Commands

### `init`

Initializes the local encryption key used to encrypt tokens saved on disk.

```shell
ksef-cli init
```

Flags:

| Flag           | Description                                                  |
|----------------|--------------------------------------------------------------|
| `--force-init` | Force initialization even if the key is already initialized. |

### `store`

Encrypts and stores a KSeF authorisation token for a NIP and environment.

```shell
ksef-cli --env TEST store \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__"
```

Flags:

| Flag                 | Environment variable | Required | Description                                   |
|----------------------|----------------------|----------|-----------------------------------------------|
| `-i`, `--identifier` |                      | Yes      | Context identifier, usually the taxpayer NIP. |
| `-t`, `--token`      | `KSEF_TOKEN`         | Yes      | KSeF authorisation token to store.            |

### `login`

Logs in to KSeF with an authorisation token. If `--token` is omitted, the CLI tries to load the encrypted authorisation token stored with `store`.

By default, the received session tokens are stored encrypted on disk.

```shell
ksef-cli --env TEST login --identifier 1234567890
```

With an explicit token:

```shell
ksef-cli --env TEST login \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__"
```

Flags:

| Flag                          | Environment variable       | Description                                                               |
|-------------------------------|----------------------------|---------------------------------------------------------------------------|
| `-i`, `--identifier`          |                            | Context identifier, usually the taxpayer NIP.                             |
| `-t`, `--token`               | `KSEF_TOKEN`               | KSeF authorisation token. If omitted, the stored encrypted token is used. |
| `-p`, `--print-session-token` | `KSEF_PRINT_SESSION_TOKEN` | Print the returned session tokens.                                        |
| `-n`, `--no-store`            |                            | Do not store returned session tokens.                                     |

Encrypted session token files are stored under:

```text
$HOME/.go-ksef-cli/<ENV>/.session_token_<NIP>
```

### `print`

Prints stored KSeF session tokens for the selected NIP and environment.

```shell
ksef-cli --env TEST print --identifier 1234567890
```

Flags:

| Flag                 | Required | Description                                   |
|----------------------|----------|-----------------------------------------------|
| `-i`, `--identifier` | Yes      | Context identifier, usually the taxpayer NIP. |

### `logout`

Logs out from KSeF and removes the stored encrypted session token pair for the selected NIP and environment.

```shell
ksef-cli --env TEST logout --identifier 1234567890
```

With an explicit token for full-auth fallback:

```shell
ksef-cli --env TEST logout \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__"
```

Flags:

| Flag                 | Environment variable | Description                                                               |
|----------------------|----------------------|---------------------------------------------------------------------------|
| `-i`, `--identifier` |                      | Context identifier, usually the taxpayer NIP.                             |
| `-t`, `--token`      | `KSEF_TOKEN`         | KSeF authorisation token. If omitted, the stored encrypted token is used. |

### `status`

Shows, per identifier, which encrypted tokens are stored locally and whether the stored session tokens are still valid.

By default it shows only the environment selected with `--env`.

```shell
ksef-cli --env TEST status
```

Show all environments:

```shell
ksef-cli status --all
```

Flags:

| Flag    | Description                                 |
|---------|---------------------------------------------|
| `--all` | Show token status for all stored environments. |

### `qr certificate`

Generates KSeF QR Code II certificate verification link and writes a QR image file.

The command:
- Reads a certificate and encrypted private key.
- Uses `KSEF_KEY_PASSWORD` from environment or asks for the password interactively.
- Builds a signed verification URL for KSeF (`CtxNip` context type).
- Saves a QR image as `<context-nip>_qr2.png` or `<context-nip>_qr2.bmp`.

Basic usage:

```shell
ksef-cli --env TEST qr certificate \
  --cert ./auth-cert.pem \
  --key ./auth-key.pem \
  --context-nip 1234567890
```

or with real Invoice data:

```shell
ksef-cli --env TEST qr certificate \
  --cert ./auth-cert.pem \
  --key ./auth-key.pem \
  --context-nip 1234567890 \
  --in ./invoice.xml
```

Read content to sign from stdin and save as BMP:

```shell
cat ./invoice.xml | ksef-cli --env TEST qr certificate \
  --cert ./auth-cert.pem \
  --key ./auth-key.pem \
  --context-nip 1234567890 \
  --in - \
  --format bmp
```

Command aliases: `ksef-cli qr cert`, `ksef-cli qr qr2`

Flags:

| Flag                  | Default | Description                                                    |
|-----------------------|---------|----------------------------------------------------------------|
| `-c`, `--cert`        |         | Required. KSeF certificate file.                               |
| `-k`, `--key`         |         | Required. Private key file.                                    |
| `-n`, `--context-nip` |         | Required. Invoice issuer NIP (KSeF context).                   |
| `-s`, `--seller-nip`  |         | Seller NIP, if different from the invoice issuer NIP.          |
| `-o`, `--out`         | `.`     | Base output path for QR code file.                             |
| `-r`, `--redirect`    |         | Optional file path for appending generated verification links. |
| `-f`, `--format`      | `png`   | Output image format: `png` or `bmp`.                           |
| `-i`, `--in`          |         | Optional file to hash and sign; use `-` to read from stdin.    |

### `send online`

Sends XML invoice files to KSeF using an interactive session. The command accepts one or more files or directories. When a directory is provided, only files with the `.xml` extension are sent.

```shell
ksef-cli --env TEST send online \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__" \
  ./invoices
```

Send a single file:

```shell
ksef-cli --env TEST send online \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__" \
  ./invoices/FA_1.xml
```

Process directories recursively:

```shell
ksef-cli --env TEST send online \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__" \
  --recursive \
  ./invoices
```

For compatibility, `send` without the explicit `online` subcommand still selects the online sending mode when the arguments match that command.

Flags:

| Flag                 | Environment variable | Description                                   |
|----------------------|----------------------|-----------------------------------------------|
| `-i`, `--identifier` |                      | Context identifier, usually the taxpayer NIP. |
| `-t`, `--token`      | `KSEF_TOKEN`         | KSeF authorisation token.                     |
| `-r`, `--recursive`  |                      | Process directory arguments recursively.      |

### `send batch`

Sends XML invoice files to KSeF using a batch session. The CLI builds the batch package from XML files, opens a batch session, uploads all encrypted batch parts, closes the batch session, and prints the batch session ID.

Closing the batch session is required by KSeF before invoices from the batch start processing. Invoice statuses for a batch session are checked with the same `session invoices` command used for online sessions.

```shell
ksef-cli --env TEST send batch \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__" \
  ./invoices
```

Process directories recursively:

```shell
ksef-cli --env TEST send batch \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__" \
  --recursive \
  ./invoices
```

Keep generated ZIP and encrypted part files for inspection:

```shell
ksef-cli --env TEST send batch \
  --identifier 1234567890 \
  --keep-files \
  --output-dir ./batch-work \
  ./invoices
```

After the command prints the batch session ID, check invoice processing status with:

```shell
ksef-cli --env TEST session invoices \
  --identifier 1234567890 \
  --session-id "__batch_session_id__"
```

Flags:

| Flag                 | Environment variable | Description                                                             |
|----------------------|----------------------|-------------------------------------------------------------------------|
| `-i`, `--identifier` |                      | Context identifier, usually the taxpayer NIP.                           |
| `-t`, `--token`      | `KSEF_TOKEN`         | KSeF authorisation token.                                               |
| `-r`, `--recursive`  |                      | Process directory arguments recursively.                                |
| `--output-dir`       |                      | Directory for temporary batch files. Defaults to the system temp dir.    |
| `--max-part-size`    |                      | Maximum plain ZIP part size in bytes before encryption. Default: 100MB. |
| `--keep-files`       |                      | Keep generated ZIP and encrypted part files after sending.              |

### `query`

Queries invoice metadata from KSeF. This command is useful for listing received or issued invoices and for exporting invoice metadata to CSV.

`--identifier` and `--date-from` are required.

Basic query:

```shell
ksef-cli --env TEST query \
  --identifier 1234567890 \
  --date-from 2026-06-01T00:00:00
```

Query a date range:

```shell
ksef-cli --env TEST query \
  --identifier 1234567890 \
  --date-from 2026-06-01T00:00:00 \
  --date-to 2026-06-30T23:59:59
```

Query buyer-side invoices and sort newest first:

```shell
ksef-cli --env TEST query \
  --identifier 1234567890 \
  --subject-type Subject2 \
  --sort-order Desc \
  --date-from 2026-06-01T00:00:00
```

Limit page size and read the next page:

```shell
ksef-cli --env TEST query \
  --identifier 1234567890 \
  --date-from 2026-06-01T00:00:00 \
  --page-size 100 \
  --page-offset 100
```

#### CSV export

Use `--export FILE` to write invoice metadata to a CSV file. The command still prints the terminal table after writing the file.

```shell
ksef-cli --env TEST query \
  --identifier 1234567890 \
  --date-from 2026-06-01T00:00:00 \
  --date-to 2026-06-30T23:59:59 \
  --export invoices-june-2026.csv
```

The CSV export contains richer metadata than the terminal table.

`third_subjects` is written as a JSON array inside a CSV field.

Query flags:

| Flag                  | Environment variable | Default            | Description                                                                    |
|-----------------------|----------------------|--------------------|--------------------------------------------------------------------------------|
| `-i`, `--identifier`  |                      |                    | Required. Context identifier, usually the taxpayer NIP.                        |
| `-t`, `--token`       | `KSEF_TOKEN`         |                    | KSeF authorisation token. If omitted, the stored encrypted token is used.      |
| `-f`, `--date-from`   |                      |                    | Required. Start of date range, for example `2026-06-01T00:00:00`.              |
| `--date-to`           |                      | KSeF current UTC   | End of date range, for example `2026-06-30T23:59:59`. When omitted, the field is not sent and KSeF applies its default. |
| `--date-type`         |                      | `PermanentStorage` | Date filter type: `Issue`, `Invoicing`, or `PermanentStorage`.                 |
| `--subject-type`      |                      | `Subject1`         | KSeF subject type: `Subject1`, `Subject2`, `Subject3`, or `SubjectAuthorized`. |
| `-s`, `--sort-order`  |                      | `Asc`              | Sort order: `Asc` or `Desc`.                                                   |
| `-o`, `--page-offset` |                      | `0`                | Page offset.                                                                   |
| `-p`, `--page-size`   |                      | `250`              | Page size. KSeF supports up to `250`.                                          |
| `--hwm`               |                      | `false`            | Restrict to permanent storage high water mark date.                            |
| `--self-invoicing`    |                      | `false`            | Include self-invoicing filter.                                                 |
| `--form-type`         |                      | `FA`               | Schema form type: `FA`, `PEF`, or `FA_RR`.                                     |
| `--export`            |                      |                    | Path to CSV export file.                                                       |

### `invoice export`

Starts a KSeF invoice export, waits until the package is ready, downloads it, decrypts it, and writes the resulting ZIP file to the selected path. The CLI does not unpack the ZIP package.

```shell
ksef-cli --env TEST invoice export \
  --identifier 1234567890 \
  --date-from 2026-06-01T00:00:00 \
  --date-to 2026-06-30T23:59:59 \
  ksef-invoices-export.zip
```

Export buyer-side invoices:

```shell
ksef-cli --env TEST invoice export \
  --identifier 1234567890 \
  --subject-type Subject2 \
  --date-from 2026-06-01T00:00:00 \
  invoices.zip
```

Invoice export flags:

| Flag                  | Environment variable | Default            | Description                                                                    |
|-----------------------|----------------------|--------------------|--------------------------------------------------------------------------------|
| `-i`, `--identifier`  |                      |                    | Required. Context identifier, usually the taxpayer NIP.                        |
| `-t`, `--token`       | `KSEF_TOKEN`         |                    | KSeF authorisation token. If omitted, the stored encrypted token is used.      |
| `-f`, `--date-from`   |                      |                    | Required. Start of date range, for example `2026-06-01T00:00:00`.              |
| `--date-to`           |                      | Current UTC time   | End of date range, for example `2026-06-30T23:59:59`.                          |
| `--date-type`         |                      | `PermanentStorage` | Date filter type: `Issue`, `Invoicing`, or `PermanentStorage`.                 |
| `--subject-type`      |                      | `Subject1`         | KSeF subject type: `Subject1`, `Subject2`, `Subject3`, or `SubjectAuthorized`. |
| `--hwm`               |                      | `false`            | Restrict to permanent storage high water mark date.                            |
| `--self-invoicing`    |                      | `false`            | Restrict to self-invoicing invoices.                                           |
| `--form-type`         |                      | `FA`               | Schema form type: `FA`, `PEF`, or `FA_RR`.                                     |
| `--only-metadata`     |                      | `false`            | Export only invoice metadata.                                                  |
| `--poll-interval`     |                      | `5s`               | Invoice export status polling interval.                                        |
| `--wait-timeout`      |                      | `30m`              | Maximum time to wait for export package. Use `0` for no timeout.               |
| `--request-timeout`   |                      | `10m`              | HTTP request timeout used by export operations.                                |

### `report invoices`

Creates CSV files from an invoice export ZIP package. The ZIP package should contain invoice XML files and `_metadata.json` in the root directory.

```shell
ksef-cli report invoices ksef-invoices-export.zip ./report
```

The command writes two files by default:

| File               | Description                                                                 |
|--------------------|-----------------------------------------------------------------------------|
| `invoices.csv`     | Invoice metadata in the same CSV layout as `query --export`.                |
| `invoice_rows.csv` | Invoice line items from `FaWiersz`, linked to metadata by `ksef_number`.     |

Report flags:

| Flag             | Default            | Description                     |
|------------------|--------------------|---------------------------------|
| `--invoices-csv` | `invoices.csv`     | Invoice metadata CSV file name. |
| `--rows-csv`     | `invoice_rows.csv` | Invoice rows CSV file name.     |

### `version`

Prints the CLI version.

```shell
ksef-cli version
```

## Common workflows

### One-off query with a token passed directly

```shell
ksef-cli --env TEST query \
  --identifier 1234567890 \
  --token "__ksef_authorisation_token__" \
  --date-from 2026-06-01T00:00:00 \
  --export invoices.csv
```

### Initialize encrypted storage and reuse the stored token

```shell
ksef-cli init
ksef-cli --env TEST store --identifier 1234567890 --token "__ksef_authorisation_token__"
ksef-cli --env TEST query --identifier 1234567890 --date-from 2026-06-01T00:00:00
```

### Use environment variables instead of repeated flags

```shell
export KSEF_ENVIRONMENT=TEST
export KSEF_TOKEN="__ksef_authorisation_token__"

ksef-cli query \
  --identifier 1234567890 \
  --date-from 2026-06-01T00:00:00 \
  --export invoices.csv
```

## Notes

- The CLI writes logs to `ksef-cli.log` in the current working directory.
- Date-time flags are passed as values like `2026-06-01T00:00:00`.
