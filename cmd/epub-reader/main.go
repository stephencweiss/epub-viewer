package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"epub-reader/pkg/analysis"
	"epub-reader/pkg/epub"
	"epub-reader/pkg/markdown"
	"epub-reader/pkg/storage"
)

var (
	dbPath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "epub-reader",
		Short: "EPUB reader and analyzer",
		Long:  "A tool for reading, analyzing, and comparing EPUB files.",
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", storage.DefaultDBPath(), "Database path")

	// Add commands
	rootCmd.AddCommand(
		analyzeCmd(),
		addCmd(),
		listCmd(),
		removeCmd(),
		exportCmd(),
		authorsCmd(),
		compareCmd(),
		infoCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// analyzeCmd analyzes an EPUB file without storing it.
func analyzeCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "analyze <epub-file>",
		Short: "Analyze an EPUB file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			// Parse EPUB
			book, err := epub.Parse(path)
			if err != nil {
				return fmt.Errorf("failed to parse EPUB: %w", err)
			}

			fmt.Printf("Title: %s\n", book.Title)
			fmt.Printf("Author: %s\n", strings.Join(book.Authors, ", "))
			fmt.Printf("Chapters: %d\n", book.ChapterCount())
			fmt.Println()

			// Analyze
			analyzer := analysis.NewAnalyzer()
			result := analyzer.AnalyzeBook(book)

			// Print results
			fmt.Println("=== Word Analysis ===")
			fmt.Printf("Total words: %d\n", result.TotalWords)
			fmt.Printf("Unique words: %d\n", result.UniqueWords)
			fmt.Printf("Vocabulary richness: %.4f\n", result.VocabularyRich)
			fmt.Printf("Hapax legomena: %d\n", result.HapaxLegomena)
			fmt.Printf("Average word length: %.2f chars\n", result.AverageWordLen)
			fmt.Println()

			fmt.Println("=== Sentence Analysis ===")
			fmt.Printf("Total sentences: %d\n", result.TotalSentences)
			fmt.Printf("Average sentence length: %.2f words\n", result.AvgSentenceLen)
			fmt.Println()

			fmt.Println("=== Paragraph Analysis ===")
			fmt.Printf("Total paragraphs: %d\n", result.TotalParagraphs)
			fmt.Printf("Average paragraph length: %.2f sentences\n", result.AvgParagraphLen)
			fmt.Println()

			fmt.Println("=== Style Analysis ===")
			fmt.Printf("Readability score: %.2f (%s)\n", result.ReadabilityScore, analysis.ReadabilityLevel(result.ReadabilityScore))
			fmt.Printf("Dialogue ratio: %.2f%%\n", result.DialogueRatio*100)
			fmt.Printf("Average syllables per word: %.2f\n", result.AvgSyllablesWord)
			fmt.Printf("Point of view: %s\n", analysis.DetectPOVStyle(book.FullText()))

			if verbose && len(result.TopWords) > 0 {
				fmt.Println()
				fmt.Println("=== Top Words ===")
				for i, wf := range result.TopWords {
					if i >= 20 {
						break
					}
					fmt.Printf("%3d. %-20s %d\n", i+1, wf.Word, wf.Count)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output including top words")
	return cmd
}

// addCmd adds an EPUB to the library.
func addCmd() *cobra.Command {
	var authorName string

	cmd := &cobra.Command{
		Use:   "add <epub-file>",
		Short: "Add an EPUB to the library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			// Parse EPUB
			book, err := epub.Parse(path)
			if err != nil {
				return fmt.Errorf("failed to parse EPUB: %w", err)
			}

			// Open database
			store, err := storage.NewSQLiteStore(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer store.Close()

			// Check if book already exists
			if existing, _ := store.GetBookByPath(path); existing != nil {
				return fmt.Errorf("book already in library (ID: %d)", existing.ID)
			}

			// Determine author
			if authorName == "" && len(book.Authors) > 0 {
				authorName = book.Authors[0]
			}
			if authorName == "" {
				authorName = "Unknown"
			}

			// Check for existing author
			var authorID int64
			existingAuthor, err := store.GetAuthorByName(authorName)
			if err == storage.ErrNotFound {
				// Check for similar authors
				similar, _ := store.FindSimilarAuthors(authorName)
				if len(similar) > 0 {
					fmt.Printf("Found similar authors:\n")
					for i, a := range similar {
						fmt.Printf("  %d. %s\n", i+1, a.Name)
					}
					fmt.Printf("  %d. Create new author '%s'\n", len(similar)+1, authorName)
					fmt.Print("Select option (default: create new): ")

					var choice int
					fmt.Scanln(&choice)
					if choice >= 1 && choice <= len(similar) {
						authorID = similar[choice-1].ID
						authorName = similar[choice-1].Name
					}
				}

				if authorID == 0 {
					// Create new author
					author, err := store.CreateAuthor(authorName)
					if err != nil {
						return fmt.Errorf("failed to create author: %w", err)
					}
					authorID = author.ID
					fmt.Printf("Created new author: %s\n", authorName)
				}
			} else if err != nil {
				return fmt.Errorf("failed to lookup author: %w", err)
			} else {
				authorID = existingAuthor.ID
				fmt.Printf("Using existing author: %s\n", existingAuthor.Name)
			}

			// Add book
			storedBook, err := store.AddBook(authorID, book.Title, path, book.Language, book.Publisher)
			if err != nil {
				return fmt.Errorf("failed to add book: %w", err)
			}

			fmt.Printf("Added book: %s (ID: %d)\n", book.Title, storedBook.ID)

			// Analyze and store
			fmt.Print("Analyzing... ")
			analyzer := analysis.NewAnalyzer()
			result := analyzer.AnalyzeBook(book)

			stored := storage.FromAnalysis(storedBook.ID, result)
			if err := store.SaveAnalysis(stored); err != nil {
				return fmt.Errorf("failed to save analysis: %w", err)
			}
			fmt.Println("done")

			fmt.Printf("  Words: %d | Unique: %d | Readability: %.1f\n",
				result.TotalWords, result.UniqueWords, result.ReadabilityScore)

			return nil
		},
	}

	cmd.Flags().StringVar(&authorName, "author", "", "Override author name")
	return cmd
}

// listCmd lists books in the library.
func listCmd() *cobra.Command {
	var authorFilter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List books in the library",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storage.NewSQLiteStore(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer store.Close()

			var books []storage.Book

			if authorFilter != "" {
				author, err := store.GetAuthorByName(authorFilter)
				if err == storage.ErrNotFound {
					// Try finding similar
					similar, _ := store.FindSimilarAuthors(authorFilter)
					if len(similar) == 0 {
						return fmt.Errorf("author not found: %s", authorFilter)
					}
					author = &similar[0]
				} else if err != nil {
					return err
				}

				books, err = store.ListBooksByAuthor(author.ID)
				if err != nil {
					return err
				}
			} else {
				books, err = store.ListBooks()
				if err != nil {
					return err
				}
			}

			if len(books) == 0 {
				fmt.Println("No books in library")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tTitle\tAuthor\tWords")
			fmt.Fprintln(w, "--\t-----\t------\t-----")

			for _, book := range books {
				words := "-"
				if a, err := store.GetAnalysis(book.ID); err == nil {
					words = fmt.Sprintf("%d", a.TotalWords)
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", book.ID, book.Title, book.AuthorName, words)
			}
			w.Flush()

			return nil
		},
	}

	cmd.Flags().StringVar(&authorFilter, "author", "", "Filter by author name")
	return cmd
}

// removeCmd removes a book from the library.
func removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <book-id>",
		Short: "Remove a book from the library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var id int64
			if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
				return fmt.Errorf("invalid book ID: %s", args[0])
			}

			store, err := storage.NewSQLiteStore(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer store.Close()

			// Get book info for confirmation
			book, err := store.GetBook(id)
			if err == storage.ErrNotFound {
				return fmt.Errorf("book not found: %d", id)
			}
			if err != nil {
				return err
			}

			if err := store.RemoveBook(id); err != nil {
				return err
			}

			fmt.Printf("Removed: %s by %s\n", book.Title, book.AuthorName)
			return nil
		},
	}
}

// exportCmd exports an EPUB to Markdown.
func exportCmd() *cobra.Command {
	var singleFile bool

	cmd := &cobra.Command{
		Use:   "export <epub-file> [output-dir]",
		Short: "Export EPUB to Markdown",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			// Determine output directory
			outputDir := "."
			if len(args) > 1 {
				outputDir = args[1]
			}

			// Parse EPUB
			book, err := epub.Parse(path)
			if err != nil {
				return fmt.Errorf("failed to parse EPUB: %w", err)
			}

			conv := markdown.NewConverter()

			if singleFile {
				// Export as single file
				md, err := conv.ConvertBook(book)
				if err != nil {
					return fmt.Errorf("failed to convert: %w", err)
				}

				// Generate filename from title
				filename := sanitizeFilename(book.Title) + ".md"
				outputPath := filepath.Join(outputDir, filename)

				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return err
				}

				if err := os.WriteFile(outputPath, []byte(md), 0644); err != nil {
					return err
				}

				fmt.Printf("Exported to: %s\n", outputPath)
			} else {
				// Export each chapter as separate file
				bookDir := filepath.Join(outputDir, sanitizeFilename(book.Title))
				if err := os.MkdirAll(bookDir, 0755); err != nil {
					return err
				}

				for i, chapter := range book.Chapters {
					md, err := conv.ConvertChapter(chapter)
					if err != nil {
						return fmt.Errorf("failed to convert chapter %d: %w", i, err)
					}

					filename := fmt.Sprintf("%02d", i+1)
					if chapter.Title != "" {
						filename += "-" + sanitizeFilename(chapter.Title)
					}
					filename += ".md"

					outputPath := filepath.Join(bookDir, filename)
					if err := os.WriteFile(outputPath, []byte(md), 0644); err != nil {
						return err
					}
				}

				fmt.Printf("Exported %d chapters to: %s/\n", len(book.Chapters), bookDir)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&singleFile, "single", false, "Export as single file instead of separate chapters")
	return cmd
}

// authorsCmd lists all authors.
func authorsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "authors",
		Short: "List all authors",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storage.NewSQLiteStore(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer store.Close()

			authors, err := store.ListAuthors()
			if err != nil {
				return err
			}

			if len(authors) == 0 {
				fmt.Println("No authors in library")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tName\tBooks\tTotal Words")
			fmt.Fprintln(w, "--\t----\t-----\t-----------")

			for _, author := range authors {
				corpus, _ := store.GetCorpusAnalysis(author.ID)
				books := 0
				words := 0
				if corpus != nil {
					books = corpus.BookCount
					words = corpus.TotalWords
				}
				fmt.Fprintf(w, "%d\t%s\t%d\t%d\n", author.ID, author.Name, books, words)
			}
			w.Flush()

			return nil
		},
	}
}

// compareCmd compares authors' writing styles.
func compareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "compare <author1> <author2>",
		Short: "Compare two authors' writing styles",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storage.NewSQLiteStore(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer store.Close()

			// Find authors
			author1, err := findAuthor(store, args[0])
			if err != nil {
				return fmt.Errorf("author 1: %w", err)
			}

			author2, err := findAuthor(store, args[1])
			if err != nil {
				return fmt.Errorf("author 2: %w", err)
			}

			// Get corpus analyses
			corpus1, err := store.GetCorpusAnalysis(author1.ID)
			if err != nil {
				return fmt.Errorf("failed to get corpus for %s: %w", author1.Name, err)
			}

			corpus2, err := store.GetCorpusAnalysis(author2.ID)
			if err != nil {
				return fmt.Errorf("failed to get corpus for %s: %w", author2.Name, err)
			}

			// Print comparison
			fmt.Printf("Comparing: %s vs %s\n", author1.Name, author2.Name)
			fmt.Println(strings.Repeat("=", 50))

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "Metric\t%s\t%s\n", author1.Name, author2.Name)
			fmt.Fprintln(w, "------\t------\t------")
			fmt.Fprintf(w, "Books\t%d\t%d\n", corpus1.BookCount, corpus2.BookCount)
			fmt.Fprintf(w, "Total Words\t%d\t%d\n", corpus1.TotalWords, corpus2.TotalWords)
			fmt.Fprintf(w, "Vocabulary Richness\t%.4f\t%.4f\n", corpus1.AvgVocabularyRich, corpus2.AvgVocabularyRich)
			fmt.Fprintf(w, "Readability\t%.1f\t%.1f\n", corpus1.AvgReadability, corpus2.AvgReadability)
			fmt.Fprintf(w, "Avg Sentence Length\t%.1f\t%.1f\n", corpus1.AvgSentenceLen, corpus2.AvgSentenceLen)
			fmt.Fprintf(w, "Avg Word Length\t%.2f\t%.2f\n", corpus1.AvgWordLen, corpus2.AvgWordLen)
			fmt.Fprintf(w, "Dialogue Ratio\t%.1f%%\t%.1f%%\n", corpus1.AvgDialogueRatio*100, corpus2.AvgDialogueRatio*100)
			w.Flush()

			return nil
		},
	}
}

// infoCmd shows detailed info about a book.
func infoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <book-id>",
		Short: "Show detailed info about a stored book",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var id int64
			if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
				return fmt.Errorf("invalid book ID: %s", args[0])
			}

			store, err := storage.NewSQLiteStore(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer store.Close()

			book, err := store.GetBook(id)
			if err == storage.ErrNotFound {
				return fmt.Errorf("book not found: %d", id)
			}
			if err != nil {
				return err
			}

			fmt.Printf("ID: %d\n", book.ID)
			fmt.Printf("Title: %s\n", book.Title)
			fmt.Printf("Author: %s\n", book.AuthorName)
			fmt.Printf("Path: %s\n", book.Path)
			fmt.Printf("Language: %s\n", book.Language)
			fmt.Printf("Publisher: %s\n", book.Publisher)
			fmt.Printf("Added: %s\n", book.AddedAt.Format("2006-01-02 15:04"))

			analysis, err := store.GetAnalysis(book.ID)
			if err == nil {
				fmt.Println()
				fmt.Println("=== Analysis ===")
				fmt.Printf("Words: %d\n", analysis.TotalWords)
				fmt.Printf("Unique words: %d\n", analysis.UniqueWords)
				fmt.Printf("Vocabulary richness: %.4f\n", analysis.VocabularyRich)
				fmt.Printf("Sentences: %d\n", analysis.TotalSentences)
				fmt.Printf("Avg sentence length: %.2f words\n", analysis.AvgSentenceLen)
				fmt.Printf("Readability: %.1f\n", analysis.ReadabilityScore)
			}

			return nil
		},
	}
}

// findAuthor finds an author by name or similar.
func findAuthor(store *storage.SQLiteStore, name string) (*storage.Author, error) {
	author, err := store.GetAuthorByName(name)
	if err == nil {
		return author, nil
	}

	if err != storage.ErrNotFound {
		return nil, err
	}

	// Try similar
	similar, _ := store.FindSimilarAuthors(name)
	if len(similar) == 1 {
		return &similar[0], nil
	}
	if len(similar) > 1 {
		return nil, fmt.Errorf("multiple authors match '%s', be more specific", name)
	}

	return nil, fmt.Errorf("author not found: %s", name)
}

// sanitizeFilename makes a string safe for use as a filename.
func sanitizeFilename(s string) string {
	// Replace problematic characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	s = replacer.Replace(s)

	// Truncate if too long
	if len(s) > 100 {
		s = s[:100]
	}

	return strings.TrimSpace(s)
}
