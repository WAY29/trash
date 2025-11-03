# trash (Windows Recycle Bin CLI)

Windows-only CLI that moves files/directories to the Recycle Bin (like rm without flags).

## Usage

```bash
trash <file-or-dir> [more paths]
```

- Expands Windows globs `*` and `?`.
- Processes each path independently. Errors are printed but other paths continue.
- Exits with code 1 if any errors occurred; otherwise 0.
- Silent on success.

## Notes

- Requires Windows. Uses `SHFileOperationW` with `FOF_ALLOWUNDO` to send items to Recycle Bin.
- Flags are disabled intentionally; this is a minimal, no-args utility.