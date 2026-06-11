package content

import (
	"strings"
	"testing"
)

func TestImageURL(t *testing.T) {
	cases := []struct {
		image, slug, want string
	}{
		{"", "my-post", ""},
		{"cover.png", "my-post", "/blog-images/my-post/cover.png"},
		{"/og-blog.png", "my-post", "/og-blog.png"},
		{"https://cdn.example.com/x.png", "my-post", "https://cdn.example.com/x.png"},
	}
	for _, c := range cases {
		if got := imageURL(c.image, c.slug); got != c.want {
			t.Errorf("imageURL(%q, %q) = %q, want %q", c.image, c.slug, got, c.want)
		}
	}
}

func TestRewriteImageSrcs(t *testing.T) {
	cases := []struct {
		name, html, want string
	}{
		{
			"relative src rewritten",
			`<p><img src="photo.png" alt="x"></p>`,
			`<p><img src="/blog-images/my-post/photo.png" alt="x"></p>`,
		},
		{
			"absolute path untouched",
			`<img src="/avatar.jpg" alt="">`,
			`<img src="/avatar.jpg" alt="">`,
		},
		{
			"external url untouched",
			`<img src="https://example.com/a.png" alt="">`,
			`<img src="https://example.com/a.png" alt="">`,
		},
		{
			"multiple images",
			`<img src="a.png" alt=""><img src="https://x.com/b.png" alt=""><img src="c.jpg" alt="">`,
			`<img src="/blog-images/my-post/a.png" alt=""><img src="https://x.com/b.png" alt=""><img src="/blog-images/my-post/c.jpg" alt="">`,
		},
		{
			"no images",
			`<p>hello</p>`,
			`<p>hello</p>`,
		},
	}
	for _, c := range cases {
		if got := rewriteImageSrcs(c.html, "my-post"); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestParsePostWithImage(t *testing.T) {
	data := []byte(`---
title: Test
date: 2026-06-11
image: cover.png
---

![diagram](arch.png)
`)
	post, err := parsePost(data, "test-post")
	if err != nil {
		t.Fatalf("parsePost: %v", err)
	}
	if post.Image != "/blog-images/test-post/cover.png" {
		t.Errorf("Image = %q, want /blog-images/test-post/cover.png", post.Image)
	}
	want := `src="/blog-images/test-post/arch.png"`
	if !strings.Contains(post.Content, want) {
		t.Errorf("Content missing %q, got: %s", want, post.Content)
	}
}
