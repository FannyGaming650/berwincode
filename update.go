package main

// Self-update: compares berwinVersion against the latest GitHub release
// and can download + launch the new installer. DNS-proofed transport
// for PCs whose hosts file blocks api.github.com.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const updateOwner = "FannyGaming650"
const updateRepo = "berwincode"
const updateAPIHost = "api.github.com"
const updateFallbackIP = "140.82.112.6"

func updateDisabled() bool {
	return os.Getenv("BERWINCODE_NO_UPDATE") == "1"
}

func dohResolveIPv4(host string) string {
	urls := []string{
		"https://cloudflare-dns.com/dns-query?name=" + host + "&type=A",
		"https://dns.google/resolve?name=" + host + "&type=A",
	}
	for _, u := range urls {
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("accept", "application/dns-json")
		resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
		if err != nil {
			continue
		}
		var m struct {
			Answer []struct {
				Type int    `json:"type"`
				Data string `json:"data"`
			} `json:"Answer"`
		}
		derr := json.NewDecoder(resp.Body).Decode(&m)
		resp.Body.Close()
		if derr != nil {
			continue
		}
		for _, a := range m.Answer {
			if a.Type == 1 && net.ParseIP(a.Data) != nil {
				return a.Data
			}
		}
	}
	return ""
}

func apiClient() *http.Client {
	plain := &http.Client{Timeout: 12 * time.Second}
	return plain
}

func apiClientSNI() *http.Client {
	var base *http.Transport
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		base = t.Clone()
	} else {
		base = &http.Transport{}
	}
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		h, p, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ip := updateFallbackIP
		if h == updateAPIHost {
			if resolved := dohResolveIPv4(h); resolved != "" {
				ip = resolved
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ip, p))
	}
	base.TLSClientConfig = &tls.Config{ServerName: updateAPIHost}
	return &http.Client{Transport: base, Timeout: 20 * time.Second}
}

func apiGet(path string) (*http.Response, error) {
	url := "https://" + updateAPIHost + path
	if resp, err := apiClient().Get(url); err == nil {
		return resp, nil
	} else {
		_ = err
	}
	return apiClientSNI().Get(url)
}

func parseVer(v string) []int {
	v = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V"))
	out := []int{0, 0, 0}
	for i, p := range strings.Split(v, ".") {
		if i >= 3 {
			break
		}
		n, _ := strconv.Atoi(strings.TrimFunc(p, func(r rune) bool {
			return r < '0' || r > '9'
		}))
		out[i] = n
	}
	return out
}

func newerAvailable(tag string) bool {
	have := parseVer(berwinVersion)
	want := parseVer(tag)
	for i := 0; i < 3; i++ {
		if want[i] != have[i] {
			return want[i] > have[i]
		}
	}
	return false
}

func latestRelease() (tag, setupURL, pageURL string, err error) {
	resp, err := apiGet("/repos/" + updateOwner + "/" + updateRepo + "/releases/latest")
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", "", fmt.Errorf("status %s", resp.Status)
	}
	var m struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return "", "", "", err
	}
	best := ""
	for _, a := range m.Assets {
		l := strings.ToLower(a.Name)
		if strings.HasSuffix(l, ".exe") && strings.Contains(l, "setup") {
			best = a.URL
			break
		}
	}
	if best == "" {
		for _, a := range m.Assets {
			if strings.HasSuffix(strings.ToLower(a.Name), ".zip") {
				best = a.URL
				break
			}
		}
	}
	if best == "" && len(m.Assets) > 0 {
		best = m.Assets[0].URL
	}
	return m.TagName, best, m.HTMLURL, nil
}

func checkForUpdate() (tag, setupURL, pageURL string, available bool) {
	if updateDisabled() {
		return "", "", "", false
	}
	tag, setupURL, pageURL, err := latestRelease()
	if err != nil || tag == "" {
		return "", "", "", false
	}
	if !newerAvailable(tag) {
		return "", "", "", false
	}
	return tag, setupURL, pageURL, true
}

func msgBoxYesNo(caption, text string) bool {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	capPtr, _ := syscall.UTF16PtrFromString(caption)
	txtPtr, _ := syscall.UTF16PtrFromString(text)
	r1, _, _ := proc.Call(0, uintptr(unsafe.Pointer(txtPtr)), uintptr(unsafe.Pointer(capPtr)), 0x24)
	return uint32(r1) == 6
}

func cmdUpgrade(interactive bool) int {
	tag, setupURL, pageURL, ok := checkForUpdate()
	if !ok {
		fmt.Printf("BerwinCode v%s is the latest version.\n", berwinVersion)
		return 0
	}
	fmt.Printf("BerwinCode %s is available (you have v%s).\n", tag, berwinVersion)
	if !interactive {
		fmt.Printf("Download it here: %s\n", pageURL)
		return 0
	}
	if !msgBoxYesNo("BerwinCode", "BerwinCode "+tag+" is available.\n\nDownload and install it now?") {
		return 0
	}
	if setupURL == "" {
		fmt.Printf("No installer found. Get it here: %s\n", pageURL)
		pauseEnter()
		return 1
	}
	dst := filepath.Join(os.TempDir(), "BerwinCode-"+strings.TrimPrefix(tag, "v")+"-setup.exe")
	fmt.Fprintf(os.Stderr, "BerwinCode: downloading %s ...\n", tag)
	if err := downloadFile(setupURL, dst); err != nil {
		fmt.Fprintf(os.Stderr, "Download failed: %v\nGet it here: %s\n", err, pageURL)
		pauseEnter()
		return 1
	}
	fmt.Fprintln(os.Stderr, "BerwinCode: starting installer, closing now...")
	_ = exec.Command(dst).Start()
	return 0
}

func maybeAutoUpdate() {
	tag, setupURL, pageURL, ok := checkForUpdate()
	if !ok {
		return
	}
	fmt.Fprintf(os.Stderr, "BerwinCode %s found - updating automatically...\n", tag)
	if setupURL == "" {
		fmt.Fprintf(os.Stderr, "No installer found. Get it here: %s\n", pageURL)
		return
	}
	dst := filepath.Join(os.TempDir(), "BerwinCode-"+strings.TrimPrefix(tag, "v")+"-setup.exe")
	if err := downloadFile(setupURL, dst); err != nil {
		fmt.Fprintf(os.Stderr, "Auto-update download failed: %v\nContinuing with v%s.\n", err, berwinVersion)
		return
	}
	_ = exec.Command(dst).Start()
	fmt.Fprintln(os.Stderr, "BerwinCode: installer started, closing so it can update. Reopen BerwinCode after.")
	os.Exit(0)
}

func maybeOfferUpdate() {
	tag, _, pageURL, ok := checkForUpdate()
	if !ok {
		return
	}
	_ = pageURL
	if msgBoxYesNo("BerwinCode", "BerwinCode "+tag+" is available (you have v"+berwinVersion+").\n\nDownload and install it now?") {
		_ = cmdUpgrade(true)
		os.Exit(0)
	}
}
