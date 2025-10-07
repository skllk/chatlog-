# Windows EXE Build Quickstart

This guide walks through cross-compiling Chatlog into a Windows executable from a Unix-like environment.

## Prerequisites

- Go 1.21 or newer installed locally (the project currently uses Go 1.24)
- Git, make, and a working shell environment
- No native CGO dependencies are required for this build; we compile with `CGO_ENABLED=0`

## Steps

1. **Change into the repository root**
   ```bash
   cd /workspace/chatlog-
   ```

2. **Cross-compile the Windows binary**
   ```bash
   GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -v -o bin/chatlog_windows_amd64.exe ./cmd/chatlog
   ```

   The verbose flag (`-v`) is optional but helpful to observe module downloads during the first build.

3. **Locate the output**
   The resulting executable is placed at:
   ```
   bin/chatlog_windows_amd64.exe
   ```

4. **Transfer to Windows**
   Copy the executable to your Windows machine (via `scp`, shared folders, etc.) and run it directly:
   ```powershell
   .\chatlog_windows_amd64.exe --help
   ```

5. **Limitations**
   - Because we build without CGO, functionality that requires native libraries (e.g., Silk-to-MP3 transcoding) falls back to the stub implementation and will report an explanatory error at runtime.
   - If you need those capabilities, build on Windows with CGO enabled and the appropriate dependencies installed.

