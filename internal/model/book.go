package model

import "time"

// Book is a model for a book
type Book struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Author    string    `json:"author"`
	Title     string    `json:"title"`
}

// NewBook creates a new book
func NewBook(title string, author string) *Book {
	return &Book{
		Title:  title,
		Author: author,
	}
}