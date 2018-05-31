package stringcoding

import (
	bd "github.com/dariodip/prefix-search/prefix-search/bitdata"
	"github.com/golang-collections/go-datastructures/bitarray"
)

type LPRC struct {
	coding                     *Coding
	Epsilon                    float64
	c                          float64
	latestCompressedBitWritten uint64
	strings                    []string
	isUncompressed             *bd.BitData
}

// LPRC (Locality Preserving Rear Coding) is a storage method
// based on RC (Rear Coding) that stores a string s in an
// uncompressed way if the latest c|s| bits do not contain
// an uncompressed string.
func NewLPRC(strings []string, epsilon float64) LPRC {
	stringsCount := uint64(len(strings))
	c := 2.0 + 2.0/epsilon
	return LPRC{New(strings),
		epsilon,
		c, 0,
		strings,
		bd.New(bitarray.NewBitArray(stringsCount), stringsCount)}
}

func calcLen(prefixLen, stringLen uint64) uint64 {
	return stringLen - prefixLen
}

// add adds the string s to the structure
func (lprc *LPRC) add(s string, index uint64) error {
	coding := lprc.coding // extracting our coding data structure

	s = s + string("\x00")
	bdS, errGbd := bd.GetBitData(s) // 1: convert string s to a bitdata bdS
	if errGbd != nil {
		return errGbd
	}
	var stringToAdd *bd.BitData
	if coding.LastString != nil { // this is not the first string
		var errGds error
		stringToAdd, errGds = coding.LastString.GetDifferentSuffix(bdS) // 2: get different suffix
		if errGds != nil {
			return errGds
		}
	} else {
		stringToAdd = bdS // 2b: this is the first string so we cannot have different suffix
	}

	saveUncompressed := saveUncompressed(stringToAdd, bdS, lprc) // should our string be saved uncompressed?
	if saveUncompressed {                                        // we have to save our string uncompressed
		stringToAdd = bdS                                         // so the string to save is the full string
		lprc.latestCompressedBitWritten = uint64(0)               // compressed bit written is now 0
		if err := lprc.isUncompressed.SetBit(index); err != nil { // We can compress s
			return err
		}
	}
	errAppendBit := coding.Strings.AppendBits(stringToAdd) // 3: append string to Strings bitdata
	if errAppendBit != nil {
		panic(errAppendBit) // we don't know if the method has written in the structure
		// so we have to stop all the process and redo... sorry :(
	}

	// 4: append different suffix' length to Lengths
	prefixLen := bdS.Len - stringToAdd.Len // get suffix length
	if coding.LastString != nil {
		errAppUL := coding.encodeEliasGamma(calcLen(prefixLen, coding.LastString.Len))
		if errAppUL != nil { // as above...
			panic(errAppUL)
		}
	}

	errSetSWO := coding.setStartsWithOffset(stringToAdd) // 5: set the bit of the next string in the Starts array
	if errSetSWO != nil {
		panic(errSetSWO)
	}
	coding.LastString = bdS // 6: update last string
	if !saveUncompressed {  // 7: if the string was saved compressed we have to update latestCompressedBitWritten counter
		lprc.latestCompressedBitWritten += stringToAdd.Len
	}
	return nil
}

// Retrieval(u, l) returns the prefix of the string string(u) with length l.
// So the returned prefix ends up in the edge (p(u), u).
func (lprc *LPRC) Retrieval(u uint64, l uint64) (string, error) {

	var (
		stringBuffer = bd.New(bitarray.NewBitArray(l), l) // let's create a buffer in order to store our prefix
	)
	isUncompressedStringU, errIsCompressed := lprc.isUncompressed.GetBit(u) // check if our string is compressed (we hope no)
	if errIsCompressed != nil {                                             // isUncompressed has gone wrong
		return "", errIsCompressed
	}
	if isUncompressedStringU { // our string is stored uncompressed
		if ll, err := lprc.getLengthInStrings(u); err != nil {
			return "", err
		} else { // no error
			if l > ll { // l is greater than our string
				l = ll // we can only return a string as big as our string
			}
		} // end else
		err := lprc.populateBuffer(stringBuffer, l, u, uint64(0)) // we get the first l bits of that string
		if err != nil {
			panic(err)
		}
	} else { // our string is stored compressed
		// we'll do Select1(V, Rank1(V, u))
		v, err := lprc.isUncompressed.Rank1(u) // extract the number of 1s before u in isUncompressed
		if err != nil {                        // i.e. the number of uncompressed strings before u
			return "", err
		}
		vPosition, err := lprc.isUncompressed.Select1(v) // extract the position of the v-th string
		if err != nil {                                  // i.e. the first uncompressed string before u
			return "", err
		}
		vStarts, err := lprc.coding.Starts.Select1(vPosition + 1) // give me the position where the string v starts in Strings
		if err != nil {                                           // where v is the first uncompressed string before u
			return "", err
		}
		vNextStarts, err := lprc.coding.Starts.Select1(vPosition + 1 + 1) // give me the position of the string next to v
		if err != nil {                                                   // in order to extract the size of string(v)
			return "", err
		}
		lengthStringV := vNextStarts - vStarts                   // that's the length of string(v)
		err = lprc.populateBuffer(stringBuffer, l, vPosition, 0) // insert the first l bits of string(v) in the buffer
		if err != nil {
			panic(err)
		}

		for i := vPosition + 1; i <= u; i++ { // for each string i between v and u (we follow the path on the trie in dfs order)
			li, err := lprc.coding.decodeIthEliasGamma(i) // li is the number of bits to remove in string(p(i)) in order to
			if err != nil {                               // obtain the prefix for string(i)
				return "", err
			}
			ni := lengthStringV - li                   // this is the length of the common prefix between string(p(i)) and string(i)
			lengthI, err := lprc.getLengthInStrings(i) // length of the suffix of string(i) in Strings
			lengthStringV = lengthI + ni               // total length of string(i)
			if i == u {                                // we found the last string
				if lengthStringV < l { // we are asking for more bit than the string has
					l = lengthStringV // l should be the total length of the string
				}
			}
			if err != nil {
				return "", err
			}
			if ni >= l {
				continue // first l bits are the same, so we can skip
			} else {
				err := lprc.populateBuffer(stringBuffer, l, i, ni) // populate the buffer
				if err != nil {
					return "", err
				}
			} // end else
		} // end for
	} //end else !isUncompressedStringU
	return stringBuffer.BitToTrimmedString()
}

func (lprc *LPRC) getLengthInStrings(i uint64) (uint64, error) {
	startPositionI, err := lprc.coding.Starts.Select1(i + 1)
	if err != nil {
		return uint64(0), err
	}
	var startPositionSuccI uint64
	if (i + 1) == uint64(len(lprc.strings)) {
		startPositionSuccI = lprc.coding.Starts.Len // u is the last string memorized!
	} else {
		startPositionSuccI, err = lprc.coding.Starts.Select1(i + 1 + 1) // We need to now where the next string starts
		if err != nil {
			return uint64(0), err
		}
	}

	return startPositionSuccI - startPositionI, nil
}

func (lprc *LPRC) populateBuffer(stringBuffer *bd.BitData, l uint64, u uint64, ni uint64) error {
	var uPosition uint64
	if (u + 1) == uint64(len(lprc.strings)) {
		uPosition = lprc.coding.Strings.Len // u is the last string memorized!
	} else {
		var err error
		uPosition, err = lprc.coding.Starts.Select1(u + 1 + 1) // We need to now where the next string starts
		if err != nil {
			return err
		}
	}
	uPosition = uPosition - 1           // The most significant bits are at the end
	for i := uint64(0); i < l-ni; i++ { // let's iterate for i = 0 up to l - 1 (l times)
		// We start from the most significant bits
		lastBit, lastBitErr := lprc.coding.Strings.GetBit(uPosition - i) // take the i-th bit of string(u)
		if lastBitErr != nil {                                           // getBit has gone wrong
			return lastBitErr
		}

		// The last ((l - 1) - ni) bits in stringBuffer are the number of most significant bit in common between
		// the two consecutive strings
		indexToUpdate := ((l - 1) - ni) - i
		if lastBit {
			stringBuffer.SetBit(indexToUpdate)
		} else {
			stringBuffer.ClearBit(indexToUpdate)
		}
	}
	return nil
}

func saveUncompressed(stringToAdd *bd.BitData, bdS *bd.BitData, lprc *LPRC) bool {
	return stringToAdd.Len == bdS.Len || float64(lprc.latestCompressedBitWritten) > lprc.c*float64(bdS.Len)
}

func (lprc *LPRC) run() error {
	for i, s := range lprc.strings {
		if err := lprc.add(s, uint64(i)); err != nil {
			return err
		}
	}
	return nil
}
