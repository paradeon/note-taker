package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type track struct {
	id       string
	title    string
	artist   string
	album    string
	genre    string
	duration int // seconds, -1 if unknown
}

func cookieArgs(browser, file string) []string {
	var args []string
	if browser != "" {
		args = append(args, "--cookies-from-browser", browser)
	}
	if file != "" {
		args = append(args, "--cookies", file)
	}
	return args
}

func parseDuration(s string) int {
	if s == "NA" || s == "" {
		return -1
	}
	// yt-dlp may emit integer or float seconds
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(math.Round(f))
	}
	return -1
}

// splitArtistTitle tries to split "Artist - Title" from a YouTube video title.
// Falls back to channel as artist and full title as song title.
func splitArtistTitle(title, channel string) (artist, songTitle string) {
	if idx := strings.Index(title, " - "); idx != -1 {
		return strings.TrimSpace(title[:idx]), strings.TrimSpace(title[idx+3:])
	}
	return channel, title
}

func fetchPlaylist(url string, ckArgs []string) ([]track, string, error) {
	args := []string{
		"--flat-playlist",
		"--no-warnings",
		"--print", "%(id)s\t%(title)s\t%(channel)s\t%(duration)s\t%(playlist_title)s",
	}
	args = append(args, ckArgs...)
	args = append(args, url)

	cmd := exec.Command("yt-dlp", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, "", fmt.Errorf("%s", msg)
		}
		return nil, "", fmt.Errorf("yt-dlp failed")
	}

	var tracks []track
	var playlistTitle string

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 5)
		if len(parts) < 5 {
			continue
		}
		id, title, channel, durStr, pTitle := parts[0], parts[1], parts[2], parts[3], parts[4]

		if playlistTitle == "" && pTitle != "NA" {
			playlistTitle = pTitle
		}

		artist, songTitle := splitArtistTitle(title, channel)
		tracks = append(tracks, track{
			id:       id,
			title:    songTitle,
			artist:   artist,
			album:    playlistTitle,
			duration: parseDuration(durStr),
		})
	}

	// back-fill album for entries parsed before playlistTitle was found
	for i := range tracks {
		if tracks[i].album == "" {
			tracks[i].album = playlistTitle
		}
	}

	return tracks, playlistTitle, scanner.Err()
}

func fetchGenre(id string, ckArgs []string) string {
	args := []string{"--no-playlist", "--no-warnings", "--print", "%(categories.0)s"}
	args = append(args, ckArgs...)
	args = append(args, "https://www.youtube.com/watch?v="+id)
	out, err := exec.Command("yt-dlp", args...).Output()
	if err != nil {
		return ""
	}
	g := strings.TrimSpace(string(out))
	if g == "NA" {
		return ""
	}
	return g
}

func writeM3U8(w io.Writer, tracks []track) {
	fmt.Fprintln(w, "#EXTM3U")
	for _, t := range tracks {
		fmt.Fprintln(w)
		display := t.title
		if t.artist != "" {
			display = t.artist + " - " + t.title
		}
		fmt.Fprintf(w, "#EXTINF:%d,%s\n", t.duration, display)
		if t.artist != "" {
			fmt.Fprintf(w, "#EXTART:%s\n", t.artist)
		}
		if t.album != "" {
			fmt.Fprintf(w, "#EXTALB:%s\n", t.album)
		}
		if t.genre != "" {
			fmt.Fprintf(w, "#EXTGENRE:%s\n", t.genre)
		}
		fmt.Fprintf(w, "https://www.youtube.com/watch?v=%s\n", t.id)
	}
}

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '_':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// parseEntrySet parses a spec like "1,3,5..7" into a set of 1-based indices.
// An empty spec means "all".
func parseEntrySet(spec string) (map[int]bool, error) {
	if spec == "" {
		return nil, nil
	}
	set := map[int]bool{}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "..") {
			sides := strings.SplitN(part, "..", 2)
			loStr, hiStr := strings.TrimSpace(sides[0]), strings.TrimSpace(sides[1])
			lo, hi := 1, math.MaxInt32
			var err error
			if loStr != "" {
				if lo, err = strconv.Atoi(loStr); err != nil || lo < 1 {
					return nil, fmt.Errorf("invalid range %q", part)
				}
			}
			if hiStr != "" {
				if hi, err = strconv.Atoi(hiStr); err != nil || hi < 1 {
					return nil, fmt.Errorf("invalid range %q", part)
				}
			}
			for i := lo; i <= hi; i++ {
				set[i] = true
			}
		} else {
			n, err := strconv.Atoi(part)
			if err != nil || n < 1 {
				return nil, fmt.Errorf("invalid id %q", part)
			}
			set[n] = true
		}
	}
	return set, nil
}

func isYouTubeURL(s string) bool {
	return strings.Contains(s, "youtube.com/") || strings.Contains(s, "youtu.be/")
}

func resolveStreamURL(url string, ckArgs []string) (string, error) {
	args := []string{"-f", "bestaudio", "-g", "--no-warnings"}
	args = append(args, ckArgs...)
	args = append(args, url)
	out, err := exec.Command("yt-dlp", args...).Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp: %w", err)
	}
	line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	if line == "" {
		return "", fmt.Errorf("no stream URL returned")
	}
	return line, nil
}

func convertM3U8(inFile, outFile string, ckArgs []string, only map[int]bool) error {
	f, err := os.Open(inFile)
	if err != nil {
		return err
	}
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	scanErr := scanner.Err()
	f.Close()
	if scanErr != nil {
		return scanErr
	}

	// number YouTube entries so -n indices are stable and predictable
	ytIndex := 0
	total := 0
	for _, l := range lines {
		if isYouTubeURL(l) {
			ytIndex++
			if only == nil || only[ytIndex] {
				total++
			}
		}
	}
	fmt.Fprintf(os.Stderr, "⟳ Resolving %d YouTube URL(s)…\n", total)

	ytIndex = 0
	done := 0
	for i, l := range lines {
		if !isYouTubeURL(l) {
			continue
		}
		ytIndex++
		if only != nil && !only[ytIndex] {
			continue
		}
		done++
		fmt.Fprintf(os.Stderr, "  [%d/%d] (#%d) %s\n", done, total, ytIndex, l)
		resolved, err := resolveStreamURL(l, ckArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %v\n", err)
			continue
		}
		lines[i] = resolved
	}

	out, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer out.Close()
	bw := bufio.NewWriter(out)
	for _, l := range lines {
		fmt.Fprintln(bw, l)
	}
	return bw.Flush()
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: yt2m3u [flags] <youtube-url>")
	fmt.Fprintln(os.Stderr, "       yt2m3u [flags] -i <input.m3u8>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  -i <file>            Input m3u8; resolve YouTube entries to stream URLs")
	fmt.Fprintln(os.Stderr, "  -n <ids>             Entries to resolve: list or range, e.g. 1,3,5..7 (default: all)")
	fmt.Fprintln(os.Stderr, "  -o <file>            Output file (default: <playlist-title>.m3u8 or <input>-resolved.m3u8)")
	fmt.Fprintln(os.Stderr, "  -g                   Fetch genre per video (slow — one request per track)")
	fmt.Fprintln(os.Stderr, "  -b <browser>         Cookies from browser (e.g. chrome, firefox, safari)")
	fmt.Fprintln(os.Stderr, "  --cookies <file>     Path to Netscape cookies file")
}

func main() {
	inFile := flag.String("i", "", "Input m3u8 file")
	entrySpec := flag.String("n", "", "Entries to resolve (e.g. 1,3,5..7)")
	outFile := flag.String("o", "", "Output file")
	doGenre := flag.Bool("g", false, "Fetch genre per video (slow)")
	browser := flag.String("b", "", "Cookies from browser")
	cookiesFilePath := flag.String("cookies", "", "Cookies file")
	flag.Usage = printUsage
	flag.Parse()

	ckArgs := cookieArgs(*browser, *cookiesFilePath)

	if *inFile != "" {
		only, err := parseEntrySet(*entrySpec)
		if err != nil {
			fmt.Fprintln(os.Stderr, "✗", err)
			os.Exit(1)
		}
		out := *outFile
		if out == "" {
			base := strings.TrimSuffix(*inFile, ".m3u8")
			out = base + "-resolved.m3u8"
		}
		if err := convertM3U8(*inFile, out, ckArgs, only); err != nil {
			fmt.Fprintln(os.Stderr, "✗", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "✓ → %s\n", out)
		return
	}

	if flag.NArg() == 0 {
		printUsage()
		os.Exit(1)
	}

	url := flag.Arg(0)

	fmt.Fprintln(os.Stderr, "⟳ Fetching playlist…")
	tracks, playlistTitle, err := fetchPlaylist(url, ckArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "✗", err)
		os.Exit(1)
	}
	if len(tracks) == 0 {
		fmt.Fprintln(os.Stderr, "✗ No tracks found")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "  Found %d tracks\n", len(tracks))

	if *doGenre {
		fmt.Fprintf(os.Stderr, "⟳ Fetching genre for %d tracks…\n", len(tracks))
		for i := range tracks {
			tracks[i].genre = fetchGenre(tracks[i].id, ckArgs)
			fmt.Fprintf(os.Stderr, "  [%d/%d] %s\n", i+1, len(tracks), tracks[i].title)
		}
	}

	out := *outFile
	if out == "" {
		name := slugify(playlistTitle)
		if name == "" {
			name = "playlist"
		}
		out = name + ".m3u8"
	}

	f, err := os.Create(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "✗", err)
		os.Exit(1)
	}
	defer f.Close()

	writeM3U8(f, tracks)
	fmt.Fprintf(os.Stderr, "✓ %d tracks → %s\n", len(tracks), out)
}
