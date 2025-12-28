commit 97cfec101131551437441f4bce21d04566161c33
Author: dacharyc <dc@dacharycarey.com>
Date:   Fri Dec 26 10:16:14 2025 -0500

    Add config support and default font size

diff --git a/README.md b/README.md
index 4a2de37..92b625b 100644
--- a/README.md
+++ b/README.md
@@ -7,6 +7,8 @@ ## Features
 - Display HTML from files or stdin
 - Native macOS WebView
 - Cmd+F find-in-page with highlight navigation
+- Zoom in/out with Cmd+/Cmd-
+- Configurable default font size
 - Sidebar for multiple files (files opened within 2 seconds are grouped together)
 - Window ID mode for live content updates
 - Dark mode support for UI elements
@@ -85,9 +87,33 @@ # The window finds the file by path, updates its content, and displays it
 ## Keyboard Shortcuts
 
 - **Cmd+F** - Find in page
+- **Cmd+Plus** - Zoom in
+- **Cmd+Minus** - Zoom out
+- **Cmd+0** - Reset zoom to 100%
 - **Cmd+W** - Close window
 - **Cmd+Q** - Quit
 
+## Configuration
+
+Fenestra supports a TOML configuration file following the [XDG Base Directory](https://specifications.freedesktop.org/basedir-spec/latest/) standard.
+
+**Location:** `$XDG_CONFIG_HOME/fenestra/config.toml` (defaults to `~/.config/fenestra/config.toml`)
+
+### Example config.toml
+
+```toml
+# Default font size in pixels (0 or omit to use browser default)
+font_size = 18
+```
+
+### Available Options
+
+| Option | Type | Default | Description |
+|--------|------|---------|-------------|
+| `font_size` | integer | 0 | Base font size in pixels. Set to 0 to use browser default. |
+
+The font size setting works alongside zoom (Cmd+/Cmd-) for additional flexibility.
+
 ## Development
 
 ```bash
diff --git a/app.go b/app.go
index dd1bd62..69d5d48 100644
--- a/app.go
+++ b/app.go
@@ -13,6 +13,7 @@ type App struct {
 	files        []FileEntry
 	currentIndex int
 	windowID     string
+	config       Config
 	mu           sync.RWMutex
 }
 
@@ -22,6 +23,7 @@ func NewApp(file FileEntry, windowID string) *App {
 		files:        []FileEntry{file},
 		currentIndex: 0,
 		windowID:     windowID,
+		config:       LoadConfig(),
 	}
 }
 
@@ -147,3 +149,8 @@ func (a *App) ReplaceFileContent(path, content, name string) {
 func (a *App) GetWindowID() string {
 	return a.windowID
 }
+
+// GetConfig returns the application configuration
+func (a *App) GetConfig() Config {
+	return a.config
+}
diff --git a/config.go b/config.go
new file mode 100644
index 0000000..edff37b
--- /dev/null
+++ b/config.go
@@ -0,0 +1,69 @@
+package main
+
+import (
+	"os"
+	"path/filepath"
+
+	"github.com/BurntSushi/toml"
+)
+
+// Config holds the application configuration
+type Config struct {
+	// FontSize is the default font size in pixels (e.g., 16, 18, 24)
+	FontSize int `toml:"font_size"`
+}
+
+// DefaultConfig returns the default configuration values
+func DefaultConfig() Config {
+	return Config{
+		FontSize: 0, // 0 means use browser default
+	}
+}
+
+// getConfigDir returns the config directory following XDG Base Directory standard
+func getConfigDir() string {
+	// Check XDG_CONFIG_HOME first
+	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
+		return filepath.Join(xdgConfigHome, "fenestra")
+	}
+	// Fall back to ~/.config/fenestra
+	home, err := os.UserHomeDir()
+	if err != nil {
+		return ""
+	}
+	return filepath.Join(home, ".config", "fenestra")
+}
+
+// getConfigPath returns the full path to the config file
+func getConfigPath() string {
+	configDir := getConfigDir()
+	if configDir == "" {
+		return ""
+	}
+	return filepath.Join(configDir, "config.toml")
+}
+
+// LoadConfig loads the configuration from the config file
+// Returns default config if file doesn't exist or can't be read
+func LoadConfig() Config {
+	config := DefaultConfig()
+
+	configPath := getConfigPath()
+	if configPath == "" {
+		return config
+	}
+
+	// Check if config file exists
+	if _, err := os.Stat(configPath); os.IsNotExist(err) {
+		return config
+	}
+
+	// Parse the config file
+	if _, err := toml.DecodeFile(configPath, &config); err != nil {
+		// Log error but continue with defaults
+		// Don't fail startup due to config issues
+		return DefaultConfig()
+	}
+
+	return config
+}
diff --git a/config_test.go b/config_test.go
new file mode 100644
index 0000000..76bfcee
--- /dev/null
+++ b/config_test.go
@@ -0,0 +1,175 @@
+package main
+
+import (
+	"os"
+	"path/filepath"
+	"testing"
+)
+
+func TestDefaultConfig(t *testing.T) {
+	config := DefaultConfig()
+	if config.FontSize != 0 {
+		t.Errorf("Expected default FontSize to be 0, got %d", config.FontSize)
+	}
+}
+
+func TestGetConfigDirWithXDGConfigHome(t *testing.T) {
+	// Save and restore XDG_CONFIG_HOME
+	original := os.Getenv("XDG_CONFIG_HOME")
+	defer os.Setenv("XDG_CONFIG_HOME", original)
+
+	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
+	dir := getConfigDir()
+	expected := "/custom/config/fenestra"
+	if dir != expected {
+		t.Errorf("Expected %s, got %s", expected, dir)
+	}
+}
+
+func TestGetConfigDirWithoutXDGConfigHome(t *testing.T) {
+	// Save and restore XDG_CONFIG_HOME
+	original := os.Getenv("XDG_CONFIG_HOME")
+	defer os.Setenv("XDG_CONFIG_HOME", original)
+
+	os.Unsetenv("XDG_CONFIG_HOME")
+	dir := getConfigDir()
+
+	home, err := os.UserHomeDir()
+	if err != nil {
+		t.Fatalf("Could not get home dir: %v", err)
+	}
+	expected := filepath.Join(home, ".config", "fenestra")
+	if dir != expected {
+		t.Errorf("Expected %s, got %s", expected, dir)
+	}
+}
+
+func TestLoadConfigNoFile(t *testing.T) {
+	// Save and restore XDG_CONFIG_HOME
+	original := os.Getenv("XDG_CONFIG_HOME")
+	defer os.Setenv("XDG_CONFIG_HOME", original)
+
+	// Point to a directory that doesn't exist
+	os.Setenv("XDG_CONFIG_HOME", "/nonexistent/path")
+	config := LoadConfig()
+
+	// Should return defaults
+	if config.FontSize != 0 {
+		t.Errorf("Expected default FontSize 0, got %d", config.FontSize)
+	}
+}
+
+func TestLoadConfigFromFile(t *testing.T) {
+	// Create temp directory
+	tmpDir, err := os.MkdirTemp("", "fenestra-config-test")
+	if err != nil {
+		t.Fatalf("Could not create temp dir: %v", err)
+	}
+	defer os.RemoveAll(tmpDir)
+
+	// Create config directory and file
+	configDir := filepath.Join(tmpDir, "fenestra")
+	if err := os.MkdirAll(configDir, 0755); err != nil {
+		t.Fatalf("Could not create config dir: %v", err)
+	}
+
+	configPath := filepath.Join(configDir, "config.toml")
+	configContent := `font_size = 24`
+	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
+		t.Fatalf("Could not write config file: %v", err)
+	}
+
+	// Save and restore XDG_CONFIG_HOME
+	original := os.Getenv("XDG_CONFIG_HOME")
+	defer os.Setenv("XDG_CONFIG_HOME", original)
+
+	os.Setenv("XDG_CONFIG_HOME", tmpDir)
+	config := LoadConfig()
+
+	if config.FontSize != 24 {
+		t.Errorf("Expected FontSize 24, got %d", config.FontSize)
+	}
+}
+
+func TestLoadConfigInvalidTOML(t *testing.T) {
+	// Create temp directory
+	tmpDir, err := os.MkdirTemp("", "fenestra-config-test")
+	if err != nil {
+		t.Fatalf("Could not create temp dir: %v", err)
+	}
+	defer os.RemoveAll(tmpDir)
+
+	// Create config directory and file with invalid TOML
+	configDir := filepath.Join(tmpDir, "fenestra")
+	if err := os.MkdirAll(configDir, 0755); err != nil {
+		t.Fatalf("Could not create config dir: %v", err)
+	}
+
+	configPath := filepath.Join(configDir, "config.toml")
+	configContent := `this is not valid toml [`
+	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
+		t.Fatalf("Could not write config file: %v", err)
+	}
+
+	// Save and restore XDG_CONFIG_HOME
+	original := os.Getenv("XDG_CONFIG_HOME")
+	defer os.Setenv("XDG_CONFIG_HOME", original)
+
+	os.Setenv("XDG_CONFIG_HOME", tmpDir)
+	config := LoadConfig()
+
+	// Should return defaults on invalid TOML
+	if config.FontSize != 0 {
+		t.Errorf("Expected default FontSize 0 on invalid TOML, got %d", config.FontSize)
+	}
+}
+
+func TestLoadConfigPartialConfig(t *testing.T) {
+	// Create temp directory
+	tmpDir, err := os.MkdirTemp("", "fenestra-config-test")
+	if err != nil {
+		t.Fatalf("Could not create temp dir: %v", err)
+	}
+	defer os.RemoveAll(tmpDir)
+
+	// Create config directory and file with only some options
+	configDir := filepath.Join(tmpDir, "fenestra")
+	if err := os.MkdirAll(configDir, 0755); err != nil {
+		t.Fatalf("Could not create config dir: %v", err)
+	}
+
+	configPath := filepath.Join(configDir, "config.toml")
+	// Empty config file - should use defaults
+	configContent := `# empty config`
+	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
+		t.Fatalf("Could not write config file: %v", err)
+	}
+
+	// Save and restore XDG_CONFIG_HOME
+	original := os.Getenv("XDG_CONFIG_HOME")
+	defer os.Setenv("XDG_CONFIG_HOME", original)
+
+	os.Setenv("XDG_CONFIG_HOME", tmpDir)
+	config := LoadConfig()
+
+	// Should use default for unspecified options
+	if config.FontSize != 0 {
+		t.Errorf("Expected default FontSize 0, got %d", config.FontSize)
+	}
+}
+
+func TestGetConfig(t *testing.T) {
+	// Save and restore XDG_CONFIG_HOME
+	original := os.Getenv("XDG_CONFIG_HOME")
+	defer os.Setenv("XDG_CONFIG_HOME", original)
+
+	// Point to nonexistent dir so we get defaults
+	os.Setenv("XDG_CONFIG_HOME", "/nonexistent/path")
+
+	app := NewApp(FileEntry{Name: "test", Content: "<html></html>"}, "")
+	config := app.GetConfig()
+
+	if config.FontSize != 0 {
+		t.Errorf("Expected default FontSize 0, got %d", config.FontSize)
+	}
+}
diff --git a/frontend/main.js b/frontend/main.js
index 0a95d54..612672c 100644
--- a/frontend/main.js
+++ b/frontend/main.js
@@ -328,8 +328,21 @@
         }
     });
 
+    // Load and apply configuration
+    async function loadConfig() {
+        try {
+            const config = await window.go.main.App.GetConfig();
+            if (config.font_size && config.font_size > 0) {
+                content.style.fontSize = config.font_size + 'px';
+            }
+        } catch (err) {
+            console.error('Error loading config:', err);
+        }
+    }
+
     // Initialize
     document.addEventListener('DOMContentLoaded', async () => {
+        await loadConfig();
         await loadContent();
         await loadFiles();
     });
diff --git a/go.mod b/go.mod
index 2a35775..10d4616 100644
--- a/go.mod
+++ b/go.mod
@@ -5,6 +5,7 @@ go 1.24.0
 toolchain go1.24.6
 
 require (
+	github.com/BurntSushi/toml v1.6.0
 	github.com/google/uuid v1.6.0
 	github.com/spf13/pflag v1.0.10
 	github.com/wailsapp/wails/v2 v2.11.0
diff --git a/go.sum b/go.sum
index f4e715a..6822220 100644
--- a/go.sum
+++ b/go.sum
@@ -1,3 +1,5 @@
+github.com/BurntSushi/toml v1.6.0 h1:dRaEfpa2VI55EwlIW72hMRHdWouJeRF7TPYhI+AUQjk=
+github.com/BurntSushi/toml v1.6.0/go.mod h1:ukJfTF/6rtPPRCnwkur4qwRxa8vTRFBF0uk2lLoLwho=
 github.com/bep/debounce v1.2.1 h1:v67fRdBA9UQu2NhLFXrSg0Brw7CexQekrBwDMM8bzeY=
 github.com/bep/debounce v1.2.1/go.mod h1:H8yggRPQKLUhUoqrJC1bO2xNya7vanpDl7xR3ISbCJ0=
 github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
