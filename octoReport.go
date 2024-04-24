package octoReport

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type Token struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Amount  string `json:"amount"`
}

type Swap struct {
	Receiver   string `json:"receiver"`
	TrxHash    string `json:"trxHash"`
	ExecutedAt string `json:"executedAt"`
	ChainId    int    `json:"chainId"`
	TokenIn    Token  `json:"tokenIn"`
	TokenOut   Token  `json:"tokenOut"`
}

type Swaps struct {
	Swaps []Swap `json:"swaps"`
}

// Public Package Function to create Swap History
func CreateSwapHistory(address string, name string, timestampFrom int64, timestampTo int64) {
	// Sanity check
	if len(address) != 42 {
		log.Fatal("Wrong address length. use 42 byte")
		return
	}

	timeFrom := time.Unix(timestampFrom, 0)

	var timeTo time.Time

	if 0 == timestampTo {
		timeTo = time.Now()
	} else {
		timeTo = time.Unix(timestampTo, 0)
	}

	fmt.Printf("Creating Swap history. address: %s, name: %s, from: %s, to: %s\n",
		address,
		name,
		timeFrom.Format("2006-01-02"),
		timeTo.Format("2006-01-02"))

	var err error

	currentTime := time.Now()
	dateString := currentTime.Format("2006-01-02")

	filenameCsv := fmt.Sprintf("out/swaps/%s_%s.csv", dateString, name)
	filenamePdf := fmt.Sprintf("out/swaps/%s_%s.pdf", dateString, name)

	swaps := getSwapHistory(address)

	createCsv(swaps, filenameCsv)

	// First, we load the CSV data.
	data := loadCSV(filenameCsv)

	// Then we create a new PDF document and write the title and the current date.
	pdf := newReport()

	// After that, we create the table header and fill the table.
	pdf = header(pdf, data[0])
	pdf = table(pdf, data[1:])

	// And we should take the opportunity and beef up our report with a nice logo.
	pdf = image(pdf)

	if pdf.Err() {
		log.Fatalf("Failed creating PDF report: %s\n", pdf.Error())
	}

	// And finally, we write out our finished record to a file.
	err = savePDF(pdf, filenamePdf)
	if err != nil {
		log.Fatalf("Cannot save PDF: %s|n", err)
	}
}

// Public Package Function to create Transaction History
func CreateTransactionHistory(address string, name string, timestampFrom int, timestampTo int) {
	// TODO
}

func createCsv(swaps Swaps, filenameCsv string) {
	file, err := os.Create(filenameCsv)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// write header
	writer.Write([]string{"Time", "Sold Token", "Sold Amount", "Bought Token", "Bought Amount"})

	for _, swap := range swaps.Swaps {
		tokenInAmount, err := strconv.ParseFloat(swap.TokenIn.Amount, 64)
		if err != nil {
			fmt.Println(err)
			return
		}
		tokenOutAmount, err := strconv.ParseFloat(swap.TokenOut.Amount, 64)
		if err != nil {
			fmt.Println(err)
			return
		}
		tokenInAmount = tokenInAmount / 1e18
		tokenOutAmount = tokenOutAmount / 1e18

		executedAtInt, err := strconv.ParseInt(swap.ExecutedAt, 10, 64)
		if err != nil {
			fmt.Println(err)
			return
		}
		executedAtTime := time.Unix(executedAtInt, 0)
		executedAtStr := executedAtTime.Format("2006-01-02 15:04")

		writer.Write([]string{executedAtStr, swap.TokenIn.Symbol, fmt.Sprintf("%.2f", roundToDigits(tokenInAmount, 6)), swap.TokenOut.Symbol, fmt.Sprintf("%.2f", roundToDigits(tokenOutAmount, 6))})
	}
}

func getSwapHistory(address string) (swaps Swaps) {
	resp, err := http.Get("https://api.octodefi.dev/trades/history?wallet=" + address) // replace with your URL
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = json.Unmarshal(body, &swaps)
	if err != nil {
		fmt.Println(err)
		return
	}

	return swaps
}

// Loading a CSV file is no problem for us, we had this last time when dealing with CSV data. We can reuse the `loadCSV()` function unchanged.
func loadCSV(path string) [][]string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Cannot open '%s': %s\n", path, err.Error())
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		log.Fatalln("Cannot read CSV data:", err.Error())
	}
	return rows
}

// ## The Initial PDF document

// Next, we create a new PDF document.
func newReport() *gofpdf.Fpdf {
	// The package provides a function named `New()` to create a PDF document with
	//
	// * landscape ("L") or portrait ("P") orientation,
	// * the unit used for expressing lengths and sizes ("mm"),
	// * the paper format ("Letter"), and
	// * the path to a font directory.
	//
	// All of these can remain empty, in which case `New()` provides suitable defaults.
	//
	// Function `New()` returns an object of type `*gofpdf.Fpdf` that
	// provides a number of methods for filling the document.
	pdf := gofpdf.New("L", "mm", "Letter", "")

	// We start by adding a new page to the document.
	pdf.AddPage()

	// Now we set the font to "Times", the style to "bold", and the size to 28 points.
	pdf.SetFont("Times", "B", 28)

	// Then we write a text cell of length 40 and height 10. There are no
	// starting coordinates used here; instead, the `Cell()` method moves
	// the current position to the end of the cell so that the next call
	// to `Cell()` continues after the previous cell.
	pdf.Cell(40, 10, "Swap History")

	// The `Ln()` function moves the current position to a new line, with
	// an optional line height parameter.
	pdf.Ln(12)

	pdf.SetFont("Times", "", 20)
	pdf.Cell(40, 10, time.Now().Format("Mon Jan 2, 2006"))
	pdf.Ln(20)

	return pdf
}

/* ### How Cell() and Ln() advance the output position

As mentioned in the comments, the `Cell()` method takes no coordinates. Instead, the PDF document maintains the current output position internally, and advances it to the right by the length of the cell being written.

Method `Ln()` moves the output position back to the left border and down by the provided value. (Passing `-1` uses the height of the recently written cell.)

HYPE[pdf](pdf.html)
*/

// ## The Table Header: Formatted Cells

// Having created the initial document, we can now create the table header.
// This time, we generate a formatted cell with a light grey as the
// background color.
func header(pdf *gofpdf.Fpdf, hdr []string) *gofpdf.Fpdf {
	pdf.SetFont("Times", "B", 16)
	pdf.SetFillColor(240, 240, 240)
	for _, str := range hdr {
		// The `CellFormat()` method takes a couple of parameters to format
		// the cell. We make use of this to create a visible border around
		// the cell, and to enable the background fill.
		pdf.CellFormat(44, 7, str, "1", 0, "", true, 0, "")
	}

	// Passing `-1` to `Ln()` uses the height of the last printed cell as
	// the line height.
	pdf.Ln(-1)
	return pdf
}

// ## The Table Body

// In the same fashion, we can create the table body.

func table(pdf *gofpdf.Fpdf, tbl [][]string) *gofpdf.Fpdf {
	// Reset font and fill color.
	pdf.SetFont("Times", "", 16)
	pdf.SetFillColor(255, 255, 255)

	// Every column gets aligned according to its contents.
	align := []string{"L", "C", "L", "C", "L"}
	for _, line := range tbl {
		for i, str := range line {
			// Again, we need the `CellFormat()` method to create a visible
			// border around the cell. We also use the `alignStr` parameter
			// here to print the cell content either left-aligned or
			// right-aligned.
			pdf.CellFormat(44, 7, str, "1", 0, align[i], false, 0, "")
		}
		pdf.Ln(-1)
	}
	return pdf
}

// ## The Image

// Next, let's not forget to impress our boss by adding a fancy image.
func image(pdf *gofpdf.Fpdf) *gofpdf.Fpdf {
	// The `ImageOptions` method takes a file path, x, y, width, and height
	// parameters, and an `ImageOptions` struct to specify a couple of options.
	pdf.ImageOptions("assets/octodefi_logo.png", 220, 10, 27, 22, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	return pdf
}

// ## Saving The Document
//
// Finally, the convenience method `OutputFileAndClose()` lets us save the
// finished document.
func savePDF(pdf *gofpdf.Fpdf, filenamePdf string) error {
	return pdf.OutputFileAndClose(filenamePdf)
}

func roundToDigits(input float64, significantDigits int) float64 {
	if input == 0 {
		return 0
	}
	d := float64(significantDigits) - math.Floor(math.Log10(math.Abs(input))) - 1
	return math.Round(input*math.Pow(10, d)) / math.Pow(10, d)
}
