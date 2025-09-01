package utils

import (
	"fmt"
	"github.com/fatih/color"
)

var (
	Red     = color.New(color.FgRed).SprintFunc()
	Green   = color.New(color.FgGreen).SprintFunc()
	Yellow  = color.New(color.FgYellow).SprintFunc()
	Blue    = color.New(color.FgBlue).SprintFunc()
	Magenta = color.New(color.FgMagenta).SprintFunc()
	Cyan    = color.New(color.FgCyan).SprintFunc()
	White   = color.New(color.FgWhite).SprintFunc()
	
	BoldRed     = color.New(color.FgRed, color.Bold).SprintFunc()
	BoldGreen   = color.New(color.FgGreen, color.Bold).SprintFunc()
	BoldYellow  = color.New(color.FgYellow, color.Bold).SprintFunc()
	BoldBlue    = color.New(color.FgBlue, color.Bold).SprintFunc()
	BoldMagenta = color.New(color.FgMagenta, color.Bold).SprintFunc()
	BoldCyan    = color.New(color.FgCyan, color.Bold).SprintFunc()
	BoldWhite   = color.New(color.FgWhite, color.Bold).SprintFunc()
)

func PrintSuccess(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", BoldGreen("✓"), fmt.Sprintf(format, args...))
}

func PrintError(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", BoldRed("✗"), fmt.Sprintf(format, args...))
}

func PrintWarning(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", BoldYellow("⚠"), fmt.Sprintf(format, args...))
}

func PrintInfo(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", BoldBlue("ℹ"), fmt.Sprintf(format, args...))
}

func PrintSection(title string) {
	fmt.Println()
	fmt.Println(BoldCyan("━━━ " + title + " ━━━"))
}

func PrintSubSection(title string) {
	fmt.Println(BoldWhite("─── " + title + " ───"))
}

func PrintKeyValue(key, value string) {
	fmt.Printf("  %s: %s\n", BoldWhite(key), value)
}

func PrintProtocolInfo(name, tag, port, status string) {
	statusColor := Green
	if status == "stopped" {
		statusColor = Red
	} else if status == "unknown" {
		statusColor = Yellow
	}
	
	fmt.Printf("  %s %s [%s] %s\n", 
		BoldCyan("•"),
		BoldWhite(name),
		Yellow("Port: "+port),
		statusColor(status))
	
	if tag != "" {
		fmt.Printf("    Tag: %s\n", tag)
	}
}

func DisableColor() {
	color.NoColor = true
}

func EnableColor() {
	color.NoColor = false
}