package main

import (
	"fmt"
	"reflect"
	"testing"
)

// ptr returns a pointer to any value. Go has no way to write a pointer to
// a literal like &"text" directly, and tests need lots of them.
func ptr[T any](v T) *T {
	return &v
}

// The descriptions below keep the exact HTML structure AniList sent for
// each title (collected in build step 1), but the story text is shortened
// to a sentence. The tests only care about the messy HTML around it.

func TestCleanSynopsis(t *testing.T) {
	tests := []struct {
		name        string
		description *string
		want        []string
	}{
		{
			name: "Attack on Titan: Source credit followed by a note",
			description: ptr("In this post-apocalyptic story, humanity has been devastated by the Titans…" +
				"\n<br><br>\n(Source: Kodansha USA)\n<br><br>\n<i>Note: Includes two extra chapters.</i>"),
			want: []string{
				"In this post-apocalyptic story, humanity has been devastated by the Titans…",
				"Note: Includes two extra chapters.",
			},
		},
		{
			name: "One Piece: newline, double br, newline, then Source at the end",
			description: ptr("As a child, Monkey D. Luffy was inspired to become a pirate…" +
				"\n<br><br>\n(Source: VIZ Media)"),
			want: []string{
				"As a child, Monkey D. Luffy was inspired to become a pirate…",
			},
		},
		{
			name: "Berserk: notes list with links",
			description: ptr("His name is Guts, the Black Swordsman…<br>\n<br>\n(Source: Dark Horse)<br>\n<br>\n" +
				"<i>Notes:<br>\n- Volumes 1-5 contain the 16 prequel chapters 0A - 0P.<br>\n" +
				"- <a href=\"https://anilist.co/manga/95821/\">Chapter 83</a> was omitted from Volume 13 due to the author’s request.<br>\n" +
				"- Volume 14 includes “<a href=\"https://anilist.co/manga/36408/\">Berserk: The Prototype</a>”.</i>"),
			want: []string{
				"His name is Guts, the Black Swordsman…",
				"Notes:",
				"- Volumes 1-5 contain the 16 prequel chapters 0A - 0P.",
				"- Chapter 83 was omitted from Volume 13 due to the author’s request.",
				"- Volume 14 includes “Berserk: The Prototype”.",
			},
		},
		{
			name: "Look Back: note ends with an unclosed <i> instead of </i>",
			description: ptr("Fujino is a fourth grader who draws a manga strip for the school newspaper…" +
				"\n<br><br>\n(Source: MANGA Plus)<br><br>\n" +
				"<i>Note: The one-shot was compiled in a special volume, released on September 3, 2021.<i>"),
			want: []string{
				"Fujino is a fourth grader who draws a manga strip for the school newspaper…",
				"Note: The one-shot was compiled in a special volume, released on September 3, 2021.",
			},
		},
		{
			name:        "Overlord: no newlines at all",
			description: ptr("What do you do when your favorite game shuts down?…<br><br>(Source: Yen Press)"),
			want: []string{
				"What do you do when your favorite game shuts down?…",
			},
		},
		{
			name: "Oyome ni Oideyo: numbered list and HTML entities",
			description: ptr("A collection of one-shots:<br><br>\n1. Oyome ni Oideyo!<br>\n" +
				"After his sister runs away from her arranged marriage…<br><br>\n" +
				"2. Mr. Valentine!<br><br>\n3. Love&hearts;Bubble&hearts;Bathtime\n"),
			want: []string{
				"A collection of one-shots:",
				"1. Oyome ni Oideyo!",
				"After his sister runs away from her arranged marriage…",
				"2. Mr. Valentine!",
				"3. Love♥Bubble♥Bathtime",
			},
		},
		{
			name: "Oyasumi Punpun: single br as line breaks",
			description: ptr("Meet Punpun Punyama.<br>\nHe wants to win a Nobel Prize and save the world.<br>\n" +
				"That’s what he wants, but what does he get…?\n<br><br>\n(Source: VIZ Media)"),
			want: []string{
				"Meet Punpun Punyama.",
				"He wants to win a Nobel Prize and save the world.",
				"That’s what he wants, but what does he get…?",
			},
		},
		{
			name:        "Karappo na Bokutachi: no Source credit, trailing newline",
			description: ptr("Akira has just run away from home…\n"),
			want:        []string{"Akira has just run away from home…"},
		},
		{
			name:        "Windows line endings and a line break inside a paragraph",
			description: ptr("First paragraph.<br><br>\r\nSecond\r\nparagraph."),
			want:        []string{"First paragraph.", "Second paragraph."},
		},
		{
			name:        "br variants and a lowercase source credit",
			description: ptr("One.<BR />Two.<br/>(source: Someone)"),
			want:        []string{"One.", "Two."},
		},
		{
			name:        "escaped angle bracket survives (tags removed before decoding)",
			description: ptr("I &lt;3 manga &amp; you"),
			want:        []string{"I <3 manga & you"},
		},
		{
			name:        "Source mention that isn't the whole paragraph is kept",
			description: ptr("Based on a story (Source: the author's blog) from 2010."),
			want:        []string{"Based on a story (Source: the author's blog) from 2010."},
		},
		{
			name:        "empty description",
			description: ptr(""),
			want:        []string{},
		},
		{
			name:        "null description",
			description: nil,
			want:        []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanSynopsis(tt.description)
			// reflect.DeepEqual also tells [] apart from nil, which matters
			// because nil would become null in the JSON instead of [].
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestDisplayTitleAndSubtitle(t *testing.T) {
	tests := []struct {
		name         string
		title        anilistTitle
		wantTitle    string
		wantSubtitle *string
	}{
		{
			name:         "Attack on Titan: English differs, so it's the subtitle",
			title:        anilistTitle{Romaji: ptr("Shingeki no Kyojin"), English: ptr("Attack on Titan"), Native: ptr("進撃の巨人")},
			wantTitle:    "Shingeki no Kyojin",
			wantSubtitle: ptr("Attack on Titan"),
		},
		{
			name:         "ONE PIECE: English and native only differ by case, so no subtitle",
			title:        anilistTitle{Romaji: ptr("ONE PIECE"), English: ptr("One Piece"), Native: ptr("ONE PIECE")},
			wantTitle:    "ONE PIECE",
			wantSubtitle: nil,
		},
		{
			name:         "Oyome ni Oideyo: no English title, so native is the subtitle",
			title:        anilistTitle{Romaji: ptr("Oyome ni Oideyo!"), English: nil, Native: ptr("お嫁においでよ！")},
			wantTitle:    "Oyome ni Oideyo!",
			wantSubtitle: ptr("お嫁においでよ！"),
		},
		{
			name:         "Karappo na Bokutachi: romaji only",
			title:        anilistTitle{Romaji: ptr("Karappo na Bokutachi")},
			wantTitle:    "Karappo na Bokutachi",
			wantSubtitle: nil,
		},
		{
			name:         "stray whitespace is trimmed (real AniList data)",
			title:        anilistTitle{Romaji: ptr("Shingeki no Kyojin Volume 0"), English: ptr("Attack on Titan Volume 0\r\n")},
			wantTitle:    "Shingeki no Kyojin Volume 0",
			wantSubtitle: ptr("Attack on Titan Volume 0"),
		},
		{
			name:         "romaji missing: English becomes the title, native the subtitle",
			title:        anilistTitle{Romaji: ptr("  "), English: ptr("Look Back"), Native: ptr("ルックバック")},
			wantTitle:    "Look Back",
			wantSubtitle: ptr("ルックバック"),
		},
		{
			name:         "no titles at all",
			title:        anilistTitle{},
			wantTitle:    "Untitled",
			wantSubtitle: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle := displayTitle(tt.title)
			if gotTitle != tt.wantTitle {
				t.Errorf("title = %q, want %q", gotTitle, tt.wantTitle)
			}

			gotSubtitle := displaySubtitle(tt.title, gotTitle)
			if !reflect.DeepEqual(gotSubtitle, tt.wantSubtitle) {
				t.Errorf("subtitle = %s, want %s", describe(gotSubtitle), describe(tt.wantSubtitle))
			}
		})
	}
}

func TestDisplayScore(t *testing.T) {
	tests := []struct {
		score *int
		want  *float64
	}{
		{ptr(84), ptr(8.4)},
		{ptr(80), ptr(8.0)},
		{ptr(100), ptr(10.0)},
		{ptr(0), ptr(0.0)},
		{nil, nil},
	}
	for _, tt := range tests {
		got := displayScore(tt.score)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("displayScore(%s) = %s, want %s", describe(tt.score), describe(got), describe(tt.want))
		}
	}
}

func TestDisplayStatus(t *testing.T) {
	tests := []struct {
		status *string
		want   *string
	}{
		{ptr("FINISHED"), ptr("Finished")},
		{ptr("RELEASING"), ptr("Ongoing")},
		{ptr("NOT_YET_RELEASED"), ptr("Not yet released")},
		{ptr("CANCELLED"), ptr("Cancelled")},
		{ptr("HIATUS"), ptr("On hiatus")},
		{ptr("SOMETHING_NEW"), nil},
		{nil, nil},
	}
	for _, tt := range tests {
		got := displayStatus(tt.status)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("displayStatus(%s) = %s, want %s", describe(tt.status), describe(got), describe(tt.want))
		}
	}
}

func TestDisplayFormat(t *testing.T) {
	tests := map[string]string{
		"MANGA":    "Manga",
		"ONE_SHOT": "One Shot",
		"NOVEL":    "NOVEL", // never requested, but passed through rather than hidden
	}
	for format, want := range tests {
		if got := displayFormat(format); got != want {
			t.Errorf("displayFormat(%q) = %q, want %q", format, got, want)
		}
	}
}

// describe prints a pointer's value (or "nil") so failure messages show
// "8.4" instead of a memory address like 0xc000012345.
func describe[T any](p *T) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", *p)
}
