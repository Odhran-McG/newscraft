package scraper

import (
	"fmt"
	"github.com/gocolly/colly"
	"log"
	"net/http"
	"newscraft/models"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func Run() {

	// There is a feature page for ynews called 'front'
	// https://news.ycombinator.com/front?day=2024-04-30
	// you can pass the date as part of this URL, paginate until all results parsed
	//	Metrics I care for:
	//		upvotes, engagment, keywords e.g. Python, Go
	//		problem on filtering, how to be sure a title with 'go' but not related to the programming language
	//
	//	The problem is low stakes here, so I care more about noise reduction as opposed to missing out on a piece of content
	//	anything truly important will reappear in some other content stream in life, reddit, work, friends etc
	//	if it doesn't then it wasn't vitally important imformation for me
	//
	//	Whilst this was an exercise in webscraping with go, a quick project to get me to up to speed on go syntax,
	//	there is a hackernews API that I could use to make more granular curated lists. https://github.com/HackerNews/API

	// Today's date in UTC
	today := time.Now().UTC().Format("2006-01-02")
	fmt.Println("Today's Date:", today)

	// Initialize collector
	c := colly.NewCollector(
		colly.AllowedDomains("news.ycombinator.com"),
	)

	// Regular expression to extract the number of upvotes
	re := regexp.MustCompile(`(\d+) points`)

	// Slice to store posts
	var posts []models.Post

	// Collect data from each page
	c.OnHTML("tr.athing", func(e *colly.HTMLElement) {
		//	temp post struct
		tempPost := models.Post{
			Title:    e.ChildText("td.title a"),
			URL:      e.ChildAttr("td.title a", "href"),
			Upvotes:  0, // initialise counts to 0, update when scraping
			Comments: 0,
		}

		//	iterate the next row to grab the post score, comments count and post date
		nextRow := e.DOM.Next()

		//	upvotes
		upvotesText := nextRow.Find("span.score").Text()
		upvotes, _ := strconv.Atoi(re.FindStringSubmatch(upvotesText)[1])
		tempPost.Upvotes = upvotes

		//	comments
		// Extract comments count and convert to integer
		commentsText := nextRow.Find("a").Last().Text()

		if commentsText == "discuss" {
			tempPost.Comments = 0 // Set to 0 if text is 'discuss'
		} else if strings.Contains(commentsText, "comment") { // Check if text includes 'comment'

			// clean up text for parsing
			cleanCommentsText := strings.Replace(commentsText, "\u00a0", "", -1)
			cleanCommentsText = strings.TrimSpace(cleanCommentsText)
			commentsText = cleanCommentsText
			// Extract only digits from the comments text
			// Compile a regex to find digits
			digitRegexp := regexp.MustCompile(`\d+`)
			commentCountStr := digitRegexp.FindString(commentsText)

			if commentCount, err := strconv.Atoi(commentCountStr); err == nil {
				tempPost.Comments = commentCount
			} else {
				// Log and handle the error if the conversion fails
				log.Println("Failed to convert comment count to integer:", err)
				tempPost.Comments = 0 // Fallback to 0 if there is an error in conversion
			}
		} else {
			tempPost.Comments = 0 // Fallback case if none of the expected formats are matched
		}

		//	post age
		ageText := nextRow.Find(".age a").Text()
		tempPost.Uploaded = ageText

		posts = append(posts, tempPost)

	})

	// Handle pagination
	c.OnHTML("a.morelink", func(e *colly.HTMLElement) {
		nextURL := e.Attr("href")
		c.Visit(e.Request.AbsoluteURL(nextURL))
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		if r.StatusCode != http.StatusOK {
			fmt.Println("Request failed with status:", r.StatusCode)
		}
		fmt.Println("Something went wrong:", err)
	})

	// Start scraping
	startURL := fmt.Sprintf("https://news.ycombinator.com/front?day=%s", today)
	c.Visit(startURL)

	// Sort posts by upvotes
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Upvotes > posts[j].Upvotes
	})

	// Ensure there are at least 5 posts to avoid slicing panics
	if len(posts) > 5 {
		posts = posts[:5]
		for _, post := range posts {
			fmt.Printf("Title: %s\nLink: %s\nUpvotes: %d \nComments: %d \n\n", post.Title, post.URL, post.Upvotes, post.Comments)
		}
	}
}
