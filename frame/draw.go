package frame

import (
	"image"

	"9fans.net/go/draw"

	"log"
)

func (f *Frame) drawtext(pt image.Point, text *draw.Image, back *draw.Image) {
	//	log.Println("DrawText at", pt, "noredraw", f.noredraw, text)
	for nb := 0; nb < f.nbox; nb++ {
		b := f.box[nb]
		f.cklinewrap(&pt, b)
		//		log.Printf("box [%d] %#v pt %v noredraw %v nrune %d\n",  nb, string(b.Ptr), pt, f.noredraw, b.Nrune)

		if !f.noredraw && b.Nrune >= 0 {
			f.Background.Bytes(pt, text, image.ZP, f.Font.Impl(), b.Ptr)
		}
		pt.X += b.Wid
	}
}

// DrawSel repaints a section of the frame, delimited by character
// positions p0 and p1, either with plain background or entirely
// highlighted, according to the flag highlighted, managing the tick
// appropriately. The point pt0 is the geometrical location of p0 on the
// screen; like all of the selection-helper routines' Point arguments, it
// must be a value generated by Ptofchar.
//
// Clarification of semantics: the point of this routine is to redraw the
// state of the Frame with selection p0,p1. In particular, this requires
// updating f.p0 and f.p1 so that other entry points (e.g. Insert) can (transparently) remove
// a pre-existing selection.
//
// Note that the original code does not remove the pre-existing selection.
// I (rjk) claim that this is clearly the wrong semantics. This function should
// arrange for the drawn selection on return to be p0, p1
func (f *Frame) DrawSel(pt image.Point, p0, p1 int, highlighted bool) {
	//	log.Println("------ DrawSel Start -----", p0, p1, f.p0, f.p1)
	//	defer func(f *Frame) {
	//		log.Println("------ DrawSel End ------", f.p0, f.p1)
	//	}(f)

	if p0 > p1 {
		panic("Drawsel0: p0 and p1 must be ordered")
	}

	var back, text *draw.Image
	log.Println("before the f.Tick", f.p0, f.p1)
	if f.ticked {
		f.Tick(f.Ptofchar(f.p0), false)
	}

	if f.p0 != f.p1 {
		// Clear the selection so that subsequent code can
		// update correctly.
		back = f.Cols[ColBack]
		text = f.Cols[ColText]
		f.Drawsel0(f.Ptofchar(f.p0), f.p0, f.p1, back, text)
	}

	if p0 == p1 {
		f.Tick(pt, highlighted)
		f.p0 = p0
		f.p1 = p1
		return
	}

	if highlighted {
		back = f.Cols[ColHigh]
		text = f.Cols[ColHText]
	} else {
		back = f.Cols[ColBack]
		text = f.Cols[ColText]
	}

	f.Drawsel0(pt, p0, p1, back, text)
	f.p0 = p0
	f.p1 = p1
}

// TODO(rjk): This function is convoluted.
// Drawsel0 is a lower-level routine, taking as arguments a background
// color back and text color text. It assumes that the tick is being
// handled (removed beforehand, replaced afterwards, as required) by its
// caller. The selection is delimited by character positions p0 and p1.
// The point pt0 is the geometrical location of p0 on the screen and must
// be a value generated by Ptofchar.
//
// Commentary: this function should conceivably not be part of the public API
//
// Function does not mutate f.p0, f.p1 (well... actually, it does.)
func (f *Frame) Drawsel0(pt image.Point, p0, p1 int, back *draw.Image, text *draw.Image) image.Point {
	//	log.Println("------Drawsel0 Start ------ ", pt, p0, p1, f.p0, f.p1)
	//	defer func(f *Frame) { log.Println("--- Drawsel0 returning ----", f.p0, f.p1  )}(f)
	p := 0
	trim := false
	x := 0
	var w int

	if p0 > p1 {
		panic("Drawsel0: p0 and p1 must be ordered")
	}

	nb := 0
	for ; nb < f.nbox && p < p1; nb++ {
		b := f.box[nb]
		nr := b.Nrune

		// TODO(rjk): There is a method for this I think. Use it.
		if nr < 0 {
			nr = 1
		}
		if p+nr <= p0 {
			// This box doesn't need to be modified.
			p += nr
			continue
		}
		if p >= p0 {
			qt := pt
			// Do we need to call this?
			f.cklinewrap(&pt, b)
			if pt.Y > qt.Y {
				f.Background.Draw(image.Rect(qt.X, qt.Y, f.Rect.Max.X, pt.Y), back, nil, qt)
			}
		}
		ptr := b.Ptr
		if p < p0 {
			// beginning of region: advance into box
			ptr = ptr[runeindex(ptr, p0-p):]
			nr -= int(p0) - p
			p = int(p0)
		}
		log.Println("after advance", nr, p)
		trim = false
		if p+nr > p1 {
			// end of region: trim box
			nr -= (p + nr) - int(p1)
			trim = true
		}

		if b.Nrune < 0 || nr == b.Nrune {
			w = b.Wid
		} else {
			w = f.Font.BytesWidth(ptr[0:runeindex(ptr, nr)])
		}
		x = pt.X + w
		if x > f.Rect.Max.X {
			x = f.Rect.Max.X
		}
		f.Background.Draw(image.Rect(pt.X, pt.Y, x, pt.Y+f.Font.DefaultHeight()), back, nil, pt)
		if b.Nrune >= 0 {
			f.Background.Bytes(pt, text, image.ZP, f.Font.Impl(), ptr[0:runeindex(ptr, nr)])
		}
		pt.X += w
		p += nr
	}

	if p1 > p0 && nb > 0 && nb < f.nbox && f.box[nb-1].Nrune > 0 && !trim {
		qt := pt
		f.cklinewrap(&pt, f.box[nb])
		if pt.Y > qt.Y {
			f.Background.Draw(image.Rect(qt.X, qt.Y, f.Rect.Max.X, pt.Y), back, nil, qt)
		}
	}
	return pt
}

// This function is not part of the documented libframe entrypoints.
// TODO(rjk): discern purpose of this code.
func (f *Frame) Redraw() {
	//	log.Println("Redraw")
	ticked := false
	var pt image.Point

	if f.p0 == f.p1 {
		ticked = f.ticked
		if ticked {
			f.Tick(f.Ptofchar(f.p0), false)
		}
		f.Drawsel0(f.Ptofchar(0), 0, f.nchars, f.Cols[ColBack], f.Cols[ColText])
		if ticked {
			f.Tick(f.Ptofchar(f.p0), true)
		}
	}

	pt = f.Ptofchar(0)
	pt = f.Drawsel0(pt, 0, f.p0, f.Cols[ColBack], f.Cols[ColText])
	pt = f.Drawsel0(pt, f.p0, f.p1, f.Cols[ColHigh], f.Cols[ColHText])
	pt = f.Drawsel0(pt, f.p1, f.nchars, f.Cols[ColBack], f.Cols[ColText])

}

func (f *Frame) tick(pt image.Point, ticked bool) {
	//	log.Println("_tick")
	if f.ticked == ticked || f.tickimage == nil || !pt.In(f.Rect) {
		return
	}

	pt.X -= f.tickscale
	r := image.Rect(pt.X, pt.Y, pt.X+frtickw*f.tickscale, pt.Y+f.Font.DefaultHeight())

	if r.Max.X > f.Rect.Max.X {
		r.Max.X = f.Rect.Max.X
	}

	if ticked {
		f.tickback.Draw(f.tickback.R, f.Background, nil, pt)
		f.Background.Draw(r, f.Display.Black, f.tickimage, image.ZP) // draws an alpha-blended box
	} else {
		// There is an issue with tick management
		f.Background.Draw(r, f.tickback, nil, image.ZP)
	}
	f.ticked = ticked
}

// Tick draws (if up is non-zero) or removes (if up is zero) the tick
// at the screen position indicated by pt.
//
// Commentary: because this code ignores selections, it is conceivably
// undesirable to use it in the public API.
func (f *Frame) Tick(pt image.Point, ticked bool) {
	if f.tickscale != f.Display.ScaleSize(1) {
		if f.ticked {
			f.tick(pt, false)
		}
		f.InitTick()
	}

	f.tick(pt, ticked)
}

func (f *Frame) _draw(pt image.Point) image.Point {
	//	log.Println("_draw")
	for nb := 0; nb < f.nbox; nb++ {
		b := f.box[nb]
		f.cklinewrap0(&pt, b)
		if pt.Y == f.Rect.Max.Y {
			f.nchars -= f.strlen(nb)
			f.delbox(nb, f.nbox-1)
			break
		}

		if b.Nrune > 0 {
			n, fits := f.canfit(pt, b)
			if !fits {
				break
			}
			if n != b.Nrune {
				f.splitbox(nb, n)
				b = f.box[nb]
			}
			pt.X += b.Wid
		} else {
			if b.Bc == '\n' {
				pt.X = f.Rect.Min.X
				pt.Y += f.Font.DefaultHeight()
			} else {
				pt.X += f.newwid(pt, b)
			}
		}
	}
	return pt
}

func (f *Frame) strlen(nb int) int {
	var n int
	for n = 0; nb < f.nbox; nb++ {
		n += nrune(f.box[nb])
	}
	return n
}
