package main

import (
	"os"
	"reflect"
	"testing"
)

// ── expandArgs ───────────────────────────────────────────────────────────────

func TestExpandArgs_AlreadySingleFlags(t *testing.T) {
	in := []string{"-d", "-c"}
	got := expandArgs(in)
	if !reflect.DeepEqual(got, in) {
		t.Errorf("got %v, want %v", got, in)
	}
}

func TestExpandArgs_CombinedBoolFlags(t *testing.T) {
	got := expandArgs([]string{"-dc"})
	want := []string{"-d", "-c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExpandArgs_ThreeCombined(t *testing.T) {
	got := expandArgs([]string{"-dct"})
	want := []string{"-d", "-c", "-t"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExpandArgs_ValueFlagNotSplit(t *testing.T) {
	// -b takes a value, so -bsafari must not be split
	in := []string{"-bsafari"}
	got := expandArgs(in)
	if !reflect.DeepEqual(got, in) {
		t.Errorf("value flag -bsafari should not be split, got %v", got)
	}
}

func TestExpandArgs_LongFlagUnchanged(t *testing.T) {
	in := []string{"--description"}
	got := expandArgs(in)
	if !reflect.DeepEqual(got, in) {
		t.Errorf("long flag should be unchanged, got %v", got)
	}
}

func TestExpandArgs_MixedCombinedAndLong(t *testing.T) {
	got := expandArgs([]string{"-dc", "--quiet"})
	want := []string{"-d", "-c", "--quiet"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExpandArgs_PositionalArgUnchanged(t *testing.T) {
	in := []string{"-d", "https://example.com"}
	got := expandArgs(in)
	if !reflect.DeepEqual(got, in) {
		t.Errorf("positional arg should be unchanged, got %v", got)
	}
}

func TestExpandArgs_Empty(t *testing.T) {
	got := expandArgs(nil)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestExpandArgs_AllBoolFlags(t *testing.T) {
	got := expandArgs([]string{"-dgTcpstlDnrCSaqh"})
	if len(got) != len("dgTcpstlDnrCSaqh") {
		t.Errorf("expected %d flags, got %v", len("dgTcpstlDnrCSaqh"), got)
	}
	for _, f := range got {
		if len(f) != 2 || f[0] != '-' {
			t.Errorf("expected single-char flag, got %q", f)
		}
	}
}

// ── cookieArgs ───────────────────────────────────────────────────────────────

func TestCookieArgs_BrowserOnly(t *testing.T) {
	got := cookieArgs("firefox", "")
	want := []string{"--cookies-from-browser", "firefox"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCookieArgs_FileOnly(t *testing.T) {
	got := cookieArgs("", "/tmp/cookies.txt")
	want := []string{"--cookies", "/tmp/cookies.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCookieArgs_Both(t *testing.T) {
	got := cookieArgs("chrome", "/tmp/cookies.txt")
	want := []string{"--cookies-from-browser", "chrome", "--cookies", "/tmp/cookies.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCookieArgs_NeitherReturnsEmpty(t *testing.T) {
	got := cookieArgs("", "")
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

// ── int64Val ─────────────────────────────────────────────────────────────────

func TestInt64Val_WithValue(t *testing.T) {
	n := int64(12345)
	if got := int64Val(&n); got != "12345" {
		t.Errorf("got %q, want %q", got, "12345")
	}
}

func TestInt64Val_Zero(t *testing.T) {
	n := int64(0)
	if got := int64Val(&n); got != "0" {
		t.Errorf("got %q, want %q", got, "0")
	}
}

func TestInt64Val_Nil(t *testing.T) {
	if got := int64Val(nil); got != "unavailable" {
		t.Errorf("got %q, want \"unavailable\"", got)
	}
}

// ── parseSubtitle ────────────────────────────────────────────────────────────

func TestParseSubtitle_VTT(t *testing.T) {
	content := "WEBVTT\n\n00:00:01.000 --> 00:00:03.000\nHello world\n\n00:00:04.000 --> 00:00:06.000\nThis is a test\n"
	f, err := os.CreateTemp(t.TempDir(), "*.vtt")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()

	got := parseSubtitle(f.Name())
	want := []string{"Hello world", "This is a test"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseSubtitle_StripsNOTELines(t *testing.T) {
	content := "WEBVTT\n\nNOTE this is a comment\n\n00:00:01.000 --> 00:00:02.000\nActual text\n"
	f, err := os.CreateTemp(t.TempDir(), "*.vtt")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()

	got := parseSubtitle(f.Name())
	want := []string{"Actual text"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseSubtitle_SRT(t *testing.T) {
	content := "1\n00:00:01,000 --> 00:00:03,000\nHello SRT\n\n2\n00:00:04,000 --> 00:00:06,000\nSecond line\n"
	f, err := os.CreateTemp(t.TempDir(), "*.srt")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(content)
	f.Close()

	got := parseSubtitle(f.Name())
	want := []string{"Hello SRT", "Second line"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseSubtitle_MissingFileReturnsNil(t *testing.T) {
	got := parseSubtitle("/tmp/no-such-subtitle-file.vtt")
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestParseSubtitle_EmptyFileReturnsNil(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.vtt")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	got := parseSubtitle(f.Name())
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}
