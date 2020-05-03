package ethash

// Package matrix implements a  library for creating and
// manipulating matrices, and performing fuzzy linear algebra.

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
)

type Matrix struct {
	rows, columns int       // the number of rows and columns.
	data          []float64 // the contents of the matrix as one long slice.
}

// Set lets you define the value of a matrix at the given row and
// column.

func (A *Matrix) Set(r int, c int, val float64) {
	A.data[findIndex(r, c, A)] = val
}

// Get retrieves the contents of the matrix at the row and column.

func (A *Matrix) Get(r int, c int) float64 {
	return A.data[findIndex(r, c, A)]
}

// Print converts the matrix into a string and then outputs it to fmt.Printf.

func (A *Matrix) Print() {

	// Find the width (in characters) that each column needs to be.  We hold these
	// widths as strings, not ints, because we're going to use these in a printf
	// function.

	columnWidths := make([]string, A.columns)

	for i := range columnWidths {
		var maxLength int
		thisColumn := A.Column(i + 1)
		for j := range thisColumn {
			thisLength := len(strconv.Itoa(int(thisColumn[j])))
			if thisLength > maxLength {
				maxLength = thisLength

			}
		}
		columnWidths[i] = strconv.Itoa(maxLength)
	}

	for i := 0; i < A.rows; i++ {
		thisRow := A.Row(i + 1)
		fmt.Printf("[")
		for j := range thisRow {
			var printFormat string
			if j == 0 {
				printFormat = "%" + columnWidths[j] + "s"
			} else {
				printFormat = " %" + columnWidths[j] + "s"
			}
			fmt.Printf(printFormat, strconv.Itoa(int(thisRow[j])))

		}
		fmt.Printf("]\n")
	}
}

func (A *Matrix) Print2() {

	// Find the width (in characters) that each column needs to be.  We hold these
	// widths as strings, not ints, because we're going to use these in a printf
	// function.

	columnWidths := make([]string, A.columns)

	for i := range columnWidths {
		var maxLength int
		thisColumn := A.Column(i + 1)
		for j := range thisColumn {
			thisLength := len(strconv.Itoa(int(thisColumn[j])))
			if thisLength > maxLength {
				maxLength = thisLength

			}
		}
		columnWidths[i] = strconv.Itoa(maxLength)
	}

	// We have the widths, so now output each element with the correct column
	// width so that they line up properly.

	for i := 0; i < A.rows; i++ {
		thisRow := A.Row(i + 1)
		fmt.Printf("[")
		for j := range thisRow {
			var printFormat string
			if j == 0 {
				printFormat = "%" + columnWidths[j] + "s"
			} else {
				printFormat = " %" + columnWidths[j] + "s"
			}
			fmt.Printf(printFormat, strconv.FormatFloat(float64(thisRow[j]), 'f', 5, 32))

		}
		fmt.Printf("]\n")
	}
}

func (A *Matrix) ToString() string {

	// Make the martrix to string
	var matrixString string
	matrixString = ""
	for i := 0; i < A.rows; i++ {
		thisRow := A.Row(i + 1)
		for j := range thisRow {
			temp := strconv.FormatFloat(float64(thisRow[j]), 'f', 6, 32)
			matrixString += temp
		}
	}
	return matrixString
}
func (A *Matrix) ToString2() string {

	// Make the martrix to string
	var matrixString string

	for i := 1; i <= A.rows; i++ {
		for j := 1; j <= A.columns; j++ {
			temp := strconv.FormatFloat(float64(A.Get(i, j)), 'f', 6, 32)
			matrixString += temp + " "
		}
	}
	return matrixString
}

// Column returns a slice that represents a column from the matrix.
// This works by examining each row, and adding the nth element of
// each to the column slice.

func (A *Matrix) Column(n int) []float64 {
	col := make([]float64, A.rows)
	for i := 1; i <= A.rows; i++ {
		col[i-1] = A.Row(i)[n-1]
	}
	return col
}

// Row returns a slice that represents a row from the matrix.

func (A *Matrix) Row(n int) []float64 {
	return A.data[findIndex(n, 1, A):findIndex(n, A.columns+1, A)]
}

// Multiply multiplies two matrices together and return the resulting matrix.
// For each element of the result matrix, we get the dot product of the
// corresponding row from matrix A and column from matrix B.

func Multiply(A, B Matrix) *Matrix {
	C := Zeros(A.rows, B.columns)
	for r := 1; r <= C.rows; r++ {
		A_row := A.Row(r)
		for c := 1; c <= C.columns; c++ {
			B_col := B.Column(c)
			C.Set(r, c, dotProduct(A_row, B_col))
		}
	}
	return &C
}

//Matrix 's FUZZY Mulit

func FMultiply(A, B Matrix) Matrix {
	C := Zeros(A.rows, B.columns)
	for r := 1; r <= C.rows; r++ {
		A_row := A.Row(r)
		for c := 1; c <= C.columns; c++ {
			B_col := B.Column(c)
			C.Set(r, c, dotFProduct(A_row, B_col))
		}
	}
	return C
}

//Get specified element from Matrix
// n is Matrix's dimension,p is zero's num
func GetElement(n int, p int, nonce uint64, row int, col int) float64 {
	var element float64 = 0
	var z int = 0
	var digt1, digt2, digt3, digt4 float64
	if row%2 == 0 {
		z = col - row
	} else {
		z = n - (col - row)
	}
	digt1 = intToFloat(z, 1)
	digt2 = intToFloat(n, 2)
	digt3 = intToFloat(p, 3)
	digt4 = uint64ToFloat(nonce, 1)
	element = (digt1 + digt2 + digt3) * digt4
	return element

}

//Change int to float,digt is zero locaton
func intToFloat(n int, digt int) float64 {
	var data float64 = 0
	var data2 float64 = 0
	var tmp int
	data = float64(n)
	tmp = n / 10
	data2 = float64(tmp)
	for data >= (10 / math.Pow10(digt)) {
		data = data / 10
		data2 = data2 / 10
	}
	data = data - data2
	return data
}

func uint64ToFloat(n uint64, digt int) float64 {
	var data float64 = 0
	data = float64(n)
	for data >= (10 / math.Pow10(digt)) {
		data = data / 10
	}
	return data
}

// Add adds two matrices together and returns the resulting matrix.  To do
// this, we just add together the corresponding elements from each matrix.

func Add(A, B Matrix) Matrix {
	C := Zeros(A.rows, A.columns)
	for r := 1; r <= A.rows; r++ {
		for c := 1; c <= A.columns; c++ {
			C.Set(r, c, A.Get(r, c)+B.Get(r, c))
		}
	}
	return C
}

// Identity creates an identity matrix with n rows and n columns.  When you
// multiply any matrix by its corresponding identity matrix, you get the
// original matrix.  The identity matrix looks like a zero-filled matrix with
// a diagonal line of one's starting at the upper left.

func Identity(n int) Matrix {
	A := Zeros(n, n)
	for i := 0; i < len(A.data); i += (n + 1) {
		A.data[i] = 1
	}
	return A
}

// Zeros creates an r x c sized matrix that's filled with zeros.  The initial
// state of an int is 0, so we don't have to do any initialization.

func Zeros(r, c int) Matrix {
	return Matrix{r, c, make([]float64, r*c)}
}

// New creates an r x c sized matrix that is filled with the provided data.
// The matrix data is represented as one long slice.

func InitMatrix(r, c int, data []float64) Matrix {
	if len(data) != r*c {
		panic("[]int data provided to matrix.New is great than the provided capacity of the matrix!'")
	}
	A := Zeros(r, c)
	A.data = data
	return A
}
func InitMatrix2(r, c int) Matrix {

	A := Zeros(r, c)

	return A
}

// findIndex takes a row and column and returns the corresponding index
// from the underlying data slice.

func findIndex(r int, c int, A *Matrix) int {
	return (r-1)*A.columns + (c - 1)
}

// dotProduct calculates the algebraic dot product of two slices.  This is just
// the sum  of the products of corresponding elements in the slices.  We use
// this when we multiply matrices together.

func dotProduct(a, b []float64) float64 {
	var total float64
	for i := 0; i < len(a); i++ {
		total += a[i] * b[i]
	}
	return total
}

// dotProduct calculates the algebraic dot product of two slices.  This is just
// the sum  of the products of corresponding elements in the slices.  We use
// this when we multiply matrices together.

func dotFProduct(a, b []float64) float64 {
	var temp float64 = 0
	for i := 0; i < len(a); i++ {
		temp = Max(temp, Min(a[i], b[i]))
	}
	return temp
}
func Min(x, y float64) float64 {
	if x < y {
		return x
	}
	return y
}
func Max(x, y float64) float64 {
	if x < y {
		return y
	}
	return x
}

func uint64ToBytes(i uint64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}
