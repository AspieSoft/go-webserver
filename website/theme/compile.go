package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AspieSoft/go-regex/v8"
	"github.com/AspieSoft/goutil/fs/v3"
	"github.com/AspieSoft/goutil/v7"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
	"gopkg.in/yaml.v3"
)

var config = map[string]string{
	"theme_name": "Smart Theme",
	"theme_version": "v0.1.1",
	"theme_uri": "github.com/AspieSoft/smart-theme",
	"theme_license": "MIT License",
}

var validExtList []string = []string{
	"css",
	"js",
	"less",
	"md",
	"txt",
	"pdf",
	"png",
	"jpg",
	"gif",
	"mp3",
	"mp4",
	"mov",
	"ogg",
	"wav",
	"webp",
	"webm",
	"weba",
	"ttf",
	"woff",
}

type ThemeConfigData struct {
	Theme string
	DefaultDarkMode bool

	Config map[string]string

	Layout struct {
		PagePaddingInline string
		PagePaddingInlineMobile string
		PageContentMaxWidth string
		PageBreakoutMaxWidth string
		PageHardMaxWidth string

		WidgetSize string
		WidgetMaxSize string

		HeaderimgWidth string
		HeaderimgHeight string
		HeaderimgHeightHome string
		HeadernavJustify string
		NavUnderlineRadius string

		SideNavWidth string

  	ShadowSize string
  	TextshadowSize string

		BorderRadius string
	}

	Topography struct {
		FontSize string
		LineHeight string
		Font map[string]string
		ImportFonts []ThemeConfigImportFont
	}

	Text struct {
		Light string
		Dark string
		LightLink string
		DarkLink string
	}

	Heading struct {
		Light string
		Dark string
		LightStrong string
		DarkStrong string
		Font string
	}

	Input struct {
		Light string
		Dark string
		Font string
	}

	Shadow struct {
		Light string
		Dark string
		LightText string
		DarkText string
	}

	Color map[string]ThemeConfigColor
	Element map[string]ThemeConfigElement
}

type ThemeConfigDataConfig struct {
	Config map[string]string
}

type ThemeConfigColor struct {
	Light string
	Dark string
	FG string
	Font string
}

type ThemeConfigElement struct {
	Light string
	Dark string
	FG string
	Font string
	Img struct {
		Light string
		Dark string
		Size string
		Pos string
		Att string
		Blend string
	}
}

type ThemeConfigImportFont struct {
	Name string
	Local string
	Src string
	Format string
	Display string
}

// this will be incramented when files change and the theme gets recompiled
// uint8 should be ok, since the hot reload script only cares whether or not
// the number is the same, and does not care if its a bigger or smaller number
var currentReversion = uint8(0);

func main(){
	port := ""
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil && p >= 3000 && p <= 65535 /* golang only accepts 16 bit port numbers */ {
			port = strconv.Itoa(p)
		}else if p, err := strconv.Atoi(os.Args[1]); err == nil && p == 0 {
			port = "0"
		}
	}

	startTime := time.Now()
	themeDir, subThemePath, defaultDarkMode, err := handleCompileTheme(port)
	printCompTime("", startTime)

	if port == "" || err != nil {
		return
	}

	if port != "0" {
		watchTestDirAndReloadClientPage()

		http.HandleFunc("/reversion.test", func(w http.ResponseWriter, r *http.Request) {
			res, err := goutil.JSON.Stringify(map[string]interface{}{
				"reversion": currentReversion,
			})

			w.Header().Set("Content-Type", "application/json")

			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("error 500"))
				return
			}

			w.WriteHeader(200)
			w.Write(res)
		})

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			url := string(regex.Comp(`[^\w_\-\/\.]+`).RepStrLit([]byte(strings.Trim(r.URL.Path, "/")), []byte{}))
			if url == "" {
				http.ServeFile(w, r, "./test/html/index.html")
				return
			}else if strings.HasPrefix(url, "theme/") || strings.HasPrefix(url, "assets/") {
				for _, ext := range validExtList {
					if strings.HasSuffix(url, "."+ext) {
						if strings.HasPrefix(url, "theme/") {
							if path, err := fs.JoinPath("./dist", strings.Replace(url, "theme/", "", 1)); err == nil {
								http.ServeFile(w, r, path)
								return
							}
						}else if strings.HasPrefix(url, "assets/") {
							if path, err := fs.JoinPath("./test/assets", strings.Replace(url, "assets/", "", 1)); err == nil {
								http.ServeFile(w, r, path)
								return
							}
						}
					}
				}
			}else{
				url = string(regex.Comp(`\.[\w_-]+$`).RepStrLit([]byte(url), []byte{}))
	
				if path, err := fs.JoinPath("./test/html", url+".html"); err == nil {
					if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
						http.ServeFile(w, r, path)
						return
					}
				}
	
				if path, err := fs.JoinPath("./test/html", url, "index.html"); err == nil {
					if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
						http.ServeFile(w, r, path)
						return
					}
				}
			}
	
			w.WriteHeader(404)
			w.Write([]byte("error 404"))
		})
	}

	go func(){
		for {
			var input string
			fmt.Print("> ")
			fmt.Scanln(&input)
			fmt.Print("\033[1A", strings.Repeat(" ", 20), "\r")

			if input == "" || input == "stop" || input == "exit" {
				fmt.Println("\x1b[1;31mStopping Server!", "\x1b[0m")

				time.Sleep(300 * time.Millisecond)
				os.Exit(0)
			}else if input == "compile" || input == "comp" {
				fmt.Print("\x1b[1;36mCompiling Theme...", "\x1b[0m")
				time.Sleep(300 * time.Millisecond)
				startTime := time.Now()

				os.RemoveAll("./dist")
				os.Mkdir("./dist", 0755)

				*subThemePath, *defaultDarkMode, _ = compileConfig()
				compileCSS(*themeDir, "style", true, *subThemePath, *defaultDarkMode)
				compileJS(*themeDir, "script", true, *subThemePath)

				time.Sleep(100 * time.Millisecond)
				currentReversion++
				printCompTime("Theme", startTime)
			}else if input == "compile config" || input == "config" || input == "compile conf" || input == "conf" || input == "compile yml" || input == "yml" {
				fmt.Print("\x1b[1;36mCompiling Config...", "\x1b[0m")
				time.Sleep(300 * time.Millisecond)
				startTime := time.Now()

				*subThemePath, *defaultDarkMode, _ = compileConfig()

				time.Sleep(100 * time.Millisecond)
				currentReversion++
				printCompTime("Config", startTime)
			}else if input == "compile css" || input == "css" || input == "compile style" || input == "style" {
				fmt.Print("\x1b[1;36mCompiling CSS...", "\x1b[0m")
				time.Sleep(300 * time.Millisecond)
				startTime := time.Now()
				
				compileCSS(*themeDir, "style", true, *subThemePath, *defaultDarkMode)

				time.Sleep(100 * time.Millisecond)
				currentReversion++
				printCompTime("CSS", startTime)
			}else if input == "compile js" || input == "js" || input == "compile script" || input == "script" {
				fmt.Print("\x1b[1;36mCompiling JS...", "\x1b[0m")
				time.Sleep(300 * time.Millisecond)
				startTime := time.Now()

				compileJS(*themeDir, "script", true, *subThemePath)

				time.Sleep(100 * time.Millisecond)
				currentReversion++
				printCompTime("JS", startTime)
			}else if input == "reload page" || input == "reload" || input == "refresh page" || input == "refresh" {
				currentReversion++
			}
		}
	}()

	if port != "0" {
		fmt.Println("\x1b[1;32mRunning Server On Port \x1b[35m"+port, "\x1b[0m")
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}else{
		fmt.Println("\x1b[1;32mRunning File Watcher", "\x1b[0m")
		for {
			time.Sleep(100 * time.Millisecond)
		}
	}
}


func printCompTime(name string, startTime time.Time){
	endTime := time.Now()

	if name != "" {
		name = " "+name
	}

	if compTime := endTime.UnixMilli() - startTime.UnixMilli(); compTime >= 1000 {
		fmt.Println("\r\x1b[1;36mCompiled"+name+" in\x1b[35m", float64(compTime) / 1000, "\x1b[36mseconds\x1b[0m")
	}else if compTime := endTime.UnixNano() - startTime.UnixNano(); compTime >= 1000 {
		fmt.Println("\r\x1b[1;36mCompiled"+name+" in\x1b[35m", float64(compTime / 1000) / 1000, "\x1b[36mmilliseconds\x1b[0m")
	}else{
		fmt.Println("\r\x1b[1;36mCompiled"+name+" in\x1b[35m", compTime, "\x1b[36mnanoseconds\x1b[0m")
	}
}


func handleCompileTheme(port string) (*string, *string, *bool, error) {
	themeDir, err := filepath.Abs("./src")
	if err != nil {
		compileConfig()
		return nil, nil, nil, err
	}

	if stat, err := os.Stat("./src"); err == nil && stat.IsDir() {
		os.RemoveAll("./dist")
		os.Mkdir("./dist", 0755)
	}

	subThemePath, defaultDarkMode, _ := compileConfig()

	if stat, err := os.Stat("./src"); err != nil || !stat.IsDir() {
		return nil, nil, nil, errors.New("src directory not found")
	}

	compileCSS(themeDir, "style", true, subThemePath, defaultDarkMode)
	compileJS(themeDir, "script", true, subThemePath)

	if port == "" {
		return nil, nil, nil, errors.New("port not specified")
	}

	lastCompile := time.Now().UnixMilli()

	watcher := fs.Watcher()
	watcher.OnAny = func(path, op string) {
		// prevent duplicate compilations from running twice in a row
		if time.Now().UnixMilli() - lastCompile < 100 {
			return
		}

		//! Do NOT Remove This Line Of Code!
		// if this method is not delayed, strange things will happen.
		// my guess is the computer operating system or file system reports the change
		// before it actually fully finishes writing the file
		time.Sleep(100 * time.Millisecond)

		lastCompile = time.Now().UnixMilli()

		path = string(regex.Comp(`^%1[\\/]?`, themeDir).RepStrLit([]byte(path), []byte{}))

		if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
			if path, darkMode, err := compileConfig(); err == nil {
				if path != subThemePath || darkMode != defaultDarkMode {
					compileCSS(themeDir, "style", true, path, darkMode)
					compileJS(themeDir, "script", true, path)
				}
				subThemePath = path
				defaultDarkMode = darkMode
			}
		}else if strings.HasSuffix(path, ".css") && !strings.HasSuffix(path, ".min.css") {
			if strings.HasPrefix(path, "themes/") || strings.HasPrefix(path, "theme/") {
				compileCSS(themeDir, "style", false, subThemePath, defaultDarkMode)
			}else if strings.ContainsRune(path, '/') {
				reg := regex.Comp(`^(.*)/([^/\\]+)\.css$`)
				name := string(reg.RepStr([]byte(path), []byte("$2")))

				if name == "config" {
					name = "theme.config"
				}else if name == "style" {
					name = "theme.style"
				}else if name == "script" {
					name = "theme.script"
				}

				if filePath, err := fs.JoinPath(themeDir, string(reg.RepStr([]byte(path), []byte("$1")))); err == nil {
					compileCSS(filePath, name, false, "", defaultDarkMode)
				}
			}else{
				compileCSS(themeDir, "style", false, subThemePath, defaultDarkMode)
			}
		}else if strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".min.js") {
			if strings.HasPrefix(path, "themes/") || strings.HasPrefix(path, "theme/") {
				compileJS(themeDir, "script", false, subThemePath)
			}else if strings.ContainsRune(path, '/') {
				reg := regex.Comp(`^(.*)/([^/\\]+)\.css$`)
				name := string(reg.RepStr([]byte(path), []byte("$2")))

				if name == "config" {
					name = "theme.config"
				}else if name == "style" {
					name = "theme.style"
				}else if name == "script" {
					name = "theme.script"
				}

				if filePath, err := fs.JoinPath(themeDir, string(reg.RepStr([]byte(path), []byte("$1")))); err == nil {
					compileJS(filePath, name, false, "")
				}
			}else{
				compileJS(themeDir, "script", false, subThemePath)
			}
		}

		time.Sleep(100 * time.Millisecond)
		currentReversion++
	}
	watcher.OnRemove = func(path, op string) (removeWatcher bool) {
		path = string(regex.Comp(`^%1[\\/]?`, themeDir).RepStrLit([]byte(path), []byte{}))

		if !strings.HasSuffix(path, ".yml") && !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".css") && !strings.HasSuffix(path, ".js") {
			reg := regex.Comp(`^(.*)/([^/\\]+)\.css$`)
			name := string(reg.RepStr([]byte(path), []byte("$2")))
			if name == "config" {
				name = "theme.config"
			}else if name == "style" {
				name = "theme.style"
			}else if name == "script" {
				name = "theme.script"
			}

			if filePath, err := fs.JoinPath("./dist", name+".css"); err == nil {
				os.Remove(filePath)
			}
			if filePath, err := fs.JoinPath("./dist", name+".min.css"); err == nil {
				os.Remove(filePath)
			}

			if filePath, err := fs.JoinPath("./dist", name+".js"); err == nil {
				os.Remove(filePath)
			}
			if filePath, err := fs.JoinPath("./dist", name+".min.js"); err == nil {
				os.Remove(filePath)
			}
		}

		return true
	}
	watcher.WatchDir(themeDir)

	return &themeDir, &subThemePath, &defaultDarkMode, nil
}

func watchTestDirAndReloadClientPage(){
	testDir, err := filepath.Abs("./test")
	if err != nil {
		return
	}

	lastCompile := time.Now().UnixMilli()

	watcher := fs.Watcher()
	watcher.OnAny = func(path, op string) {
		// prevent duplicate compilations from running twice in a row
		if time.Now().UnixMilli() - lastCompile < 100 {
			return
		}

		time.Sleep(100 * time.Millisecond)

		lastCompile = time.Now().UnixMilli()

		currentReversion++
	}
	watcher.WatchDir(testDir)
}


func compileConfig() (string, bool, error) {
	themeConfig, themePath, dist, inDistFolder, err := getThemeConfig()
	if err != nil {
		return "", false, err
	}

	res := []byte(":root {\n")

	res = append(res, regex.JoinBytes(
		//* layout
		`  --page-padding-inline: `, themeConfig.Layout.PagePaddingInline, ";\n",
		`  --page-padding-inline-mobile: `, themeConfig.Layout.PagePaddingInlineMobile, ";\n",
		`  --page-content-max-width: `, themeConfig.Layout.PageContentMaxWidth, ";\n",
		`  --page-breakout-max-width: `, themeConfig.Layout.PageBreakoutMaxWidth, ";\n",
		`  --page-hard-max-width: `, themeConfig.Layout.PageHardMaxWidth, ";\n",

		`  --widget-size: `, themeConfig.Layout.WidgetSize, ";\n",
		`  --widget-max-size: `, themeConfig.Layout.WidgetMaxSize, ";\n",

		`  --headerimg-width: `, themeConfig.Layout.HeaderimgWidth, ";\n",
		`  --headerimg-height: `, themeConfig.Layout.HeaderimgHeight, ";\n",
		`  --headerimg-height-home: `, themeConfig.Layout.HeaderimgHeightHome, ";\n",
		`  --headernav-justify: `, themeConfig.Layout.HeadernavJustify, ";\n",
		`  --nav-underline-radius: `, themeConfig.Layout.NavUnderlineRadius, ";\n",

		`  --side-nav-width: `, themeConfig.Layout.SideNavWidth, ";\n",

		`  --shadow-size: `, themeConfig.Layout.ShadowSize, ";\n",
		`  --textshadow-size: `, themeConfig.Layout.TextshadowSize, ";\n",

		`  --border-radius: `, themeConfig.Layout.BorderRadius, ";\n",

		//* topography
		`  --font-size: `, themeConfig.Topography.FontSize, ";\n",
		`  --line-height: `, themeConfig.Topography.LineHeight, ";\n",

		`  --ff-sans: `, themeConfig.Topography.Font["sans"], ";\n",
		`  --ff-serif: `, themeConfig.Topography.Font["serif"], ";\n",
		`  --ff-mono: `, themeConfig.Topography.Font["mono"], ";\n",
		`  --ff-cursive: `, themeConfig.Topography.Font["cursive"], ";\n",
		`  --ff-logo: `, themeConfig.Topography.Font["logo"], ";\n",

		//* shadow
		`  --shadow-light: `, themeConfig.Shadow.Light, ";\n",
		`  --shadow-dark: `, themeConfig.Shadow.Dark, ";\n",
		`  --textshadow-light: `, themeConfig.Shadow.LightText, ";\n",
		`  --textshadow-dark: `, themeConfig.Shadow.DarkText, ";\n",
	)...)

	for key, val := range themeConfig.Config {
		res = append(res, []byte("  --"+key+": "+val+";\n")...)
	}

	var colorTextLight colorful.Color
	var colorTextDark colorful.Color
	{ //* text
		lightText := colorToHsl(colorFromString(themeConfig.Text.Light, "#ffffff"))
		darkText := colorToHsl(colorFromString(themeConfig.Text.Dark, "#0f0f0f"))

		if lightText[1] > 0.1 {
			lightText[1] = 0.1
		}
		if darkText[1] > 0.1 {
			darkText[1] = 0.1
		}

		if lightText[2] < 0.6 {
			lightText[2] = 0.6
		}
		if darkText[2] > 0.4 {
			darkText[2] = 0.4
		}

		res = append(res, regex.JoinBytes(
			`  --text-light: hsl(var(--text-light-h), var(--text-light-s), var(--text-light-l)); `,
				`--text-light-h: var(--fg-h); `,
				`--text-light-s: `, hslStringPart(lightText[1], "%"), "; ",
				`--text-light-l: `, hslStringPart(lightText[2], "%"), ";\n",

			`  --text-dark: hsl(var(--text-dark-h), var(--text-dark-s), var(--text-dark-l)); `,
				`--text-dark-h: var(--fg-h); `,
				`--text-dark-s: `, hslStringPart(darkText[1], "%"), "; ",
				`--text-dark-l: `, hslStringPart(darkText[2], "%"), ";\n",
		)...)

		colorTextLight = colorful.Hsl(lightText[0], lightText[1], lightText[2])
		colorTextDark = colorful.Hsl(darkText[0], darkText[1], darkText[2])
	}

	{ //* link
		lightLink := colorToHsl(colorFromString(themeConfig.Text.LightLink, "#3db9eb"))
		darkLink := colorToHsl(colorFromString(themeConfig.Text.DarkLink, "#1a84cb"))

		if lightLink[0] < 180 {
			lightLink[0] = 180
		}else if lightLink[0] > 260 {
			lightLink[0] = 260
		}

		if darkLink[0] < 180 {
			darkLink[0] = 180
		}else if darkLink[0] > 260 {
			darkLink[0] = 260
		}

		if lightLink[1] < 0.15 {
			lightLink[1] = 0.15
		}
		if darkLink[1] < 0.15 {
			darkLink[1] = 0.15
		}

		if lightLink[2] < 0.5 {
			lightLink[2] = 0.5
		}
		if darkLink[2] > 0.49 {
			darkLink[2] = 0.49
		}

		res = append(res, regex.JoinBytes(
			`  --link-light: hsl(var(--link-light-h), var(--link-light-s), var(--link-light-l)); `,
				`--link-light-h: `, hslStringPart(lightLink[0], "deg"), "; ",
				`--link-light-s: `, hslStringPart(lightLink[1], "%"), "; ",
				`--link-light-l: `, hslStringPart(lightLink[2], "%"), ";\n",

			`  --link-dark: hsl(var(--link-dark-h), var(--link-dark-s), var(--link-dark-l)); `,
				`--link-dark-h: `, hslStringPart(darkLink[0], "deg"), "; ",
				`--link-dark-s: `, hslStringPart(darkLink[1], "%"), "; ",
				`--link-dark-l: `, hslStringPart(darkLink[2], "%"), ";\n",
		)...)
	}

	{ //* heading
		if themeConfig.Heading.Light == "auto" {
			res = append(res, []byte(
				`  --heading-light: hsl(var(--fg-h), var(--fg-s), 0.65);`+"\n",
			)...)
		}else if themeConfig.Heading.Light == "text" {
			res = append(res, []byte(
				`  --heading-light: hsl(var(--text-light-h), var(--text-light-s), var(--text-light-l));`+"\n",
			)...)
		}else if _, ok := themeConfig.Color[themeConfig.Heading.Light]; ok {
			res = append(res, []byte(
				`  --heading-light: hsl(var(--`+themeConfig.Heading.Light+`-light-h), var(--`+themeConfig.Heading.Light+`-light-s), 0.65);`+"\n",
			)...)
		}else{
			res = append(res, []byte(
				`  --heading-light: hsl(var(--text-light-h), var(--text-light-s), var(--text-light-l));`+"\n",
			)...)
		}

		if themeConfig.Heading.Dark == "auto" {
			res = append(res, []byte(
				`  --heading-dark: hsl(var(--fg-h), var(--fg-s), 0.35);`+"\n",
			)...)
		}else if themeConfig.Heading.Dark == "text" {
			res = append(res, []byte(
				`  --heading-dark: hsl(var(--text-dark-h), var(--text-dark-s), var(--text-dark-l));`+"\n",
			)...)
		}else if _, ok := themeConfig.Color[themeConfig.Heading.Dark]; ok {
			res = append(res, []byte(
				`  --heading-dark: hsl(var(--`+themeConfig.Heading.Dark+`-dark-h), var(--`+themeConfig.Heading.Dark+`-Dark-s), 0.35);`+"\n",
			)...)
		}else{
			res = append(res, []byte(
				`  --heading-dark: hsl(var(--text-dark-h), var(--text-dark-s), var(--text-dark-l));`+"\n",
			)...)
		}

		if themeConfig.Heading.LightStrong == "auto" {
			res = append(res, []byte(
				`  --strongheading-light: linear-gradient(45deg, hsl(calc(var(--fg-h) + 5), calc(var(--fg-s) - 0.1), 0.55), hsl(calc(var(--fg-h) - 5), calc(var(--fg-s) + 0.1), 0.7));`+"\n",
			)...)
		}else if themeConfig.Heading.LightStrong == "text" {
			res = append(res, []byte(
				`  --strongheading-light: linear-gradient(45deg, hsl(calc(var(--text-light-h) + 5), calc(var(--text-light-s) - 0.1), 0.55), hsl(calc(var(--text-light-h) - 5), calc(var(--text-light-s) + 0.1), 0.7));`+"\n",
			)...)
		}else if _, ok := themeConfig.Color[themeConfig.Heading.LightStrong]; ok {
			res = append(res, []byte(
				`  --strongheading-light: linear-gradient(45deg, hsl(calc(var(--`+themeConfig.Heading.LightStrong+`-light-h) + 5), calc(var(--`+themeConfig.Heading.LightStrong+`-light-s) - 0.1), 0.55), hsl(calc(var(--`+themeConfig.Heading.LightStrong+`-light-h) - 5), calc(var(--`+themeConfig.Heading.LightStrong+`-light-s) + 0.1), 0.7));`+"\n",
			)...)
		}else if strings.Contains(themeConfig.Heading.LightStrong, "gradient(") || strings.Contains(themeConfig.Heading.LightStrong, "url(") {
			res = append(res, []byte(
				`  --strongheading-light: `+themeConfig.Heading.LightStrong+`;`+"\n",
			)...)
		}else{
			res = append(res, []byte(
				`  --strongheading-light: linear-gradient(45deg, hsl(calc(var(--fg-h) + 5), calc(var(--fg-s) - 0.1), 0.55), hsl(calc(var(--fg-h) - 5), calc(var(--fg-s) + 0.1), 0.7));`+"\n",
			)...)
		}

		if themeConfig.Heading.DarkStrong == "auto" {
			res = append(res, []byte(
				`  --strongheading-dark: linear-gradient(45deg, hsl(calc(var(--fg-h) + 5), calc(var(--fg-s) - 0.1), 0.3), hsl(calc(var(--fg-h) - 5), calc(var(--fg-s) + 0.1), 0.4));`+"\n",
			)...)
		}else if themeConfig.Heading.DarkStrong == "text" {
			res = append(res, []byte(
				`  --strongheading-dark: linear-gradient(45deg, hsl(calc(var(--text-dark-h) + 5), calc(var(--text-dark-s) - 0.1), 0.3), hsl(calc(var(--text-dark-h) - 5), calc(var(--text-dark-s) + 0.1), 0.4));`+"\n",
			)...)
		}else if _, ok := themeConfig.Color[themeConfig.Heading.DarkStrong]; ok {
			res = append(res, []byte(
				`  --strongheading-dark: linear-gradient(45deg, hsl(calc(var(--`+themeConfig.Heading.DarkStrong+`-dark-h) + 5), calc(var(--`+themeConfig.Heading.DarkStrong+`-dark-s) - 0.1), 0.3), hsl(calc(var(--`+themeConfig.Heading.DarkStrong+`-dark-h) - 5), calc(var(--`+themeConfig.Heading.DarkStrong+`-dark-s) + 0.1), 0.4));`+"\n",
			)...)
		}else if strings.Contains(themeConfig.Heading.DarkStrong, "gradient(") || strings.Contains(themeConfig.Heading.DarkStrong, "url(") {
			res = append(res, []byte(
				`  --strongheading-dark: `+themeConfig.Heading.DarkStrong+`;`+"\n",
			)...)
		}else{
			res = append(res, []byte(
				`  --strongheading-dark: linear-gradient(45deg, hsl(calc(var(--fg-h) + 5), calc(var(--fg-s) - 0.1), 0.3), hsl(calc(var(--fg-h) - 5), calc(var(--fg-s) + 0.1), 0.4));`+"\n",
			)...)
		}

		if themeConfig.Heading.Font != "auto" {
			if _, ok := themeConfig.Topography.Font[themeConfig.Heading.Font]; ok {
				res = append(res, []byte(
					`  --heading-font: var(--ff-`+themeConfig.Heading.Font+`);`+"\n",
				)...)
			}else{
				res = append(res, []byte(
					`  --heading-font: `+themeConfig.Heading.Font+`;`+"\n",
				)...)
			}
		}
	}

	{ //* input
		if themeConfig.Input.Light == "auto" {
			res = append(res, []byte(
				`  --input-light: hsl(var(--fg-h), var(--fg-s), var(--fg-l));`+"\n",
			)...)
		}else if themeConfig.Input.Light == "text" {
			res = append(res, []byte(
				`  --input-light: hsl(var(--text-light-h), var(--text-light-s), var(--text-light-l));`+"\n",
			)...)
		}else if _, ok := themeConfig.Color[themeConfig.Input.Light]; ok {
			res = append(res, []byte(
				`  --input-light: hsl(var(--`+themeConfig.Input.Light+`-light-h), var(--`+themeConfig.Input.Light+`-light-s), var(--`+themeConfig.Input.Light+`-light-l));`+"\n",
			)...)
		}else{
			res = append(res, []byte(
				`  --input-light: hsl(var(--fg-h), var(--fg-s), var(--fg-l));`+"\n",
			)...)
		}

		if themeConfig.Input.Dark == "auto" {
			res = append(res, []byte(
				`  --input-dark: hsl(var(--fg-h), var(--fg-s), var(--fg-l));`+"\n",
			)...)
		}else if themeConfig.Input.Dark == "text" {
			res = append(res, []byte(
				`  --input-dark: hsl(var(--text-dark-h), var(--text-dark-s), var(--text-dark-l));`+"\n",
			)...)
		}else if _, ok := themeConfig.Color[themeConfig.Input.Dark]; ok {
			res = append(res, []byte(
				`  --input-dark: hsl(var(--`+themeConfig.Input.Dark+`-dark-h), var(--`+themeConfig.Input.Dark+`-dark-s), var(--`+themeConfig.Input.Dark+`-dark-l));`+"\n",
			)...)
		}else{
			res = append(res, []byte(
				`  --input-dark: hsl(var(--fg-h), var(--fg-s), var(--fg-l));`+"\n",
			)...)
		}

		if themeConfig.Input.Font != "auto" {
			if _, ok := themeConfig.Topography.Font[themeConfig.Input.Font]; ok {
				res = append(res, []byte(
					`  --input-font: var(--ff-`+themeConfig.Input.Font+`);`+"\n",
				)...)
			}else{
				res = append(res, []byte(
					`  --input-font: `+themeConfig.Input.Font+`;`+"\n",
				)...)
			}
		}
	}


	{ //* color
		var colorPrimaryLight [3]float64
		var colorPrimaryDark [3]float64
		if color, ok := themeConfig.Color["primary"]; ok {
			colorPrimaryLight = colorToHsl(colorFromString(color.Light, "#00a3cc"))
			colorPrimaryDark = colorToHsl(colorFromString(color.Dark, "#007a99"))
		}else{
			colorPrimaryLight = colorToHsl(colorFromString("#00a3cc", ""))
			colorPrimaryDark = colorToHsl(colorFromString("#007a99", ""))
		}

		var colorAccentLight [3]float64
		var colorAccentDark [3]float64
		if color, ok := themeConfig.Color["accent"]; ok {
			colorAccentLight = colorToHsl(colorFromString(color.Light, "#12ba2e"))
			colorAccentDark = colorToHsl(colorFromString(color.Dark, "#0e8b23"))
		}else{
			colorAccentLight = colorToHsl(colorFromString("#12ba2e", ""))
			colorAccentDark = colorToHsl(colorFromString("#0e8b23", ""))
		}

		var colorWarnLight [3]float64
		var colorWarnDark [3]float64
		if color, ok := themeConfig.Color["warn"]; ok {
			colorWarnLight = colorToHsl(colorFromString(color.Light, "#ba1c1c"))
			colorWarnDark = colorToHsl(colorFromString(color.Dark, "#8e1515"))
		}else{
			colorWarnLight = colorToHsl(colorFromString("#ba1c1c", ""))
			colorWarnDark = colorToHsl(colorFromString("#8e1515", ""))
		}

		commonColors := []string{
			"primary",
			"accent",
			"warn",
			"dark",
			"light",
		}

		for _, name := range commonColors {
			if color, ok := themeConfig.Color[name]; ok {
				if name == "primary" {
					color.Light = colorful.Hsl(colorPrimaryLight[0], colorPrimaryLight[1], colorPrimaryLight[2]).Hex()
					color.Dark = colorful.Hsl(colorPrimaryLight[0], colorPrimaryDark[1], colorPrimaryDark[2]).Hex()

					res = append(res, regex.JoinBytes(
						`  --primary--light: hsl(var(--primary--light-h), var(--primary--light-s), var(--primary--light-l)); `,
							`--primary--light-h: `, hslStringPart(colorPrimaryLight[0], "deg"), "; ",
							`--primary--light-s: `, hslStringPart(colorPrimaryLight[1], "%"), "; ",
							`--primary--light-l: `, hslStringPart(colorPrimaryLight[2], "%"), ";\n",

						`  --primary--dark: hsl(var(--primary--dark-h), var(--primary--dark-s), var(--primary--dark-l)); `,
							`--primary--dark-h: `, hslStringPart(colorPrimaryLight[0], "deg"), "; ",
							`--primary--dark-s: `, hslStringPart(colorPrimaryDark[1], "%"), "; ",
							`--primary--dark-l: `, hslStringPart(colorPrimaryDark[2], "%"), ";\n",
					)...)
				}else if name == "accent" {
					var h float64
					if colorful.Hsl(colorAccentLight[0], 100, 100).AlmostEqualRgb(colorful.Hsl(colorPrimaryLight[0], 100, 100)) {
						h = colorPrimaryLight[0]
					}else{
						h = colorAccentLight[0]
					}

					color.Light = colorful.Hsl(h, colorAccentLight[1], colorAccentLight[2]).Hex()
					color.Dark = colorful.Hsl(h, colorAccentDark[1], colorAccentDark[2]).Hex()

					res = append(res, regex.JoinBytes(
						`  --accent--light: hsl(var(--accent--light-h), var(--accent--light-s), var(--accent--light-l)); `,
							`--accent--light-h: `, hslStringPart(h, "deg"), "; ",
							`--accent--light-s: `, hslStringPart(colorAccentLight[1], "%"), "; ",
							`--accent--light-l: `, hslStringPart(colorAccentLight[2], "%"), ";\n",

						`  --accent--dark: hsl(var(--accent--dark-h), var(--accent--dark-s), var(--accent--dark-l)); `,
							`--accent--dark-h: `, hslStringPart(h, "deg"), "; ",
							`--accent--dark-s: `, hslStringPart(colorAccentDark[1], "%"), "; ",
							`--accent--dark-l: `, hslStringPart(colorAccentDark[2], "%"), ";\n",
					)...)
				}else if name == "warn" {
					if colorWarnLight[0] > 64 && colorWarnLight[0] < 264 {
						if colorWarnLight[0] <= 164 {
							colorWarnLight[0] = 64
						}else{
							colorWarnLight[0] = 264
						}
					}

					if colorful.Hsl(colorWarnLight[0], 100, 100).AlmostEqualRgb(colorful.Hsl(colorPrimaryLight[0], 100, 100)) || colorful.Hsl(colorWarnLight[0], 100, 100).AlmostEqualRgb(colorful.Hsl(colorAccentLight[0], 100, 100)) {
						colorHueTryList := []float64{
							0, // red
							270, // purple
							300, // light purple
							330, // pink
							54, // yellow
							32, // orange
							colorWarnLight[0],
						}

						for _, hue := range colorHueTryList {
							if !(colorful.Hsl(colorWarnLight[0], 100, 100).AlmostEqualRgb(colorful.Hsl(colorPrimaryLight[0], 100, 100)) || colorful.Hsl(colorWarnLight[0], 100, 100).AlmostEqualRgb(colorful.Hsl(colorAccentLight[0], 100, 100))) {
								break
							}
							colorWarnLight[0] = hue
						}
					}

					color.Light = colorful.Hsl(colorWarnLight[0], colorWarnLight[1], colorWarnLight[2]).Hex()
					color.Dark = colorful.Hsl(colorWarnLight[0], colorWarnDark[1], colorWarnDark[2]).Hex()

					res = append(res, regex.JoinBytes(
						`  --warn--light: hsl(var(--warn--light-h), var(--warn--light-s), var(--warn--light-l)); `,
							`--warn--light-h: `, hslStringPart(colorWarnLight[0], "deg"), "; ",
							`--warn--light-s: `, hslStringPart(colorWarnLight[1], "%"), "; ",
							`--warn--light-l: `, hslStringPart(colorWarnLight[2], "%"), ";\n",

						`  --warn--dark: hsl(var(--warn--dark-h), var(--warn--dark-s), var(--warn--dark-l)); `,
							`--warn--dark-h: `, hslStringPart(colorWarnLight[0], "deg"), "; ",
							`--warn--dark-s: `, hslStringPart(colorWarnDark[1], "%"), "; ",
							`--warn--dark-l: `, hslStringPart(colorWarnDark[2], "%"), ";\n",
					)...)
				}else if name == "dark" {
					colorLight := colorToHsl(colorFromString(color.Light, "#2b2b2b"))
					colorDark := colorToHsl(colorFromString(color.Dark, "#2b2b2b"))

					if colorLight[1] > 15 {
						colorLight[1] = 15
					}
					if colorDark[1] > 15 {
						colorDark[1] = 15
					}

					if colorLight[2] > 49 {
						colorLight[2] = 49
					}
					if colorDark[2] > 49 {
						colorDark[2] = 49
					}

					color.Light = colorful.Hsl(colorLight[0], colorLight[1], colorLight[2]).Hex()
					color.Dark = colorful.Hsl(colorLight[0], colorDark[1], colorDark[2]).Hex()

					res = append(res, regex.JoinBytes(
						`  --dark--light: hsl(var(--dark--light-h), var(--dark--light-s), var(--dark--light-l)); `,
							`--dark--light-h: var(--fg-h); `,
							`--dark--light-s: `, hslStringPart(colorLight[1], "%"), "; ",
							`--dark--light-l: `, hslStringPart(colorLight[2], "%"), ";\n",

						`  --dark--dark: hsl(var(--dark--dark-h), var(--dark--dark-s), var(--dark--dark-l)); `,
							`--dark--dark-h: var(--fg-h); `,
							`--dark--dark-s: `, hslStringPart(colorDark[1], "%"), "; ",
							`--dark--dark-l: `, hslStringPart(colorDark[2], "%"), ";\n",
					)...)
				}else if name == "light" {
					colorLight := colorToHsl(colorFromString(color.Light, "#f0f0f0"))
					colorDark := colorToHsl(colorFromString(color.Dark, "#474747"))

					if colorLight[1] > 15 {
						colorLight[1] = 15
					}
					if colorDark[1] > 15 {
						colorDark[1] = 15
					}

					if colorLight[2] < 50 {
						colorLight[2] = 50
					}
					if colorDark[2] > 49 {
						colorDark[2] = 49
					}

					color.Light = colorful.Hsl(colorLight[0], colorLight[1], colorLight[2]).Hex()
					color.Dark = colorful.Hsl(colorLight[0], colorDark[1], colorDark[2]).Hex()

					res = append(res, regex.JoinBytes(
						`  --light--light: hsl(var(--light--light-h), var(--light--light-s), var(--light--light-l)); `,
							`--light--light-h: var(--fg-h); `,
							`--light--light-s: `, hslStringPart(colorLight[1], "%"), "; ",
							`--light--light-l: `, hslStringPart(colorLight[2], "%"), ";\n",

						`  --light--dark: hsl(var(--light--dark-h), var(--light--dark-s), var(--light--dark-l)); `,
							`--light--dark-h: var(--fg-h); `,
							`--light--dark-s: `, hslStringPart(colorDark[1], "%"), "; ",
							`--light--dark-l: `, hslStringPart(colorDark[2], "%"), ";\n",
					)...)
				}

				themeConfig.Color[name] = color

				// add text
				var light, dark, lightCont, darkCont string
				if contrastRatio(colorTextLight, colorFromString(color.Light, "#fff")) >= contrastRatio(colorTextDark, colorFromString(color.Light, "#fff")) {
					light = "light"
					lightCont = "dark"
				}else{
					light = "dark"
					lightCont = "light"
				}

				if contrastRatio(colorTextLight, colorFromString(color.Dark, "#000")) >= contrastRatio(colorTextDark, colorFromString(color.Dark, "#000")) {
					dark = "light"
					darkCont = "dark"
				}else{
					dark = "dark"
					darkCont = "light"
				}

				res = append(res, regex.JoinBytes(
					//* text
					`  --`, name, `--light-text: hsl(var(--`, name, `--light-text-h), var(--`, name, `--light-text-s), var(--`, name, `--light-text-l)); `,
						`--`, name, `--light-text-h: var(--text-`, light, `-h)`, "; ",
						`--`, name, `--light-text-s: var(--text-`, light, `-s)`, "; ",
						`--`, name, `--light-text-l: var(--text-`, light, `-l)`, ";\n",

					`  --`, name, `--dark-text: hsl(var(--`, name, `--dark-text-h), var(--`, name, `--dark-text-s), var(--`, name, `--dark-text-l)); `,
						`--`, name, `--dark-text-h: var(--text-`, dark, `-h)`, "; ",
						`--`, name, `--dark-text-s: var(--text-`, dark, `-s)`, "; ",
						`--`, name, `--dark-text-l: var(--text-`, dark, `-l)`, ";\n",

					//* link
					`  --`, name, `--light-link: hsl(var(--`, name, `--light-link-h), var(--`, name, `--light-link-s), var(--`, name, `--light-link-l)); `,
						`--`, name, `--light-link-h: var(--link-`, light, `-h)`, "; ",
						`--`, name, `--light-link-s: var(--link-`, light, `-s)`, "; ",
						`--`, name, `--light-link-l: var(--link-`, light, `-l)`, ";\n",

					`  --`, name, `--dark-link: hsl(var(--`, name, `--dark-link-h), var(--`, name, `--dark-link-s), var(--`, name, `--dark-link-l)); `,
						`--`, name, `--dark-link-h: var(--link-`, dark, `-h)`, "; ",
						`--`, name, `--dark-link-s: var(--link-`, dark, `-s)`, "; ",
						`--`, name, `--dark-link-l: var(--link-`, dark, `-l)`, ";\n",

					//* heading
					`  --`, name, `--light-heading: var(--heading-`, light, `)`, "; ",
						`--`, name, `--light-strongheading: var(--strongheading-`, light, `)`, ";\n",

					`  --`, name, `--dark-heading: var(--heading-`, dark, `)`, "; ",
						`--`, name, `--dark-strongheading: var(--strongheading-`, dark, `)`, ";\n",

					//* input
					`  --`, name, `--light-input: var(--input-`, light, `)`, ";\n",
					`  --`, name, `--dark-input: var(--input-`, dark, `)`, ";\n",

					//* shadow
					`  --`, name, `--light-shadow: var(--shadow-`, lightCont, `)`, "; ",
						`--`, name, `--light-textshadow: var(--textshadow-`, lightCont, `)`, ";\n",

					`  --`, name, `--dark-shadow: var(--shadow-`, darkCont, `)`, "; ",
						`--`, name, `--dark-textshadow: var(--textshadow-`, darkCont, `)`, ";\n",
				)...)

				if color.FG == "text" {
					res = append(res, regex.JoinBytes(
						`  --`, name, `--light-fg: hsl(var(--`, name, `--light-fg-h), var(--`, name, `--light-fg-s), var(--`, name, `--light-fg-l)); `,
							`--`, name, `--light-fg-h: var(--`, name, `--light-text-h)`, "; ",
							`--`, name, `--light-fg-s: var(--`, name, `--light-text-s)`, "; ",
							`--`, name, `--light-fg-l: var(--`, name, `--light-text-l)`, "; ",
							`--`, name, `--light-fg-text: var(--text-`, lightCont, `)`, ";\n",

						`  --`, name, `--dark-fg: hsl(var(--`, name, `--dark-fg-h), var(--`, name, `--dark-fg-s), var(--`, name, `--dark-fg-l)); `,
							`--`, name, `--dark-fg-h: var(--`, name, `--dark-text-h)`, "; ",
							`--`, name, `--dark-fg-s: var(--`, name, `--dark-text-s)`, "; ",
							`--`, name, `--dark-fg-l: var(--`, name, `--dark-text-l)`, "; ",
							`--`, name, `--dark-fg-text: var(--text-`, darkCont, `)`, ";\n",
					)...)
				}else if color.FG == "auto" {
					res = append(res, regex.JoinBytes(
						`  --`, name, `--light-fg: hsl(var(--`, name, `--light-fg-h), var(--`, name, `--light-fg-s), var(--`, name, `--light-fg-l)); `,
							`--`, name, `--light-fg-h: var(--fg--`, light, `-h)`, "; ",
							`--`, name, `--light-fg-s: var(--fg--`, light, `-s)`, "; ",
							`--`, name, `--light-fg-l: var(--fg--`, light, `-l)`, "; ",
	
						`  --`, name, `--dark-fg: hsl(var(--`, name, `--dark-fg-h), var(--`, name, `--dark-fg-s), var(--`, name, `--dark-fg-l)); `,
							`--`, name, `--dark-fg-h: var(--fg--`, dark, `-h)`, "; ",
							`--`, name, `--dark-fg-s: var(--fg--`, dark, `-s)`, "; ",
							`--`, name, `--dark-fg-l: var(--fg--`, dark, `-l)`, "; ",
					)...)
				}else{
					if fgColor, ok := themeConfig.Color[color.FG]; ok {
						var light, dark string
						if contrastRatio(colorFromString(fgColor.Light, "#fff"), colorFromString(color.Light, "#fff")) >= contrastRatio(colorFromString(fgColor.Dark, "#000"), colorFromString(color.Light, "#fff")) {
							light = "light"
						}else{
							light = "dark"
						}

						if contrastRatio(colorFromString(fgColor.Light, "#fff"), colorFromString(color.Dark, "#000")) >= contrastRatio(colorFromString(fgColor.Dark, "#000"), colorFromString(color.Dark, "#000")) {
							dark = "light"
						}else{
							dark = "dark"
						}

						res = append(res, regex.JoinBytes(
							`  --`, name, `--light-fg: hsl(var(--`, name, `--light-fg-h), var(--`, name, `--light-fg-s), var(--`, name, `--light-fg-l)); `,
								`--`, name, `--light-fg-h: var(--`, color.FG, `--`, light, `-h)`, "; ",
								`--`, name, `--light-fg-s: var(--`, color.FG, `--`, light, `-s)`, "; ",
								`--`, name, `--light-fg-l: var(--`, color.FG, `--`, light, `-l)`, ";\n",

							`  --`, name, `--dark-fg: hsl(var(--`, name, `--dark-fg-h), var(--`, name, `--dark-fg-s), var(--`, name, `--dark-fg-l)); `,
								`--`, name, `--dark-fg-h: var(--`, color.FG, `--`, dark, `-h)`, "; ",
								`--`, name, `--dark-fg-s: var(--`, color.FG, `--`, dark, `-s)`, "; ",
								`--`, name, `--dark-fg-l: var(--`, color.FG, `--`, dark, `-l)`, ";\n",
						)...)
					}
				}

				if color.Font != "auto" {
					if _, ok := themeConfig.Topography.Font[color.Font]; ok {
						res = append(res, []byte(
							`  --`+name+`-font: var(--ff-`+color.Font+`);`+"\n",
						)...)
					}else{
						res = append(res, []byte(
							`  --`+name+`-font: `+color.Font+`;`+"\n",
						)...)
					}
				}
			}
		}

		for name, color := range themeConfig.Color {
			if goutil.Contains(commonColors, name) {
				continue
			}

			colorLight := colorToHsl(colorFromString(color.Light, "#fff"))
			colorDark := colorToHsl(colorFromString(color.Dark, "#000"))

			var h float64
			if colorful.Hsl(colorLight[0], 100, 100).AlmostEqualRgb(colorful.Hsl(colorAccentLight[0], 100, 100)) {
				h = colorAccentLight[0]
			}else if colorful.Hsl(colorLight[0], 100, 100).AlmostEqualRgb(colorful.Hsl(colorPrimaryLight[0], 100, 100)) {
				h = colorPrimaryLight[0]
			}else if colorful.Hsl(colorLight[0], 100, 100).AlmostEqualRgb(colorful.Hsl(colorWarnLight[0], 100, 100)) {
				h = colorAccentLight[0]
			}else{
				h = colorLight[0]
			}

			color.Light = colorful.Hsl(h, colorLight[1], colorLight[2]).Hex()
			color.Dark = colorful.Hsl(h, colorDark[1], colorDark[2]).Hex()

			res = append(res, regex.JoinBytes(
				`  --`, name, `--light: hsl(var(--`, name, `--light-h), var(--`, name, `--light-s), var(--`, name, `--light-l)); `,
					`--`, name, `--light-h: `, hslStringPart(h, "deg"), "; ",
					`--`, name, `--light-s: `, hslStringPart(colorLight[1], "%"), "; ",
					`--`, name, `--light-l: `, hslStringPart(colorLight[2], "%"), ";\n",

				`  --`, name, `--dark: hsl(var(--`, name, `--dark-h), var(--`, name, `--dark-s), var(--`, name, `--dark-l)); `,
					`--`, name, `--dark-h: `, hslStringPart(h, "deg"), "; ",
					`--`, name, `--dark-s: `, hslStringPart(colorDark[1], "%"), "; ",
					`--`, name, `--dark-l: `, hslStringPart(colorDark[2], "%"), ";\n",
			)...)

			themeConfig.Color[name] = color

			// add text
			var light, dark, lightCont, darkCont string
			if contrastRatio(colorTextLight, colorFromString(color.Light, "#fff")) >= contrastRatio(colorTextDark, colorFromString(color.Light, "#fff")) {
				light = "light"
				lightCont = "dark"
			}else{
				light = "dark"
				lightCont = "light"
			}

			if contrastRatio(colorTextLight, colorFromString(color.Dark, "#000")) >= contrastRatio(colorTextDark, colorFromString(color.Dark, "#000")) {
				dark = "light"
				darkCont = "dark"
			}else{
				dark = "dark"
				darkCont = "light"
			}

			res = append(res, regex.JoinBytes(
				//* text
				`  --`, name, `--light-text: hsl(var(--`, name, `--light-text-h), var(--`, name, `--light-text-s), var(--`, name, `--light-text-l)); `,
					`--`, name, `--light-text-h: var(--text-`, light, `-h)`, "; ",
					`--`, name, `--light-text-s: var(--text-`, light, `-s)`, "; ",
					`--`, name, `--light-text-l: var(--text-`, light, `-l)`, ";\n",

				`  --`, name, `--dark-text: hsl(var(--`, name, `--dark-text-h), var(--`, name, `--dark-text-s), var(--`, name, `--dark-text-l)); `,
					`--`, name, `--dark-text-h: var(--text-`, dark, `-h)`, "; ",
					`--`, name, `--dark-text-s: var(--text-`, dark, `-s)`, "; ",
					`--`, name, `--dark-text-l: var(--text-`, dark, `-l)`, ";\n",

				//* link
				`  --`, name, `--light-link: hsl(var(--`, name, `--light-link-h), var(--`, name, `--light-link-s), var(--`, name, `--light-link-l)); `,
					`--`, name, `--light-link-h: var(--link-`, light, `-h)`, "; ",
					`--`, name, `--light-link-s: var(--link-`, light, `-s)`, "; ",
					`--`, name, `--light-link-l: var(--link-`, light, `-l)`, ";\n",

				`  --`, name, `--dark-link: hsl(var(--`, name, `--dark-link-h), var(--`, name, `--dark-link-s), var(--`, name, `--dark-link-l)); `,
					`--`, name, `--dark-link-h: var(--link-`, dark, `-h)`, "; ",
					`--`, name, `--dark-link-s: var(--link-`, dark, `-s)`, "; ",
					`--`, name, `--dark-link-l: var(--link-`, dark, `-l)`, ";\n",

				//* heading
				`  --`, name, `--light-heading: var(--heading-`, light, `)`, "; ",
					`--`, name, `--light-strongheading: var(--strongheading-`, light, `)`, ";\n",

				`  --`, name, `--dark-heading: var(--heading-`, dark, `)`, "; ",
					`--`, name, `--dark-strongheading: var(--strongheading-`, dark, `)`, ";\n",

				//* input
				`  --`, name, `--light-input: var(--input-`, light, `)`, ";\n",
				`  --`, name, `--dark-input: var(--input-`, dark, `)`, ";\n",

				//* shadow
				`  --`, name, `--light-shadow: var(--shadow-`, lightCont, `)`, "; ",
					`--`, name, `--light-textshadow: var(--textshadow-`, lightCont, `)`, ";\n",

				`  --`, name, `--dark-shadow: var(--shadow-`, darkCont, `)`, "; ",
					`--`, name, `--dark-textshadow: var(--textshadow-`, darkCont, `)`, ";\n",
			)...)

			if color.FG == "text" {
				res = append(res, regex.JoinBytes(
					`  --`, name, `--light-fg: hsl(var(--`, name, `--light-fg-h), var(--`, name, `--light-fg-s), var(--`, name, `--light-fg-l)); `,
						`--`, name, `--light-fg-h: var(--`, name, `--light-text-h)`, "; ",
						`--`, name, `--light-fg-s: var(--`, name, `--light-text-s)`, "; ",
						`--`, name, `--light-fg-l: var(--`, name, `--light-text-l)`, "; ",
						`--`, name, `--light-fg-text: var(--text-`, lightCont, `)`, ";\n",

					`  --`, name, `--dark-fg: hsl(var(--`, name, `--dark-fg-h), var(--`, name, `--dark-fg-s), var(--`, name, `--dark-fg-l)); `,
						`--`, name, `--dark-fg-h: var(--`, name, `--dark-text-h)`, "; ",
						`--`, name, `--dark-fg-s: var(--`, name, `--dark-text-s)`, "; ",
						`--`, name, `--dark-fg-l: var(--`, name, `--dark-text-l)`, "; ",
						`--`, name, `--dark-fg-text: var(--text-`, darkCont, `)`, ";\n",
				)...)
			}else if color.FG == "auto" {
				res = append(res, regex.JoinBytes(
					`  --`, name, `--light-fg: hsl(var(--`, name, `--light-fg-h), var(--`, name, `--light-fg-s), var(--`, name, `--light-fg-l)); `,
						`--`, name, `--light-fg-h: var(--fg--`, light, `-h)`, "; ",
						`--`, name, `--light-fg-s: var(--fg--`, light, `-s)`, "; ",
						`--`, name, `--light-fg-l: var(--fg--`, light, `-l)`, "; ",
						`--`, name, `--light-fg-text: var(--fg--`, light, `-text)`, ";\n",

					`  --`, name, `--dark-fg: hsl(var(--`, name, `--dark-fg-h), var(--`, name, `--dark-fg-s), var(--`, name, `--dark-fg-l)); `,
						`--`, name, `--dark-fg-h: var(--fg--`, dark, `-h)`, "; ",
						`--`, name, `--dark-fg-s: var(--fg--`, dark, `-s)`, "; ",
						`--`, name, `--dark-fg-l: var(--fg--`, dark, `-l)`, "; ",
						`--`, name, `--dark-fg-text: var(--fg--`, dark, `-text)`, ";\n",
				)...)
			}else{
				if fgColor, ok := themeConfig.Color[color.FG]; ok {
					var light, dark string
					if contrastRatio(colorFromString(fgColor.Light, "#fff"), colorFromString(color.Light, "#fff")) >= contrastRatio(colorFromString(fgColor.Dark, "#000"), colorFromString(color.Light, "#fff")) {
						light = "light"
					}else{
						light = "dark"
					}

					if contrastRatio(colorFromString(fgColor.Light, "#fff"), colorFromString(color.Dark, "#000")) >= contrastRatio(colorFromString(fgColor.Dark, "#000"), colorFromString(color.Dark, "#000")) {
						dark = "light"
					}else{
						dark = "dark"
					}

					res = append(res, regex.JoinBytes(
						`  --`, name, `--light-fg: hsl(var(--`, name, `--light-fg-h), var(--`, name, `--light-fg-s), var(--`, name, `--light-fg-l)); `,
							`--`, name, `--light-fg-h: var(--`, color.FG, `--`, light, `-h)`, "; ",
							`--`, name, `--light-fg-s: var(--`, color.FG, `--`, light, `-s)`, "; ",
							`--`, name, `--light-fg-l: var(--`, color.FG, `--`, light, `-l)`, "; ",
							`--`, name, `--light-fg-text: var(--`, color.FG, `--`, light, `-text)`, ";\n",

						`  --`, name, `--dark-fg: hsl(var(--`, name, `--dark-fg-h), var(--`, name, `--dark-fg-s), var(--`, name, `--dark-fg-l)); `,
							`--`, name, `--dark-fg-h: var(--`, color.FG, `--`, dark, `-h)`, "; ",
							`--`, name, `--dark-fg-s: var(--`, color.FG, `--`, dark, `-s)`, "; ",
							`--`, name, `--dark-fg-l: var(--`, color.FG, `--`, dark, `-l)`, "; ",
							`--`, name, `--dark-fg-text: var(--`, color.FG, `--`, dark, `-text)`, ";\n",
					)...)
				}
			}

			if color.Font != "auto" {
				if _, ok := themeConfig.Topography.Font[color.Font]; ok {
					res = append(res, []byte(
						`  --`+name+`-font: var(--ff-`+color.Font+`);`+"\n",
					)...)
				}else{
					res = append(res, []byte(
						`  --`+name+`-font: `+color.Font+`;`+"\n",
					)...)
				}
			}
		}
	}


	{ //* element
		for name, color := range themeConfig.Element {
			colorLight := colorToHsl(colorFromString(color.Light, "#fff"))
			colorDark := colorToHsl(colorFromString(color.Dark, "#000"))

			// text contrast
			var light, dark, lightCont, darkCont string
			if contrastRatio(colorTextLight, colorFromString(color.Light, "#fff")) >= contrastRatio(colorTextDark, colorFromString(color.Light, "#fff")) {
				light = "light"
				lightCont = "dark"
			}else{
				light = "dark"
				lightCont = "light"
			}

			if contrastRatio(colorTextLight, colorFromString(color.Dark, "#000")) >= contrastRatio(colorTextDark, colorFromString(color.Dark, "#000")) {
				dark = "light"
				darkCont = "dark"
			}else{
				dark = "dark"
				darkCont = "light"
			}

			res = append(res, regex.JoinBytes(
				//* color
				`  --`, name, `--light: hsl(var(--`, name, `--light-h), var(--`, name, `--light-s), var(--`, name, `--light-l))`, "; ",
					`--`, name, `--light-h: var(--fg-h)`, "; ",
					`--`, name, `--light-s: `, hslStringPart(colorLight[1], "%"), "; ",
					`--`, name, `--light-l: `, hslStringPart(colorLight[2], "%"), ";\n",

				`  --`, name, `--dark: hsl(var(--`, name, `--dark-h), var(--`, name, `--dark-s), var(--`, name, `--dark-l))`, "; ",
					`--`, name, `--dark-h: var(--fg-h)`, "; ",
					`--`, name, `--dark-s: `, hslStringPart(colorDark[1], "%"), "; ",
					`--`, name, `--dark-l: `, hslStringPart(colorDark[2], "%"), ";\n",

				//* text
				`  --`, name, `--light-text: hsl(var(--`, name, `--light-text-h), var(--`, name, `--light-text-s), var(--`, name, `--light-text-l)); `,
					`--`, name, `--light-text-h: var(--text-`, light, `-h)`, "; ",
					`--`, name, `--light-text-s: var(--text-`, light, `-s)`, "; ",
					`--`, name, `--light-text-l: var(--text-`, light, `-l)`, ";\n",

				`  --`, name, `--dark-text: hsl(var(--`, name, `--dark-text-h), var(--`, name, `--dark-text-s), var(--`, name, `--dark-text-l)); `,
					`--`, name, `--dark-text-h: var(--text-`, dark, `-h)`, "; ",
					`--`, name, `--dark-text-s: var(--text-`, dark, `-s)`, "; ",
					`--`, name, `--dark-text-l: var(--text-`, dark, `-l)`, ";\n",

				//* link
				`  --`, name, `--light-link: hsl(var(--`, name, `--light-link-h), var(--`, name, `--light-link-s), var(--`, name, `--light-link-l)); `,
					`--`, name, `--light-link-h: var(--link-`, light, `-h)`, "; ",
					`--`, name, `--light-link-s: var(--link-`, light, `-s)`, "; ",
					`--`, name, `--light-link-l: var(--link-`, light, `-l)`, ";\n",

				`  --`, name, `--dark-link: hsl(var(--`, name, `--dark-link-h), var(--`, name, `--dark-link-s), var(--`, name, `--dark-link-l)); `,
					`--`, name, `--dark-link-h: var(--link-`, dark, `-h)`, "; ",
					`--`, name, `--dark-link-s: var(--link-`, dark, `-s)`, "; ",
					`--`, name, `--dark-link-l: var(--link-`, dark, `-l)`, ";\n",

				//* heading
				`  --`, name, `--light-heading: var(--heading-`, light, `)`, "; ",
					`--`, name, `--light-strongheading: var(--strongheading-`, light, `)`, ";\n",

				`  --`, name, `--dark-heading: var(--heading-`, dark, `)`, "; ",
					`--`, name, `--dark-strongheading: var(--strongheading-`, dark, `)`, ";\n",

				//* input
				`  --`, name, `--light-input: var(--input-`, light, `)`, ";\n",
				`  --`, name, `--dark-input: var(--input-`, dark, `)`, ";\n",

				//* shadow
				`  --`, name, `--light-shadow: var(--shadow-`, lightCont, `)`, "; ",
					`--`, name, `--light-textshadow: var(--textshadow-`, lightCont, `)`, ";\n",

				`  --`, name, `--dark-shadow: var(--shadow-`, darkCont, `)`, "; ",
					`--`, name, `--dark-textshadow: var(--textshadow-`, darkCont, `)`, ";\n",
			)...)

			if color.FG == "text" {
				res = append(res, regex.JoinBytes(
					`  --`, name, `--light-fg: hsl(var(--`, name, `--light-fg-h), var(--`, name, `--light-fg-s), var(--`, name, `--light-fg-l)); `,
						`--`, name, `--light-fg-h: var(--`, name, `--light-text-h)`, "; ",
						`--`, name, `--light-fg-s: var(--`, name, `--light-text-s)`, "; ",
						`--`, name, `--light-fg-l: var(--`, name, `--light-text-l)`, "; ",
						`--`, name, `--light-fg-text: var(--text-`, lightCont, `)`, ";\n",

					`  --`, name, `--dark-fg: hsl(var(--`, name, `--dark-fg-h), var(--`, name, `--dark-fg-s), var(--`, name, `--dark-fg-l)); `,
						`--`, name, `--dark-fg-h: var(--`, name, `--dark-text-h)`, "; ",
						`--`, name, `--dark-fg-s: var(--`, name, `--dark-text-s)`, "; ",
						`--`, name, `--dark-fg-l: var(--`, name, `--dark-text-l)`, "; ",
						`--`, name, `--dark-fg-text: var(--text-`, darkCont, `)`, ";\n",
				)...)
			}else if color.FG == "auto" {
				res = append(res, regex.JoinBytes(
					`  --`, name, `--light-fg: hsl(var(--`, name, `--light-fg-h), var(--`, name, `--light-fg-s), var(--`, name, `--light-fg-l)); `,
						`--`, name, `--light-fg-h: var(--fg--`, light, `-h)`, "; ",
						`--`, name, `--light-fg-s: var(--fg--`, light, `-s)`, "; ",
						`--`, name, `--light-fg-l: var(--fg--`, light, `-l)`, "; ",
						`--`, name, `--light-fg-text: var(--fg--`, light, `-text)`, ";\n",

					`  --`, name, `--dark-fg: hsl(var(--`, name, `--dark-fg-h), var(--`, name, `--dark-fg-s), var(--`, name, `--dark-fg-l)); `,
						`--`, name, `--dark-fg-h: var(--fg--`, dark, `-h)`, "; ",
						`--`, name, `--dark-fg-s: var(--fg--`, dark, `-s)`, "; ",
						`--`, name, `--dark-fg-l: var(--fg--`, dark, `-l)`, "; ",
						`--`, name, `--dark-fg-text: var(--fg--`, dark, `-text)`, ";\n",
				)...)
			}else{
				if fgColor, ok := themeConfig.Color[color.FG]; ok {
					var light, dark string
					if contrastRatio(colorFromString(fgColor.Light, "#fff"), colorFromString(color.Light, "#fff")) >= contrastRatio(colorFromString(fgColor.Dark, "#000"), colorFromString(color.Light, "#fff")) {
						light = "light"
					}else{
						light = "dark"
					}

					if contrastRatio(colorFromString(fgColor.Light, "#fff"), colorFromString(color.Dark, "#000")) >= contrastRatio(colorFromString(fgColor.Dark, "#000"), colorFromString(color.Dark, "#000")) {
						dark = "light"
					}else{
						dark = "dark"
					}

					res = append(res, regex.JoinBytes(
						`  --`, name, `--light-fg: hsl(var(--`, name, `--light-fg-h), var(--`, name, `--light-fg-s), var(--`, name, `--light-fg-l)); `,
							`--`, name, `--light-fg-h: var(--`, color.FG, `--`, light, `-h)`, "; ",
							`--`, name, `--light-fg-s: var(--`, color.FG, `--`, light, `-s)`, "; ",
							`--`, name, `--light-fg-l: var(--`, color.FG, `--`, light, `-l)`, "; ",
							`--`, name, `--light-fg-text: var(--`, color.FG, `--`, light, `-text)`, ";\n",

						`  --`, name, `--dark-fg: hsl(var(--`, name, `--dark-fg-h), var(--`, name, `--dark-fg-s), var(--`, name, `--dark-fg-l)); `,
							`--`, name, `--dark-fg-h: var(--`, color.FG, `--`, dark, `-h)`, "; ",
							`--`, name, `--dark-fg-s: var(--`, color.FG, `--`, dark, `-s)`, "; ",
							`--`, name, `--dark-fg-l: var(--`, color.FG, `--`, dark, `-l)`, "; ",
							`--`, name, `--dark-fg-text: var(--`, color.FG, `--`, dark, `-text)`, ";\n",
					)...)
				}
			}

			if color.Font != "auto" {
				if _, ok := themeConfig.Topography.Font[color.Font]; ok {
					res = append(res, []byte(
						`  --`+name+`-font: var(--ff-`+color.Font+`);`+"\n",
					)...)
				}else{
					res = append(res, []byte(
						`  --`+name+`-font: `+color.Font+`;`+"\n",
					)...)
				}
			}

			if !goutil.IsZeroOfUnderlyingType(color.Img) && (color.Img.Light != "" || color.Img.Dark != "") {
				if color.Img.Light == "" {
					color.Img.Light = color.Img.Dark
				}else if color.Img.Dark == "" {
					color.Img.Dark = color.Img.Light
				}

				res = append(res, regex.JoinBytes(
					`  --`, name, `--img-light: `, color.Img.Light, "; ",
						`--`, name, `--img-dark: `, color.Img.Dark, "; ",
						`--`, name, `--img-size: `, color.Img.Size, "; ",
						`--`, name, `--img-pos: `, color.Img.Pos, "; ",
						`--`, name, `--img-att: `, color.Img.Att, "; ",
						`--`, name, `--img-blend: `, color.Img.Blend, ";\n",
				)...)
			}
		}
	}


	res = append(res, '}', '\n')

	res = append(res, regex.JoinBytes(
		"\n:root {\n",

		`  --bg-a: 1`, "; ",
			`--fg-a: 1`, ";\n",

			compileConfigFG("primary"),
			compileConfigElm("base", themeConfig.DefaultDarkMode),

			"\n  @media(prefers-color-scheme: light){color-scheme: light;}\n",
				"  @media(prefers-color-scheme: dark){color-scheme: dark;}\n",

		"}\n",
	)...)

	if len(themeConfig.Topography.ImportFonts) != 0 {
		res = append(res, '\n')
		res = append(res, compileConfigFonts(themeConfig.Topography.ImportFonts)...)
	}

	if inDistFolder {
		os.WriteFile("./dist/config.css", res, 0755)
	}

	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	if res, err := m.Bytes("text/css", res); err == nil {
		os.WriteFile(dist, res, 0755)
	}else{
		os.WriteFile(dist, res, 0755)
	}

	return themePath, themeConfig.DefaultDarkMode, nil
}

func getThemeConfig() (ThemeConfigData, string, string, bool, error) {
	buf, err := os.ReadFile("./src/config.yml")
	dist := "./dist/config.min.css"
	inDistFolder := true
	if err != nil {
		buf, err = os.ReadFile("./theme.yml")
		dist = "./theme.min.css"
		inDistFolder = false
	}
	if err != nil {
		buf, err = os.ReadFile("./config.yml")
		dist = "./config.min.css"
		inDistFolder = false
	}
	if err != nil {
		return ThemeConfigData{}, "", "", false, err
	}


	// get extra config
	themeConfigExtra := ThemeConfigDataConfig{}
	err = yaml.Unmarshal(buf, &themeConfigExtra)
	if err != nil {
		return ThemeConfigData{}, "", "", false, err
	}
	if themeConfigExtra.Config == nil {
		themeConfigExtra.Config = map[string]string{}
	}


	// simplify config keys
	buf = regex.Comp(`(?m)^(\s*\w+)([_-][\w_-]+|):`).RepFunc(buf, func(data func(int) []byte) []byte {
		b := bytes.Split(bytes.ReplaceAll(data(2), []byte{'_'}, []byte{'-'}), []byte{'-'})
		return bytes.ToLower(regex.JoinBytes(data(1), bytes.Join(b, []byte{}), ':'))
	})

	buf = regex.Comp(`(?m)^\s*\w+:\s*none(?:\s*#.*?|)$`).RepStrLit(buf, []byte{})

	// get main theme config
	themeConfig := ThemeConfigData{}
	err = yaml.Unmarshal(buf, &themeConfig)
	if err != nil {
		return ThemeConfigData{}, "", "", false, err
	}


	// separate overriding root settings (subthemes should only set defaults for these settings)
	themeConfigColor := map[string]ThemeConfigColor{}
	for key, val := range themeConfig.Color {
		themeConfigColor[key] = val
	}

	themeConfigElement := map[string]ThemeConfigElement{}
	for key, val := range themeConfig.Element {
		themeConfigElement[key] = val
	}

	themeConfigFont := map[string]string{}
	for key, val := range themeConfig.Topography.Font {
		themeConfigFont[key] = val
	}

	themeConfigImportFonts := make([]ThemeConfigImportFont, len(themeConfig.Topography.ImportFonts))
	copy(themeConfigImportFonts, themeConfig.Topography.ImportFonts)


	var themePath string
	if themeConfig.Theme != "" && themeConfig.Theme != "none" {
		// get subtheme config
		var buf []byte
		if themePath, err = fs.JoinPath("./src/themes", themeConfig.Theme, "config.yml"); err == nil {
			buf, err = os.ReadFile(themePath)
		}
		if err != nil {
			if themePath, err = fs.JoinPath("./src/theme", themeConfig.Theme, "config.yml"); err == nil {
				buf, err = os.ReadFile(themePath)
			}
		}
		if err != nil {
			themePath = ""
		}else{
			themeConfigExtraDef := ThemeConfigDataConfig{}
			err = yaml.Unmarshal(buf, &themeConfigExtraDef)
			if err != nil {
				return ThemeConfigData{}, "", "", false, err
			}

			for key, val := range themeConfigExtraDef.Config {
				if _, ok := themeConfigExtra.Config[key]; !ok {
					themeConfigExtra.Config[key] = val
				}
			}

			// simplify config keys
			buf = regex.Comp(`(?m)^(\s*\w+)([_-][\w_-]+|):`).RepFunc(buf, func(data func(int) []byte) []byte {
				b := bytes.Split(bytes.ReplaceAll(data(2), []byte{'_'}, []byte{'-'}), []byte{'-'})
				return bytes.ToLower(regex.JoinBytes(data(1), bytes.Join(b, []byte{}), ':'))
			})
		
			buf = regex.Comp(`(?m)^\s*\w+:\s*none(?:\s*#.*?|)$`).RepStrLit(buf, []byte{})

			// mearge subtheme config (should override root config)
			err = yaml.Unmarshal(buf, &themeConfig)
			if err != nil {
				return ThemeConfigData{}, "", "", false, err
			}

			// mearge empty root config settings with subtheme config (should Not override root config)
			for key, val := range themeConfig.Color {
				if _, ok := themeConfigColor[key]; !ok {
					themeConfigColor[key] = val
				}
			}

			for key, val := range themeConfig.Element {
				if _, ok := themeConfigElement[key]; !ok {
					themeConfigElement[key] = val
				}
			}

			for key, val := range themeConfig.Topography.Font {
				if _, ok := themeConfigFont[key]; !ok {
					themeConfigFont[key] = val
				}
			}

			for _, val := range themeConfig.Topography.ImportFonts {
				hasFont := false
				for _, font := range themeConfigImportFonts {
					if font.Name == val.Name {
						hasFont = true
						break
					}
				}

				if !hasFont {
					themeConfigImportFonts = append(themeConfigImportFonts, val)
				}
			}
		}
	}

	themeConfig.Config = themeConfigExtra.Config
	themeConfig.Color = themeConfigColor
	themeConfig.Element = themeConfigElement
	themeConfig.Topography.Font = themeConfigFont
	themeConfig.Topography.ImportFonts = themeConfigImportFonts


	if themePath != "" {
		themePath = string(regex.Comp(`[/\\]+config\.yml$`).RepStrLit([]byte(themePath), []byte{}))
	}

	return themeConfig, themePath, dist, inDistFolder, nil
}


func compileConfigFonts(importFonts []ThemeConfigImportFont) []byte {
	res := []byte{}
	for _, font := range importFonts {
		if font.Name == "" && font.Local == "" {
			continue
		}else if font.Local == "" {
			font.Local = font.Name
		}else if font.Name == "" {
			font.Name = font.Local
		}

		if font.Format == "" {
			font.Format = "truetype"
		}

		if font.Display == "" {
			font.Display = "auto"
		}

		fontSrc := regex.Comp(`{(.*?)}`).Split([]byte(font.Src))
		if len(fontSrc) == 3 {
			fontSrcData := bytes.Split(fontSrc[1], []byte{'|'})
			for _, srcData := range fontSrcData {
				var fontWeight []byte
				srcList := [][]byte{}
				if i := bytes.IndexByte(srcData, ':'); i != -1 && i < len(srcData)-1 {
					fontWeight = srcData[:i]
					srcList = bytes.Split(srcData[i+1:], []byte{','})
				}else if !bytes.Contains(srcData, []byte{':'}) {
					srcList = bytes.Split(srcData, []byte{','})
				}

				if len(srcList) == 0 {
					continue
				}

				for i, src := range srcList {
					fontStyle := []byte{}
					if i == 1 {
						fontStyle = []byte(` font-style: italic;`)
					}else if i == 2 {
						fontStyle = []byte(` font-style: oblique;`)
					}else if i > 0 {
						break
					}

					css := regex.JoinBytes(
						`@font-face {`,
							`font-family: '`, font.Name, `';`,
							` src: local('`, font.Local, `'),`,
								` url('`, fontSrc[0], src, fontSrc[2], `') format('`, font.Format, `');`,
							` font-display: `, font.Display, ';',
							fontStyle,
					)
					if len(fontWeight) != 0 {
						css = regex.JoinBytes(css, 
							` font-weight: `, fontWeight, ';',
						)
					}
					css = append(css, '}', '\n')
					res = append(res, css...)
				}
			}
		}else if len(fontSrc) == 1 {
			res = append(res, regex.JoinBytes(
				`@font-face {`,
					`font-family: '`, font.Name, `';`,
					` src: local('`, font.Local, `'),`,
						` url('`, fontSrc[0], `') format('`, font.Format, `');`,
					` font-display: `, font.Display, ';',
				'}', '\n',
			)...)
		}
	}
	return res
}


func compileConfigElm(name string, defaultDarkMode bool) []byte {
	var main, alt string
	if defaultDarkMode {
		main = "dark"
		alt = "light"
	}else{
		main = "light"
		alt = "dark"
	}

	return regex.JoinBytes(
		`  --bg: hsla(var(--bg-h), var(--bg-s), var(--bg-l), var(--bg-a))`, "; ",
			`--bg-h: var(--`, name, `--`, main, `-h)`, "; ",
			`--bg-s: var(--`, name, `--`, main, `-s)`, "; ",
			`--bg-l: var(--`, name, `--`, main, `-l)`, ";\n",

		`  --fg: hsla(var(--fg-h), var(--fg-s), var(--fg-l), var(--fg-a))`, "; ",
			`--fg-h: var(--`, name, `--`, main, `-fg-h)`, "; ",
			`--fg-s: var(--`, name, `--`, main, `-fg-s)`, "; ",
			`--fg-l: var(--`, name, `--`, main, `-fg-l)`, "; ",
			`--fg-text: var(--`, name, `--`, main, `-fg-text)`, ";\n",

		`  --text: var(--`, name, `--`, main, `-text)`, "; ",
			`--text-h: var(--`, name, `--`, main, `-text-h)`, "; ",
			`--text-s: var(--`, name, `--`, main, `-text-s)`, "; ",
			`--text-l: var(--`, name, `--`, main, `-text-l)`, "; ",
			`--link: var(--`, name, `--`, main, `-link)`, "; ",
			`--link-h: var(--`, name, `--`, main, `-link-h)`, "; ",
			`--link-s: var(--`, name, `--`, main, `-link-s)`, "; ",
			`--link-l: var(--`, name, `--`, main, `-link-l)`, "; ",
			`--heading: var(--`, name, `--`, main, `-heading)`, "; ",
			`--stringheading: var(--`, name, `--`, main, `-strongheading)`, "; ",
			`--input: var(--`, name, `--`, main, `-input)`, "; ",
			// `--shadow: var(--`, name, `--`, main, `-shadow)`, "; ",
			`--textshadow: var(--`, name, `--`, main, `-textshadow)`, "; ",
			`--font: var(--`, name, `-font, var(--ff-sans))`, ";\n",

		`  --img: var(--`, name, `--img-`, main, `, none)`, "; ",
			`--img-size: var(--`, name, `--img-size, cover)`, "; ",
			`--img-pos: var(--`, name, `--img-pos, center)`, "; ",
			`--img-att: var(--`, name, `--img-att, fixed)`, "; ",
			`--img-blend: var(--`, name, `--img-blend, fixed)`, ";\n",

		"\n  @media (prefers-color-scheme: ", alt, ") {\n",

		`    --bg-h: var(--`, name, `--`, alt, `-h)`, "; ",
				`--bg-s: var(--`, name, `--`, alt, `-s)`, "; ",
				`--bg-l: var(--`, name, `--`, alt, `-l)`, ";\n",

		`    --fg-h: var(--`, name, `--`, alt, `-fg-h)`, "; ",
				`--fg-s: var(--`, name, `--`, alt, `-fg-s)`, "; ",
				`--fg-l: var(--`, name, `--`, alt, `-fg-l)`, "; ",
				`--fg-text: var(--`, name, `--`, alt, `-fg-text)`, ";\n",

		`    --text: var(--`, name, `--`, alt, `-text)`, "; ",
				`--text-h: var(--`, name, `--`, alt, `-text-h)`, "; ",
				`--text-s: var(--`, name, `--`, alt, `-text-s)`, "; ",
				`--text-l: var(--`, name, `--`, alt, `-text-l)`, "; ",
				`--link: var(--`, name, `--`, alt, `-link)`, "; ",
				`--link-h: var(--`, name, `--`, alt, `-link-h)`, "; ",
				`--link-s: var(--`, name, `--`, alt, `-link-s)`, "; ",
				`--link-l: var(--`, name, `--`, alt, `-link-l)`, "; ",
				`--heading: var(--`, name, `--`, alt, `-heading)`, "; ",
				`--stringheading: var(--`, name, `--`, alt, `-strongheading)`, "; ",
				`--input: var(--`, name, `--`, alt, `-input)`, "; ",
				// `--shadow: var(--`, name, `--`, alt, `-shadow)`, "; ",
				`--textshadow: var(--`, name, `--`, alt, `-textshadow)`, ";\n",

		`    --img: var(--`, name, `--img-`, alt, `, none)`, ";\n",

		"  }\n",

		"  & > * {\n",
		`    --shadow: var(--`, name, `--`, main, `-shadow)`, ";\n",
		`    @media(prefers-color-scheme: `, alt, `){--shadow: var(--`, name, `--`, alt, `-shadow)`, ";}\n",
		"  }\n",
	)
}

func compileConfigColor(name string, defaultDarkMode bool) []byte {
	var main, alt string
	if defaultDarkMode {
		main = "dark"
		alt = "light"
	}else{
		main = "light"
		alt = "dark"
	}

	return regex.JoinBytes(
		`  --bg: hsla(var(--bg-h), var(--bg-s), var(--bg-l), var(--bg-a))`, "; ",
			`--bg-h: var(--`, name, `--`, main, `-h)`, "; ",
			`--bg-s: var(--`, name, `--`, main, `-s)`, "; ",
			`--bg-l: var(--`, name, `--`, main, `-l)`, ";\n",

		`  --fg: hsla(var(--fg-h), var(--fg-s), var(--fg-l), var(--fg-a))`, "; ",
			`--fg-h: var(--`, name, `--`, main, `-fg-h)`, "; ",
			`--fg-s: var(--`, name, `--`, main, `-fg-s)`, "; ",
			`--fg-l: var(--`, name, `--`, main, `-fg-l)`, "; ",
			`--fg-text: var(--`, name, `--`, main, `-fg-text)`, ";\n",

		`  --text: var(--`, name, `--`, main, `-text)`, "; ",
			`--text-h: var(--`, name, `--`, main, `-text-h)`, "; ",
			`--text-s: var(--`, name, `--`, main, `-text-s)`, "; ",
			`--text-l: var(--`, name, `--`, main, `-text-l)`, "; ",
			`--link: var(--`, name, `--`, main, `-link)`, "; ",
			`--link-h: var(--`, name, `--`, main, `-link-h)`, "; ",
			`--link-s: var(--`, name, `--`, main, `-link-s)`, "; ",
			`--link-l: var(--`, name, `--`, main, `-link-l)`, "; ",
			`--heading: var(--`, name, `--`, main, `-heading)`, "; ",
			`--stringheading: var(--`, name, `--`, main, `-strongheading)`, "; ",
			`--input: var(--`, name, `--`, main, `-input)`, "; ",
			// `--shadow: var(--`, name, `--`, main, `-shadow)`, "; ",
			`--textshadow: var(--`, name, `--`, main, `-textshadow)`, "; ",
			`--font: var(--`, name, `-font, var(--ff-sans))`, ";\n",

		`  --img: none`, ";\n",

		"\n  @media (prefers-color-scheme: ", alt, ") {\n",

		`    --bg-h: var(--`, name, `--`, alt, `-h)`, "; ",
				`--bg-s: var(--`, name, `--`, alt, `-s)`, "; ",
				`--bg-l: var(--`, name, `--`, alt, `-l)`, ";\n",

		`    --fg-h: var(--`, name, `--`, alt, `-fg-h)`, "; ",
				`--fg-s: var(--`, name, `--`, alt, `-fg-s)`, "; ",
				`--fg-l: var(--`, name, `--`, alt, `-fg-l)`, "; ",
				`--fg-text: var(--`, name, `--`, alt, `-fg-text)`, ";\n",

		`    --text: var(--`, name, `--`, alt, `-text)`, "; ",
				`--text-h: var(--`, name, `--`, alt, `-text-h)`, "; ",
				`--text-s: var(--`, name, `--`, alt, `-text-s)`, "; ",
				`--text-l: var(--`, name, `--`, alt, `-text-l)`, "; ",
				`--link: var(--`, name, `--`, alt, `-link)`, "; ",
				`--link-h: var(--`, name, `--`, alt, `-link-h)`, "; ",
				`--link-s: var(--`, name, `--`, alt, `-link-s)`, "; ",
				`--link-l: var(--`, name, `--`, alt, `-link-l)`, "; ",
				`--heading: var(--`, name, `--`, alt, `-heading)`, "; ",
				`--stringheading: var(--`, name, `--`, alt, `-strongheading)`, "; ",
				`--input: var(--`, name, `--`, alt, `-input)`, "; ",
				// `--shadow: var(--`, name, `--`, alt, `-shadow)`, "; ",
				`--textshadow: var(--`, name, `--`, alt, `-textshadow)`, ";\n",

		"  }\n",

		"  & > * {\n",
		`    --shadow: var(--`, name, `--`, main, `-shadow)`, ";\n",
		`    @media(prefers-color-scheme: `, alt, `){--shadow: var(--`, name, `--`, alt, `-shadow)`, ";}\n",
		"  }\n",
	)
}

func compileConfigFG(name string) []byte {
	return regex.JoinBytes(
		`  --fg--light-h: var(--`, name, `--light-h)`, "; ",
			`--fg--light-s: var(--`, name, `--light-s)`, "; ",
			`--fg--light-l: var(--`, name, `--light-l)`, "; ",
			`--fg--light-text: var(--`, name, `--light-text)`, ";\n",

		`  --fg--dark-h: var(--`, name, `--dark-h)`, "; ",
			`--fg--dark-s: var(--`, name, `--dark-s)`, "; ",
			`--fg--dark-l: var(--`, name, `--dark-l)`, "; ",
			`--fg--dark-text: var(--`, name, `--dark-text)`, ";\n",
	)
}


func compileCSS(inDir string, outName string, init bool, subThemePath string, defaultDarkMode bool){
	if stat, err := os.Stat("./dist"); err != nil || !stat.IsDir() {
		return
	}

	res := []byte{}

	if files, err := os.ReadDir(inDir); err == nil {
		for _, file := range files {
			if file.IsDir() {
				if init {
					name := file.Name()
					if name == "themes" || name == "theme" {
						continue
					}

					if name == "config" {
						name = "theme.config"
					}else if name == "style" {
						name = "theme.style"
					}else if name == "script" {
						name = "theme.script"
					}

					if filePath, err := fs.JoinPath(inDir, name); err == nil {
						compileCSS(filePath, name, init, "", defaultDarkMode)
					}
				}
			}else if strings.HasSuffix(file.Name(), ".css") && !strings.HasSuffix(file.Name(), ".min.css") {
				if filePath, err := fs.JoinPath(inDir, file.Name()); err == nil {
					if buf, err := os.ReadFile(filePath); err == nil {
						res = append(res, '\n')
						res = append(res, buf...)
					}
				}
			}
		}
	}

	if outName == "style" && subThemePath != "" {
		if files, err := os.ReadDir(subThemePath); err == nil {
			for _, file := range files {
				if file.IsDir() {
					if init {
						name := file.Name()
						if name == "config" {
							name = "theme.config"
						}else if name == "style" {
							name = "theme.style"
						}else if name == "script" {
							name = "theme.script"
						}
	
						if filePath, err := fs.JoinPath(subThemePath, name); err == nil {
							compileCSS(filePath, name, init, "", defaultDarkMode)
						}
					}
				}else if strings.HasSuffix(file.Name(), ".css") && !strings.HasSuffix(file.Name(), ".min.css") {
					if filePath, err := fs.JoinPath(subThemePath, file.Name()); err == nil {
						if buf, err := os.ReadFile(filePath); err == nil {
							res = append(res, '\n')
							res = append(res, buf...)
						}
					}
				}
			}
		}
	}

	if len(res) == 0 {
		return
	}

	res = compileMethodsCSS(res, defaultDarkMode)


	if path, err := fs.JoinPath("./dist", outName+".css"); err == nil {
		os.WriteFile(path, regex.JoinBytes(`/*! `, config["theme_name"], ' ', config["theme_version"], ` | `, config["theme_license"], ` | `, config["theme_uri"], ` */`, '\n', res), 0755)
	}


	// fix browser support with nested @media elements
	res = compileNestedCSS(res, true)


	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	if res, err := m.Bytes("text/css", res); err == nil {
		res = regex.JoinBytes(`/*! `, config["theme_name"], ' ', config["theme_version"], ` | `, config["theme_license"], ` | `, config["theme_uri"], ` */`, '\n', res)

		if path, err := fs.JoinPath("./dist", outName+".min.css"); err == nil {
			os.WriteFile(path, res, 0755)
		}

		if outName == "style" {
			if path, err := fs.JoinPath(inDir, "normalize.min.css"); err == nil {
				if buf, err := os.ReadFile(path); err == nil {
					res = append([]byte{'\n'}, res...)
					res = append(buf, res...)
				}
			}

			if path, err := fs.JoinPath("./dist", outName+".norm.min.css"); err == nil {
				os.WriteFile(path, res, 0755)
			}
		}

	}else{
		res = regex.JoinBytes(`/*! `, config["theme_name"], ' ', config["theme_version"], ` | `, config["theme_license"], ` | `, config["theme_uri"], ` */`, '\n', res)

		if path, err := fs.JoinPath("./dist", outName+".min.css"); err == nil {
			os.WriteFile(path, res, 0755)
		}

		if outName == "style" {
			if path, err := fs.JoinPath(inDir, "normalize.min.css"); err == nil {
				if buf, err := os.ReadFile(path); err == nil {
					res = append([]byte{'\n'}, res...)
					res = append(buf, res...)
				}
			}
	
			if path, err := fs.JoinPath("./dist", outName+".norm.min.css"); err == nil {
				os.WriteFile(path, res, 0755)
			}
		}
	}
}

func compileJS(inDir string, outName string, init bool, themePath string){
	if stat, err := os.Stat("./dist"); err != nil || !stat.IsDir() {
		return
	}

	res := []byte{}

	if files, err := os.ReadDir(inDir); err == nil {
		for _, file := range files {
			if file.IsDir() {
				if init {
					name := file.Name()
					if name == "themes" || name == "theme" {
						continue
					}

					if name == "config" {
						name = "theme.config"
					}else if name == "style" {
						name = "theme.style"
					}else if name == "script" {
						name = "theme.script"
					}

					if filePath, err := fs.JoinPath(inDir, name); err == nil {
						compileJS(filePath, name, init, "")
					}
				}
			}else if strings.HasSuffix(file.Name(), ".js") && !strings.HasSuffix(file.Name(), ".min.js") {
				if filePath, err := fs.JoinPath(inDir, file.Name()); err == nil {
					if buf, err := os.ReadFile(filePath); err == nil {
						res = append(res, '\n')
						res = append(res, buf...)
					}
				}
			}
		}
	}

	if outName == "script" && themePath != "" {
		if files, err := os.ReadDir(themePath); err == nil {
			for _, file := range files {
				if file.IsDir() {
					if init {
						name := file.Name()
						if name == "themes" || name == "theme" {
							continue
						}
	
						if name == "config" {
							name = "theme.config"
						}else if name == "style" {
							name = "theme.style"
						}else if name == "script" {
							name = "theme.script"
						}
	
						if filePath, err := fs.JoinPath(themePath, name); err == nil {
							compileJS(filePath, name, init, "")
						}
					}
				}else if strings.HasSuffix(file.Name(), ".js") && !strings.HasSuffix(file.Name(), ".min.js") {
					if filePath, err := fs.JoinPath(themePath, file.Name()); err == nil {
						if buf, err := os.ReadFile(filePath); err == nil {
							res = append(res, '\n')
							res = append(res, buf...)
						}
					}
				}
			}
		}
	}

	if len(res) == 0 {
		return
	}

	if path, err := fs.JoinPath("./dist", outName+".js"); err == nil {
		os.WriteFile(path, regex.JoinBytes(`/*! `, config["theme_name"], ' ', config["theme_version"], ` | `, config["theme_license"], ` | `, config["theme_uri"], ` */`, '\n', res), 0755)
	}

	m := minify.New()
	m.AddFunc("text/javascript", js.Minify)
	if res, err := m.Bytes("text/javascript", res); err == nil {
		res = regex.JoinBytes(`/*! `, config["theme_name"], ' ', config["theme_version"], ` | `, config["theme_license"], ` | `, config["theme_uri"], ` */`, '\n', ';', res, ';')

		if path, err := fs.JoinPath("./dist", outName+".min.js"); err == nil {
			os.WriteFile(path, res, 0755)
		}
	}else{
		res = regex.JoinBytes(`/*! `, config["theme_name"], ' ', config["theme_version"], ` | `, config["theme_license"], ` | `, config["theme_uri"], ` */`, '\n', res)

		if path, err := fs.JoinPath("./dist", outName+".min.js"); err == nil {
			os.WriteFile(path, res, 0755)
		}
	}
}

func compileMethodsCSS(buf []byte, defaultDarkMode bool) []byte {
	// encode bracket indexes
	buf = regex.Comp(`\%!([\w]+)!\%`).RepFunc(buf, func(data func(int) []byte) []byte {
		return regex.JoinBytes(`%!o!%`, data(1), `%!c!%`)
	})

	ind := 0
	buf = regex.Comp(`\{|\}`).RepFunc(buf, func(data func(int) []byte) []byte {
		if data(0)[0] == '}' && ind > 0 {
			ind--
			return regex.JoinBytes(`%!`, ind, `!%}`)
		}
		r := regex.JoinBytes(`{%!`, ind, `!%`)
		ind++
		return r
	})


	// compile for loop
	buf = regex.Comp(`(?s)@for\s*\((.*?)\)\s*\{(\%![0-9]+!\%)(.*?)\2\}`).RepFunc(buf, func(data func(int) []byte) []byte {
		argsB := regex.Comp(`,\s*`).Split(data(1))
		args := []int{}
		var varName string
		for _, b := range argsB {
			if n, err := strconv.Atoi(string(b)); err == nil {
				args = append(args, n)
			}else{
				varName = string(b)
			}
		}

		for len(args) < 3 {
			args = append(args, 0)
		}

		if args[0] == args[1] {
			return []byte{}
		}

		if args[0] > args[1] {
			t := args[0]
			args[0] = args[1]
			args[1] = t
		}

		if args[2] == 0 {
			args[2] = 1
		}else if args[2] < 0 {
			if args[0] < args[1] {
				t := args[0]
				args[0] = args[1]
				args[1] = t
			}
		}

		res := []byte{}
		if args[2] > 0 {
			for i := args[0]; i <= args[1]; i += args[2] {
				if varName != "" {
					res = append(res, regex.Comp(`\{(\%![0-9]+!\%)%1\1\}`, varName).RepFunc(data(3), func(data func(int) []byte) []byte {
						return []byte(strconv.Itoa(i))
					})...)
				}else{
					res = append(res, data(3)...)
				}
			}
		}else if args[2] < 0 {
			for i := args[0]; i >= args[1]; i += args[2] {
				if varName != "" {
					res = append(res, regex.Comp(`\{(\%![0-9]+!\%)%1\1\}`, varName).RepFunc(data(3), func(data func(int) []byte) []byte {
						return []byte(strconv.Itoa(i))
					})...)
				}else{
					res = append(res, data(3)...)
				}
			}
		}

		return res
	})


	// decode bracket indexes
	buf = regex.Comp(`\%!([\w]+)!\%`).RepFunc(buf, func(data func(int) []byte) []byte {
		if data(0)[0] == 'o' {
			return []byte(`%!`)
		}else if data(0)[0] == 'c' {
			return []byte(`!%`)
		}
		return []byte{}
	})


	// compile element and color vars
	buf = regex.Comp(`(?s)@(elm|color|fg)\s*\(([\w_-]+)\)\s*;`).RepFunc(buf, func(data func(int) []byte) []byte {
		tag := string(data(1))
		val := string(data(2))
		if tag == "elm" {
			return compileConfigElm(val, defaultDarkMode)
		}else if tag == "color" {
			return compileConfigColor(val, defaultDarkMode)
		}else if tag == "fg" {
			return compileConfigFG(val)
		}
		return []byte{}
	})


	// compile mobile and desktop width
	buf = regex.Comp(`mobile-width`).RepStrLit(buf, []byte("800px"))
	buf = regex.Comp(`desktop-width`).RepStrLit(buf, []byte("1400px"))


	// compile random number (int) method
	buf = regex.Comp(`\.rand\((-?[0-9]+|)(,\s*-?[0-9]+|)\)`).RepFunc(buf, func(data func(int) []byte) []byte {
		var n1, n2 int

		if n, err := strconv.Atoi(string(data(1))); err == nil {
			n1 = n
		}

		if n, err := strconv.Atoi(string(regex.Comp(`,\s*`).RepStrLit(data(2), []byte{}))); err == nil {
			n2 = n
		}

		if n1 == 0 && n2 == 0 {
			return []byte(strconv.Itoa(rand.Intn(100)))
		}

		if n1 == n2 {
			return []byte(strconv.Itoa(n1))
		}else if n1 > n2 {
			t := n2
			n2 = n1
			n1 = t
		}

		return []byte(strconv.Itoa(rand.Intn(n2 - n1) + n1))
	})

	return buf
}


func compileNestedCSS(buf []byte, init bool) []byte {
	if init {
		buf = regex.Comp(`/\*(?:[\r\n\t\s ]|.)*?\*/`).RepStr(buf, []byte{})
		buf = regex.Comp(`([{};])`).RepStr(buf, []byte("$1\n"))

		// encode bracket indexes
		buf = regex.Comp(`\%!([\w]+)!\%`).RepFunc(buf, func(data func(int) []byte) []byte {
			return regex.JoinBytes(`%!o!%`, data(1), `%!c!%`)
		})
	
		ind := 0
		buf = regex.Comp(`\{|\}`).RepFunc(buf, func(data func(int) []byte) []byte {
			if data(0)[0] == '}' && ind > 0 {
				ind--
				return regex.JoinBytes(`%!`, ind, `!%}`)
			}
			r := regex.JoinBytes(`{%!`, ind, `!%`)
			ind++
			return r
		})
	}


	buf = regex.Comp(`(?m)^\s*(.*?)\{%!([0-9]+)!%((?:[r\n\t\s ]|.)*?)%!\2!%\}`).RepFunc(buf, func(data func(int) []byte) []byte {
		selStr := data(1)
		if bytes.HasPrefix(selStr, []byte{'@'}) {
			if regex.Comp(`^\s*@[\w_-]+\s*[\w_-]*\s*$`).Match(selStr) {
				return regex.JoinBytes(selStr, '{', []byte("%!"), data(2), []byte("!%"), data(3), []byte("%!"), data(2), []byte("!%"), '}')
			}
			selStr = regex.JoinBytes([]byte("(%!"), bytes.TrimSpace(selStr), []byte("!%)"))
		}
		
		sel := bytes.Split(selStr, []byte{','})
		for i, v := range sel {
			sel[i] = bytes.TrimSpace(v)
		}

		addCSS := []byte{}
		b := regex.Comp(`(?m)^\s*(.*?)\{%!([0-9]+)!%((?:[r\n\t\s ]|.)*?)%!\2!%\}`).RepFunc(data(3), func(data func(int) []byte) []byte {
			selStr := data(1)
			if bytes.HasPrefix(selStr, []byte{'@'}) {
				if regex.Comp(`^\s*@[\w_-]+\s*[\w_-]*\s*\{`).Match(selStr) {
					return regex.JoinBytes(selStr, '{', []byte("%!"), data(2), []byte("!%"), data(3), []byte("%!"), data(2), []byte("!%"), '}')
				}
				selStr = regex.JoinBytes([]byte("(%!"), bytes.TrimSpace(selStr), []byte("!%)"))
			}

			newSel := [][]byte{}
			s := bytes.Split(selStr, []byte{','})
			for _, v := range s {
				v = bytes.TrimSpace(v)
				if !bytes.ContainsRune(v, '&') {
					v = append([]byte("& "), v...)
				}

				
				for _, selI := range sel {
					newSel = append(newSel, bytes.ReplaceAll(v, []byte{'&'}, selI))
				}
			}

			addCSS = compileNestedCSS(regex.JoinBytes(addCSS, bytes.Join(newSel, []byte{','}), '{', []byte("%!"), data(2), []byte("!%"), data(3), []byte("%!"), data(2), []byte("!%"), '}', '\n'), false)
			return []byte{}
		})

		return regex.JoinBytes(selStr, '{', []byte("%!"), data(2), []byte("!%"), compileNestedCSS(b, false), []byte("%!"), data(2), []byte("!%"), '}', '\n', addCSS)
	})


	if init {
		buf = regex.Comp(`(?m)^\s*(.*?)\{%!([0-9]+)!%((?:[r\n\t\s ]|.)*?)%!\2!%\}`).RepFunc(buf, func(data func(int) []byte) []byte {
			if len(bytes.TrimSpace(data(3))) == 0 {
				return []byte{}
			}
			return data(0)
		})

		buf = regex.Comp(`(?m)^\s*(.*?)\{%!([0-9]+)!%((?:[r\n\t\s ]|.)*?)%!\2!%\}`).RepFunc(buf, func(data func(int) []byte) []byte {
			if !regex.Comp(`\(%!(@.*?)!%\)`).Match(data(1)) {
				return data(0)
			}

			query := [][]byte{}
			sel := regex.Comp(`\(%!(@.*?)!%\)`).RepFunc(data(1), func(data func(int) []byte) []byte {
				query = append(query, data(1))
				return []byte{}
			})

			if len(bytes.TrimSpace(sel)) != 0 {
				sel = append(sel, '{')
			}

			res := regex.JoinBytes(bytes.Join(query, []byte{'{'}), '{', sel, []byte("%!"), data(2), []byte("!%"), data(3), []byte("%!"), data(2), []byte("!%"), bytes.Repeat([]byte{'}'}, len(query)))
			if len(bytes.TrimSpace(sel)) != 0 {
				res = append(res, '}')
			}

			return res
		})


		// decode bracket indexes
		buf = regex.Comp(`\%!([\w]+)!\%`).RepFunc(buf, func(data func(int) []byte) []byte {
			if data(0)[0] == 'o' {
				return []byte(`%!`)
			}else if data(0)[0] == 'c' {
				return []byte(`!%`)
			}
			return []byte{}
		})
	}

	return buf
}


func colorFromString(str string, defHex string) colorful.Color {
	if reg := regex.Comp(`(?i)^\s*(hs[lv]|rgb|hcl)a?\s*\(\s*([0-9]+(?:deg|))\s*,\s*([0-9]+(?:%|\.[0-9]+|))\s*,\s*([0-9]+(?:%|\.[0-9]+|))(?:\s*,.*?|)\)\s*$`); reg.Match([]byte(str)) {
		var color colorful.Color
		hasCol := false
		reg.RepFunc([]byte(str), func(data func(int) []byte) []byte {
			col := [3]float64{}

			if bytes.HasSuffix(data(2), []byte("deg")) {
				if n, err := strconv.ParseFloat(string(data(2)[:len(data(2))-3]), 64); err == nil {
					col[0] = n
				}
			}else{
				if n, err := strconv.ParseFloat(string(data(2)), 64); err == nil {
					col[0] = n
				}
			}

			if bytes.HasSuffix(data(3), []byte("%")) {
				if n, err := strconv.ParseFloat(string(data(3)[:len(data(3))-1]), 64); err == nil {
					col[1] = n / 100
				}
			}else{
				if n, err := strconv.ParseFloat(string(data(3)), 64); err == nil {
					col[1] = n
				}
			}

			if bytes.HasSuffix(data(4), []byte("%")) {
				if n, err := strconv.ParseFloat(string(data(4)[:len(data(4))-1]), 64); err == nil {
					col[2] = n / 100
				}
			}else{
				if n, err := strconv.ParseFloat(string(data(4)), 64); err == nil {
					col[2] = n
				}
			}

			switch string(bytes.ToLower(data(1))) {
			case "hsl":
				color = colorful.Hsl(col[0], col[1], col[2])
				hasCol = true
			case "hsv":
				color = colorful.Hsv(col[0], col[1], col[2])
				hasCol = true
			case "rgb":
				color = colorful.LinearRgb(col[0], col[1], col[2])
				hasCol = true
			case "hcl":
				color = colorful.Hcl(col[0], col[1], col[2])
				hasCol = true
			default:
				hasCol = false
			}
			return nil
		}, true)

		if hasCol {
			return color
		}else if color, err := colorful.Hex(defHex); err == nil {
			return color
		}else{
			return colorful.Color{}
		}
	}

	if strings.HasPrefix(str, "#") {
		if color, err := colorful.Hex(str); err == nil {
			return color
		}else if color, err := colorful.Hex(defHex); err == nil {
			return color
		}
	}

	return colorful.Color{}
}

func colorToHsl(color colorful.Color) [3]float64 {
	h, s, l := color.Hsl()
	return [3]float64{h, s, l}
}

func contrastRatio(fg colorful.Color, bg colorful.Color) int16 {
	return int16(math.Max(fg.DistanceRgb(bg), fg.DistanceLuv(bg)) * 100)
}

func hslStringPart(f float64, t string) string {
	if t == "deg" {
		return strconv.FormatFloat(math.Round(f), 'f', 0, 32) + "deg"
	}else if t == "%" {
		return strconv.FormatFloat(math.Round(f*100), 'f', 0, 32) + "%"
	}

	return strconv.FormatFloat(f, 'f', 2, 32)
}
