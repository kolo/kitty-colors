package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type Palette map[string]string

var (
	config struct {
		KittyDir string
		ThemeDir string
	}

	puttyColourRx = regexp.MustCompile(`^"Colour\d+"="\d+,\d+,\d+"$`)
)

func init() {
	flag.StringVar(&config.KittyDir, "kitty-dir", config.KittyDir, "Set path to kitty directory")
	flag.StringVar(&config.ThemeDir, "theme-dir", config.ThemeDir, "Set path to base16-putty repository")

	toml.DecodeFile(filepath.Join(os.Getenv("USERPROFILE"), ".kitty-colors"), &config)
}

func main() {
	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "usage: kitty-colours [--kittyDir DIR] [--themeDir DIR] session theme\n")
		os.Exit(2)
	}

	session := flag.Args()[0]
	theme := flag.Args()[1]

	themePath := filepath.Join(config.ThemeDir, fmt.Sprintf("base16-%s.reg", theme))
	if !fileExists(themePath) {
		fail(fmt.Errorf("can't find theme %s", theme))
	}

	palette, err := loadPalette(themePath)
	if err != nil {
		fail(err)
	}
	if len(palette) != 22 {
		fail(fmt.Errorf("color palette is not complete"))
	}

	sessionPath := filepath.Join(config.KittyDir, "Sessions", session)
	if !fileExists(sessionPath) {
		fail(fmt.Errorf("can't find session %s", session))
	}

	if err = writePaletteToSession(palette, sessionPath); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return false
	}
	return true
}

func loadPalette(path string) (Palette, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	palette := Palette{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := scanner.Text()
		if puttyColourRx.MatchString(l) {
			color := strings.SplitN(l, "=", 2)
			if len(color) != 2 {
				return nil, fmt.Errorf("cannot load putty colors, string %q is invalid", l)
			}
			palette[strings.Trim(color[0], `"`)] = strings.Trim(color[1], `"`)
		}
	}

	return palette, nil
}

func writePaletteToSession(palette Palette, sessionPath string) error {
	b, err := ioutil.ReadFile(sessionPath)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(sessionPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "Colour") {
			name := strings.SplitN(l, "\\", 2)[0]
			if newColor, ok := palette[name]; ok {
				l = fmt.Sprintf("%s\\%s\\", name, newColor)
			}
		}
		fmt.Fprintln(f, l)
	}

	return nil
}
