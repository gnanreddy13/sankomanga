package main

// This file defines the JSON our API sends (see docs/api-contract.md) and
// the functions that build it from AniList's raw data.
//
// Nullable fields are pointers WITHOUT `omitempty`: a nil pointer is
// written as "field": null, so every key is always present, as the
// contract promises.

// searchResponse is the body of GET /api/search.
type searchResponse struct {
	Query       string      `json:"query"`
	Page        int         `json:"page"`
	HasNextPage bool        `json:"hasNextPage"`
	Results     []mangaCard `json:"results"`
}

// mangaCard is one search result.
type mangaCard struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	CoverURL   *string  `json:"coverUrl"`
	CoverColor *string  `json:"coverColor"`
	Year       *int     `json:"year"`
	Format     string   `json:"format"`
	Score      *float64 `json:"score"`
}

// mangaDetail is the body of GET /api/manga/{id}.
type mangaDetail struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Subtitle   *string  `json:"subtitle"`
	Format     string   `json:"format"`
	Status     *string  `json:"status"`
	Year       *int     `json:"year"`
	Chapters   *int     `json:"chapters"`
	Volumes    *int     `json:"volumes"`
	Score      *float64 `json:"score"`
	Genres     []string `json:"genres"`
	Synopsis   []string `json:"synopsis"`
	CoverURL   *string  `json:"coverUrl"`
	CoverColor *string  `json:"coverColor"`
	URL        string   `json:"url"`
}

// showcaseResponse is the body of GET /api/showcase.
type showcaseResponse struct {
	Covers []showcaseCover `json:"covers"`
}

// showcaseCover is one decorative cover on the home page. id and title
// aren't shown today, but they cost nothing and keep the door open for
// making covers clickable later.
type showcaseCover struct {
	ID         int     `json:"id"`
	Title      string  `json:"title"`
	CoverURL   string  `json:"coverUrl"`
	CoverColor *string `json:"coverColor"`
}

// toShowcaseResponse builds the showcase response. Titles without a
// cover image are skipped: a cover is the only thing the showcase shows.
func toShowcaseResponse(popular []anilistMedia) showcaseResponse {
	covers := make([]showcaseCover, 0, len(popular))
	for _, m := range popular {
		if m.CoverImage.Large == nil || *m.CoverImage.Large == "" {
			continue
		}
		covers = append(covers, showcaseCover{
			ID:         m.ID,
			Title:      displayTitle(m.Title),
			CoverURL:   *m.CoverImage.Large,
			CoverColor: m.CoverImage.Color,
		})
	}
	return showcaseResponse{Covers: covers}
}

// toSearchResponse builds the search response from one page of results.
func toSearchResponse(query string, page int, result anilistSearchResult) searchResponse {
	// make(..., 0, n) creates an empty (not nil) list, so the JSON is
	// "results": [] when nothing matched, never "results": null.
	cards := make([]mangaCard, 0, len(result.Media))
	for _, media := range result.Media {
		cards = append(cards, toCard(media))
	}

	return searchResponse{
		Query:       query,
		Page:        page,
		HasNextPage: result.HasNextPage,
		Results:     cards,
	}
}

// toCard builds a search result card. Cards use the smaller cover image
// (coverImage.large) because they're shown small, in a grid.
func toCard(m anilistMedia) mangaCard {
	return mangaCard{
		ID:         m.ID,
		Title:      displayTitle(m.Title),
		CoverURL:   m.CoverImage.Large,
		CoverColor: m.CoverImage.Color,
		Year:       m.StartDate.Year,
		Format:     displayFormat(m.Format),
		Score:      displayScore(m.AverageScore),
	}
}

// toDetail builds a title page. It uses the biggest cover image
// (coverImage.extraLarge) because the title page shows it large.
func toDetail(m anilistMedia) mangaDetail {
	title := displayTitle(m.Title)

	// AniList could send "genres": null; we promise a list.
	genres := m.Genres
	if genres == nil {
		genres = []string{}
	}

	return mangaDetail{
		ID:         m.ID,
		Title:      title,
		Subtitle:   displaySubtitle(m.Title, title),
		Format:     displayFormat(m.Format),
		Status:     displayStatus(m.Status),
		Year:       m.StartDate.Year,
		Chapters:   m.Chapters,
		Volumes:    m.Volumes,
		Score:      displayScore(m.AverageScore),
		Genres:     genres,
		Synopsis:   cleanSynopsis(m.Description),
		CoverURL:   m.CoverImage.ExtraLarge,
		CoverColor: m.CoverImage.Color,
		URL:        m.SiteURL,
	}
}
