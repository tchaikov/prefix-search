package stringcoding

import (
	bd "github.com/dariodip/prefix-search/prefix-search/bitdata"
	"github.com/golang-collections/go-datastructures/bitarray"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Unit test in order to check out if the method getBitData
// works on a string of a single character
func TestGetBitDataSingleChar(t *testing.T) {
	const s1 = "a" // 01100001
	var (
		a       = assert.New(t)
		b1, e1  = bd.GetBitData(s1)
		s1check = []bool{true, false, false, false, false, true, true} // binary for a (reverse)
	)

	a.Nil(e1, "Error in conversion 'a'")
	a.Equal(uint64(8), b1.Len, "Length of 'a' in bits should be 7")
	for i, bitCheck := range s1check {
		check1bit, _ := b1.GetBit(uint64(i))
		a.Equal(bitCheck, check1bit, "Not equals in position: "+string(i))
	}
}

func TestAddAndGetUnaryLength(t *testing.T) {
	a := assert.New(t)
	const (
		n1              = uint64(5)         // length to set
		expectedLength  = uint64(1 + 5 + 1) // first 0 + 5 * '1' + last 0
		expected1sCount = uint64(5)
	)
	var (
		c           = New([]string{"ciaos"})
		onesCounter uint64      // counter of 1s
		lastIndex   = uint64(1) // last index of the array
	)

	t.Log("Add unary value")
	err := c.addUnaryLength(n1)
	a.Nil(err, "Error should be nil")
	t.Log("Get lengths array")
	lengthBitData := c.Lengths
	a.NotNil(lengthBitData, "Lengths array should not be nil")
	a.Equal(expectedLength, lengthBitData.Len, "Len value should be 6 (5 + 1)")
	a.Equal(expectedLength, c.NextLengthsIndex, "NextLengthsIndex should be 6")
	for lastIndex >= 0 {
		bit, err := lengthBitData.GetBit(lastIndex)
		a.Nil(err, "Error should be nil")
		if !bit {
			break
		} else {
			onesCounter++
		}
		lastIndex++
	}
	a.Equal(expected1sCount, onesCounter, "Array contains 5 ones")

	checkUnaryToInt, err := c.unaryToInt(uint64(0))
	a.Nil(err, "Error should be nil")
	a.Equal(expected1sCount, checkUnaryToInt, "Check unary to int should be 5")
}

// Unit test in order to check out if the method getBitData
// works on a string of two characters
func TestGetBitDataTwoChar(t *testing.T) {
	const s2 = "ab" //01100001 01100010
	var (
		a       = assert.New(t)
		b2, e2  = bd.GetBitData(s2)
		s2check = []bool{false, true, false, false, false, true, true, false, true, false, false, false, false,
			true, true} // binary for ab (reverse)
	)

	a.Nil(e2, "Error in conversion 'aa'")
	a.Equal(uint64(16), b2.Len, "Length of 'aa' in bits should be 14")
	for i, bitCheck := range s2check {
		check2bit, _ := b2.GetBit(uint64(i))
		a.Equal(bitCheck, check2bit, "Not equals in position: "+string(i))
	}
}

// Unit test in order to check out if the method GetLengthInBit
// works on a string of two characters
func TestGetLengthInBit(t *testing.T) {
	a := assert.New(t)
	s1 := "ciao" // 4 * 8 bit
	a.Equal(bd.GetLengthInBit(s1), uint64(4*8))

	s2 := "∂iao" // 3 * 8 bit + 24
	a.Equal(bd.GetLengthInBit(s2), uint64(3*8+24))

	s3 := "世iao" // 3 * 8 bit + 24
	a.Equal(bd.GetLengthInBit(s3), uint64(3*8+24))

	c1 := "a"
	a.Equal(bd.GetLengthInBit(c1), uint64(8))
}

// Unit test in order to check out if the method add works
// by inserting 1 or 2 strings
func TestCoding_Add(t *testing.T) {
	const (
		s1      = "cia"
		s2      = "cicccc"
		s3      = "c"
		epsilon = 20 // c = 2.1
	)

	var (
		a           = assert.New(t)
		lprc        = NewLPRC([]string{s1, s2, s3}, epsilon)
		stringsBits = []bool{ // expected Strings final state
			true, false, false, false, false, true, true, false,
			true, false, false, true, false, true, true, false,
			true, true, false, false, false, true, true, false,
			true, true, false, false, false, true, true, false,
			true, true, false, false, false, true, true, false,
			true, true, false, false, false, true, true, false,
			true, true,
			true, true, false, false, false, true, true, false,
		}

		startsBits = []bool{ // expected Starts final state
			true, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			true, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			false, false, false, false, false, false, false, false,
			false, false,
			true, false, false, false, false, false, false, false,
		}
	)

	a.NotEqual(s1, s2, s3, "The three test strings must not be equal")

	err := lprc.add(s1, 0)
	a.Nil(err, "Cannot add string %s. %s", s1, err)
	// Strings and LastString check, since we have only one strings for now we can use getBitData
	s1bits, err := bd.GetBitData(s1)
	a.Nil(err, "Something goes wrong while converting %s in a BitData: %s", s1, err)

	// Checking the behavior when we add the first strings
	for i := uint64(0); i < lprc.coding.Strings.Len; i++ {
		bit, err := lprc.coding.Strings.GetBit(i)
		a.Nil(err, "Cannot access bit %d. %s", i, err)
		expectedBit, err := s1bits.GetBit(i)
		a.Nil(err, "Cannot access bit %d. %s", i, err)
		a.Equal(bit, expectedBit, "Wrong bit at position %d. Found %t, expected %t",
			i, bit, expectedBit)
	}

	a.Equal(s1bits.Len, lprc.coding.Strings.Len, "Wrong len on Strings")

	a.Equal(s1bits, lprc.coding.LastString, "Wrong conversion on LastString, should be equal to %s", s1)
	a.Equal(s1bits.Len, lprc.coding.LastString.Len, "Wrong len on LastString, should be %d", s1bits.Len)

	// Starts check
	bit, err := lprc.coding.Starts.GetBit(0)
	a.Nil(err, "Cannot access bit %d. %s", 0, err)
	a.Equal(bit, true, "Wrong bit at position %d. Found %t, expected %t", 0, bit, true)
	for i := uint64(1); i < lprc.coding.Starts.Len; i++ {
		bit, err := lprc.coding.Starts.GetBit(i)
		a.Nil(err, "Cannot access bit %d. %s", i, err)
		a.Equal(bit, false, "Wrong bit at position %d. Found %t, expected %t", i, bit, false)
	}

	// Lengths
	s1len, err := lprc.coding.unaryToInt(0)
	a.Nil(err, "Something goes wrong: %s", err)
	a.Equal(s1len, bd.GetLengthInBit(s1), "Some bit are missing in Lengths. Found %d, expected %d", s1len,
		len(s1))

	a.Equal(lprc.latestCompressedBitWritten, uint64(0), "The string should not be compressed")

	// Now we check the insertion of more strings
	err = lprc.add(s2, 1)
	a.Nil(err, "Cannot add string %s. %s", s2, err)

	// Check if lastString is now s2
	s2bits, err := bd.GetBitData(s2)
	a.Nil(err, "Something goes wrong while converting %s in a BitData: %s", s1, err)
	a.Equal(s2bits, lprc.coding.LastString, "Wrong conversion on LastString, should be equal to %s",
		s2)
	a.Equal(s2bits.Len, lprc.coding.LastString.Len, "Wrong len on LastString, should be %d", s2bits.Len)

	compressedS2, err := s1bits.GetDifferentSuffix(s2bits)
	a.Nil(err, "Something goes wrong while generating the different suffix between %s and %s: %s",
		s2, s1, err)
	a.Equal(compressedS2.Len, lprc.latestCompressedBitWritten,
		"wrong latest compressed bit written. Found %d, expected %d",
		compressedS2.Len, lprc.latestCompressedBitWritten)

	s2suffixLen, err := lprc.coding.unaryToInt(bd.GetLengthInBit(s1) + 1)
	a.Nil(err, "Something goes wrong: %s", err)
	a.Equal(s2suffixLen, uint64(26), "Some bit are missing in Lengths. Found %d, expected %d",
		s2suffixLen, uint64(26))

	s3bits, err := bd.GetBitData(s3)
	a.Nil(err, "Something goes wrong while converting %s in a BitData: %s", s3, err)

	// Due to the value of c, now s3 should be uncompressed even if it can in theory be completely compressed
	err = lprc.add(s3, 2)
	a.Nil(err, "Cannot add string %s. %s", s3, err)
	a.Equal(s3bits, lprc.coding.LastString, "Wrong conversion on LastString, should be equal to %s", s3)
	a.Equal(s3bits.Len, lprc.coding.LastString.Len, "Wrong len on LastString, should be %d", s3bits.Len)
	a.Equal(lprc.latestCompressedBitWritten, uint64(0), "String %s should be uncompressed", s3)

	s3len, err := lprc.coding.unaryToInt(bd.GetLengthInBit(s1) + 1 + s2suffixLen + 1)
	a.Equal(s3len, s3bits.Len, "Wrong bit len for %s. Found %d, expected %d", s3, s3len, s3bits.Len)

	// Check the structure final state
	// Strings check
	for i := uint64(0); i < lprc.coding.Strings.Len; i++ {
		bit, err := lprc.coding.Strings.GetBit(i)
		a.Nil(err, "Cannot access bit %d. %s", i, err)
		a.Equal(bit, stringsBits[i], "Wrong bit at position %d. Found %t, expected %t",
			i, bit, stringsBits[i])
	}

	// Starts check
	for i := uint64(0); i < lprc.coding.Starts.Len; i++ {
		bit, err := lprc.coding.Starts.GetBit(i)
		a.Nil(err, "Cannot access bit %d. %s", i, err)
		a.Equal(bit, startsBits[i], "Wrong bit at position %d. Found %t, expected %t",
			i, bit, startsBits[i])
	}
}

func TestSetStartsWithOffset(t *testing.T) {
	var (
		a            = assert.New(t)
		c            = New([]string{"ciao"})
		bitD         = bd.New(bitarray.NewBitArray(8), 0)
		checkBit     = []bool{false, true, false}
		expectedBits = []bool{true, false, false, true, false, false}
	)

	for i, bit := range checkBit {
		err := bitD.AppendBit(bit)
		a.Nil(err, "Cannot set bit %d", i)
	}

	err1 := c.setStartsWithOffset(bitD)
	a.Nil(err1, "Something goes wrong %s", err1)
	err2 := c.setStartsWithOffset(bitD)
	a.Nil(err2, "Something goes wrong %s", err2)

	a.Equal(c.Starts.Len, bitD.Len*2, "Lengths are not equal. Found %d, expected %d",
		c.Starts.Len, bitD.Len*2)

	for i := uint64(0); i < c.Starts.Len; i++ {
		bit, err := c.Starts.GetBit(i)
		a.Nil(err, "Cannot access to bit %d", i)
		a.Equal(bit, expectedBits[i], "Wrong bit at position %d. Found %t, expected %t",
			i, bit, expectedBits[i])
	}
}
