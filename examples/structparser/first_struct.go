package structs

import (
	"context"

	"github.com/wricardo/code-surgeon/examples/structparser/other"
)

// this is here just test that we don't care about it, just structs
var someVariable string

// this is here just test that we don't care about it, just structs
func someFunction() {
}

type SpecialString string

type SomeFunc func(a string) error

// FirstStruct this is the comment for the first struct.
// This is new line.
type FirstStruct struct {
	Int                                  int  `json:"int" bson:"int"`
	Int8                                 int8 `bson:"int8"`
	Int16                                int16
	Int32                                int32
	Int64                                int64
	Uint                                 uint
	Uintptr                              uintptr
	Uint8                                uint8
	Uint16                               uint16
	Uint32                               uint32
	Uint64                               uint64
	Float32                              float32
	Float64                              float64
	Complex64                            complex64
	Complex128                           complex128
	Byte                                 byte
	Rune                                 rune
	String                               string
	SpecialString                        SpecialString
	SecondStruct                         SecondStruct
	ArrayInt                             [3]int
	SliceString                          []string
	SlicePointerString                   []*string
	PointerSliceString                   *[]string
	PointerSlicePointerString            *[]*string
	ChanString                           chan string
	RChanString                          <-chan string
	SChanString                          chan<- string
	MapStringString                      map[string]string
	MapPointerStringString               map[*string]string
	MapPointerStringPointerString        map[*string]*string
	PointerMapStringString               *map[string]string
	PointerMapPointerStringPointerString *map[*string]*string
	Func                                 SomeFunc
	PointerFunc                          *SomeFunc
	MapStringSliceString                 map[string][]string
	MapStringSlicePointerString          map[string][]*string
	MapPointerStringSlicePointerString   map[*string][]*string
	MapChanPointerStringStruct           map[chan *string]SecondStruct
	PackageStruct                        other.Struct
	PointerPackageStruct                 *other.Struct
	SlicePointerPackageStruct            []*other.Struct
	MapStringPackageStruct               map[string]other.Struct
	ChanPackagePointerStruct             chan *other.Struct
}

// MyTestMethod this is the comment for the MyTestMethod method.
func (s *FirstStruct) MyTestMethod(ctx context.Context, x, y []string, z int) (a, b string, c int) {
	return "", "", 0
}

// CommentsAndDocs this is the comment for the CommentsAndDocs struct.
type CommentsAndDocs struct {
	// this is line 1 of comment 001
	SingleDoc int

	// this is line 1 of comment 001
	// this is line 2 of comment 002
	MultiLineDoc int

	// this is line 1 of comment 003
	// this is line 2 of comment 004
	MixedSpacesDoc int

	// this is line 1 of comment 005
	/* this is line 2 of comment 006 */
	MixedTypesDoc int

	// this is line 1 of comment 007
	DocAndComment int // comment 008

	// this should be ignored

	CommentNoSpaces int // comment abc

	/* this is line 1 of comment 009*/
	StarDoc int /*comment 010 */

	/* this is line 1 of comment 010*/
	CommentWithTag int `json:"comment_with_tag"` // comment 11

	// 001
	//002
	//   003
	/*004*/
	/* 005*/
	/*006 */
	/* 007 */
	/** 008 **/
	/*   009   */
	CrazyDoc int
}
