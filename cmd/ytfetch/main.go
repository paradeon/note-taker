package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var vidIDRe = regexp.MustCompile(`(?:v=|youtu\.be/|shorts/)([A-Za-z0-9_-]{11})`)

// timestamp patterns to strip from VTT/SRT transcripts
var (
	vttTimestampRe = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}`)
	srtSeqRe       = regexp.MustCompile(`^\d+\s*$`)
)

type meta struct {
	Title                string         `json:"title"`
	Description          string         `json:"description"`
	Categories           []string       `json:"categories"`
	Tags                 []string       `json:"tags"`
	LikeCount            *int64         `json:"like_count"`
	ViewCount            *int64         `json:"view_count"`
	Uploader             string         `json:"uploader"`
	Creator              string         `json:"creator"`
	Channel              string         `json:"channel"`
	ChannelURL           string         `json:"channel_url"`
	UploaderURL          string         `json:"uploader_url"`
	ChannelFollowerCount *int64         `json:"channel_follower_count"`
	RelatedVideos        []relatedVideo `json:"related_videos"`
	Comments             []comment      `json:"comments"`
}

type relatedVideo struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Channel string `json:"channel"`
}

type comment struct {
	Author    string `json:"author"`
	Text      string `json:"text"`
	LikeCount *int64 `json:"like_count"`
}

type dislikeData struct {
	Dislikes *int64   `json:"dislikes"`
	Rating   *float64 `json:"rating"`
}

func int64Val(p *int64) string {
	if p == nil {
		return "unavailable"
	}
	return fmt.Sprintf("%d", *p)
}

func section(quiet bool, title string) {
	if !quiet {
		fmt.Printf("━━━ %-60s\n", title+" ")
	}
}

func fetchMeta(url string) (*meta, error) {
	out, err := exec.Command("yt-dlp",
		"--dump-single-json", "--no-playlist", "--quiet", url).Output()
	if err != nil || len(out) == 0 {
		return nil, fmt.Errorf("failed to fetch metadata — check the URL or your network")
	}
	var m meta
	if err := json.Unmarshal(out, &m); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}
	return &m, nil
}

func fetchCommentsMeta(url string) (*meta, error) {
	out, err := exec.Command("yt-dlp",
		"--write-comments",
		"--extractor-args", "youtube:comment_sort=top,max_comments=20",
		"--dump-single-json",
		"--no-playlist",
		"--no-warnings",
		url).Output()
	if err != nil || len(out) == 0 {
		return nil, fmt.Errorf("could not fetch comments")
	}
	var m meta
	if err := json.Unmarshal(out, &m); err != nil {
		return nil, fmt.Errorf("failed to parse comments JSON: %w", err)
	}
	return &m, nil
}

func printDescription(m *meta, quiet bool) {
	section(quiet, "DESCRIPTION")
	desc := m.Description
	if desc == "" {
		desc = "No description available."
	}
	fmt.Println(desc)
	if !quiet {
		fmt.Println()
	}
}

func printGenre(m *meta, quiet bool) {
	section(quiet, "GENRE / CATEGORY")
	category := strings.Join(m.Categories, ", ")
	var tags []string
	for i, t := range m.Tags {
		if i >= 8 {
			break
		}
		tags = append(tags, t)
	}
	tagStr := strings.Join(tags, ", ")
	if quiet {
		if category != "" {
			fmt.Println(category)
		} else if tagStr != "" {
			fmt.Println(tagStr)
		}
		return
	}
	fmt.Println("Category :", category)
	fmt.Println("Tags     :", tagStr)
	fmt.Println()
}

func printTitle(m *meta, quiet bool) {
	section(quiet, "TITLE")
	title := m.Title
	if title == "" {
		title = "unavailable"
	}
	if quiet {
		fmt.Println(title)
		return
	}
	fmt.Println("Title :", title)
	fmt.Println()
}

func printLikes(m *meta, quiet bool) {
	section(quiet, "LIKES")
	v := int64Val(m.LikeCount)
	if quiet {
		fmt.Println(v)
		return
	}
	fmt.Println("Likes :", v)
	fmt.Println()
}

func printDislikes(vidID string, quiet bool) {
	section(quiet, "DISLIKES (estimated)")
	resp, err := http.Get("https://returnyoutubedislikeapi.com/votes?videoId=" + vidID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "✗ Could not reach returnyoutubedislike API.")
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var d dislikeData
	if err := json.Unmarshal(body, &d); err != nil {
		fmt.Fprintln(os.Stderr, "✗ Failed to parse dislike data.")
		return
	}
	dislikes := int64Val(d.Dislikes)
	rating := "unavailable"
	if d.Rating != nil {
		rating = fmt.Sprintf("%.2f", *d.Rating)
	}
	if quiet {
		fmt.Println(dislikes)
		return
	}
	fmt.Printf("Dislikes : %s  (source: returnyoutubedislike.com)\n", dislikes)
	fmt.Printf("Rating   : %s / 5\n", rating)
	fmt.Println()
}

func printViews(m *meta, quiet bool) {
	section(quiet, "VIEWS")
	v := int64Val(m.ViewCount)
	if quiet {
		fmt.Println(v)
		return
	}
	fmt.Println("Views :", v)
	fmt.Println()
}

func printCreator(m *meta, quiet bool) {
	section(quiet, "CREATOR")
	creator := m.Uploader
	if creator == "" {
		creator = m.Creator
	}
	if creator == "" {
		creator = m.Channel
	}
	if creator == "" {
		creator = "unavailable"
	}
	if quiet {
		fmt.Println(creator)
		return
	}
	fmt.Println("Creator :", creator)
	fmt.Println()
}

func printChannel(m *meta, quiet bool) {
	section(quiet, "CHANNEL")
	name := m.Channel
	if name == "" {
		name = m.Uploader
	}
	if name == "" {
		name = "unavailable"
	}
	url := m.ChannelURL
	if url == "" {
		url = m.UploaderURL
	}
	if url == "" {
		url = "unavailable"
	}
	if quiet {
		fmt.Println(name, " ", url)
		return
	}
	fmt.Println("Name :", name)
	fmt.Println("URL  :", url)
	fmt.Println()
}

func printSubs(m *meta, quiet bool) {
	section(quiet, "SUBSCRIBERS")
	v := int64Val(m.ChannelFollowerCount)
	if quiet {
		fmt.Println(v)
		return
	}
	fmt.Println("Subscribers :", v)
	fmt.Println()
}

func printTranscript(url, vidID string, quiet bool) {
	section(quiet, "TRANSCRIPT")
	subDir := "ytfetch_" + vidID
	if err := os.MkdirAll(subDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "✗ Could not create subtitle directory.")
		return
	}

	cmd := exec.Command("yt-dlp",
		"--write-subs", "--write-auto-subs",
		"--sub-langs", "en.*",
		"--sub-format", "vtt/srt/best",
		"--skip-download",
		"--quiet",
		"-o", filepath.Join(subDir, "%(title)s.%(ext)s"),
		url)
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	files, _ := filepath.Glob(filepath.Join(subDir, "*.vtt"))
	if len(files) == 0 {
		files, _ = filepath.Glob(filepath.Join(subDir, "*.srt"))
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "✗ No subtitles found (video may not have captions).")
		return
	}

	lines := parseSubtitle(files[0])
	if quiet {
		for _, l := range lines {
			fmt.Println(l)
		}
		return
	}
	fmt.Println("✓ Transcript saved to:")
	for _, f := range files {
		fmt.Println(" ", f)
	}
	fmt.Println()
	fmt.Println("── Preview (first 30 lines) ──────────────────────────────────")
	for i, l := range lines {
		if i >= 30 {
			break
		}
		fmt.Println(l)
	}
	fmt.Println()
}

func parseSubtitle(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "WEBVTT") ||
			strings.HasPrefix(l, "NOTE") ||
			vttTimestampRe.MatchString(l) ||
			srtSeqRe.MatchString(l) ||
			strings.TrimSpace(l) == "" {
			continue
		}
		lines = append(lines, l)
	}
	return lines
}

func printComments(url string, quiet bool) {
	section(quiet, "COMMENTS / DISCUSSIONS")
	m, err := fetchCommentsMeta(url)
	if err != nil || len(m.Comments) == 0 {
		fmt.Fprintln(os.Stderr, "✗ No comments found.")
		return
	}
	if quiet {
		for _, c := range m.Comments {
			fmt.Println(c.Text)
		}
		return
	}
	fmt.Printf("Top %d comments:\n\n", len(m.Comments))
	for _, c := range m.Comments {
		likes := int64Val(c.LikeCount)
		fmt.Printf("👤 %s  ❤ %s\n%s\n%s\n", c.Author, likes, c.Text,
			strings.Repeat("─", 41))
	}
	fmt.Println()
}

func printThumbnail(vidID string, quiet bool) {
	section(quiet, "THUMBNAIL")
	outFile := "ytfetch_thumbnail_" + vidID + ".jpg"
	urls := []string{
		"https://img.youtube.com/vi/" + vidID + "/maxresdefault.jpg",
		"https://img.youtube.com/vi/" + vidID + "/hqdefault.jpg",
	}
	var usedURL string
	for _, u := range urls {
		resp, err := http.Get(u)
		if err != nil || resp.StatusCode != 200 {
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		f, err := os.Create(outFile)
		if err != nil {
			resp.Body.Close()
			fmt.Fprintln(os.Stderr, "✗ Could not create thumbnail file.")
			return
		}
		_, copyErr := io.Copy(f, resp.Body)
		resp.Body.Close()
		f.Close()
		if copyErr == nil {
			usedURL = u
			break
		}
	}
	if usedURL == "" {
		fmt.Fprintln(os.Stderr, "✗ Could not download thumbnail.")
		return
	}
	if quiet {
		fmt.Println(outFile)
		return
	}
	fmt.Println("✓ Thumbnail saved:", outFile)
	fmt.Println("  URL:", usedURL)
	fmt.Println()
}

func printSimilar(m *meta, vidID string, quiet bool) {
	section(quiet, "SIMILAR / RELATED VIDEOS")
	if len(m.RelatedVideos) > 0 {
		limit := 10
		if len(m.RelatedVideos) < limit {
			limit = len(m.RelatedVideos)
		}
		for _, v := range m.RelatedVideos[:limit] {
			url := "https://youtu.be/" + v.ID
			if quiet {
				fmt.Println(url)
			} else {
				fmt.Printf("• %s\n  Channel : %s\n  URL     : %s\n\n", v.Title, v.Channel, url)
			}
		}
		if !quiet {
			fmt.Println()
		}
		return
	}

	// fallback: mix playlist
	if !quiet {
		fmt.Println("⟳ Trying YouTube mix playlist…")
	}
	mixURL := "https://www.youtube.com/watch?v=" + vidID + "&list=RD" + vidID
	out, err := exec.Command("yt-dlp",
		"--flat-playlist",
		"--print", "%(title)s|https://youtu.be/%(id)s|%(channel)s",
		"--playlist-end", "10",
		"--quiet",
		mixURL).Output()
	if err != nil || len(out) == 0 {
		fmt.Fprintln(os.Stderr, "✗ No related videos found.")
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		if quiet {
			fmt.Println(parts[1])
		} else {
			fmt.Printf("• %s\n  Channel : %s\n  URL     : %s\n\n", parts[0], parts[2], parts[1])
		}
	}
	if !quiet {
		fmt.Println()
	}
}

func printHelp() {
	fmt.Println(`Usage: ytfetch [flags] <youtube-url>

Flags:
  -d, --description   Print video description
  -g, --genre         Print video category / genre
  -T, --transcript    Download transcript (subtitles)
  -c, --comments      Fetch top comments / discussions
  -p, --thumbnail     Download thumbnail image
  -s, --similar       List similar / related videos
  -t, --title         Print video title
  -l, --likes         Print like count
  -D, --dislikes      Print dislike count (via returnyoutubedislike.com)
  -n, --views         Print view count
  -r, --creator       Print uploader / creator name
  -C, --channel       Print channel name + URL
  -S, --subs          Print subscriber count
  -a, --all           Fetch everything
  -q, --quiet         Only output raw value(s), no labels or chrome
  -h, --help          Show this help

Dependencies: yt-dlp, jq, curl`)
}

func main() {
	var (
		doDescription bool
		doGenre       bool
		doTranscript  bool
		doComments    bool
		doThumbnail   bool
		doSimilar     bool
		doTitle       bool
		doLikes       bool
		doDislikes    bool
		doViews       bool
		doCreator     bool
		doChannel     bool
		doSubs        bool
		doAll         bool
		doQuiet       bool
		doHelp        bool
	)

	flag.BoolVar(&doDescription, "d", false, "")
	flag.BoolVar(&doDescription, "description", false, "Print video description")
	flag.BoolVar(&doGenre, "g", false, "")
	flag.BoolVar(&doGenre, "genre", false, "Print video category / genre")
	flag.BoolVar(&doTranscript, "T", false, "")
	flag.BoolVar(&doTranscript, "transcript", false, "Download transcript")
	flag.BoolVar(&doComments, "c", false, "")
	flag.BoolVar(&doComments, "comments", false, "Fetch top comments")
	flag.BoolVar(&doThumbnail, "p", false, "")
	flag.BoolVar(&doThumbnail, "thumbnail", false, "Download thumbnail image")
	flag.BoolVar(&doSimilar, "s", false, "")
	flag.BoolVar(&doSimilar, "similar", false, "List similar / related videos")
	flag.BoolVar(&doTitle, "t", false, "")
	flag.BoolVar(&doTitle, "title", false, "Print video title")
	flag.BoolVar(&doLikes, "l", false, "")
	flag.BoolVar(&doLikes, "likes", false, "Print like count")
	flag.BoolVar(&doDislikes, "D", false, "")
	flag.BoolVar(&doDislikes, "dislikes", false, "Print dislike count")
	flag.BoolVar(&doViews, "n", false, "")
	flag.BoolVar(&doViews, "views", false, "Print view count")
	flag.BoolVar(&doCreator, "r", false, "")
	flag.BoolVar(&doCreator, "creator", false, "Print uploader / creator name")
	flag.BoolVar(&doChannel, "C", false, "")
	flag.BoolVar(&doChannel, "channel", false, "Print channel name + URL")
	flag.BoolVar(&doSubs, "S", false, "")
	flag.BoolVar(&doSubs, "subs", false, "Print subscriber count")
	flag.BoolVar(&doAll, "a", false, "")
	flag.BoolVar(&doAll, "all", false, "Fetch everything")
	flag.BoolVar(&doQuiet, "q", false, "")
	flag.BoolVar(&doQuiet, "quiet", false, "Only output raw values")
	flag.BoolVar(&doHelp, "h", false, "Show help")
	flag.BoolVar(&doHelp, "help", false, "Show help")
	flag.Usage = printHelp
	flag.Parse()

	if doHelp || flag.NArg() == 0 {
		printHelp()
		return
	}

	if doAll {
		doDescription = true
		doGenre = true
		doTranscript = true
		doComments = true
		doThumbnail = true
		doSimilar = true
		doTitle = true
		doLikes = true
		doDislikes = true
		doViews = true
		doCreator = true
		doChannel = true
		doSubs = true
	}

	url := flag.Arg(0)
	m := vidIDRe.FindStringSubmatch(url)
	if m == nil {
		fmt.Fprintln(os.Stderr, "✗ Could not extract video ID from URL:", url)
		os.Exit(1)
	}
	vidID := m[1]

	if !doQuiet {
		fmt.Println("▶ Video ID:", vidID)
		fmt.Println("  URL     :", url)
		fmt.Println()
	}

	needMeta := doDescription || doGenre || doTitle || doLikes || doViews ||
		doCreator || doChannel || doSubs || doSimilar
	var md *meta
	if needMeta {
		if !doQuiet {
			fmt.Println("⟳ Fetching metadata…")
		}
		var err error
		md, err = fetchMeta(url)
		if err != nil {
			fmt.Fprintln(os.Stderr, "✗", err)
			os.Exit(1)
		}
	}

	if doDescription {
		printDescription(md, doQuiet)
	}
	if doGenre {
		printGenre(md, doQuiet)
	}
	if doTitle {
		printTitle(md, doQuiet)
	}
	if doLikes {
		printLikes(md, doQuiet)
	}
	if doDislikes {
		printDislikes(vidID, doQuiet)
	}
	if doViews {
		printViews(md, doQuiet)
	}
	if doCreator {
		printCreator(md, doQuiet)
	}
	if doChannel {
		printChannel(md, doQuiet)
	}
	if doSubs {
		printSubs(md, doQuiet)
	}
	if doTranscript {
		printTranscript(url, vidID, doQuiet)
	}
	if doComments {
		printComments(url, doQuiet)
	}
	if doThumbnail {
		printThumbnail(vidID, doQuiet)
	}
	if doSimilar {
		printSimilar(md, vidID, doQuiet)
	}

	if !doQuiet {
		fmt.Println("✓ Done.")
	}
}
