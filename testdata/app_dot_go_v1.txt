commit ad46a5c0d9f4b92d24d3a2c07570f65dd21483eb
Author: dacharyc <dc@dacharycarey.com>
Date:   Fri Dec 26 09:48:07 2025 -0500

    Add hidden flag to support git difftool use case

diff --git a/README.md b/README.md
index 7f8b83d..4a2de37 100644
--- a/README.md
+++ b/README.md
@@ -117,6 +117,34 @@ # Remove all fenestra sockets
 rm -rf ~/.fenestra/
 ```
 
+## Architecture
+
+### Background Process Model
+
+Fenestra uses a background process model to ensure the CLI always exits immediately - essential for integration with tools like `git difftool` that wait for each command to complete before proceeding.
+
+When you run `fenestra`:
+1. The CLI checks for an existing fenestra window (via Unix socket)
+2. If found, it sends the file via IPC and exits immediately
+3. If not found, it spawns the GUI as a background process, waits for the socket to be ready, then exits
+
+This means every `fenestra` invocation returns immediately, while windows run independently in the background.
+
+### Wails v2 Limitation
+
+Wails v2 only supports a single window per application process. This means each "window group" (files opened within 2 seconds) runs in its own process with its own Wails stack.
+
+The original Swift version (Fenestro) used a single-process architecture where one app managed multiple windows via NSDocument. This was more memory-efficient but macOS-specific.
+
+### Future: Wails v3
+
+[Wails v3](https://v3alpha.wails.io/whats-new/) introduces native multi-window support, which would allow fenestra to use a single-process daemon architecture:
+- One persistent process managing all windows
+- Lower memory footprint
+- Simpler IPC (all windows in one process)
+
+When Wails v3 reaches stable release, consider refactoring to this architecture. The relevant tracking issue is [wailsapp/wails#1480](https://github.com/wailsapp/wails/issues/1480).
+
 ## History
 
 Fenestra is a rewrite of [Fenestro](https://github.com/masukomi/fenestro), originally written in Swift. This Go version uses Wails for native macOS WebView integration.
diff --git a/background_test.go b/background_test.go
new file mode 100644
index 0000000..185dedc
--- /dev/null
+++ b/background_test.go
@@ -0,0 +1,235 @@
+package main
+
+import (
+	"encoding/json"
+	"net"
+	"os"
+	"path/filepath"
+	"testing"
+	"time"
+)
+
+// TestSocketPolling verifies the socket polling logic used in spawnGUIBackground
+func TestSocketPolling(t *testing.T) {
+	socketPath := filepath.Join(os.TempDir(), "fenestra-test-polling.sock")
+	os.Remove(socketPath)
+
+	// Start a goroutine that creates the socket after a delay
+	go func() {
+		time.Sleep(100 * time.Millisecond)
+		listener, err := net.Listen("unix", socketPath)
+		if err != nil {
+			t.Errorf("Failed to create socket: %v", err)
+			return
+		}
+		defer listener.Close()
+		// Keep socket alive for the test
+		time.Sleep(500 * time.Millisecond)
+	}()
+
+	// Poll for socket (simulating spawnGUIBackground logic)
+	deadline := time.Now().Add(1 * time.Second)
+	found := false
+	for time.Now().Before(deadline) {
+		if _, err := os.Stat(socketPath); err == nil {
+			found = true
+			break
+		}
+		time.Sleep(10 * time.Millisecond)
+	}
+
+	if !found {
+		t.Error("Socket polling failed to detect socket creation")
+	}
+}
+
+// TestSocketPollingTimeout verifies polling times out correctly
+func TestSocketPollingTimeout(t *testing.T) {
+	socketPath := filepath.Join(os.TempDir(), "fenestra-test-polling-timeout.sock")
+	os.Remove(socketPath)
+
+	// Poll for non-existent socket with short timeout
+	deadline := time.Now().Add(100 * time.Millisecond)
+	found := false
+	for time.Now().Before(deadline) {
+		if _, err := os.Stat(socketPath); err == nil {
+			found = true
+			break
+		}
+		time.Sleep(10 * time.Millisecond)
+	}
+
+	if found {
+		t.Error("Socket polling should have timed out for non-existent socket")
+	}
+}
+
+// TestTempFileCreation verifies temp file is created with correct content
+func TestTempFileCreation(t *testing.T) {
+	content := "<html><body>Test content</body></html>"
+
+	tmpFile, err := os.CreateTemp("", "fenestra-*.html")
+	if err != nil {
+		t.Fatalf("Failed to create temp file: %v", err)
+	}
+	tmpPath := tmpFile.Name()
+	defer os.Remove(tmpPath)
+
+	if _, err := tmpFile.WriteString(content); err != nil {
+		tmpFile.Close()
+		t.Fatalf("Failed to write to temp file: %v", err)
+	}
+	tmpFile.Close()
+
+	// Verify content
+	readContent, err := os.ReadFile(tmpPath)
+	if err != nil {
+		t.Fatalf("Failed to read temp file: %v", err)
+	}
+
+	if string(readContent) != content {
+		t.Errorf("Content mismatch: got %q, want %q", string(readContent), content)
+	}
+
+	// Verify file has fenestra prefix
+	if !filepath.HasPrefix(filepath.Base(tmpPath), "fenestra-") {
+		t.Errorf("Temp file should have fenestra- prefix, got %s", filepath.Base(tmpPath))
+	}
+}
+
+// TestTempFileCleanup verifies temp file is removed after reading when --temp-file flag is used
+func TestTempFileCleanup(t *testing.T) {
+	content := "<html><body>Cleanup test</body></html>"
+
+	// Create temp file (simulating what spawnGUIBackground does)
+	tmpFile, err := os.CreateTemp("", "fenestra-*.html")
+	if err != nil {
+		t.Fatalf("Failed to create temp file: %v", err)
+	}
+	tmpPath := tmpFile.Name()
+
+	if _, err := tmpFile.WriteString(content); err != nil {
+		tmpFile.Close()
+		os.Remove(tmpPath)
+		t.Fatalf("Failed to write temp file: %v", err)
+	}
+	tmpFile.Close()
+
+	// Simulate what the child process does with --temp-file flag:
+	// Read content then delete
+	readContent, err := os.ReadFile(tmpPath)
+	if err != nil {
+		t.Fatalf("Failed to read temp file: %v", err)
+	}
+
+	if string(readContent) != content {
+		t.Errorf("Content mismatch before cleanup")
+	}
+
+	// Simulate cleanup (as done in main.go when tempFile flag is true)
+	os.Remove(tmpPath)
+
+	// Verify file is gone
+	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
+		t.Error("Temp file should have been removed")
+		os.Remove(tmpPath) // Cleanup if test failed
+	}
+}
+
+// TestSequentialCLISimulation simulates the sequential CLI invocation pattern
+// that git difftool uses: each invocation should be able to connect to the socket
+func TestSequentialCLISimulation(t *testing.T) {
+	app := NewApp(FileEntry{Name: "initial", Content: "<html>initial</html>"}, "")
+
+	socketPath := filepath.Join(os.TempDir(), "fenestra-test-cli-sim.sock")
+	os.Remove(socketPath)
+
+	server, err := NewIPCServer(app, socketPath, true) // Use timeout mode like sidebar
+	if err != nil {
+		t.Fatalf("NewIPCServer() failed: %v", err)
+	}
+	server.Start()
+	defer server.Close()
+
+	// Simulate 3 sequential CLI invocations
+	// Each one should:
+	// 1. Find the socket exists (first one creates it, subsequent ones find it)
+	// 2. Connect and send file
+	// 3. Exit (return) immediately
+
+	for i := 0; i < 3; i++ {
+		// Verify socket exists (what CLI does before connecting)
+		if _, err := os.Stat(socketPath); os.IsNotExist(err) {
+			t.Fatalf("Socket should exist for invocation %d", i)
+		}
+
+		// Connect and send (simulating TrySendToSidebarInstance)
+		conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
+		if err != nil {
+			t.Fatalf("Invocation %d failed to connect: %v", i, err)
+		}
+
+		cmd := IPCCommand{
+			Cmd: "add-file",
+			Entry: FileEntry{
+				Name:    "file" + string(rune('1'+i)) + ".html",
+				Path:    "/tmp/file" + string(rune('1'+i)) + ".html",
+				Content: "<html>File " + string(rune('1'+i)) + "</html>",
+			},
+		}
+
+		encoder := json.NewEncoder(conn)
+		if err := encoder.Encode(cmd); err != nil {
+			conn.Close()
+			t.Fatalf("Invocation %d failed to send: %v", i, err)
+		}
+		conn.Close()
+
+		// Small delay between invocations (simulating real CLI timing)
+		time.Sleep(20 * time.Millisecond)
+	}
+
+	// Give server time to process all files
+	time.Sleep(50 * time.Millisecond)
+
+	// Verify all files arrived
+	files := app.GetFiles()
+	expectedCount := 4 // initial + 3 added
+	if len(files) != expectedCount {
+		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
+	}
+}
+
+// TestFirstInvocationCreatesSocket verifies that the first invocation creates the socket
+// and subsequent invocations can immediately connect
+func TestFirstInvocationCreatesSocket(t *testing.T) {
+	socketPath := filepath.Join(os.TempDir(), "fenestra-test-first-invoke.sock")
+	os.Remove(socketPath)
+
+	// Verify socket doesn't exist initially
+	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
+		t.Fatal("Socket should not exist initially")
+	}
+
+	// Create app and server (simulating first invocation)
+	app := NewApp(FileEntry{Name: "first", Content: "<html>first</html>"}, "")
+	server, err := NewIPCServer(app, socketPath, true)
+	if err != nil {
+		t.Fatalf("NewIPCServer() failed: %v", err)
+	}
+	server.Start()
+	defer server.Close()
+
+	// Socket should exist immediately after NewIPCServer (before Start even)
+	// This is the guarantee that spawnGUIBackground relies on
+	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
+		t.Error("Socket should exist immediately after NewIPCServer")
+	}
+
+	// Subsequent invocation should be able to connect immediately
+	conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
+	if err != nil {
+		t.Fatalf("Subsequent invocation failed to connect: %v", err)
+	}
+	conn.Close()
+}
diff --git a/main.go b/main.go
index d03c4b1..14d90b0 100644
--- a/main.go
+++ b/main.go
@@ -6,7 +6,10 @@
 	"fmt"
 	"io"
 	"os"
+	"os/exec"
 	"path/filepath"
+	"syscall"
+	"time"
 
 	"github.com/google/uuid"
 	flag "github.com/spf13/pflag"
@@ -26,6 +29,8 @@
 	displayName string
 	windowID    string
 	showVersion bool
+	internalGUI bool // Hidden flag: run as GUI subprocess
+	tempFile    bool // Hidden flag: delete file after reading (for stdin content)
 )
 
 func init() {
@@ -33,6 +38,10 @@ func init() {
 	flag.StringVarP(&displayName, "name", "n", "", "Display name for the window title")
 	flag.StringVar(&windowID, "id", "", "Window ID: use 'new' to generate ID, or provide existing UUID to target that window")
 	flag.BoolVarP(&showVersion, "version", "v", false, "Show version")
+	flag.BoolVar(&internalGUI, "internal-gui", false, "Internal: run as GUI subprocess")
+	flag.BoolVar(&tempFile, "temp-file", false, "Internal: delete file after reading")
+	flag.CommandLine.MarkHidden("internal-gui")
+	flag.CommandLine.MarkHidden("temp-file")
 }
 
 func main() {
@@ -45,6 +54,7 @@ func main() {
 
 	// Determine content source and create FileEntry
 	var entry FileEntry
+	var fromStdin bool
 
 	if filePath != "" {
 		// Load from file path
@@ -66,6 +76,10 @@ func main() {
 		if entry.Name == "" {
 			entry.Name = filepath.Base(filePath)
 		}
+		// If this was a temp file (from stdin in parent), clean it up after reading
+		if tempFile {
+			os.Remove(absPath)
+		}
 	} else if !isTerminal(os.Stdin) {
 		// Read from stdin
 		content, err := io.ReadAll(os.Stdin)
@@ -81,6 +95,7 @@ func main() {
 		if entry.Name == "" {
 			entry.Name = "stdin"
 		}
+		fromStdin = true
 	} else {
 		// No input provided
 		fmt.Println("Usage: fenestra [-p path] [-n name] [-id [window-id]]")
@@ -104,15 +119,22 @@ func main() {
 	// Check if we're using window ID mode
 	isWindowIDMode := windowID != ""
 
+	// Handle window ID "new" - generate UUID before any IPC or spawning
+	if isWindowIDMode && windowID == "new" {
+		windowID = uuid.New().String()
+		fmt.Println(windowID)
+	}
+
+	// If this is the GUI subprocess, run the GUI directly
+	if internalGUI {
+		runGUI(entry, windowID, isWindowIDMode)
+		return
+	}
+
+	// CLI invocation - try to send to existing instance first
 	if isWindowIDMode {
-		// Window ID mode
-		if windowID == "new" {
-			// Generate new window ID
-			windowID = uuid.New().String()
-			// Print the ID to stdout for the caller to capture
-			fmt.Println(windowID)
-		} else {
-			// Validate that window ID is a valid UUID
+		if windowID != "" {
+			// Validate UUID format (skip if we just generated it above)
 			if _, err := uuid.Parse(windowID); err != nil {
 				fmt.Fprintf(os.Stderr, "Error: Invalid window ID format (expected UUID): %s\n", windowID)
 				os.Exit(1)
@@ -121,7 +143,6 @@ func main() {
 			if TrySendToWindowInstance(windowID, entry) {
 				os.Exit(0)
 			}
-			// Window doesn't exist, create new one with that ID
 		}
 	} else {
 		// Sidebar mode - try to send to existing instance
@@ -130,6 +151,83 @@ func main() {
 		}
 	}
 
+	// No existing instance - spawn GUI in background and exit
+	if err := spawnGUIBackground(entry, windowID, fromStdin); err != nil {
+		fmt.Fprintf(os.Stderr, "Error spawning GUI: %v\n", err)
+		os.Exit(1)
+	}
+	os.Exit(0)
+}
+
+// spawnGUIBackground spawns the GUI as a background process and waits for the socket to be ready
+func spawnGUIBackground(entry FileEntry, windowID string, fromStdin bool) error {
+	exe, err := os.Executable()
+	if err != nil {
+		return fmt.Errorf("failed to get executable path: %w", err)
+	}
+
+	args := []string{"--internal-gui"}
+
+	// Handle content: if from stdin, write to temp file; otherwise use original path
+	if fromStdin {
+		tmpFile, err := os.CreateTemp("", "fenestra-*.html")
+		if err != nil {
+			return fmt.Errorf("failed to create temp file: %w", err)
+		}
+		if _, err := tmpFile.WriteString(entry.Content); err != nil {
+			tmpFile.Close()
+			os.Remove(tmpFile.Name())
+			return fmt.Errorf("failed to write temp file: %w", err)
+		}
+		tmpFile.Close()
+		args = append(args, "-p", tmpFile.Name(), "--temp-file")
+	} else {
+		args = append(args, "-p", entry.Path)
+	}
+
+	// Pass display name if it was explicitly set
+	if displayName != "" {
+		args = append(args, "-n", displayName)
+	}
+
+	// Pass window ID if set
+	if windowID != "" {
+		args = append(args, "-id", windowID)
+	}
+
+	// Spawn the child process detached
+	cmd := exec.Command(exe, args...)
+	cmd.SysProcAttr = &syscall.SysProcAttr{
+		Setsid: true, // Create new session so child survives parent exit
+	}
+	// Don't inherit stdin (child reads from file), but keep stderr for errors
+	cmd.Stderr = os.Stderr
+
+	if err := cmd.Start(); err != nil {
+		return fmt.Errorf("failed to start GUI process: %w", err)
+	}
+
+	// Wait for socket to be created (guarantees subsequent invocations can connect)
+	var socketPath string
+	if windowID != "" {
+		socketPath = getWindowSocketPath(windowID)
+	} else {
+		socketPath = getSidebarSocketPath()
+	}
+
+	deadline := time.Now().Add(5 * time.Second)
+	for time.Now().Before(deadline) {
+		if _, err := os.Stat(socketPath); err == nil {
+			return nil // Socket exists, child is ready
+		}
+		time.Sleep(10 * time.Millisecond)
+	}
+
+	return fmt.Errorf("timeout waiting for GUI to start")
+}
+
+// runGUI runs the Wails application (called from GUI subprocess)
+func runGUI(entry FileEntry, windowID string, isWindowIDMode bool) {
 	// Create app with the file entry
 	app := NewApp(entry, windowID)
 
