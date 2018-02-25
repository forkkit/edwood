package frame

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"9fans.net/go/draw"
)

func TestRunIndex(t *testing.T) {

	testvector := []struct {
		thestring string
		arg       int
		want      int
	}{
		{"", 0, 0},
		{"a\x02b", 0, 0},
		{"a\x02b", 1, 1},
		{"a\x02b", 2, 2},
		{"a\x02日本b", 0, 0},
		{"a\x02日本b", 1, 1},
		{"a\x02日本b", 2, 2},
		{"a\x02日本b", 3, 5},
		{"a\x02日本b", 4, 8},
		{"Kröger", 3, 4},
	}

	for _, ps := range testvector {
		b := ps.thestring

		if got, want := runeindex([]byte(b), ps.arg), ps.want; got != want {
			t.Errorf("comparing %#v at %d got %d, want %d", b, ps.arg, got, want)
		}
	}
}

const fixedwidth = 10

// makeBox creates somewhat realistic test boxes in 10pt fixed width font.
func makeBox(s string) *frbox {

	r, _ := utf8.DecodeRuneInString(s)

	switch s {
	case "\t":
		return &frbox{
			Wid:    5000,
			Nrune:  -1,
			Ptr:    []byte(s),
			Bc:     r,
			Minwid: 10,
		}

	case "\n":
		return &frbox{
			Wid:    5000,
			Nrune:  -1,
			Ptr:    []byte(s),
			Bc:     r,
			Minwid: 0,
		}
	default:
		nrune := strings.Count(s, "") - 1
		return &frbox{
			Wid:   fixedwidth * nrune,
			Nrune: nrune,
			Ptr:   []byte(s),
			// Remaining fields not used.
		}
	}
}

type Fakemetrics int

func (w Fakemetrics) BytesWidth(s []byte) int {
	return int(w) * (strings.Count(string(s), "") - 1)
}

func (w Fakemetrics) 	DefaultHeight() int { return 13 }

func (w Fakemetrics) 	Impl() *draw.Font { return nil }

func (w Fakemetrics) 	StringWidth(s string) int { 
	return int(w) * (strings.Count(s, "") - 1)
}

func (w Fakemetrics) 	RunesWidth(r []rune) int {
	return len(r) * int(w)
}

func TestTruncatebox(t *testing.T) {
	frame := &Frame{
		Font: Fakemetrics(fixedwidth),
		nbox:   0,
		nalloc: 0,
	}

	testvector := []struct {
		before string
		after  string
		at     int
	}{
		{"ab", "a", 1},
		{"abc", "a", 2},
		{"a\x02日本b", "a", 4},
	}

	for _, ps := range testvector {
		pb := makeBox(ps.before)
		ab := makeBox(ps.after)

		frame.truncatebox(pb, ps.at)
		if ab.Nrune != pb.Nrune || string(ab.Ptr) != string(pb.Ptr) {
			t.Errorf("truncating %#v (%#v) at %d failed to provide %#v. Gave %#v (%s)\n",
				makeBox(ps.before), ps.before, ps.at, ps.after, pb, string(pb.Ptr))
		}

		if ab.Wid != pb.Wid {
			t.Errorf("wrong width: got %d, want %d for %s", pb.Wid, ab.Wid,  string(pb.Ptr))
		}
	}

}

func TestChopbox(t *testing.T) {
	frame := &Frame{
		Font: Fakemetrics(fixedwidth),
		nbox:   0,
		nalloc: 0,
	}

	testvector := []struct {
		before string
		after  string
		at     int
	}{
		{"ab", "b", 1},
		{"abc", "c", 2},
		{"a\x02日本b", "本b", 3},
	}

	for _, ps := range testvector {
		pb := makeBox(ps.before)
		ab := makeBox(ps.after)

		frame.chopbox(pb, ps.at)
		if ab.Nrune != pb.Nrune || string(ab.Ptr) != string(pb.Ptr) {
			t.Errorf("truncating %#v (%#v) at %d failed to provide %#v. Gave %#v (%s)\n",
				makeBox(ps.before), ps.before, ps.at, ps.after, pb, string(pb.Ptr))
		}

		if ab.Wid != pb.Wid {
			t.Errorf("wrong width: got %d, want %d for %s", pb.Wid, ab.Wid,  string(pb.Ptr))
		}
	}

}

func TestAddbox(t *testing.T) {
	hellobox := makeBox("hi")
	worldbox := makeBox("world")

	comparecore(t, "TestAddbox", []TestStim{
		{
			"empty frame",
			&Frame{
				nbox:   0,
				nalloc: 0,
			},
			r0(func(f *Frame) { f.addbox(0, 1)  }),
			1, 26,
			[]*frbox{},
			0,
			false,
		},
		{
			"one element frame",
			&Frame{
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{hellobox, nil},
			},
			r0(func(f *Frame) { f.addbox(0, 1) }),
			2, 2,
			[]*frbox{hellobox, hellobox},
			0,
			false,
		},
		{
			"two element frame",
			&Frame{
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hellobox, worldbox},
			},
			r0(func(f *Frame) { f.addbox(0, 1) }),
			3, 28,
			[]*frbox{hellobox, hellobox, worldbox},
			0,
			false,
		},
		{
			"two element frame",
			&Frame{
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hellobox, worldbox},
			},
			r0(func(f *Frame) { f.addbox(1, 1) }),
			3, 28,
			[]*frbox{hellobox, worldbox, worldbox},
			0,
			false,
		},
	})
}

type TestStim struct {
	name       string
	frame      *Frame
	stim       func(*Frame) (int, bool)
	nbox       int
	nalloc     int
	afterboxes []*frbox
	result int
	boolresult bool
}

func r0(of  func (*Frame)) (func(*Frame) (int, bool)) {
	return func(f *Frame) (int , bool ){
		of(f)
		return 0, false
	}
}

func r1(of func (*Frame) int) (func (*Frame) (int, bool)) {
	return func(f *Frame) (int , bool ){
		return of(f), false
	}
}

func comparecore(t *testing.T, prefix string, testvector []TestStim) {
	for _, tv := range testvector {
		result, boolresult  := tv.stim(tv.frame)
		if got, want := tv.frame.nbox, tv.nbox; got != want {
			t.Errorf("%s-%s: nbox got %d but want %d\n", prefix, tv.name, got, want)
		}
		if got, want := tv.frame.nalloc, tv.nalloc; got != want {
			t.Errorf("%s-%s: nalloc got %d but want %d\n", prefix, tv.name, got, want)
		}

		if got, want := result, tv.result; got != want {
			t.Errorf("%s-%s: running stim got %d but want %d\n", prefix, tv.name, got, want)
		}
		if got, want := boolresult, tv.boolresult; got != want {
			t.Errorf("%s-%s: running stim bool got %v but want %v\n", prefix, tv.name, got, want)
		}

		if tv.frame.box == nil {
			t.Errorf("%s-%s: ran add but did not succeed in creating boxex", prefix, tv.name)
		}

		// First part of box array must match the provided afterboxes slice.
		for i, _ := range tv.afterboxes {
			if got, want := tv.frame.box[i], tv.afterboxes[i]; !reflect.DeepEqual(got, want) {
				switch {
				case got == nil && want != nil:
					t.Errorf("%s-%s: result box [%d] mismatch: got nil want %#v (%s)", prefix, tv.name, i, want, string(want.Ptr))
				case got != nil && want == nil:
					t.Errorf("%s-%s: result box [%d] mismatch: got %#v (%s) want nil", prefix, tv.name, i, got, string(got.Ptr))
				case got.Ptr == nil && want.Ptr == nil:
					t.Errorf("%s-%s: result box [%d] mismatch: got %#v (nil) want %#v (nil)", prefix, tv.name, i, got, want)
				case got.Ptr == nil && want.Ptr != nil:
					t.Errorf("%s-%s: result box [%d] mismatch: got %#v (nil) want %#v (%s)", prefix, tv.name, i, got, want, string(want.Ptr))
				case want.Ptr == nil && got.Ptr != nil:
					t.Errorf("%s-%s: result box [%d] mismatch: got %#v (%s) want %#v (nil)", prefix, tv.name, i, got, string(got.Ptr), want)
				case want.Ptr != nil && got.Ptr != nil:
					t.Errorf("%s-%s: result box [%d] mismatch: got %#v (%s) want %#v (%s)", prefix, tv.name, i, got, string(got.Ptr), want, string(want.Ptr))
				}
			}
		}

		// Remaining part of box array must merely exist.
		for i, b := range tv.frame.box[len(tv.afterboxes):] {
			if b != nil {
				t.Errorf("%s-%s: result box [%d] should be nil", prefix, tv.name, i+len(tv.afterboxes))
			}
		}
	}
}

func TestFreebox(t *testing.T) {
	hellobox := makeBox("hi")
	worldbox := makeBox("world")

	comparecore(t, "TestFreebox", []TestStim{
		{
			"one element frame",
			&Frame{
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{hellobox, nil},
			},
			r0(func(f *Frame) { f.freebox(0, 0) }),
			1, 2,
			[]*frbox{nil},
			0,
			false,
		},
		{
			"two element frame 0",
			&Frame{
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hellobox, worldbox},
			},
			r0(func(f *Frame) { f.freebox(0, 0) }),
			2, 2,
			[]*frbox{nil, worldbox},
			0,
			false,
		},
		{
			"two element frame 1",
			&Frame{
				nbox:   3,
				nalloc: 3,
				box:    []*frbox{hellobox, worldbox, hellobox},
			},
			r0(func(f *Frame) { f.freebox(1, 1) }),
			3, 3,
			[]*frbox{hellobox, nil, hellobox},
			0,
			false,
		},
	})
}

func TestClosebox(t *testing.T) {
	hellobox := makeBox("hi")
	worldbox := makeBox("world")

	comparecore(t, "TestClosebox", []TestStim{
		{
			"one element frame",
			&Frame{
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{hellobox, nil},
			},
			r0(func(f *Frame) { f.closebox(0, 0) }),
			0, 2,
			[]*frbox{nil},
			0,
			false,
		},
		{
			"two element frame 0",
			&Frame{
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hellobox, worldbox},
			},
			r0(func(f *Frame) { f.closebox(0, 0) }),
			1, 2,
			[]*frbox{worldbox},
			0,
			false,
		},
		{
			"two element frame 1",
			&Frame{
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hellobox, worldbox},
			},
			r0(func(f *Frame) { f.closebox(1, 1) }),
			1, 2,
			[]*frbox{hellobox},
			0,
			false,
		},
		{
			"three element frame",
			&Frame{
				nbox:   3,
				nalloc: 3,
				box:    []*frbox{hellobox, worldbox, hellobox},
			},
			r0(func(f *Frame) { f.closebox(1, 1) }),
			2, 3,
			[]*frbox{hellobox, hellobox},
			0,
			false,
		},
	})
}

func TestDupbox(t *testing.T) {
	hellobox := makeBox("hi")

	stim := []TestStim{
		{
			"one element frame",
			&Frame{
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{hellobox, nil},
			},
			r0(func(f *Frame) { f.dupbox(0) }),
			2, 2,
			[]*frbox{hellobox, hellobox},
			0,
			false,
		},
	}
	comparecore(t, "TestDupbox", stim)

	// Specifically must verify that the box string is different.
	if stim[0].frame.box[0] == stim[0].frame.box[1] {
		t.Errorf("dupbox failed to make a copy of the backing rune string")
	}

}

func TestSplitbox(t *testing.T) {
	hibox := makeBox("hi")
	worldbox := makeBox("world")
	zerobox := makeBox("")

	comparecore(t, "TestSplitbox", []TestStim{
		{
			"one element frame",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{ makeBox("hiworld"), nil},
			},
			r0(func(f *Frame) { f.splitbox(0, 2) }),
			2, 2,
			[]*frbox{ hibox, worldbox },
			0,
			false,
		}, 
		{
			"two element frame 1",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   2,
				nalloc: 3,
				box:    []*frbox{worldbox,  makeBox("hiworld"), nil},
			},
			r0(func(f *Frame) { f.splitbox(1, 2) }),
			3, 3,
			[]*frbox{ worldbox, hibox, worldbox },
			0,
			false,
		},
		{
			"one element 0, 0",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{makeBox("hi"), nil},
			},
			r0(func(f *Frame) { f.splitbox(0, 0) }),
			2, 2,
			[]*frbox{ zerobox, hibox},
			0,
			false,
		},
		{
			"one element 0, 2",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{makeBox("hi"), nil},
			},
			r0(func(f *Frame) { f.splitbox(0, 2) }),
			2, 2,
			[]*frbox{  hibox, zerobox},
			0,
			false,
		},
		{
			"one element 0, 2",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{makeBox("hi"), nil},
			},
			r0(func(f *Frame) { f.splitbox(0, 2) }),
			2, 2,
			[]*frbox{  hibox, zerobox},
			0,
			false,
		},
	})
}

func TestMergebox(t *testing.T) {
	hibox := makeBox("hi")
	worldbox := makeBox("world")
	hiworldbox := makeBox("hiworld")
	zerobox := makeBox("")

	comparecore(t, "TestMergebox", []TestStim{
		{
			"two -> 1",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{ hibox, worldbox},
			},
			r0(func(f *Frame) { f.mergebox(0) }),
			1, 2,
			[]*frbox{ hiworldbox },
			0,
			false,
		}, 
		{
			"two null -> 1",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{ hibox, zerobox},
			},
			r0(func(f *Frame) { f.mergebox(0) }),
			1, 2,
			[]*frbox{ hibox },
			0,
			false,
		}, 
		{
			"three -> 2",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   3,
				nalloc: 3,
				box:    []*frbox{ makeBox("hi"), worldbox, hibox},
			},
			r0(func(f *Frame) { f.mergebox(0) }),
			2, 3,
			[]*frbox{ hiworldbox, hibox },
			0,
			false,
		}, 
		{
			"three -> 1",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   3,
				nalloc: 3,
				box:    []*frbox{ makeBox("hi"), makeBox("world"), makeBox("hi")},
			},
			r0(func(f *Frame) {
				f.mergebox(1)
				f.mergebox(0)
			}),
			1, 3,
			[]*frbox{ makeBox("hiworldhi") },
			0,
			false,
		}, 
	})
}

func TestFindbox(t *testing.T) {
	hibox := makeBox("hi")
	worldbox := makeBox("world")
	hiworldbox := makeBox("hiworld")
//	zerobox := makeBox("")

	comparecore(t, "TestFindbox", []TestStim{
		{
			"find in 1",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				box:    []*frbox{ makeBox("hiworld")},
			},
			r1(func(f *Frame) int { return f.findbox(0, 0, 2) }),
			2, 27,
			[]*frbox{ hibox, worldbox },
			1,
			false,
		}, 
		{
			"find at beginning",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				box:    []*frbox{ makeBox("hiworld")},
			},
			r1(func(f *Frame) int { return f.findbox(0, 0, 0) }),
			1, 1,
			[]*frbox{ hiworldbox },
			0,
			false,
		}, 
		{
			"find at edge",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{ makeBox("hi"), makeBox("world") },
			},
			r1(func(f *Frame) int { return f.findbox(0, 0, 2) }),
			2, 2,
			[]*frbox{ hibox, worldbox },
			1,
			false,
		}, 
		{
			"find continuing",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{ makeBox("hi"), makeBox("world") },
			},
			r1(func(f *Frame) int { return f.findbox(1, 0, 2) }),
			3, 28,
			[]*frbox{ hibox, makeBox("wo"), makeBox("rld") },
			2,
			false,
		}, 
		{
			"find in empty",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   0,
				nalloc: 2,
				box:    []*frbox{ nil, nil },
			},
			r1(func(f *Frame) int { return f.findbox(0, 0, 0) }),
			0, 2,
			[]*frbox{  },
			0,
			false,
		}, 
	})
}
