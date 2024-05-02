package models

// Post represents a Hacker News post
type Post struct {
	Title    string
	URL      string
	Upvotes  int
	Comments int
	Uploaded string
}
