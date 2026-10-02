package image

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/disintegration/imaging"
	"github.com/lucasb-eyer/go-colorful"
)

func ValidateFileSize(filePath string, sizeLimit int64) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("error getting file info: %w", err)
	}

	if fileInfo.Size() > sizeLimit {
		return true, nil
	}

	return false, nil
}

func ToString(width int, img image.Image) string {
	img = imaging.Resize(img, width, 0, imaging.Lanczos)
	b := img.Bounds()
	imageWidth := b.Max.X
	h := b.Max.Y
	str := strings.Builder{}

	for heightCounter := 0; heightCounter < h; heightCounter += 2 {
		for x := range imageWidth {
			c1, _ := colorful.MakeColor(img.At(x, heightCounter))
			color1 := lipgloss.Color(c1.Hex())

			var color2 color.Color
			if heightCounter+1 < h {
				c2, _ := colorful.MakeColor(img.At(x, heightCounter+1))
				color2 = lipgloss.Color(c2.Hex())
			} else {
				color2 = color1
			}

			str.WriteString(lipgloss.NewStyle().Foreground(color1).
				Background(color2).Render("▀"))
		}

		str.WriteString("\n")
	}

	return str.String()
}

func ImagePreview(width int, filename string) (string, error) {
	imageContent, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer imageContent.Close()

	img, _, err := image.Decode(imageContent)
	if err != nil {
		return "", err
	}

	imageString := ToString(width, img)

	return imageString, nil
}

// CaptureClipboardImage attempts to extract an image from the system clipboard
// using native platform tools (wl-paste on Wayland, xclip on X11, pngpaste on macOS,
// or PowerShell on Windows) and saves it to destDir.
func CaptureClipboardImage(destDir string) (string, error) {
	if destDir == "" {
		destDir = os.TempDir()
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("creating image destination directory: %w", err)
	}

	targetPath := fmt.Sprintf("%s/clipboard_%d.png", strings.TrimRight(destDir, "/"), os.Getpid())

	// Try Wayland first (wl-paste)
	if _, err := exec.LookPath("wl-paste"); err == nil {
		cmd := exec.Command("wl-paste", "-t", "image/png")
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			if err := os.WriteFile(targetPath, out, 0600); err == nil {
				return targetPath, nil
			}
		}
	}

	// Try X11 (xclip)
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard", "-t", "image/png", "-o")
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			if err := os.WriteFile(targetPath, out, 0600); err == nil {
				return targetPath, nil
			}
		}
	}

	// Try macOS (pngpaste)
	if _, err := exec.LookPath("pngpaste"); err == nil {
		cmd := exec.Command("pngpaste", targetPath)
		if err := cmd.Run(); err == nil {
			if info, statErr := os.Stat(targetPath); statErr == nil && info.Size() > 0 {
				return targetPath, nil
			}
		}
	}

	// Try Windows (PowerShell)
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		psScript := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$img = [System.Windows.Forms.Clipboard]::GetImage()
if ($img -ne $null) {
    $img.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
    exit 0
}
exit 1
`, targetPath)
		cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", psScript)
		if err := cmd.Run(); err == nil {
			if info, statErr := os.Stat(targetPath); statErr == nil && info.Size() > 0 {
				return targetPath, nil
			}
		}
	}

	return "", fmt.Errorf("no image found on clipboard or clipboard utility not found (supported: wl-paste, xclip, pngpaste, powershell)")
}
