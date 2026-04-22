package main

import "testing"

func TestParseMDURL(t *testing.T) {
	cases := []struct {
		input   string
		desc    string
		rawURL  string
		ok      bool
	}{
		{
			input:  "[Hello World](https://example.com)",
			desc:   "Hello World",
			rawURL: "https://example.com",
			ok:     true,
		},
		{
			input:  "[蛇と蜘蛛 [中文字幕] - Hanime1.me](https://hanime1.me/watch?v=404983)",
			desc:   "蛇と蜘蛛 [中文字幕] - Hanime1.me",
			rawURL: "https://hanime1.me/watch?v=404983",
			ok:     true,
		},
		{
			input:  "[[outer [inner] text]](https://example.com)",
			desc:   "[outer [inner] text]",
			rawURL: "https://example.com",
			ok:     true,
		},
		{
			input:  "[no url]()",
			desc:   "no url",
			rawURL: "",
			ok:     true,
		},
		// not a markdown link
		{input: "https://example.com", ok: false},
		{input: "", ok: false},
		// missing closing paren
		{input: "[desc](https://example.com", ok: false},
		// missing opening bracket on URL
		{input: "[desc]https://example.com", ok: false},
		// unmatched bracket in desc
		{input: "[desc(https://example.com)", ok: false},
	}

	for _, tc := range cases {
		desc, rawURL, ok := parseMDURL(tc.input)
		if ok != tc.ok {
			t.Errorf("parseMDURL(%q): ok=%v, want %v", tc.input, ok, tc.ok)
			continue
		}
		if !ok {
			continue
		}
		if desc != tc.desc {
			t.Errorf("parseMDURL(%q): desc=%q, want %q", tc.input, desc, tc.desc)
		}
		if rawURL != tc.rawURL {
			t.Errorf("parseMDURL(%q): rawURL=%q, want %q", tc.input, rawURL, tc.rawURL)
		}
	}
}
